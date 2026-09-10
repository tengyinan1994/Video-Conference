package sys

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"hotgo/addons/conference/consts"
	"hotgo/addons/conference/model/entity"
	"hotgo/addons/conference/model/input/sysin"
	"hotgo/addons/conference/service"
	"hotgo/internal/library/cache"
	"hotgo/internal/library/contexts"
	iservice "hotgo/internal/service"

	"github.com/gogf/gf/v2/database/gdb"
	"github.com/gogf/gf/v2/encoding/gjson"
	"github.com/gogf/gf/v2/errors/gerror"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gtime"
	"github.com/gogf/gf/v2/util/gconv"
	"github.com/livekit/protocol/livekit"
	lksdk "github.com/livekit/server-sdk-go/v2"
)

type sSysMeeting struct{}

func NewSysMeeting() *sSysMeeting {
	return &sSysMeeting{}
}

func init() {
	service.RegisterSysMeeting(NewSysMeeting())
}

func meetingModel(ctx context.Context) *gdb.Model {
	return g.DB().Model(consts.MeetingTable).Safe().Ctx(ctx)
}

func (s *sSysMeeting) Create(ctx context.Context, in *sysin.MeetingCreateInp) (res *sysin.MeetingItemModel, err error) {
	if err = in.Filter(ctx); err != nil {
		return
	}
	user := contexts.GetUser(ctx)
	if user == nil || user.Id <= 0 {
		return nil, gerror.New("请先登录")
	}
	if err = service.SysMeetingType().AssertExists(ctx, in.TypeId); err != nil {
		return
	}

	hostId := in.HostId
	hostName := in.HostName
	if hostId <= 0 {
		hostId = user.Id
	}
	if hostName == "" {
		if hostId == user.Id {
			hostName = displayName(user.Username, user.RealName)
		} else {
			hostName = fmt.Sprintf("用户%d", hostId)
		}
	}

	now := gtime.Now()
	status := consts.MeetingStatusScheduled
	if !in.StartAt.After(now) {
		status = consts.MeetingStatusOngoing
	}

	roomName, err := generateRoomName()
	if err != nil {
		return nil, err
	}
	shareCode, err := generateShareCode()
	if err != nil {
		return nil, err
	}

	data := g.Map{
		"title":          in.Title,
		"room_name":      roomName,
		"host_id":        hostId,
		"host_name":      hostName,
		"start_at":       in.StartAt,
		"end_at":         in.EndAt,
		"status":         status,
		"share_code":     shareCode,
		"created_by":     user.Id,
		"created_at":     now,
		"updated_at":     now,
		"record_enabled": boolToTiny(in.RecordEnabled),
		"type_id":        in.TypeId,
	}
	id, err := meetingModel(ctx).Data(data).InsertAndGetId()
	if err != nil {
		return nil, gerror.Wrap(err, "创建会议室失败")
	}

	m := &entity.Meeting{}
	if err = meetingModel(ctx).Where("id", id).Scan(m); err != nil {
		return nil, gerror.Wrap(err, "读取会议室失败")
	}
	res = toMeetingItem(m, user.Id)
	attachMeetingTypeNames(ctx, []*sysin.MeetingItemModel{res})
	return
}

func (s *sSysMeeting) List(ctx context.Context, in *sysin.MeetingListInp) (list []*sysin.MeetingItemModel, err error) {
	if err = in.Filter(ctx); err != nil {
		return
	}
	user := contexts.GetUser(ctx)
	userId := int64(0)
	if user != nil {
		userId = user.Id
	}

	now := gtime.Now()

	mod := meetingModel(ctx)
	switch in.Tab {
	case consts.MeetingListTabEnded:
		mod = mod.WhereIn("status", g.Slice{consts.MeetingStatusEnded, consts.MeetingStatusReleased})
	case consts.MeetingListTabOngoing:
		mod = mod.WhereNotIn("status", g.Slice{consts.MeetingStatusEnded, consts.MeetingStatusReleased}).
			Where("(status = ? OR start_at <= ? OR started_at IS NOT NULL)", consts.MeetingStatusOngoing, now)
	case consts.MeetingListTabScheduled:
		mod = mod.WhereNotIn("status", g.Slice{consts.MeetingStatusEnded, consts.MeetingStatusReleased}).
			WhereNot("status", consts.MeetingStatusOngoing).
			Where("start_at > ?", now).
			WhereNull("started_at")
	default:
		// all：表中全部会议（删除为硬删，不会出现在此）
	}

	var rows []*entity.Meeting
	if err = mod.OrderDesc("start_at").Scan(&rows); err != nil {
		return nil, gerror.Wrap(err, "查询会议室失败")
	}
	list = make([]*sysin.MeetingItemModel, 0, len(rows))
	for _, m := range rows {
		if m == nil {
			continue
		}
		item := toMeetingItem(m, userId)
		list = append(list, item)
	}
	attachMeetingTypeNames(ctx, list)
	attachMeetingRecordings(ctx, list)
	attachMeetingMinutes(ctx, list)
	return
}

