package sysin

import (
	"context"
	"strings"
	"unicode/utf8"

	"hotgo/addons/conference/consts"
	"hotgo/internal/model/input/form"

	"github.com/gogf/gf/v2/errors/gerror"
	"github.com/gogf/gf/v2/os/gtime"
)

// MeetingCreateInp 创建会议室
type MeetingCreateInp struct {
	Title         string      `json:"title" v:"required#会议名称不能为空" dc:"会议名称"`
	HostId        int64       `json:"hostId" dc:"主持人用户ID，默认当前用户"`
	HostName      string      `json:"hostName" dc:"主持人显示名"`
	StartAt       *gtime.Time `json:"startAt" v:"required#请填写开始时间" dc:"开始时间"`
	EndAt         *gtime.Time `json:"endAt" v:"required#请填写结束时间" dc:"结束时间"`
	RecordEnabled bool        `json:"recordEnabled" dc:"是否开启录制（默认关；开则主持人进房自动起第一段）"`
}

func (in *MeetingCreateInp) Filter(ctx context.Context) (err error) {
	in.Title = strings.TrimSpace(in.Title)
	in.HostName = strings.TrimSpace(in.HostName)
	if in.Title == "" {
		return gerror.New("会议名称不能为空")
	}
	if utf8.RuneCountInString(in.Title) > consts.MaxMeetingTitleLen {
		return gerror.Newf("会议名称最长 %d 个字符", consts.MaxMeetingTitleLen)
	}
	if in.StartAt == nil || in.EndAt == nil {
		return gerror.New("请填写会议时间段")
	}
	if !in.EndAt.After(in.StartAt) {
		return gerror.New("结束时间必须晚于开始时间")
	}
	return
}

// MeetingListInp 会议室列表
type MeetingListInp struct {
	Tab string `json:"tab" dc:"列表分区：ongoing / scheduled / all / ended"`
}

func (in *MeetingListInp) Filter(ctx context.Context) (err error) {
	in.Tab = strings.TrimSpace(in.Tab)
	if in.Tab == "" {
		in.Tab = "all"
	}
	switch in.Tab {
	case "ongoing", "scheduled", "all", "ended":
		return nil
	default:
		return gerror.New("无效的列表分区")
	}
}

// MeetingItemModel 列表项
type MeetingItemModel struct {
	Id            int64       `json:"id"`
	Title         string      `json:"title"`
	RoomName      string      `json:"roomName"`
	HostId        int64       `json:"hostId"`
	HostName      string      `json:"hostName"`
	StartAt       *gtime.Time `json:"startAt"`
	EndAt         *gtime.Time `json:"endAt"`
	Status        string      `json:"status"`
	ShareCode     string      `json:"shareCode"`
	ShareUrl      string      `json:"shareUrl" dc:"相对路径 /join/{shareCode}"`
	IsHost        bool        `json:"isHost" dc:"当前用户是否主持人"`
	Tab           string      `json:"tab" dc:"ongoing / scheduled / ended"`
	Attendees     []string                  `json:"attendees" dc:"参会显示名去重列表"`
	RecordEnabled bool                      `json:"recordEnabled" dc:"创建时录制开关"`
	Recordings    []*RecordingSegmentModel  `json:"recordings" dc:"录制分段（含回放地址）"`
}

// MeetingReleaseInp 结束会议室（保留记录，计入历史）
type MeetingReleaseInp struct {
	Id int64 `json:"id" v:"required#会议ID不能为空" dc:"会议ID"`
}

func (in *MeetingReleaseInp) Filter(ctx context.Context) (err error) {
	if in.Id <= 0 {
		return gerror.New("会议ID不能为空")
	}
	return
}

// MeetingDeleteInp 删除会议室（硬删，建错重建场景）
type MeetingDeleteInp struct {
	Id int64 `json:"id" v:"required#会议ID不能为空" dc:"会议ID"`
}

