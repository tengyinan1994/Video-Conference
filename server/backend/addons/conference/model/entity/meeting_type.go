package entity

import "github.com/gogf/gf/v2/os/gtime"

// MeetingType 会议类型
type MeetingType struct {
	Id        int64       `json:"id"        orm:"id"`
	Name      string      `json:"name"      orm:"name"`
	CreatedAt *gtime.Time `json:"createdAt" orm:"created_at"`
	UpdatedAt *gtime.Time `json:"updatedAt" orm:"updated_at"`
}
