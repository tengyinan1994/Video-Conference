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
	"github.com/gogf/gf/v2/errors/gerror"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gtime"
	"github.com/gogf/gf/v2/util/gconv"
	"github.com/livekit/protocol/livekit"
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
	if !in.StartAt.After(now) && in.EndAt.After(now.Add(-time.Duration(consts.MeetingReleaseGraceHours)*time.Hour)) {
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
		"title":      in.Title,
		"room_name":  roomName,
		"host_id":    hostId,
		"host_name":  hostName,
		"start_at":   in.StartAt,
		"end_at":     in.EndAt,
		"status":     status,
		"share_code": shareCode,
		"created_by": user.Id,
		"created_at": now,
		"updated_at": now,
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
	graceEnd := now.Add(-time.Duration(consts.MeetingReleaseGraceHours) * time.Hour)

	mod := meetingModel(ctx)
	switch in.Tab {
	case consts.MeetingListTabEnded:
		mod = mod.WhereIn("status", g.Slice{consts.MeetingStatusEnded, consts.MeetingStatusReleased})
	case consts.MeetingListTabOngoing:
		mod = mod.WhereNotIn("status", g.Slice{consts.MeetingStatusEnded, consts.MeetingStatusReleased}).
			WhereLTE("start_at", now).WhereGT("end_at", graceEnd)
	case consts.MeetingListTabScheduled:
		mod = mod.WhereNotIn("status", g.Slice{consts.MeetingStatusEnded, consts.MeetingStatusReleased}).
			WhereGT("start_at", now)
	default:
		// all：表中全部会议（删除为硬删，不会出现在此）
	}

	var rows []*entity.Meeting
	if err = mod.OrderAsc("start_at").Scan(&rows); err != nil {
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
	return
}

func (s *sSysMeeting) Release(ctx context.Context, in *sysin.MeetingReleaseInp) (err error) {
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
		return gerror.New("会议室不存在")
	}
	if isEndedStatus(m.Status) {
		return nil
	}
	if m.HostId != user.Id && !iservice.AdminMember().VerifySuperId(ctx, user.Id) {
		return gerror.New("仅主持人可结束会议室")
	}
	return s.endMeeting(ctx, m, true)
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

	if _, err = meetingModel(ctx).Where("id", m.Id).Delete(); err != nil {
		return gerror.Wrap(err, "删除会议室失败")
	}
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

	now := gtime.Now()
	if _, err = meetingModel(ctx).Where("id", m.Id).Data(g.Map{
		"title":      in.Title,
		"start_at":   in.StartAt,
		"end_at":     in.EndAt,
		"updated_at": now,
	}).Update(); err != nil {
		return nil, gerror.Wrap(err, "更新会议室失败")
	}
	m.Title = in.Title
	m.StartAt = in.StartAt
	m.EndAt = in.EndAt
	m.UpdatedAt = now
	res = toMeetingItem(m, user.Id)
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
	canJoin := item.Tab != consts.MeetingListTabEnded
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
	if isEndedStatus(m.Status) || isPastAutoRelease(m) {
		return gerror.New("会议室已结束，无法加入")
	}
	return nil
}

// AutoReleaseExpired 定时：结束时间超过宽限期的未结束会议 → 标记为已结束
func (s *sSysMeeting) AutoReleaseExpired(ctx context.Context) (count int, err error) {
	deadline := gtime.Now().Add(-time.Duration(consts.MeetingReleaseGraceHours) * time.Hour)
	var rows []*entity.Meeting
	if err = meetingModel(ctx).
		WhereNotIn("status", g.Slice{consts.MeetingStatusEnded, consts.MeetingStatusReleased}).
		WhereLTE("end_at", deadline).
		Scan(&rows); err != nil {
		return 0, gerror.Wrap(err, "扫描过期会议室失败")
	}
	for _, m := range rows {
		if m == nil {
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
	s.cleanupLiveKitRoom(ctx, m.RoomName)
	return nil
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

func toMeetingItem(m *entity.Meeting, userId int64) *sysin.MeetingItemModel {
	now := gtime.Now()
	tab := consts.MeetingListTabScheduled
	status := m.Status

	if isEndedStatus(status) || isPastAutoRelease(m) {
		tab = consts.MeetingListTabEnded
		status = consts.MeetingStatusEnded
	} else if m.StartAt != nil && !m.StartAt.After(now) {
		tab = consts.MeetingListTabOngoing
		status = consts.MeetingStatusOngoing
	} else {
		status = consts.MeetingStatusScheduled
	}

	return &sysin.MeetingItemModel{
		Id:        m.Id,
		Title:     m.Title,
		RoomName:  m.RoomName,
		HostId:    m.HostId,
		HostName:  m.HostName,
		StartAt:   m.StartAt,
		EndAt:     m.EndAt,
		Status:    status,
		ShareCode: m.ShareCode,
		ShareUrl:  "/join/" + m.ShareCode,
		IsHost:    userId > 0 && m.HostId == userId,
		Tab:       tab,
	}
}

func isPastAutoRelease(m *entity.Meeting) bool {
	if m == nil || m.EndAt == nil {
		return false
	}
	deadline := m.EndAt.Add(time.Duration(consts.MeetingReleaseGraceHours) * time.Hour)
	return !gtime.Now().Before(deadline)
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
	graceEnd := now.Add(-time.Duration(consts.MeetingReleaseGraceHours) * time.Hour)
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
			WhereLTE("start_at", now).WhereGT("end_at", graceEnd)
	case consts.MeetingStatusScheduled:
		mod = mod.WhereNotIn("status", g.Slice{consts.MeetingStatusEnded, consts.MeetingStatusReleased}).
			WhereGT("start_at", now)
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
	now := gtime.Now()

	// 编辑：任意状态可改名称与时间
	if in.Id > 0 {
		var m *entity.Meeting
		if err = meetingModel(ctx).Where("id", in.Id).Scan(&m); err != nil {
			return gerror.Wrap(err, "查询会议失败")
		}
		if m == nil {
			return gerror.New("会议不存在")
		}
		data := g.Map{
			"title":      in.Title,
			"start_at":   in.StartAt,
			"end_at":     in.EndAt,
			"updated_at": now,
		}
		if in.HostId > 0 {
			data["host_id"] = in.HostId
		}
		if in.HostName != "" {
			data["host_name"] = in.HostName
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
	if !in.StartAt.After(now) && in.EndAt.After(now.Add(-time.Duration(consts.MeetingReleaseGraceHours)*time.Hour)) {
		status = consts.MeetingStatusOngoing
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
		"title":      in.Title,
		"room_name":  roomName,
		"host_id":    hostId,
		"host_name":  hostName,
		"start_at":   in.StartAt,
		"end_at":     in.EndAt,
		"status":     status,
		"share_code": shareCode,
		"created_by": user.Id,
		"created_at": now,
		"updated_at": now,
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
		Id:         m.Id,
		Title:      m.Title,
		RoomName:   m.RoomName,
		HostId:     m.HostId,
		HostName:   m.HostName,
		StartAt:    m.StartAt,
		EndAt:      m.EndAt,
		Status:     item.Status,
		ShareCode:  m.ShareCode,
		ShareUrl:   item.ShareUrl,
		Tab:        item.Tab,
		CreatedBy:  m.CreatedBy,
		CreatedAt:  m.CreatedAt,
		UpdatedAt:  m.UpdatedAt,
		ReleasedAt: m.ReleasedAt,
	}
}
