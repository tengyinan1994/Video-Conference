package api

import (
	"context"

	meetingtype "hotgo/addons/conference/api/api/meeting_type"
	"hotgo/addons/conference/model/input/sysin"
	"hotgo/addons/conference/service"
)

// MeetingTypeOptions 会议端会议类型接口（需登录）
var MeetingTypeOptions = cMeetingTypeOptions{}

type cMeetingTypeOptions struct{}

func (c *cMeetingTypeOptions) Options(ctx context.Context, req *meetingtype.OptionsReq) (res *meetingtype.OptionsRes, err error) {
	list, err := service.SysMeetingType().Options(ctx)
	if err != nil {
		return
	}
	if list == nil {
		list = []*sysin.MeetingTypeOptionModel{}
	}
	res = &meetingtype.OptionsRes{List: list}
	return
}
