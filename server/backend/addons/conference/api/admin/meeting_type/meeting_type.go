package meetingtype

import (
	"hotgo/addons/conference/model/input/sysin"
	"hotgo/internal/model/input/form"

	"github.com/gogf/gf/v2/frame/g"
)

// ListReq 管理端会议类型列表
type ListReq struct {
	g.Meta `path:"/meetingType/list" method:"get" tags:"会议类型" summary:"获取会议类型列表"`
	sysin.AdminMeetingTypeListInp
}

type ListRes struct {
	form.PageRes
	List []*sysin.AdminMeetingTypeListModel `json:"list" dc:"数据列表"`
}

// ViewReq 管理端会议类型详情
type ViewReq struct {
	g.Meta `path:"/meetingType/view" method:"get" tags:"会议类型" summary:"获取会议类型详情"`
	sysin.AdminMeetingTypeViewInp
}

type ViewRes struct {
	*sysin.AdminMeetingTypeViewModel
}

// EditReq 管理端新增/编辑会议类型
type EditReq struct {
	g.Meta `path:"/meetingType/edit" method:"post" tags:"会议类型" summary:"新增/编辑会议类型"`
	sysin.AdminMeetingTypeEditInp
}

type EditRes struct{}

// DeleteReq 管理端删除会议类型（支持批量）
type DeleteReq struct {
	g.Meta `path:"/meetingType/delete" method:"post" tags:"会议类型" summary:"删除会议类型"`
	sysin.AdminMeetingTypeDeleteInp
}

type DeleteRes struct{}

// OptionReq 管理端会议类型选项（不分页，供会议编辑下拉使用）
type OptionReq struct {
	g.Meta `path:"/meetingType/option" method:"get" tags:"会议类型" summary:"获取会议类型选项"`
}

type OptionRes struct {
	List []*sysin.MeetingTypeOptionModel `json:"list" dc:"类型选项"`
}
