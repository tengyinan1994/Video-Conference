package sysin

import (
	"context"
	"io"
	"strings"

	"github.com/gogf/gf/v2/errors/gerror"
	"github.com/gogf/gf/v2/os/gtime"
)

// RecordingStartInp 开始一段录制
type RecordingStartInp struct {
	Room string `json:"room" v:"required#房间名不能为空" dc:"LiveKit 房间名"`
}

func (in *RecordingStartInp) Filter(ctx context.Context) (err error) {
	in.Room = strings.TrimSpace(in.Room)
	if in.Room == "" {
		return gerror.New("房间名不能为空")
	}
	return
}

// RecordingStopInp 停止当前录制段
type RecordingStopInp struct {
	Room string `json:"room" v:"required#房间名不能为空" dc:"LiveKit 房间名"`
}

func (in *RecordingStopInp) Filter(ctx context.Context) (err error) {
	in.Room = strings.TrimSpace(in.Room)
	if in.Room == "" {
		return gerror.New("房间名不能为空")
	}
	return
}

// RecordingStatusInp 查询录制状态
type RecordingStatusInp struct {
	Room      string `json:"room" dc:"房间名"`
	MeetingId int64  `json:"meetingId" dc:"会议ID"`
}

func (in *RecordingStatusInp) Filter(ctx context.Context) (err error) {
	in.Room = strings.TrimSpace(in.Room)
	if in.Room == "" && in.MeetingId <= 0 {
		return gerror.New("请提供房间名或会议ID")
	}
	return
}

// RecordingSegmentModel 录制段
type RecordingSegmentModel struct {
	Id          int64       `json:"id"`
	MeetingId   int64       `json:"meetingId"`
	RoomName    string      `json:"roomName"`
	EgressId    string      `json:"egressId"`
	Seq         int         `json:"seq"`
	Status      string      `json:"status"`
	ObjectKey   string      `json:"objectKey"`
	FileSize    int64       `json:"fileSize"`
	StartedAt   *gtime.Time `json:"startedAt"`
	EndedAt     *gtime.Time `json:"endedAt"`
	ErrorMsg    string      `json:"errorMsg"`
	PlayUrl     string      `json:"playUrl,omitempty" dc:"完成段的临时播放地址"`
	DownloadUrl string      `json:"downloadUrl,omitempty" dc:"完成段的临时下载地址（附件）"`
}

// UnsatisfiableRangeError HTTP 416
type UnsatisfiableRangeError struct {
	Size int64
}

func (e *UnsatisfiableRangeError) Error() string {
	return "requested range not satisfiable"
}

// RecordingStatusModel 录制状态
type RecordingStatusModel struct {
	Active   bool                     `json:"active" dc:"是否有进行中的录制"`
	Segments []*RecordingSegmentModel `json:"segments"`
}

// RecordingStartModel 开始录制结果
type RecordingStartModel struct {
	*RecordingSegmentModel
}

// RecordingStopModel 停止录制结果
type RecordingStopModel struct {
	*RecordingSegmentModel
}

// RecordingPlayStream 回放流（HTTP 层按 Range 写出，不走 JSON）
type RecordingPlayStream struct {
	Body     io.ReadCloser
	Filename string
	Size     int64
	Start    int64
	End      int64
	Partial  bool
}