func (s *sSysMeeting) Release(ctx context.Context, in *sysin.MeetingReleaseInp) (res *sysin.MeetingReleaseModel, err error) {
	if err = in.Filter(ctx); err != nil {
		return
	}
	user := contexts.GetUser(ctx)
	if user == nil || user.Id <= 0 {
		return nil, gerror.New("请先登录")
	}

	var m *entity.Meeting
	if err = meetingModel(ctx).Where("id", in.Id).Scan(&m); err != nil {
		return nil, gerror.Wrap(err, "查询会议室失败")
	}
	if m == nil {
		return nil, gerror.New("会议室不存在")
	}
	if isEndedStatus(m.Status) {
		return &sysin.MeetingReleaseModel{Minutes: minutesSnapshot(ctx, m.Id)}, nil
	}
	if m.HostId != user.Id && !iservice.AdminMember().VerifySuperId(ctx, user.Id) {
		return nil, gerror.New("仅主持人可结束会议室")
	}
	if err = s.endMeeting(ctx, m, true); err != nil {
		return nil, err
	}
	return &sysin.MeetingReleaseModel{Minutes: minutesSnapshot(ctx, m.Id)}, nil
}

func (s *sSysMeeting) Delete(ctx context.Context, in *sysin.MeetingDeleteInp) (err error) {
	if err = in.Filter(ctx); err != nil {
		return
	}
	user := contexts.GetUser(ctx)
	if user == nil || user.Id <= 0 {
		return gerror.New("请先登录")
	}

	var m *entity.Meeting
	if err = meetingModel(ctx).Where("id", in.Id).Scan(&m); err != nil {
		return gerror.Wrap(err, "查询会议室失败")
	}
	if m == nil {
		return nil
	}
	if m.HostId != user.Id && !iservice.AdminMember().VerifySuperId(ctx, user.Id) {
		return gerror.New("仅主持人可删除会议室")
	}
	if isEndedStatus(m.Status) {
		return gerror.New("已结束的会议请在管理后台删除")
	}

	if _, err = meetingModel(ctx).Where("id", m.Id).Delete(); err != nil {
		return gerror.Wrap(err, "删除会议室失败")
	}
	service.SysRecording().StopAllForMeeting(ctx, m.Id, m.RoomName)
	service.SysRecording().StopAllAiForMeeting(ctx, m.Id, m.RoomName)
	s.cleanupLiveKitRoom(ctx, m.RoomName)
	return nil
}

func (s *sSysMeeting) Update(ctx context.Context, in *sysin.MeetingUpdateInp) (res *sysin.MeetingItemModel, err error) {
	if err = in.Filter(ctx); err != nil {
		return
	}
	user := contexts.GetUser(ctx)
	if user == nil || user.Id <= 0 {
		return nil, gerror.New("请先登录")
	}

	var m *entity.Meeting
	if err = meetingModel(ctx).Where("id", in.Id).Scan(&m); err != nil {
		return nil, gerror.Wrap(err, "查询会议室失败")
	}
	if m == nil {
		return nil, gerror.New("会议室不存在")
	}
	if m.HostId != user.Id && !iservice.AdminMember().VerifySuperId(ctx, user.Id) {
		return nil, gerror.New("仅主持人可修改会议室")
	}

	item := toMeetingItem(m, user.Id)
	if item.Tab != consts.MeetingListTabScheduled {
		return nil, gerror.New("进行中或已结束的会议不可修改")
	}

	if in.TypeId != nil {
		if err = service.SysMeetingType().AssertExists(ctx, *in.TypeId); err != nil {
			return
		}
	}

	now := gtime.Now()
	data := g.Map{
		"title":      in.Title,
		"start_at":   in.StartAt,
		"end_at":     in.EndAt,
		"updated_at": now,
	}
	if in.RecordEnabled != nil {
		data["record_enabled"] = boolToTiny(*in.RecordEnabled)
	}
	if in.TypeId != nil {
		data["type_id"] = *in.TypeId
	}
	if _, err = meetingModel(ctx).Where("id", m.Id).Data(data).Update(); err != nil {
		return nil, gerror.Wrap(err, "更新会议室失败")
	}
	m.Title = in.Title
	m.StartAt = in.StartAt
	m.EndAt = in.EndAt
	m.UpdatedAt = now
	if in.RecordEnabled != nil {
		m.RecordEnabled = boolToTiny(*in.RecordEnabled)
	}
	if in.TypeId != nil {
		m.TypeId = *in.TypeId
	}
	res = toMeetingItem(m, user.Id)
	attachMeetingTypeNames(ctx, []*sysin.MeetingItemModel{res})
	return
}

