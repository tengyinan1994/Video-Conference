package sys

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"hotgo/addons/conference/consts"
	"hotgo/addons/conference/model"
	"hotgo/addons/conference/model/input/sysin"
	"hotgo/addons/conference/service"
	"hotgo/internal/library/cache"

	"github.com/gogf/gf/v2/errors/gerror"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/net/ghttp"
	"github.com/livekit/protocol/auth"
)

type sSysToken struct{}

func NewSysToken() *sSysToken {
	return &sSysToken{}
}

func init() {
	service.RegisterSysToken(NewSysToken())
}

func (s *sSysToken) Create(ctx context.Context, in *sysin.TokenCreateInp) (res *sysin.TokenCreateModel, err error) {
	if err = in.Filter(ctx); err != nil {
		return
	}

	cfg, err := s.loadConfig(ctx)
	if err != nil {
		return
	}
	if !cfg.AllowAnonymousToken {
		return nil, gerror.New("当前未开放匿名进房，请先登录")
	}
	if err = s.checkRateLimit(ctx, cfg.RateLimitPerMinute); err != nil {
		return
	}

	identity, err := generateIdentity()
	if err != nil {
		return nil, gerror.Wrap(err, "生成参与者身份失败")
	}

	ttl := time.Duration(cfg.TokenTTL) * time.Second
	expiresAt := time.Now().Add(ttl).Unix()

	at := auth.NewAccessToken(cfg.ApiKey, cfg.ApiSecret)
	grant := &auth.VideoGrant{
		RoomJoin: true,
		Room:     in.Room,
	}
	grant.SetCanPublish(true)
	grant.SetCanSubscribe(true)
	grant.SetCanPublishData(true)

	at.SetVideoGrant(grant).
		SetIdentity(identity).
		SetName(in.Nickname).
		SetValidFor(ttl)

	token, err := at.ToJWT()
	if err != nil {
		return nil, gerror.Wrap(err, "签发会议 Token 失败")
	}

	res = &sysin.TokenCreateModel{
		ServerUrl: cfg.Url,
		Room:      in.Room,
		Identity:  identity,
		Nickname:  in.Nickname,
		Token:     token,
		ExpiresAt: expiresAt,
	}
	return
}

func (s *sSysToken) loadConfig(ctx context.Context) (cfg *model.LiveKitConfig, err error) {
	cfg = &model.LiveKitConfig{
		TokenTTL:           consts.DefaultTokenTTL,
		AllowAnonymousToken: true,
		RateLimitPerMinute: consts.DefaultRateLimitPerMinute,
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

func (s *sSysToken) checkRateLimit(ctx context.Context, limit int) error {
	r := ghttp.RequestFromCtx(ctx)
	ip := "unknown"
	if r != nil {
		ip = r.GetClientIp()
	}
	key := consts.RateLimitCachePrefix + ip
	val, err := cache.Instance().Get(ctx, key)
	if err != nil {
		return gerror.Wrap(err, "读取限流计数失败")
	}
	count := 0
	if val != nil && !val.IsNil() {
		count = val.Int()
	}
	if count >= limit {
		return gerror.New("请求过于频繁，请稍后再试")
	}
	count++
	if err = cache.Instance().Set(ctx, key, count, time.Minute); err != nil {
		return gerror.Wrap(err, "写入限流计数失败")
	}
	return nil
}

func generateIdentity() (string, error) {
	buf := make([]byte, 8)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return fmt.Sprintf("u_%s", hex.EncodeToString(buf)), nil
}
