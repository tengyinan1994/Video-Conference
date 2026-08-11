package crons

import (
	"context"

	"hotgo/addons/conference/service"
	"hotgo/internal/library/cron"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gcron"
	"github.com/gogf/gf/v2/os/gctx"
)

func init() {
	// 注册到 HotGo 任务中心（可在后台配置）
	cron.Register(MeetingAutoRelease)

	// 同时直接挂到 gcron，确保部署后无需再配后台任务也能跑
	_, err := gcron.AddSingleton(gctx.GetInitCtx(), "0 */5 * * * *", func(ctx context.Context) {
		count, err := service.SysMeeting().AutoReleaseExpired(ctx)
		if err != nil {
			g.Log().Warningf(ctx, "conference auto release err: %+v", err)
			return
		}
		if count > 0 {
			g.Log().Infof(ctx, "conference auto released %d meeting(s)", count)
		}
	}, "conferenceMeetingAutoRelease")
	if err != nil {
		g.Log().Warningf(gctx.GetInitCtx(), "register conference auto release cron failed: %+v", err)
	}
}

var MeetingAutoRelease = &cMeetingAutoRelease{name: "conferenceMeetingAutoRelease"}

type cMeetingAutoRelease struct{ name string }

func (c *cMeetingAutoRelease) GetName() string { return c.name }

func (c *cMeetingAutoRelease) Execute(ctx context.Context, parser *cron.Parser) (err error) {
	count, err := service.SysMeeting().AutoReleaseExpired(ctx)
	if err != nil {
		return err
	}
	if parser != nil && parser.Logger != nil {
		parser.Logger.Infof(ctx, "conference auto released %d meeting(s)", count)
	}
	return nil
}