func (s *sSysMeeting) ShareView(ctx context.Context, in *sysin.MeetingShareViewInp) (res *sysin.MeetingShareViewModel, err error) {
	if err = in.Filter(ctx); err != nil {
		return
	}
	m, err := s.GetByShareCode(ctx, in.Code)
	if err != nil {
		return
	}
	if m == nil {
		return nil, gerror.New("会议不存在或链接无效")
	}
	item := toMeetingItem(m, 0)
	canJoin := item.Tab != consts.MeetingListTabEnded && !isBeforeJoinWindow(m)
	res = &sysin.MeetingShareViewModel{
		Title:     m.Title,
		RoomName:  m.RoomName,
		HostName:  m.HostName,
		StartAt:   m.StartAt,
		EndAt:     m.EndAt,
		Status:    item.Status,
		ShareCode: m.ShareCode,
		CanJoin:   canJoin,
	}
	return
}

func (s *sSysMeeting) GetByShareCode(ctx context.Context, code string) (m *entity.Meeting, err error) {
	code = strings.TrimSpace(code)
	if code == "" {
		return nil, nil
	}
	if err = meetingModel(ctx).Where("share_code", code).Scan(&m); err != nil {
		return nil, gerror.Wrap(err, "查询会议室失败")
	}
	return
}

func (s *sSysMeeting) GetByRoomName(ctx context.Context, room string) (m *entity.Meeting, err error) {
	room = strings.TrimSpace(room)
	if room == "" {
		return nil, nil
	}
	if err = meetingModel(ctx).Where("room_name", room).Scan(&m); err != nil {
		return nil, gerror.Wrap(err, "查询会议室失败")
	}
	return
}

func (s *sSysMeeting) AssertJoinable(ctx context.Context, m *entity.Meeting) error {
	if m == nil {
		return gerror.New("会议室不存在")
	}
	if isEndedStatus(m.Status) {
		return gerror.New("会议室已结束，无法加入")
	}
	if isBeforeJoinWindow(m) {
		return gerror.New("会议尚未开始")
	}
	return nil
}

// AutoReleaseExpired 定时：已到预定结束时间的未结束会议，若会议室无人则标记为已结束；若仍有人则继续计时。
func (s *sSysMeeting) AutoReleaseExpired(ctx context.Context) (count int, err error) {
	var rows []*entity.Meeting
	if err = meetingModel(ctx).
		WhereNotIn("status", g.Slice{consts.MeetingStatusEnded, consts.MeetingStatusReleased}).
		WhereLTE("end_at", gtime.Now()).
		Scan(&rows); err != nil {
		return 0, gerror.Wrap(err, "扫描过期会议室失败")
	}
	if len(rows) == 0 {
		return 0, nil
	}

	client, _, err := newRoomServiceClient(ctx)
	if err != nil {
		return 0, gerror.Wrap(err, "获取 LiveKit 客户端失败")
	}

	for _, m := range rows {
		if m == nil {
			continue
		}
		empty, checkErr := roomEmpty(ctx, client, m.RoomName)
		if checkErr != nil {
			g.Log().Warningf(ctx, "auto release check room empty meeting id=%d room=%s err=%+v", m.Id, m.RoomName, checkErr)
			continue
		}
		if !empty {
			// 到点后会议室仍有人 → 继续计时，不结束
			continue
		}
		if endErr := s.endMeeting(ctx, m, false); endErr != nil {
			g.Log().Warningf(ctx, "auto end meeting id=%d err=%+v", m.Id, endErr)
			continue
		}
		count++
	}
	return
}

