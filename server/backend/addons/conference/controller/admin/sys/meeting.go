package sys

import (
	"context"

	"hotgo/addons/conference/api/admin/meeting"
	"hotgo/addons/conference/model/input/sysin"
	"hotgo/addons/conference/service"
)

var (
	Meeting = cMeeting{}
)

type cMeeting struct{}

// List 会议列表
func (c *cMeeting) List(ctx context.Context, req *meeting.ListReq) (res *meeting.ListRes, err error) {
	list, totalCount, err := service.SysMeeting().AdminList(ctx, &req.AdminMeetingListInp)
	if err != nil {
		return
	}
	if list == nil {
		list = []*sysin.AdminMeetingListModel{}
	}
	res = new(meeting.ListRes)
	res.List = list
	res.PageRes.Pack(req, totalCount)
	return
}

// View 会议详情
func (c *cMeeting) View(ctx context.Context, req *meeting.ViewReq) (res *meeting.ViewRes, err error) {
	data, err := service.SysMeeting().AdminView(ctx, &req.AdminMeetingViewInp)
	if err != nil {
		return
	}
	res = &meeting.ViewRes{AdminMeetingViewModel: data}
	return
}

// Edit 新增/编辑会议
func (c *cMeeting) Edit(ctx context.Context, req *meeting.EditReq) (res *meeting.EditRes, err error) {
	err = service.SysMeeting().AdminEdit(ctx, &req.AdminMeetingEditInp)
	return
}

// Delete 删除会议（任意状态）
func (c *cMeeting) Delete(ctx context.Context, req *meeting.DeleteReq) (res *meeting.DeleteRes, err error) {
	err = service.SysMeeting().AdminDelete(ctx, &req.AdminMeetingDeleteInp)
	return
}

// Release 结束会议
func (c *cMeeting) Release(ctx context.Context, req *meeting.ReleaseReq) (res *meeting.ReleaseRes, err error) {
	err = service.SysMeeting().AdminRelease(ctx, &req.AdminMeetingReleaseInp)
	return
}
