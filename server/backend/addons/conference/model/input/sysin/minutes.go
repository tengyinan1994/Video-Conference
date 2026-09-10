package sysin

import (
	"context"
	"strings"

	"github.com/gogf/gf/v2/encoding/gjson"
	"github.com/gogf/gf/v2/errors/gerror"
	"github.com/gogf/gf/v2/os/gtime"
)

// MinutesViewInp 查询会议纪要
type MinutesViewInp struct {
	MeetingId int64 `json:"meetingId" in:"query" v:"required#会议ID不能为空" dc:"会议ID"`
}

func (in *MinutesViewInp) Filter(ctx context.Context) (err error) {
	if in.MeetingId <= 0 {
		return gerror.New("会议ID不能为空")
	}
	return
}

// MinutesRegenerateInp 重新生成纪要
type MinutesRegenerateInp struct {
	MeetingId int64 `json:"meetingId" v:"required#会议ID不能为空" dc:"会议ID"`
}

func (in *MinutesRegenerateInp) Filter(ctx context.Context) (err error) {
	if in.MeetingId <= 0 {
		return gerror.New("会议ID不能为空")
	}
	return
}

// MinutesCallbackInp Worker 回调
type MinutesCallbackInp struct {
	MeetingId          int64       `json:"meetingId"`
	Status             string      `json:"status"`
	Transcript         string      `json:"transcript"`
	Summary            string      `json:"summary"`
	Structured         *gjson.Json `json:"structured"`
	SourceRecordingIds []int64     `json:"sourceRecordingIds"`
	ErrorMsg           string      `json:"errorMsg"`
	Model              string      `json:"model"`
}

func (in *MinutesCallbackInp) Filter(ctx context.Context) (err error) {
	if in.MeetingId <= 0 {
		return gerror.New("会议ID不能为空")
	}
	in.Status = strings.TrimSpace(in.Status)
	if in.Status == "" {
		return gerror.New("状态不能为空")
	}
	in.ErrorMsg = strings.TrimSpace(in.ErrorMsg)
	in.Model = strings.TrimSpace(in.Model)
	return
}

// MinutesModel 纪要对外模型
type MinutesModel struct {
	MeetingId          int64       `json:"meetingId"`
	Status             string      `json:"status"`
	Transcript         string      `json:"transcript,omitempty"`
	Summary            string      `json:"summary,omitempty"`
	Structured         *gjson.Json `json:"structured,omitempty"`
	SourceRecordingIds []int64     `json:"sourceRecordingIds,omitempty"`
	ErrorMsg           string      `json:"errorMsg,omitempty"`
	Model              string      `json:"model,omitempty"`
	GeneratedAt        *gtime.Time `json:"generatedAt,omitempty"`
}
