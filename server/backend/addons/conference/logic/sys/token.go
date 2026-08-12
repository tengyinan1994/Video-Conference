package sys

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"hotgo/addons/conference/consts"
	"hotgo/addons/conference/model/entity"
	"hotgo/addons/conference/model/input/sysin"
	"hotgo/addons/conference/service"
	"hotgo/internal/library/cache"
	"hotgo/internal/library/contexts"
	"hotgo/internal/library/token"
	"hotgo/internal/model"

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
	if err = s.checkRateLimit(ctx, cfg.RateLimitPerMinute); err != nil {
		return
	}

	user := optionalLoginUser(ctx)

	var meeting *entity.Meeting
	if in.ShareCode != "" {
		meeting, err = service.SysMeeting().GetByShareCode(ctx, in.ShareCode)
		if err != nil {
			return
		}
		if meeting == nil {
			return nil, gerror.New("会议不存在或链接无效")
		}
	} else {
		if user == nil {
			return nil, gerror.New("请先登录后再进入会议")
		}
		meeting, err = service.SysMeeting().GetByRoomName(ctx, in.Room)
		if err != nil {
			return
		}
		if meeting == nil {
			return nil, gerror.New("会议室不存在，请从大厅进入或使用分享链接")
		}
	}

	if err = service.SysMeeting().AssertJoinable(ctx, meeting); err != nil {
		return
	}

	if user == nil {
		if !cfg.AllowAnonymousToken {
			return nil, gerror.New("当前未开放游客进房，请先登录")
		}
		if in.ShareCode == "" {
			return nil, gerror.New("请先登录后再进入会议")
		}
	}

	identity, isHost, err := s.resolveIdentityAndHost(ctx, meeting, user)
	if err != nil {
		return
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
		Room:     meeting.RoomName,
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

	jwtToken, err := at.ToJWT()
	if err != nil {
		return nil, gerror.Wrap(err, "签发会议 Token 失败")
	}

	// 进房必经 Token；不依赖 LiveKit webhook（本机 --dev 常无 webhook）
	if err = service.SysMeeting().AppendAttendee(ctx, meeting.RoomName, in.Nickname); err != nil {
		g.Log().Warningf(ctx, "append attendee on token create failed room=%s nick=%s err=%+v", meeting.RoomName, in.Nickname, err)
	}

	res = &sysin.TokenCreateModel{
		ServerUrl: cfg.Url,
		Room:      meeting.RoomName,
		Title:     meeting.Title,
		Identity:  identity,
		Nickname:  in.Nickname,
		Token:     jwtToken,
		ExpiresAt: expiresAt,
		IsHost:    isHost,
	}
	return
}

func optionalLoginUser(ctx context.Context) *model.Identity {
	if u := contexts.GetUser(ctx); u != nil && u.Id > 0 {
		return u
	}
	r := ghttp.RequestFromCtx(ctx)
	if r == nil {
		return nil
	}
	user, err := token.ParseLoginUser(r)
	if err != nil || user == nil || user.Id <= 0 {
		return nil
	}
	contexts.SetUser(ctx, user)
	return user
}

func (s *sSysToken) resolveIdentityAndHost(ctx context.Context, meeting *entity.Meeting, user *model.Identity) (identity string, isHost bool, err error) {
	if user != nil {
		// 每次进房唯一 identity，防止同账号再进时踢掉已在房会话
		identity, err = newMemberIdentity(user.Id)
		if err != nil {
			return "", false, gerror.Wrap(err, "生成参与者身份失败")
		}
		// 主持人固定为会议室预定人，不转让、不抢占
		isHost = meeting.HostId == user.Id
		return identity, isHost, nil
	}

	identity, err = generateIdentity()
	if err != nil {
		return "", false, gerror.Wrap(err, "生成参与者身份失败")
	}
	return identity, false, nil
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
	return fmt.Sprintf("g_%s", hex.EncodeToString(buf)), nil
}
