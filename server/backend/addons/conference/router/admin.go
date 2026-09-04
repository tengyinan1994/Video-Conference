package router

import (
	"context"

	"hotgo/addons/conference/controller/api"
	"hotgo/addons/conference/global"
	"hotgo/addons/conference/router/genrouter"
	"hotgo/internal/consts"
	"hotgo/internal/library/addons"
	"hotgo/internal/service"

	"github.com/gogf/gf/v2/net/ghttp"
)

// Admin 后台路由
func Admin(ctx context.Context, group *ghttp.RouterGroup) {
	prefix := addons.RouterPrefix(ctx, consts.AppAdmin, global.GetSkeleton().Name)
	group.Group(prefix, func(group *ghttp.RouterGroup) {
		group.Middleware(service.Middleware().AdminAuth)
		// 回放/下载走 HotGo 代理，避免 HTTPS 后台直链 HTTP RustFS 被浏览器拦截
		group.GET("/recording/download", api.HandleRecordingDownload)
		group.GET("/recording/play", api.HandleRecordingPlay)
	})
	genrouter.Register(ctx, group)
}
