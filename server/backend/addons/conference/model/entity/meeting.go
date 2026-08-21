package entity

import (
	"github.com/gogf/gf/v2/encoding/gjson"
	"github.com/gogf/gf/v2/os/gtime"
)

// Meeting 业务会议室
type Meeting struct {
	Id         int64       `json:"id"         orm:"id"`
	Title      string      `json:"title"      orm:"title"`
	RoomName   string      `json:"roomName"   orm:"room_name"`
	HostId     int64       `json:"hostId"     orm:"host_id"`
	HostName   string      `json:"hostName"   orm:"host_name"`
	StartAt    *gtime.Time `json:"startAt"    orm:"start_at"`
	EndAt      *gtime.Time `json:"endAt"      orm:"end_at"`
	Status     string      `json:"status"     orm:"status"`
	ShareCode  string      `json:"shareCode"  orm:"share_code"`
	CreatedBy  int64       `json:"createdBy"  orm:"created_by"`
	CreatedAt  *gtime.Time `json:"createdAt"  orm:"created_at"`
	UpdatedAt  *gtime.Time `json:"updatedAt"  orm:"updated_at"`
	ReleasedAt    *gtime.Time `json:"releasedAt"    orm:"released_at"`
	Attendees     *gjson.Json `json:"attendees"     orm:"attendees"`
	RecordEnabled int         `json:"recordEnabled" orm:"record_enabled"`
}
