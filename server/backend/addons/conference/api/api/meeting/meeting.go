package meeting

import (
	"hotgo/addons/conference/model/input/sysin"

	"github.com/gogf/gf/v2/frame/g"
)

type CreateReq struct {
	g.Meta `path:"/meeting/create" method:"post" tags:"视频会议会议室" summary:"创建会议室"`
	sysin.MeetingCreateInp
}

type CreateRes struct {
	*sysin.MeetingItemModel
}

type ListReq struct {
	g.Meta `path:"/meeting/list" method:"get" tags:"视频会议会议室" summary:"会议室列表"`
	sysin.MeetingListInp
}

type ListRes struct {
	List []*sysin.MeetingItemModel `json:"list"`
}

type ReleaseReq struct {
	g.Meta `path:"/meeting/release" method:"post" tags:"视频会议会议室" summary:"结束会议室（保留历史）"`
	sysin.MeetingReleaseInp
}

type ReleaseRes struct{}

type DeleteReq struct {
	g.Meta `path:"/meeting/delete" method:"post" tags:"视频会议会议室" summary:"删除会议室（硬删）"`
	sysin.MeetingDeleteInp
}

type DeleteRes struct{}

type UpdateReq struct {
	g.Meta `path:"/meeting/update" method:"post" tags:"视频会议会议室" summary:"更新会议室（名称与时间）"`
	sysin.MeetingUpdateInp
}

type UpdateRes struct {
	*sysin.MeetingItemModel
}

type ShareViewReq struct {
	g.Meta `path:"/meeting/shareView" method:"get" tags:"视频会议会议室" summary:"分享链接会议信息"`
	sysin.MeetingShareViewInp
}

type ShareViewRes struct {
	*sysin.MeetingShareViewModel
}