// roomEmpty 判断会议室是否无真人参会者（忽略录制 Egress，仅 Egress 在房视为空）
func roomEmpty(ctx context.Context, client *lksdk.RoomServiceClient, room string) (bool, error) {
	if strings.TrimSpace(room) == "" {
		return true, nil
	}
	participants, err := listRoomParticipants(ctx, client, room)
	if err != nil {
		return false, err
	}
	return !roomHasHumanParticipant(participants), nil
}

// roomHasHumanParticipant 参会者中是否存在真人（忽略录制 Egress）
func roomHasHumanParticipant(participants []*livekit.ParticipantInfo) bool {
	for _, p := range participants {
		if p == nil {
			continue
		}
		if isEgressIdentity(p.Identity) || isEgressIdentity(p.Name) {
			continue
		}
		return true
	}
	return false
}

func (s *sSysMeeting) endMeeting(ctx context.Context, m *entity.Meeting, syncEndAt bool) error {
	now := gtime.Now()
	data := g.Map{
		"status":      consts.MeetingStatusEnded,
		"released_at": now,
		"updated_at":  now,
	}
	if syncEndAt {
		data["end_at"] = now
	}
	_, err := meetingModel(ctx).Where("id", m.Id).Data(data).Update()
	if err != nil {
		return gerror.Wrap(err, "更新会议室状态失败")
	}
	service.SysRecording().StopAllForMeeting(ctx, m.Id, m.RoomName)
	service.SysRecording().StopAllAiForMeeting(ctx, m.Id, m.RoomName)
	if enqErr := service.SysMinutes().TryEnqueue(ctx, m.Id, false); enqErr != nil {
		g.Log().Warningf(ctx, "conference enqueue minutes on end meeting=%d err=%+v", m.Id, enqErr)
	}
	// 等 Egress 收尾再删房，避免 AI/回放仍 STARTING 时被 DeleteRoom 打成 Start signal not received
	roomName := m.RoomName
	meetingId := m.Id
	go s.cleanupLiveKitRoomAfterEgress(context.Background(), meetingId, roomName)
	return nil
}

func (s *sSysMeeting) cleanupLiveKitRoomAfterEgress(ctx context.Context, meetingId int64, roomName string) {
	deadline := time.Now().Add(20 * time.Second)
	for time.Now().Before(deadline) {
		if !hasInFlightRecordingAnyPurpose(ctx, meetingId) {
			break
		}
		time.Sleep(500 * time.Millisecond)
	}
	s.cleanupLiveKitRoom(ctx, roomName)
}

func (s *sSysMeeting) cleanupLiveKitRoom(ctx context.Context, roomName string) {
	if roomName == "" {
		return
	}
	client, _, lkErr := newRoomServiceClient(ctx)
	if lkErr == nil {
		_, _ = client.DeleteRoom(ctx, &livekit.DeleteRoomRequest{Room: roomName})
	}
	_, _ = cache.Instance().Remove(ctx, consts.HostCachePrefix+roomName)
}

func isEndedStatus(status string) bool {
	return status == consts.MeetingStatusEnded || status == consts.MeetingStatusReleased
}

func minutesSnapshot(ctx context.Context, meetingId int64) *sysin.MinutesModel {
	if meetingId <= 0 {
		return nil
	}
	byID, err := service.SysMinutes().ListByMeetingIDs(ctx, []int64{meetingId})
	if err != nil {
		g.Log().Warningf(ctx, "minutes snapshot meeting=%d err=%+v", meetingId, err)
		return nil
	}
	return byID[meetingId]
}

// meetingOngoing 会议是否处于进行中：已有首个真人进房（started_at 非空），
// 或已到预定开始时间（start_at <= now）。用于展示状态推导与列表分区判断。
func meetingOngoing(m *entity.Meeting, now *gtime.Time) bool {
	if m == nil || isEndedStatus(m.Status) {
		return false
	}
	if m.Status == consts.MeetingStatusOngoing {
		return true
	}
	if m.StartedAt != nil {
		return true
	}
	return m.StartAt != nil && !m.StartAt.After(now)
}

