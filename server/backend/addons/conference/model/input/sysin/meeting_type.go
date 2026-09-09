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

// ========== 会议端（下拉选项） ==========

// MeetingTypeOptionModel 会议类型选项
type MeetingTypeOptionModel struct {
	Id   int64  `json:"id"   dc:"类型ID"`
	Name string `json:"name" dc:"类型名称"`
}

// ========== 管理端 ==========

// AdminMeetingTypeListInp 管理端会议类型列表
type AdminMeetingTypeListInp struct {
	form.PageReq
	Id   int64  `json:"id"   dc:"类型ID"`
	Name string `json:"name" dc:"类型名称"`
}

func (in *AdminMeetingTypeListInp) Filter(ctx context.Context) (err error) {
	in.Name = strings.TrimSpace(in.Name)
	return
}

// AdminMeetingTypeListModel 管理端列表项
type AdminMeetingTypeListModel struct {
	Id        int64       `json:"id"        dc:"类型ID"`
	Name      string      `json:"name"      dc:"类型名称"`
	CreatedAt *gtime.Time `json:"createdAt" dc:"创建时间"`
	UpdatedAt *gtime.Time `json:"updatedAt" dc:"更新时间"`
}

// AdminMeetingTypeViewInp 管理端会议类型详情
type AdminMeetingTypeViewInp struct {
	Id int64 `json:"id" v:"required#类型ID不能为空" dc:"类型ID"`
}

func (in *AdminMeetingTypeViewInp) Filter(ctx context.Context) (err error) {
	if in.Id <= 0 {
		return gerror.New("类型ID不能为空")
	}
	return
}

// AdminMeetingTypeViewModel 管理端会议类型详情
type AdminMeetingTypeViewModel struct {
	*AdminMeetingTypeListModel
}

// AdminMeetingTypeEditInp 管理端新增/编辑会议类型
type AdminMeetingTypeEditInp struct {
	Id   int64  `json:"id"   dc:"类型ID，大于0为编辑"`
	Name string `json:"name" v:"required#类型名称不能为空" dc:"类型名称"`
}

func (in *AdminMeetingTypeEditInp) Filter(ctx context.Context) (err error) {
	in.Name = strings.TrimSpace(in.Name)
	if in.Name == "" {
		return gerror.New("类型名称不能为空")
	}
	if utf8.RuneCountInString(in.Name) > consts.MaxMeetingTypeNameLen {
		return gerror.Newf("类型名称最长 %d 个字符", consts.MaxMeetingTypeNameLen)
	}
	return
}

// AdminMeetingTypeDeleteInp 管理端删除会议类型（支持批量）
type AdminMeetingTypeDeleteInp struct {
	Id interface{} `json:"id" v:"required#类型ID不能为空" dc:"类型ID，支持批量"`
}

func (in *AdminMeetingTypeDeleteInp) Filter(ctx context.Context) (err error) {
	return
}
