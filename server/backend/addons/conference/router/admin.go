package router

import (
	"context"

	"hotgo/addons/conference/router/genrouter"

	"github.com/gogf/gf/v2/net/ghttp"
)

// Admin 后台路由
func Admin(ctx context.Context, group *ghttp.RouterGroup) {
	genrouter.Register(ctx, group)
}