func toMeetingItem(m *entity.Meeting, userId int64) *sysin.MeetingItemModel {
	now := gtime.Now()
	tab := consts.MeetingListTabScheduled
	status := m.Status

	if isEndedStatus(status) {
		tab = consts.MeetingListTabEnded
		status = consts.MeetingStatusEnded
	} else if meetingOngoing(m, now) {
		tab = consts.MeetingListTabOngoing
		status = consts.MeetingStatusOngoing
	} else {
		status = consts.MeetingStatusScheduled
	}

	return &sysin.MeetingItemModel{
		Id:            m.Id,
		Title:         m.Title,
		RoomName:      m.RoomName,
		HostId:        m.HostId,
		HostName:      m.HostName,
		StartAt:       m.StartAt,
		ActualStartAt: m.StartedAt,
		EndAt:         m.EndAt,
		Status:        status,
		ShareCode:     m.ShareCode,
		ShareUrl:      "/join/" + m.ShareCode,
		IsHost:        userId > 0 && m.HostId == userId,
		Tab:           tab,
		Attendees:     attendeesFromJSON(m.Attendees),
		RecordEnabled: m.RecordEnabled != 0,
		TypeId:        m.TypeId,
	}
}

func boolToTiny(v bool) int {
	if v {
		return 1
	}
	return 0
}

func attendeesFromJSON(j *gjson.Json) []string {
	if j == nil || j.IsNil() {
		return []string{}
	}
	var names []string
	if err := j.Scan(&names); err != nil || names == nil {
		return []string{}
	}
	out := make([]string, 0, len(names))
	for _, n := range names {
		n = strings.TrimSpace(n)
		if n == "" || isEgressIdentity(n) {
			continue
		}
		out = append(out, n)
	}
	return out
}

// AppendAttendee 进房时把显示名去重追加到会议 attendees。
// 首个参会者早于预定开始时间进房时，顺带记录 started_at 作为会议实际开始时间：
// 会议计时起点 = started_at（有人提前入会）或 start_at（无人提前入会，到点即开始）。
func (s *sSysMeeting) AppendAttendee(ctx context.Context, roomName, displayName string) (err error) {
	roomName = strings.TrimSpace(roomName)
	displayName = strings.TrimSpace(displayName)
	if roomName == "" || displayName == "" {
		return nil
	}
	if isEgressIdentity(displayName) {
		return nil
	}

	return g.DB().Transaction(ctx, func(ctx context.Context, tx gdb.TX) error {
		var m *entity.Meeting
		if err := tx.Model(consts.MeetingTable).Ctx(ctx).
			Where("room_name", roomName).
			LockUpdate().
			Scan(&m); err != nil {
			return gerror.Wrap(err, "查询会议室失败")
		}
		if m == nil {
			return nil
		}

		names := attendeesFromJSON(m.Attendees)
		for _, n := range names {
			if n == displayName {
				return nil
			}
		}
		names = append(names, displayName)

		data := g.Map{
			"attendees":  gjson.New(names),
			"updated_at": gtime.Now(),
		}
		now := gtime.Now()
		if len(names) == 1 && m.StartedAt == nil && !isEndedStatus(m.Status) {
			// 首个真人进房即视为进行中：即使早于预定开始时间，状态也立刻变为 ongoing
			data["status"] = consts.MeetingStatusOngoing
			m.Status = consts.MeetingStatusOngoing
			if m.StartAt != nil && m.StartAt.After(now) {
				// 首个参会者提前入会：从此刻开始累计会议时长（其余参会者以此时间为计时起点）
				data["started_at"] = now
				m.StartedAt = now
			}
		}
		if _, err := tx.Model(consts.MeetingTable).Ctx(ctx).
			Where("id", m.Id).
			Data(data).
			Update(); err != nil {
			return gerror.Wrap(err, "更新参会名单失败")
		}
		return nil
	})
}

// isBeforeJoinWindow 是否尚未进入可入会窗口（开始前 MeetingEarlyJoinMinutes 分钟）
func isBeforeJoinWindow(m *entity.Meeting) bool {
	if m == nil || m.StartAt == nil {
		return false
	}
	openAt := m.StartAt.Add(-time.Duration(consts.MeetingEarlyJoinMinutes) * time.Minute)
	return gtime.Now().Before(openAt)
}

func displayName(username, realName string) string {
	realName = strings.TrimSpace(realName)
	if realName != "" {
		return realName
	}
	return username
}

func generateRoomName() (string, error) {
	buf := make([]byte, 6)
	if _, err := rand.Read(buf); err != nil {
		return "", gerror.Wrap(err, "生成房间名失败")
	}
	return fmt.Sprintf("m_%s", hex.EncodeToString(buf)), nil
}

