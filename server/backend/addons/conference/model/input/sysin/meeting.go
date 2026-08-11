package sysin

import (
	"context"
	"strings"
	"unicode/utf8"

	"hotgo/addons/conference/consts"

	"github.com/gogf/gf/v2/errors/gerror"
	"github.com/gogf/gf/v2/os/gtime"
)

// MeetingCreateInp 创建会议室
type MeetingCreateInp struct {
	Title    string      `json:"title" v:"required#会议名称不能为空" dc:"会议名称"`
	HostId   int64       `json:"hostId" dc:"主持人用户ID，默认当前用户"`
	HostName string      `json:"hostName" dc:"主持人显示名"`
	StartAt  *gtime.Time `json:"startAt" v:"required#请填写开始时间" dc:"开始时间"`
	EndAt    *gtime.Time `json:"endAt" v:"required#请填写结束时间" dc:"结束时间"`
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
	Id        int64       `json:"id"`
	Title     string      `json:"title"`
	RoomName  string      `json:"roomName"`
	HostId    int64       `json:"hostId"`
	HostName  string      `json:"hostName"`
	StartAt   *gtime.Time `json:"startAt"`
	EndAt     *gtime.Time `json:"endAt"`
	Status    string      `json:"status"`
	ShareCode string      `json:"shareCode"`
	ShareUrl  string      `json:"shareUrl" dc:"相对路径 /join/{shareCode}"`
	IsHost    bool        `json:"isHost" dc:"当前用户是否主持人"`
	Tab       string      `json:"tab" dc:"ongoing / scheduled / ended"`
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
	Id      int64       `json:"id" v:"required#会议ID不能为空" dc:"会议ID"`
	Title   string      `json:"title" v:"required#会议名称不能为空" dc:"会议名称"`
	StartAt *gtime.Time `json:"startAt" v:"required#请填写开始时间" dc:"开始时间"`
	EndAt   *gtime.Time `json:"endAt" v:"required#请填写结束时间" dc:"结束时间"`
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
