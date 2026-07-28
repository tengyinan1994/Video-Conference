package sys

import (
	"context"
	"strings"

	"hotgo/addons/conference/consts"
	"hotgo/addons/conference/model"

	"github.com/gogf/gf/v2/errors/gerror"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/livekit/protocol/livekit"
	lksdk "github.com/livekit/server-sdk-go/v2"
)

func loadLiveKitConfig(ctx context.Context) (cfg *model.LiveKitConfig, err error) {
	cfg = &model.LiveKitConfig{
		TokenTTL:            consts.DefaultTokenTTL,
		AllowAnonymousToken: true,
		RateLimitPerMinute:  consts.DefaultRateLimitPerMinute,
	}
	if err = g.Cfg().MustGet(ctx, "livekit").Scan(cfg); err != nil {
		return nil, gerror.Wrap(err, "读取 LiveKit 配置失败")
	}
	if cfg.Url == "" || cfg.ApiKey == "" || cfg.ApiSecret == "" {
		return nil, gerror.New("LiveKit 配置不完整，请检查 livekit.url / apiKey / apiSecret")
	}
	if cfg.TokenTTL <= 0 {
		cfg.TokenTTL = consts.DefaultTokenTTL
	}
	if cfg.RateLimitPerMinute <= 0 {
		cfg.RateLimitPerMinute = consts.DefaultRateLimitPerMinute
	}
	return
}

// httpURLFromLiveKit 将 ws(s):// 转为 RoomService 所需的 http(s)://
func httpURLFromLiveKit(url string) string {
	u := strings.TrimSpace(url)
	switch {
	case strings.HasPrefix(u, "wss://"):
		return "https://" + strings.TrimPrefix(u, "wss://")
	case strings.HasPrefix(u, "ws://"):
		return "http://" + strings.TrimPrefix(u, "ws://")
	default:
		return u
	}
}

func newRoomServiceClient(ctx context.Context) (*lksdk.RoomServiceClient, *model.LiveKitConfig, error) {
	cfg, err := loadLiveKitConfig(ctx)
	if err != nil {
		return nil, nil, err
	}
	client := lksdk.NewRoomServiceClient(httpURLFromLiveKit(cfg.Url), cfg.ApiKey, cfg.ApiSecret)
	return client, cfg, nil
}

func listRoomParticipants(ctx context.Context, client *lksdk.RoomServiceClient, room string) ([]*livekit.ParticipantInfo, error) {
	res, err := client.ListParticipants(ctx, &livekit.ListParticipantsRequest{Room: room})
	if err != nil {
		// 房间尚不存在时视为空房
		msg := err.Error()
		if strings.Contains(msg, "not found") || strings.Contains(msg, "does not exist") || strings.Contains(msg, "room not found") {
			return nil, nil
		}
		return nil, gerror.Wrap(err, "查询房间参与者失败")
	}
	if res == nil {
		return nil, nil
	}
	return res.Participants, nil
}