func generateShareCode() (string, error) {
	buf := make([]byte, 8)
	if _, err := rand.Read(buf); err != nil {
		return "", gerror.Wrap(err, "生成分享码失败")
	}
	return hex.EncodeToString(buf), nil
}

// ========== 管理端（管理员可操作任意状态会议） ==========

func (s *sSysMeeting) AdminList(ctx context.Context, in *sysin.AdminMeetingListInp) (list []*sysin.AdminMeetingListModel, totalCount int, err error) {
	if err = in.Filter(ctx); err != nil {
		return
	}

	now := gtime.Now()
	mod := meetingModel(ctx)

	if in.Id > 0 {
		mod = mod.Where("id", in.Id)
	}
	if in.Title != "" {
		mod = mod.WhereLike("title", "%"+in.Title+"%")
	}
	if in.HostName != "" {
		mod = mod.WhereLike("host_name", "%"+in.HostName+"%")
	}
	if in.Keyword != "" {
		kw := "%" + in.Keyword + "%"
		mod = mod.Where("(title LIKE ? OR host_name LIKE ? OR room_name LIKE ? OR share_code LIKE ?)", kw, kw, kw, kw)
	}
	if len(in.StartAt) == 2 {
		mod = mod.WhereBetween("start_at", in.StartAt[0], in.StartAt[1])
	}

	switch in.Status {
	case consts.MeetingStatusEnded:
		mod = mod.WhereIn("status", g.Slice{consts.MeetingStatusEnded, consts.MeetingStatusReleased})
	case consts.MeetingStatusOngoing:
		mod = mod.WhereNotIn("status", g.Slice{consts.MeetingStatusEnded, consts.MeetingStatusReleased}).
			Where("(status = ? OR start_at <= ? OR started_at IS NOT NULL)", consts.MeetingStatusOngoing, now)
	case consts.MeetingStatusScheduled:
		mod = mod.WhereNotIn("status", g.Slice{consts.MeetingStatusEnded, consts.MeetingStatusReleased}).
			WhereNot("status", consts.MeetingStatusOngoing).
			Where("start_at > ?", now).
			WhereNull("started_at")
	}

	mod = mod.Page(in.Page, in.PerPage).OrderDesc("id")

	var rows []*entity.Meeting
	if err = mod.ScanAndCount(&rows, &totalCount, false); err != nil {
		return nil, 0, gerror.Wrap(err, "查询会议列表失败")
	}

	list = make([]*sysin.AdminMeetingListModel, 0, len(rows))
	for _, m := range rows {
		if m == nil {
			continue
		}
		list = append(list, toAdminMeetingItem(m))
	}
	attachAdminMeetingTypeNames(ctx, list)
	attachAdminMeetingRecordings(ctx, list)
	return
}

func (s *sSysMeeting) AdminView(ctx context.Context, in *sysin.AdminMeetingViewInp) (res *sysin.AdminMeetingViewModel, err error) {
	if err = in.Filter(ctx); err != nil {
		return
	}
	var m *entity.Meeting
	if err = meetingModel(ctx).Where("id", in.Id).Scan(&m); err != nil {
		return nil, gerror.Wrap(err, "查询会议失败")
	}
	if m == nil {
		return nil, gerror.New("会议不存在")
	}
	res = &sysin.AdminMeetingViewModel{AdminMeetingListModel: toAdminMeetingItem(m)}
	attachAdminMeetingTypeNames(ctx, []*sysin.AdminMeetingListModel{res.AdminMeetingListModel})
	attachAdminMeetingRecordings(ctx, []*sysin.AdminMeetingListModel{res.AdminMeetingListModel})
	return
}

