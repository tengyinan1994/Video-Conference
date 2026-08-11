package api

import (
	"context"

	"hotgo/addons/conference/api/api/meeting"
	"hotgo/addons/conference/model/input/sysin"
	"hotgo/addons/conference/service"
)

// MeetingPublic 公开会议室接口
var MeetingPublic = cMeetingPublic{}

type cMeetingPublic struct{}

func (c *cMeetingPublic) ShareView(ctx context.Context, req *meeting.ShareViewReq) (res *meeting.ShareViewRes, err error) {
	data, err := service.SysMeeting().ShareView(ctx, &req.MeetingShareViewInp)
	if err != nil {
		return
	}
	res = &meeting.ShareViewRes{MeetingShareViewModel: data}
	return
}

// Meeting 需登录的会议室接口
var Meeting = cMeeting{}

type cMeeting struct{}

func (c *cMeeting) Create(ctx context.Context, req *meeting.CreateReq) (res *meeting.CreateRes, err error) {
	data, err := service.SysMeeting().Create(ctx, &req.MeetingCreateInp)
	if err != nil {
		return
	}
	res = &meeting.CreateRes{MeetingItemModel: data}
	return
}

func (c *cMeeting) List(ctx context.Context, req *meeting.ListReq) (res *meeting.ListRes, err error) {
	list, err := service.SysMeeting().List(ctx, &req.MeetingListInp)
	if err != nil {
		return
	}
	if list == nil {
		list = make([]*sysin.MeetingItemModel, 0)
	}
	res = &meeting.ListRes{List: list}
	return
}

func (c *cMeeting) Release(ctx context.Context, req *meeting.ReleaseReq) (res *meeting.ReleaseRes, err error) {
	err = service.SysMeeting().Release(ctx, &req.MeetingReleaseInp)
	res = new(meeting.ReleaseRes)
	return
}

func (c *cMeeting) Delete(ctx context.Context, req *meeting.DeleteReq) (res *meeting.DeleteRes, err error) {
	err = service.SysMeeting().Delete(ctx, &req.MeetingDeleteInp)
	res = new(meeting.DeleteRes)
	return
}

func (c *cMeeting) Update(ctx context.Context, req *meeting.UpdateReq) (res *meeting.UpdateRes, err error) {
	data, err := service.SysMeeting().Update(ctx, &req.MeetingUpdateInp)
	if err != nil {
		return
	}
	res = &meeting.UpdateRes{MeetingItemModel: data}
	return
}
