package entity

import (
	"github.com/gogf/gf/v2/encoding/gjson"
	"github.com/gogf/gf/v2/os/gtime"
)

// Minutes 会后 AI 会议纪要（一场会一行）
type Minutes struct {
	Id                 int64       `json:"id"                 orm:"id"`
	MeetingId          int64       `json:"meetingId"          orm:"meeting_id"`
	Status             string      `json:"status"             orm:"status"`
	Transcript         string      `json:"transcript"         orm:"transcript"`
	Summary            string      `json:"summary"            orm:"summary"`
	Structured         *gjson.Json `json:"structured"         orm:"structured"`
	SourceRecordingIds *gjson.Json `json:"sourceRecordingIds" orm:"source_recording_ids"`
	ErrorMsg           string      `json:"errorMsg"           orm:"error_msg"`
	Model              string      `json:"model"              orm:"model"`
	GeneratedAt        *gtime.Time `json:"generatedAt"        orm:"generated_at"`
	CreatedAt          *gtime.Time `json:"createdAt"          orm:"created_at"`
	UpdatedAt          *gtime.Time `json:"updatedAt"          orm:"updated_at"`
}