func (s *sSysMeeting) AdminEdit(ctx context.Context, in *sysin.AdminMeetingEditInp) (err error) {
	if err = in.Filter(ctx); err != nil {
		return
	}
	user := contexts.GetUser(ctx)
	if user == nil || user.Id <= 0 {
		return gerror.New("请先登录")
	}
	if in.TypeId != nil {
		if err = service.SysMeetingType().AssertExists(ctx, *in.TypeId); err != nil {
			return
		}
	}
	now := gtime.Now()

	// 编辑：任意状态可改名称、时间与会议类型
	if in.Id > 0 {
		var m *entity.Meeting
		if err = meetingModel(ctx).Where("id", in.Id).Scan(&m); err != nil {
			return gerror.Wrap(err, "查询会议失败")
		}
		if m == nil {
			return gerror.New("会议不存在")
		}
		data := g.Map{
			"title":          in.Title,
			"start_at":       in.StartAt,
			"end_at":         in.EndAt,
			"updated_at":     now,
			"record_enabled": boolToTiny(in.RecordEnabled),
		}
		if in.HostId > 0 {
			data["host_id"] = in.HostId
		}
		if in.HostName != "" {
			data["host_name"] = in.HostName
		}
		if in.TypeId != nil {
			data["type_id"] = *in.TypeId
		}
		if _, err = meetingModel(ctx).Where("id", in.Id).Data(data).Update(); err != nil {
			return gerror.Wrap(err, "更新会议失败")
		}
		return nil
	}

	// 新增
	hostId := in.HostId
	hostName := in.HostName
	if hostId <= 0 {
		hostId = user.Id
	}
	if hostName == "" {
		if hostId == user.Id {
			hostName = displayName(user.Username, user.RealName)
		} else {
			hostName = fmt.Sprintf("用户%d", hostId)
		}
	}
	status := consts.MeetingStatusScheduled
	if !in.StartAt.After(now) {
		status = consts.MeetingStatusOngoing
	}
	typeId := int64(0)
	if in.TypeId != nil {
		typeId = *in.TypeId
	}
	roomName, err := generateRoomName()
	if err != nil {
		return err
	}
	shareCode, err := generateShareCode()
	if err != nil {
		return err
	}
	_, err = meetingModel(ctx).Data(g.Map{
		"title":          in.Title,
		"room_name":      roomName,
		"host_id":        hostId,
		"host_name":      hostName,
		"start_at":       in.StartAt,
		"end_at":         in.EndAt,
		"status":         status,
		"share_code":     shareCode,
		"created_by":     user.Id,
		"created_at":     now,
		"updated_at":     now,
		"record_enabled": boolToTiny(in.RecordEnabled),
		"type_id":        typeId,
	}).Insert()
	if err != nil {
		return gerror.Wrap(err, "创建会议失败")
	}
	return nil
}

func (s *sSysMeeting) AdminDelete(ctx context.Context, in *sysin.AdminMeetingDeleteInp) (err error) {
	if err = in.Filter(ctx); err != nil {
		return
	}
	ids := gconv.Int64s(in.Id)
	if len(ids) == 0 {
		return gerror.New("会议ID不能为空")
	}

	var rows []*entity.Meeting
	if err = meetingModel(ctx).WhereIn("id", ids).Scan(&rows); err != nil {
		return gerror.Wrap(err, "查询会议失败")
	}
	if _, err = meetingModel(ctx).WhereIn("id", ids).Delete(); err != nil {
		return gerror.Wrap(err, "删除会议失败")
	}
	for _, m := range rows {
		if m == nil {
			continue
		}
		service.SysRecording().StopAllForMeeting(ctx, m.Id, m.RoomName)
		service.SysRecording().StopAllAiForMeeting(ctx, m.Id, m.RoomName)
		s.cleanupLiveKitRoom(ctx, m.RoomName)
	}
	return nil
}

func (s *sSysMeeting) AdminRelease(ctx context.Context, in *sysin.AdminMeetingReleaseInp) (err error) {
	if err = in.Filter(ctx); err != nil {
		return
	}
	var m *entity.Meeting
	if err = meetingModel(ctx).Where("id", in.Id).Scan(&m); err != nil {
		return gerror.Wrap(err, "查询会议失败")
	}
	if m == nil {
		return gerror.New("会议不存在")
	}
	if isEndedStatus(m.Status) {
		return nil
	}
	return s.endMeeting(ctx, m, true)
}

func toAdminMeetingItem(m *entity.Meeting) *sysin.AdminMeetingListModel {
	item := toMeetingItem(m, 0)
	return &sysin.AdminMeetingListModel{
		Id:            m.Id,
		Title:         m.Title,
		RoomName:      m.RoomName,
		HostId:        m.HostId,
		HostName:      m.HostName,
		StartAt:       m.StartAt,
		EndAt:         m.EndAt,
		Status:        item.Status,
		ShareCode:     m.ShareCode,
		ShareUrl:      item.ShareUrl,
		Tab:           item.Tab,
		CreatedBy:     m.CreatedBy,
		CreatedAt:     m.CreatedAt,
		UpdatedAt:     m.UpdatedAt,
		ReleasedAt:    m.ReleasedAt,
		Attendees:     item.Attendees,
		RecordEnabled: item.RecordEnabled,
		TypeId:        item.TypeId,
		Recordings:    item.Recordings,
	}
}

