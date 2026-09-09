package meetingtype

import (
	"hotgo/addons/conference/model/input/sysin"

	"github.com/gogf/gf/v2/frame/g"
)

// OptionsReq 会议端会议类型选项（供会议列表筛选与新建会议下拉使用）
type OptionsReq struct {
	g.Meta `path:"/meetingType/options" method:"get" tags:"视频会议会议类型" summary:"获取会议类型选项"`
}

type OptionsRes struct {
	List []*sysin.MeetingTypeOptionModel `json:"list" dc:"类型选项"`
}
