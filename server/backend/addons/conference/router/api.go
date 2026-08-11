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

// Api 前台路由：公开接口在前，需登录接口走 ApiAuth
func Api(ctx context.Context, group *ghttp.RouterGroup) {
	prefix := addons.RouterPrefix(ctx, consts.AppApi, global.GetSkeleton().Name)
	group.Group(prefix, func(group *ghttp.RouterGroup) {
		group.Bind(
			api.AuthPublic,
			api.MeetingPublic,
			api.Token,
			api.Room,
		)

		group.Group("/", func(group *ghttp.RouterGroup) {
			group.Middleware(service.Middleware().ApiAuth)
			group.Bind(
				api.Auth,
				api.Meeting,
			)
		})
	})
}