// attachMeetingTypeNames 回填会议类型名称（一次批量查询，避免 N+1）
func attachMeetingTypeNames(ctx context.Context, list []*sysin.MeetingItemModel) {
	if len(list) == 0 {
		return
	}
	ids := make([]int64, 0, len(list))
	for _, item := range list {
		if item != nil && item.TypeId > 0 {
			ids = append(ids, item.TypeId)
		}
	}
	names, err := service.SysMeetingType().NamesByIDs(ctx, ids)
	if err != nil {
		g.Log().Warningf(ctx, "attach meeting type names failed: %+v", err)
		return
	}
	for _, item := range list {
		if item == nil {
			continue
		}
		item.TypeName = names[item.TypeId]
	}
}

// attachAdminMeetingTypeNames 回填管理端会议列表的会议类型名称
func attachAdminMeetingTypeNames(ctx context.Context, list []*sysin.AdminMeetingListModel) {
	if len(list) == 0 {
		return
	}
	ids := make([]int64, 0, len(list))
	for _, item := range list {
		if item != nil && item.TypeId > 0 {
			ids = append(ids, item.TypeId)
		}
	}
	names, err := service.SysMeetingType().NamesByIDs(ctx, ids)
	if err != nil {
		g.Log().Warningf(ctx, "attach admin meeting type names failed: %+v", err)
		return
	}
	for _, item := range list {
		if item == nil {
			continue
		}
		item.TypeName = names[item.TypeId]
	}
}

func attachMeetingRecordings(ctx context.Context, list []*sysin.MeetingItemModel) {
	if len(list) == 0 {
		return
	}
	ids := make([]int64, 0, len(list))
	for _, item := range list {
		if item != nil && item.Id > 0 {
			ids = append(ids, item.Id)
		}
	}
	byID, err := service.SysRecording().ListByMeetingIDs(ctx, ids)
	if err != nil {
		g.Log().Warningf(ctx, "attach meeting recordings failed: %+v", err)
		return
	}
	for _, item := range list {
		if item == nil {
			continue
		}
		item.Recordings = byID[item.Id]
		if item.Recordings == nil {
			item.Recordings = []*sysin.RecordingSegmentModel{}
		}
	}
}

func attachMeetingMinutes(ctx context.Context, list []*sysin.MeetingItemModel) {
	if len(list) == 0 {
		return
	}
	ids := make([]int64, 0, len(list))
	for _, item := range list {
		if item != nil && item.Id > 0 && isEndedStatus(item.Status) {
			ids = append(ids, item.Id)
		}
	}
	if len(ids) == 0 {
		return
	}
	byID, err := service.SysMinutes().ListByMeetingIDs(ctx, ids)
	if err != nil {
		g.Log().Warningf(ctx, "attach meeting minutes failed: %+v", err)
		return
	}
	for _, item := range list {
		if item == nil {
			continue
		}
		if m := byID[item.Id]; m != nil {
			// 列表不带全文转写，减小体积
			item.Minutes = &sysin.MinutesModel{
				MeetingId:   m.MeetingId,
				Status:      m.Status,
				Summary:     m.Summary,
				Structured:  m.Structured,
				ErrorMsg:    m.ErrorMsg,
				Model:       m.Model,
				GeneratedAt: m.GeneratedAt,
			}
		}
	}
}

func attachAdminMeetingRecordings(ctx context.Context, list []*sysin.AdminMeetingListModel) {
	if len(list) == 0 {
		return
	}
	ids := make([]int64, 0, len(list))
	for _, item := range list {
		if item != nil && item.Id > 0 {
			ids = append(ids, item.Id)
		}
	}
	byID, err := service.SysRecording().ListByMeetingIDs(ctx, ids)
	if err != nil {
		g.Log().Warningf(ctx, "attach admin meeting recordings failed: %+v", err)
		return
	}
	for _, item := range list {
		if item == nil {
			continue
		}
		item.Recordings = byID[item.Id]
		if item.Recordings == nil {
			item.Recordings = []*sysin.RecordingSegmentModel{}
		}
	}
}
