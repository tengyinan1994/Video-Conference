package meeting

import (
	"hotgo/addons/conference/model/input/sysin"
	"hotgo/internal/model/input/form"

	"github.com/gogf/gf/v2/frame/g"
)

// ListReq 管理端会议列表
type ListReq struct {
	g.Meta `path:"/meeting/list" method:"get" tags:"会议管理" summary:"获取会议列表"`
	sysin.AdminMeetingListInp
}

type ListRes struct {
	form.PageRes
	List []*sysin.AdminMeetingListModel `json:"list" dc:"数据列表"`
}

// ViewReq 管理端会议详情
type ViewReq struct {
	g.Meta `path:"/meeting/view" method:"get" tags:"会议管理" summary:"获取会议详情"`
	sysin.AdminMeetingViewInp
}

type ViewRes struct {
	*sysin.AdminMeetingViewModel
}

// EditReq 管理端新增/编辑会议
type EditReq struct {
	g.Meta `path:"/meeting/edit" method:"post" tags:"会议管理" summary:"新增/编辑会议"`
	sysin.AdminMeetingEditInp
}

type EditRes struct{}

// DeleteReq 管理端删除会议（任意状态）
type DeleteReq struct {
	g.Meta `path:"/meeting/delete" method:"post" tags:"会议管理" summary:"删除会议"`
	sysin.AdminMeetingDeleteInp
}

type DeleteRes struct{}

// ReleaseReq 管理端结束会议
type ReleaseReq struct {
	g.Meta `path:"/meeting/release" method:"post" tags:"会议管理" summary:"结束会议"`
	sysin.AdminMeetingReleaseInp
}

type ReleaseRes struct{}
