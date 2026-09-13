package api

import (
	"fmt"
	"io"
	"strconv"

	"hotgo/addons/conference/service"
	"hotgo/internal/library/response"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/net/ghttp"
)

// HandleClientDownload 走业务鉴权后从 RustFS releases 桶代理下载客户端安装包
func HandleClientDownload(r *ghttp.Request) {
	ctx := r.Context()
	platform := r.Get("platform").String()
	obj, filename, size, err := service.SysClientRelease().OpenForDownload(ctx, platform)
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
		g.Log().Warningf(ctx, "conference download client release failed platform=%s err=%+v", platform, err)
	}
}
