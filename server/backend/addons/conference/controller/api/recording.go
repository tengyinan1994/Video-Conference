package api

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"

	"hotgo/addons/conference/api/api/recording"
	"hotgo/addons/conference/model/input/sysin"
	"hotgo/addons/conference/service"
	"hotgo/internal/library/response"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/net/ghttp"
)

// RecordingPublic 公开录制查询（全员可见「录制中」）
var RecordingPublic = cRecordingPublic{}

type cRecordingPublic struct{}

func (c *cRecordingPublic) Status(ctx context.Context, req *recording.StatusReq) (res *recording.StatusRes, err error) {
	data, err := service.SysRecording().Status(ctx, &req.RecordingStatusInp)
	if err != nil {
		return
	}
	res = &recording.StatusRes{RecordingStatusModel: data}
	return
}

// Recording 需登录的录制控制
var Recording = cRecording{}

type cRecording struct{}

func (c *cRecording) Start(ctx context.Context, req *recording.StartReq) (res *recording.StartRes, err error) {
	data, err := service.SysRecording().Start(ctx, &req.RecordingStartInp)
	if err != nil {
		return
	}
	res = &recording.StartRes{RecordingStartModel: data}
	return
}

func (c *cRecording) Stop(ctx context.Context, req *recording.StopReq) (res *recording.StopRes, err error) {
	data, err := service.SysRecording().Stop(ctx, &req.RecordingStopInp)
	if err != nil {
		return
	}
	res = &recording.StopRes{RecordingStopModel: data}
	return
}

// HandleRecordingDownload 走业务鉴权后从 RustFS 拉文件，触发浏览器下载（避免直链被当成在线播放）
func HandleRecordingDownload(r *ghttp.Request) {
	ctx := r.Context()
	id := r.Get("id").Int64()
	obj, filename, size, err := service.SysRecording().OpenForDownload(ctx, id)
	if err != nil {
		response.JsonExit(r, 1, err.Error())
		return
	}
	defer obj.Close()

	r.Response.Header().Set("Content-Type", "application/octet-stream")
	r.Response.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	if size > 0 {
		r.Response.Header().Set("Content-Length", strconv.FormatInt(size, 10))
	}
	if _, err = io.Copy(r.Response.Writer, obj); err != nil {
		g.Log().Warningf(ctx, "conference download recording failed id=%d err=%+v", id, err)
	}
}

// HandleRecordingPlay 在线回放：支持 Range，供 <video> 边下边播
func HandleRecordingPlay(r *ghttp.Request) {
	ctx := r.Context()
	id := r.Get("id").Int64()
	st, err := service.SysRecording().OpenForPlay(ctx, id, r.Header.Get("Range"))
	if err != nil {
		var unsat *sysin.UnsatisfiableRangeError
		if errors.As(err, &unsat) {
			r.Response.Header().Set("Accept-Ranges", "bytes")
			r.Response.Header().Set("Content-Range", fmt.Sprintf("bytes */%d", unsat.Size))
			r.Response.Header().Set("Content-Type", "video/mp4")
			r.Response.RawWriter().WriteHeader(http.StatusRequestedRangeNotSatisfiable)
			return
		}
		response.JsonExit(r, 1, err.Error())
		return
	}
	defer st.Body.Close()

	r.Response.Header().Set("Accept-Ranges", "bytes")
	r.Response.Header().Set("Content-Type", "video/mp4")
	r.Response.Header().Set("Content-Disposition", fmt.Sprintf(`inline; filename="%s"`, st.Filename))
	r.Response.Header().Set("Cache-Control", "private, max-age=0, no-transform")
	r.Response.Header().Set("X-Content-Type-Options", "nosniff")
	r.Response.Header().Set("Access-Control-Expose-Headers", "Content-Range, Accept-Ranges, Content-Length, Content-Type")

	raw := r.Response.RawWriter()
	if st.Partial {
		length := st.End - st.Start + 1
		if length < 0 {
			length = 0
		}
		r.Response.Header().Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", st.Start, st.End, st.Size))
		r.Response.Header().Set("Content-Length", strconv.FormatInt(length, 10))
		raw.WriteHeader(http.StatusPartialContent)
	} else {
		if st.Size > 0 {
			r.Response.Header().Set("Content-Length", strconv.FormatInt(st.Size, 10))
		}
		r.Response.Status = http.StatusOK
	}

	if _, err = io.Copy(raw, st.Body); err != nil {
		g.Log().Warningf(ctx, "conference play recording failed id=%d err=%+v", id, err)
	}
}
