package api

import (
	"bytes"
	"context"
	"io"
	"strings"

	"hotgo/addons/conference/model"
	"hotgo/addons/conference/service"

	"github.com/gogf/gf/v2/errors/gerror"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/net/ghttp"
	"github.com/livekit/protocol/auth"
	"github.com/livekit/protocol/webhook"
)

const livekitEventParticipantJoined = "participant_joined"

// HandleLiveKitWebhook LiveKit 进房 webhook：去重追加参会昵称
func HandleLiveKitWebhook(r *ghttp.Request) {
	ctx := r.Context()

	cfg, err := loadWebhookLiveKitConfig(ctx)
	if err != nil {
		g.Log().Warningf(ctx, "conference webhook: load livekit config failed: %+v", err)
		r.Response.WriteStatus(500)
		return
	}

	// 中间件可能已读过 Body，恢复给 LiveKit 校验用
	if body := r.GetBody(); len(body) > 0 {
		r.Request.Body = io.NopCloser(bytes.NewReader(body))
	}

	provider := auth.NewSimpleKeyProvider(cfg.ApiKey, cfg.ApiSecret)
	event, err := webhook.ReceiveWebhookEvent(r.Request, provider)
	if err != nil {
		g.Log().Warningf(ctx, "conference webhook: verify failed: %+v", err)
		r.Response.WriteStatus(401)
		return
	}

	if event.GetEvent() != livekitEventParticipantJoined {
		r.Response.WriteStatus(200)
		return
	}

	room := ""
	if event.GetRoom() != nil {
		room = strings.TrimSpace(event.GetRoom().GetName())
	}
	displayName := ""
	if p := event.GetParticipant(); p != nil {
		displayName = strings.TrimSpace(p.GetName())
		if displayName == "" {
			displayName = strings.TrimSpace(p.GetIdentity())
		}
	}
	if room == "" || displayName == "" {
		r.Response.WriteStatus(200)
		return
	}

	if err = service.SysMeeting().AppendAttendee(ctx, room, displayName); err != nil {
		g.Log().Warningf(ctx, "conference webhook: append attendee failed room=%s name=%s err=%+v", room, displayName, err)
		r.Response.WriteStatus(500)
		return
	}
	r.Response.WriteStatus(200)
}

func loadWebhookLiveKitConfig(ctx context.Context) (cfg *model.LiveKitConfig, err error) {
	cfg = &model.LiveKitConfig{
		TokenTTL:            900,
		AllowAnonymousToken: true,
		RateLimitPerMinute:  30,
	}
	if err = g.Cfg().MustGet(ctx, "livekit").Scan(cfg); err != nil {
		return nil, gerror.Wrap(err, "读取 LiveKit 配置失败")
	}
	if cfg.Url == "" || cfg.ApiKey == "" || cfg.ApiSecret == "" {
		return nil, gerror.New("LiveKit 配置不完整，请检查 livekit.url / apiKey / apiSecret")
	}
	return cfg, nil
}
