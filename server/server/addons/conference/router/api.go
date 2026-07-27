package router

import (
	"context"

	"hotgo/addons/conference/controller/api"
	"hotgo/addons/conference/global"
	"hotgo/internal/consts"
	"hotgo/internal/library/addons"
	"hotgo/internal/service"

	"github.com/gogf/gf/v2/net/ghttp"
)

// Api 前台路由（公开 Token 接口放在 ApiAuth 之前）
func Api(ctx context.Context, group *ghttp.RouterGroup) {
	prefix := addons.RouterPrefix(ctx, consts.AppApi, global.GetSkeleton().Name)
	group.Group(prefix, func(group *ghttp.RouterGroup) {
		group.Bind(
			api.Token,
		)
		group.Middleware(service.Middleware().ApiAuth)
	})
}
