package sys

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"hotgo/addons/conference/consts"
	"hotgo/addons/conference/model/input/sysin"
	"hotgo/addons/conference/service"
	"hotgo/internal/library/cache"

	"github.com/gogf/gf/v2/errors/gerror"
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

type participantMeta struct {
	Role string `json:"role"`
}

func (s *sSysToken) Create(ctx context.Context, in *sysin.TokenCreateInp) (res *sysin.TokenCreateModel, err error) {
	if err = in.Filter(ctx); err != nil {
		return
	}

	cfg, err := loadLiveKitConfig(ctx)
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

	isHost, err := s.assignHost(ctx, in.Room, identity)
	if err != nil {
		return nil, err
	}

	ttl := time.Duration(cfg.TokenTTL) * time.Second
	expiresAt := time.Now().Add(ttl).Unix()

	role := consts.RoleMember
	if isHost {
		role = consts.RoleHost
	}
	metaBytes, _ := json.Marshal(participantMeta{Role: role})

	at := auth.NewAccessToken(cfg.ApiKey, cfg.ApiSecret)
	grant := &auth.VideoGrant{
		RoomJoin: true,
		Room:     in.Room,
	}
	grant.SetCanPublish(true)
	grant.SetCanSubscribe(true)
	grant.SetCanPublishData(true)
	if isHost {
		grant.RoomAdmin = true
	}

	at.SetVideoGrant(grant).
		SetIdentity(identity).
		SetName(in.Nickname).
		SetMetadata(string(metaBytes)).
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
		IsHost:    isHost,
	}
	return
}

// assignHost：按「房间维度」抢占主持人，与昵称无关。
// 规则：
// 1) Redis 里尚无主持人 → 当前取 Token 者成为主持人
// 2) 房间在 LiveKit 侧已空（全员离会）→ 清掉旧主持缓存，重新抢占（避免「空房永远无主持」）
// 3) 房内还有人，但缓存里的主持人不在房内 → 当前取 Token 者接任
// 4) 否则仍为普通成员
func (s *sSysToken) assignHost(ctx context.Context, room, identity string) (isHost bool, err error) {
	key := consts.HostCachePrefix + room
	hostTTL := time.Duration(consts.HostCacheTTL) * time.Second

	client, _, err := newRoomServiceClient(ctx)
	if err != nil {
		return false, err
	}
	participants, err := listRoomParticipants(ctx, client, room)
	if err != nil {
		return false, err
	}

	live := make(map[string]struct{}, len(participants))
	for _, p := range participants {
		if p != nil && p.Identity != "" {
			live[p.Identity] = struct{}{}
		}
	}

	// 空房：旧主持缓存作废，重新抢（并发时只有一人能 SetIfNotExist 成功）
	if len(live) == 0 {
		if _, remErr := cache.Instance().Remove(ctx, key); remErr != nil {
			return false, gerror.Wrap(remErr, "清除空房主持人缓存失败")
		}
		ok, setErr := cache.Instance().SetIfNotExist(ctx, key, identity, hostTTL)
		if setErr != nil {
			return false, gerror.Wrap(setErr, "写入主持人缓存失败")
		}
		return ok, nil
	}

	ok, err := cache.Instance().SetIfNotExist(ctx, key, identity, hostTTL)
	if err != nil {
		return false, gerror.Wrap(err, "写入主持人缓存失败")
	}
	if ok {
		return true, nil
	}

	val, err := cache.Instance().Get(ctx, key)
	if err != nil {
		return false, gerror.Wrap(err, "读取主持人缓存失败")
	}
	currentHost := ""
	if val != nil && !val.IsNil() {
		currentHost = val.String()
	}
	// 房内已有人，但缓存主持人已不在 → 接任
	if currentHost == "" {
		if err = cache.Instance().Set(ctx, key, identity, hostTTL); err != nil {
			return false, gerror.Wrap(err, "写入主持人缓存失败")
		}
		return true, nil
	}
	if _, inRoom := live[currentHost]; !inRoom {
		if err = cache.Instance().Set(ctx, key, identity, hostTTL); err != nil {
			return false, gerror.Wrap(err, "写入主持人缓存失败")
		}
		return true, nil
	}

	return false, nil
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
