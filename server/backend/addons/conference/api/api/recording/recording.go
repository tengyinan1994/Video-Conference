package recording

import (
	"hotgo/addons/conference/model/input/sysin"

	"github.com/gogf/gf/v2/frame/g"
)

type StartReq struct {
	g.Meta `path:"/recording/start" method:"post" tags:"视频会议录制" summary:"开始录制（主持人）"`
	sysin.RecordingStartInp
}

type StartRes struct {
	*sysin.RecordingStartModel
}

type StopReq struct {
	g.Meta `path:"/recording/stop" method:"post" tags:"视频会议录制" summary:"停止录制（主持人）"`
	sysin.RecordingStopInp
}

type StopRes struct {
	*sysin.RecordingStopModel
}

type StatusReq struct {
	g.Meta `path:"/recording/status" method:"get" tags:"视频会议录制" summary:"录制状态与分段列表"`
	sysin.RecordingStatusInp
}

type StatusRes struct {
	*sysin.RecordingStatusModel
}