func (in *MeetingDeleteInp) Filter(ctx context.Context) (err error) {
	if in.Id <= 0 {
		return gerror.New("会议ID不能为空")
	}
	return
}

// MeetingUpdateInp 更新会议室（名称与时间）
type MeetingUpdateInp struct {
	Id            int64       `json:"id" v:"required#会议ID不能为空" dc:"会议ID"`
	Title         string      `json:"title" v:"required#会议名称不能为空" dc:"会议名称"`
	StartAt       *gtime.Time `json:"startAt" v:"required#请填写开始时间" dc:"开始时间"`
	EndAt         *gtime.Time `json:"endAt" v:"required#请填写结束时间" dc:"结束时间"`
	RecordEnabled *bool       `json:"recordEnabled" dc:"是否开启录制；nil 表示不改"`
}

func (in *MeetingUpdateInp) Filter(ctx context.Context) (err error) {
	if in.Id <= 0 {
		return gerror.New("会议ID不能为空")
	}
	in.Title = strings.TrimSpace(in.Title)
	if in.Title == "" {
		return gerror.New("会议名称不能为空")
	}
	if utf8.RuneCountInString(in.Title) > consts.MaxMeetingTitleLen {
		return gerror.Newf("会议名称最长 %d 个字符", consts.MaxMeetingTitleLen)
	}
	if in.StartAt == nil || in.EndAt == nil {
		return gerror.New("请填写会议时间段")
	}
	if !in.EndAt.After(in.StartAt) {
		return gerror.New("结束时间必须晚于开始时间")
	}
	return
}

// MeetingShareViewInp 分享页查询
type MeetingShareViewInp struct {
	Code string `json:"code" v:"required#分享码不能为空" dc:"分享短码"`
}

func (in *MeetingShareViewInp) Filter(ctx context.Context) (err error) {
	in.Code = strings.TrimSpace(in.Code)
	if in.Code == "" {
		return gerror.New("分享码不能为空")
	}
	return
}

// MeetingShareViewModel 分享页信息
type MeetingShareViewModel struct {
	Title     string      `json:"title"`
	RoomName  string      `json:"roomName"`
	HostName  string      `json:"hostName"`
	StartAt   *gtime.Time `json:"startAt"`
	EndAt     *gtime.Time `json:"endAt"`
	Status    string      `json:"status"`
	ShareCode string      `json:"shareCode"`
	CanJoin   bool        `json:"canJoin"`
}

// ========== 管理端 ==========

// AdminMeetingListInp 管理端会议列表
type AdminMeetingListInp struct {
	form.PageReq
	Id       int64         `json:"id" dc:"会议ID"`
	Title    string        `json:"title" dc:"会议名称"`
	HostName string        `json:"hostName" dc:"主持人"`
	Status   string        `json:"status" dc:"有效状态：scheduled / ongoing / ended，空为全部"`
	Keyword  string        `json:"keyword" dc:"关键词（名称/主持人/房间/分享码）"`
	StartAt  []*gtime.Time `json:"startAt" dc:"开始时间范围"`
}

func (in *AdminMeetingListInp) Filter(ctx context.Context) (err error) {
	in.Title = strings.TrimSpace(in.Title)
	in.HostName = strings.TrimSpace(in.HostName)
	in.Keyword = strings.TrimSpace(in.Keyword)
	in.Status = strings.TrimSpace(in.Status)
	if in.Status == "" {
		return nil
	}
	switch in.Status {
	case consts.MeetingStatusScheduled, consts.MeetingStatusOngoing, consts.MeetingStatusEnded:
		return nil
	default:
		return gerror.New("无效的会议状态")
	}
}

