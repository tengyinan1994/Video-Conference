package sys

import (
	"context"

	meetingtype "hotgo/addons/conference/api/admin/meeting_type"
	"hotgo/addons/conference/model/input/sysin"
	"hotgo/addons/conference/service"
)

var (
	MeetingType = cMeetingType{}
)

type cMeetingType struct{}

// List 会议类型列表
func (c *cMeetingType) List(ctx context.Context, req *meetingtype.ListReq) (res *meetingtype.ListRes, err error) {
	list, totalCount, err := service.SysMeetingType().AdminList(ctx, &req.AdminMeetingTypeListInp)
	if err != nil {
		return
	}
	if list == nil {
		list = []*sysin.AdminMeetingTypeListModel{}
	}
	res = new(meetingtype.ListRes)
	res.List = list
	res.PageRes.Pack(req, totalCount)
	return
}

// View 会议类型详情
func (c *cMeetingType) View(ctx context.Context, req *meetingtype.ViewReq) (res *meetingtype.ViewRes, err error) {
	data, err := service.SysMeetingType().AdminView(ctx, &req.AdminMeetingTypeViewInp)
	if err != nil {
		return
	}
	res = &meetingtype.ViewRes{AdminMeetingTypeViewModel: data}
	return
}

// Edit 新增/编辑会议类型
func (c *cMeetingType) Edit(ctx context.Context, req *meetingtype.EditReq) (res *meetingtype.EditRes, err error) {
	err = service.SysMeetingType().AdminEdit(ctx, &req.AdminMeetingTypeEditInp)
	return
}

// Delete 删除会议类型（支持批量）
func (c *cMeetingType) Delete(ctx context.Context, req *meetingtype.DeleteReq) (res *meetingtype.DeleteRes, err error) {
	err = service.SysMeetingType().AdminDelete(ctx, &req.AdminMeetingTypeDeleteInp)
	return
}

// Option 会议类型选项（供会议编辑下拉使用）
func (c *cMeetingType) Option(ctx context.Context, req *meetingtype.OptionReq) (res *meetingtype.OptionRes, err error) {
	list, err := service.SysMeetingType().Options(ctx)
	if err != nil {
		return
	}
	if list == nil {
		list = []*sysin.MeetingTypeOptionModel{}
	}
	res = &meetingtype.OptionRes{List: list}
	return
}
