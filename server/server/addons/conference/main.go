package conference

import (
	"context"
	"sync"

	"hotgo/addons/conference/global"
	_ "hotgo/addons/conference/logic"
	"hotgo/addons/conference/router"
	"hotgo/internal/library/addons"
	"hotgo/internal/service"

	"github.com/gogf/gf/v2/net/ghttp"
	"github.com/gogf/gf/v2/os/gctx"
)

type module struct {
	skeleton *addons.Skeleton
	ctx      context.Context
	sync.Mutex
}

func init() {
	newModule()
}

func newModule() {
	m := &module{
		skeleton: &addons.Skeleton{
			Label:       "视频会议",
			Name:        "conference",
			Group:       1,
			Logo:        "",
			Brief:       "LiveKit 会议 Token 与会控能力",
			Description: "为独立会议客户端签发 LiveKit 进房 Token，后续扩展预约与会控",
			Author:      "Video-Conference",
			Version:     "v1.0.0",
		},
		ctx: gctx.New(),
	}
	addons.RegisterModule(m)
}

func (m *module) Start(option *addons.Option) (err error) {
	global.Init(m.ctx, m.skeleton)
	option.Server.Group("/", func(group *ghttp.RouterGroup) {
		group.Middleware(service.Middleware().Addon)
		router.Api(m.ctx, group)
	})
	return
}

func (m *module) Stop() (err error) {
	return
}

func (m *module) Ctx() context.Context {
	return m.ctx
}

func (m *module) GetSkeleton() *addons.Skeleton {
	return m.skeleton
}

func (m *module) Install(ctx context.Context) (err error) {
	return
}

func (m *module) Upgrade(ctx context.Context) (err error) {
	return
}

func (m *module) UnInstall(ctx context.Context) (err error) {
	return
}