// AdminMeetingListModel 管理端列表项
type AdminMeetingListModel struct {
	Id            int64       `json:"id" dc:"会议ID"`
	Title         string      `json:"title" dc:"会议名称"`
	RoomName      string      `json:"roomName" dc:"房间名"`
	HostId        int64       `json:"hostId" dc:"主持人ID"`
	HostName      string      `json:"hostName" dc:"主持人"`
	StartAt       *gtime.Time `json:"startAt" dc:"开始时间"`
	EndAt         *gtime.Time `json:"endAt" dc:"结束时间"`
	Status        string      `json:"status" dc:"有效状态"`
	ShareCode     string      `json:"shareCode" dc:"分享码"`
	ShareUrl      string      `json:"shareUrl" dc:"分享路径"`
	Tab           string      `json:"tab" dc:"分区"`
	CreatedBy     int64       `json:"createdBy" dc:"创建者"`
	CreatedAt     *gtime.Time `json:"createdAt" dc:"创建时间"`
	UpdatedAt     *gtime.Time `json:"updatedAt" dc:"更新时间"`
	ReleasedAt    *gtime.Time `json:"releasedAt" dc:"结束时间点"`
	Attendees     []string                 `json:"attendees" dc:"参会显示名去重列表"`
	RecordEnabled bool                     `json:"recordEnabled" dc:"是否开启录制"`
	Recordings    []*RecordingSegmentModel `json:"recordings" dc:"录制分段"`
}

// AdminMeetingViewInp 管理端会议详情
type AdminMeetingViewInp struct {
	Id int64 `json:"id" v:"required#会议ID不能为空" dc:"会议ID"`
}

func (in *AdminMeetingViewInp) Filter(ctx context.Context) (err error) {
	if in.Id <= 0 {
		return gerror.New("会议ID不能为空")
	}
	return
}

// AdminMeetingViewModel 管理端会议详情
type AdminMeetingViewModel struct {
	*AdminMeetingListModel
}

// AdminMeetingEditInp 管理端新增/编辑会议（不受状态限制）
type AdminMeetingEditInp struct {
	Id            int64       `json:"id" dc:"会议ID，大于0为编辑"`
	Title         string      `json:"title" v:"required#会议名称不能为空" dc:"会议名称"`
	HostId        int64       `json:"hostId" dc:"主持人用户ID"`
	HostName      string      `json:"hostName" dc:"主持人显示名"`
	StartAt       *gtime.Time `json:"startAt" v:"required#请填写开始时间" dc:"开始时间"`
	EndAt         *gtime.Time `json:"endAt" v:"required#请填写结束时间" dc:"结束时间"`
	RecordEnabled bool        `json:"recordEnabled" dc:"是否开启录制"`
}

func (in *AdminMeetingEditInp) Filter(ctx context.Context) (err error) {
	in.Title = strings.TrimSpace(in.Title)
	in.HostName = strings.TrimSpace(in.HostName)
	if in.Title == "" {
		return gerror.New("会议名称不能为空")
	}
	if utf8.RuneCountInString(in.Title) > consts.MaxMeetingTitleLen {
		return gerror.Newf("会议名称最长 %d 个字符", consts.MaxMeetingTitleLen)
	}
	if in.StartAt == nil || in.EndAt == nil {
		return gerror.New("请填写会议时间段")
	}
	if !in.EndAt.After(in.StartAt) {
		return gerror.New("结束时间必须晚于开始时间")
	}
	return
}

// AdminMeetingDeleteInp 管理端删除会议（任意状态可删）
type AdminMeetingDeleteInp struct {
	Id interface{} `json:"id" v:"required#会议ID不能为空" dc:"会议ID，支持批量"`
}

func (in *AdminMeetingDeleteInp) Filter(ctx context.Context) (err error) {
	return
}

// AdminMeetingReleaseInp 管理端结束会议
type AdminMeetingReleaseInp struct {
	Id int64 `json:"id" v:"required#会议ID不能为空" dc:"会议ID"`
}

func (in *AdminMeetingReleaseInp) Filter(ctx context.Context) (err error) {
	if in.Id <= 0 {
		return gerror.New("会议ID不能为空")
	}
	return
}
