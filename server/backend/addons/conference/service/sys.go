package service

import (
	"context"
	"io"

	"hotgo/addons/conference/model/entity"
	"hotgo/addons/conference/model/input/sysin"

	"github.com/livekit/protocol/livekit"
)

type (
	ISysToken interface {
		Create(ctx context.Context, in *sysin.TokenCreateInp) (res *sysin.TokenCreateModel, err error)
	}
	ISysRoom interface {
		Kick(ctx context.Context, in *sysin.RoomKickInp) (err error)
		MuteAll(ctx context.Context, in *sysin.RoomMuteAllInp) (res *sysin.RoomMuteAllModel, err error)
		ClaimHost(ctx context.Context, in *sysin.RoomClaimHostInp) (res *sysin.RoomClaimHostModel, err error)
	}
	ISysRecording interface {
		Start(ctx context.Context, in *sysin.RecordingStartInp) (res *sysin.RecordingStartModel, err error)
		Stop(ctx context.Context, in *sysin.RecordingStopInp) (res *sysin.RecordingStopModel, err error)
		Status(ctx context.Context, in *sysin.RecordingStatusInp) (res *sysin.RecordingStatusModel, err error)
		TryAutoStart(ctx context.Context, meeting *entity.Meeting, startedBy int64) (err error)
		StopAllForMeeting(ctx context.Context, meetingId int64, roomName string)
		HandleEgressWebhook(ctx context.Context, info *livekit.EgressInfo)
		ListByMeetingIDs(ctx context.Context, meetingIDs []int64) (map[int64][]*sysin.RecordingSegmentModel, error)
		OpenForDownload(ctx context.Context, id int64) (rc io.ReadCloser, filename string, size int64, err error)
		OpenForPlay(ctx context.Context, id int64, rangeHeader string) (st *sysin.RecordingPlayStream, err error)
	}
	ISysMeeting interface {
		Create(ctx context.Context, in *sysin.MeetingCreateInp) (res *sysin.MeetingItemModel, err error)
		List(ctx context.Context, in *sysin.MeetingListInp) (list []*sysin.MeetingItemModel, err error)
		Release(ctx context.Context, in *sysin.MeetingReleaseInp) (err error)
		Delete(ctx context.Context, in *sysin.MeetingDeleteInp) (err error)
		Update(ctx context.Context, in *sysin.MeetingUpdateInp) (res *sysin.MeetingItemModel, err error)
		ShareView(ctx context.Context, in *sysin.MeetingShareViewInp) (res *sysin.MeetingShareViewModel, err error)
		GetByShareCode(ctx context.Context, code string) (m *entity.Meeting, err error)
		GetByRoomName(ctx context.Context, room string) (m *entity.Meeting, err error)
		AssertJoinable(ctx context.Context, m *entity.Meeting) error
		AutoReleaseExpired(ctx context.Context) (count int, err error)
		AppendAttendee(ctx context.Context, roomName, displayName string) (err error)
		// 管理端：不受主持人/状态限制
		AdminList(ctx context.Context, in *sysin.AdminMeetingListInp) (list []*sysin.AdminMeetingListModel, totalCount int, err error)
		AdminView(ctx context.Context, in *sysin.AdminMeetingViewInp) (res *sysin.AdminMeetingViewModel, err error)
		AdminEdit(ctx context.Context, in *sysin.AdminMeetingEditInp) (err error)
		AdminDelete(ctx context.Context, in *sysin.AdminMeetingDeleteInp) (err error)
		AdminRelease(ctx context.Context, in *sysin.AdminMeetingReleaseInp) (err error)
	}
	ISysAuth interface {
		Captcha(ctx context.Context) (res *sysin.AuthCaptchaModel, err error)
		LoginConfig(ctx context.Context) (res *sysin.AuthLoginConfigModel, err error)
		Login(ctx context.Context, in *sysin.AuthLoginInp) (res *sysin.AuthLoginModel, err error)
		Logout(ctx context.Context) (err error)
		Me(ctx context.Context) (res *sysin.AuthMeModel, err error)
	}
)

var (
	localSysToken     ISysToken
	localSysRoom      ISysRoom
	localSysMeeting   ISysMeeting
	localSysAuth      ISysAuth
	localSysRecording ISysRecording
)

func SysToken() ISysToken {
	if localSysToken == nil {
		panic("implement not found for interface ISysToken, forgot register?")
	}
	return localSysToken
}

func RegisterSysToken(i ISysToken) {
	localSysToken = i
}

func SysRoom() ISysRoom {
	if localSysRoom == nil {
		panic("implement not found for interface ISysRoom, forgot register?")
	}
	return localSysRoom
}

func RegisterSysRoom(i ISysRoom) {
	localSysRoom = i
}

func SysMeeting() ISysMeeting {
	if localSysMeeting == nil {
		panic("implement not found for interface ISysMeeting, forgot register?")
	}
	return localSysMeeting
}

func RegisterSysMeeting(i ISysMeeting) {
	localSysMeeting = i
}

func SysAuth() ISysAuth {
	if localSysAuth == nil {
		panic("implement not found for interface ISysAuth, forgot register?")
	}
	return localSysAuth
}

func RegisterSysAuth(i ISysAuth) {
	localSysAuth = i
}

func SysRecording() ISysRecording {
	if localSysRecording == nil {
		panic("implement not found for interface ISysRecording, forgot register?")
	}
	return localSysRecording
}

func RegisterSysRecording(i ISysRecording) {
	localSysRecording = i
}
