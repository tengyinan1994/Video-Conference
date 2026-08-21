package entity

import (
	"github.com/gogf/gf/v2/os/gtime"
)

// Recording 会议录制分段（一次启停 = 一行）
type Recording struct {
	Id        int64       `json:"id"        orm:"id"`
	MeetingId int64       `json:"meetingId" orm:"meeting_id"`
	RoomName  string      `json:"roomName"  orm:"room_name"`
	EgressId  string      `json:"egressId"  orm:"egress_id"`
	Seq       int         `json:"seq"       orm:"seq"`
	Status    string      `json:"status"    orm:"status"`
	ObjectKey string      `json:"objectKey" orm:"object_key"`
	FileSize  int64       `json:"fileSize"  orm:"file_size"`
	StartedAt *gtime.Time `json:"startedAt" orm:"started_at"`
	EndedAt   *gtime.Time `json:"endedAt"   orm:"ended_at"`
	StartedBy int64       `json:"startedBy" orm:"started_by"`
	ErrorMsg  string      `json:"errorMsg"  orm:"error_msg"`
	CreatedAt *gtime.Time `json:"createdAt" orm:"created_at"`
	UpdatedAt *gtime.Time `json:"updatedAt" orm:"updated_at"`
}
