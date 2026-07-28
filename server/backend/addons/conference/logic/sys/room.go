package sys

import (
	"context"
	"encoding/json"
	"time"

	"hotgo/addons/conference/consts"
	"hotgo/addons/conference/model/input/sysin"
	"hotgo/addons/conference/service"
	"hotgo/internal/library/cache"

	"github.com/gogf/gf/v2/errors/gerror"
	"github.com/livekit/protocol/livekit"
)

type sSysRoom struct{}

func NewSysRoom() *sSysRoom {
	return &sSysRoom{}
}

func init() {
	service.RegisterSysRoom(NewSysRoom())
}

func (s *sSysRoom) Kick(ctx context.Context, in *sysin.RoomKickInp) (err error) {
	if err = in.Filter(ctx); err != nil {
		return
	}
	if err = s.assertHost(ctx, in.Room, in.RequesterIdentity); err != nil {
		return
	}

	client, _, err := newRoomServiceClient(ctx)
	if err != nil {
		return
	}
	_, err = client.RemoveParticipant(ctx, &livekit.RoomParticipantIdentity{
		Room:     in.Room,
		Identity: in.TargetIdentity,
	})
	if err != nil {
		return gerror.Wrap(err, "踢出参与者失败")
	}
	return
}

func (s *sSysRoom) MuteAll(ctx context.Context, in *sysin.RoomMuteAllInp) (res *sysin.RoomMuteAllModel, err error) {
	if err = in.Filter(ctx); err != nil {
		return
	}
	if err = s.assertHost(ctx, in.Room, in.RequesterIdentity); err != nil {
		return
	}

	client, _, err := newRoomServiceClient(ctx)
	if err != nil {
		return
	}
	participants, err := listRoomParticipants(ctx, client, in.Room)
	if err != nil {
		return
	}

	muted := 0
	for _, p := range participants {
		if p == nil || p.Identity == in.RequesterIdentity {
			continue
		}
		for _, t := range p.Tracks {
			if t == nil {
				continue
			}
			if t.Type != livekit.TrackType_AUDIO {
				continue
			}
			if t.Source != livekit.TrackSource_MICROPHONE && t.Source != livekit.TrackSource_UNKNOWN {
				continue
			}
			if t.Muted {
				continue
			}
			_, muteErr := client.MutePublishedTrack(ctx, &livekit.MuteRoomTrackRequest{
				Room:     in.Room,
				Identity: p.Identity,
				TrackSid: t.Sid,
				Muted:    true,
			})
			if muteErr != nil {
				return nil, gerror.Wrapf(muteErr, "静音 %s 失败", p.Identity)
			}
			muted++
		}
	}

	res = &sysin.RoomMuteAllModel{MutedCount: muted}
	return
}

// ClaimHost 进房后同步/接任主持人：
// - 本来就是缓存中的主持 → 续期并同步 metadata
// - 房内只剩自己，或原主持已不在房 → 接任
// - 否则保持普通成员
func (s *sSysRoom) ClaimHost(ctx context.Context, in *sysin.RoomClaimHostInp) (res *sysin.RoomClaimHostModel, err error) {
	if err = in.Filter(ctx); err != nil {
		return
	}

	client, _, err := newRoomServiceClient(ctx)
	if err != nil {
		return
	}
	participants, err := listRoomParticipants(ctx, client, in.Room)
	if err != nil {
		return
	}

	live := make(map[string]struct{}, len(participants))
	inRoom := false
	for _, p := range participants {
		if p == nil || p.Identity == "" {
			continue
		}
		live[p.Identity] = struct{}{}
		if p.Identity == in.RequesterIdentity {
			inRoom = true
		}
	}
	if !inRoom {
		return nil, gerror.New("你不在该房间内，无法认领主持人")
	}

	key := consts.HostCachePrefix + in.Room
	hostTTL := time.Duration(consts.HostCacheTTL) * time.Second

	val, err := cache.Instance().Get(ctx, key)
	if err != nil {
		return nil, gerror.Wrap(err, "读取主持人信息失败")
	}
	currentHost := ""
	if val != nil && !val.IsNil() {
		currentHost = val.String()
	}

	canClaim := false
	switch {
	case currentHost == in.RequesterIdentity:
		canClaim = true
	case currentHost == "":
		canClaim = true
	case len(live) == 1:
		// 房里只剩自己：无论缓存挂着谁，都应能控场
		canClaim = true
	default:
		if _, ok := live[currentHost]; !ok {
			canClaim = true
		}
	}

	res = &sysin.RoomClaimHostModel{IsHost: false}
	if !canClaim {
		return res, nil
	}

	if err = cache.Instance().Set(ctx, key, in.RequesterIdentity, hostTTL); err != nil {
		return nil, gerror.Wrap(err, "写入主持人缓存失败")
	}
	meta, _ := json.Marshal(map[string]string{"role": consts.RoleHost})
	_, updErr := client.UpdateParticipant(ctx, &livekit.UpdateParticipantRequest{
		Room:     in.Room,
		Identity: in.RequesterIdentity,
		Metadata: string(meta),
	})
	if updErr != nil {
		// 缓存已写上，metadata 失败不阻断（前端仍可按 isHost 显示会控）
		_ = updErr
	}
	res.IsHost = true
	return
}

func (s *sSysRoom) assertHost(ctx context.Context, room, requesterIdentity string) error {
	key := consts.HostCachePrefix + room
	val, err := cache.Instance().Get(ctx, key)
	if err != nil {
		return gerror.Wrap(err, "读取主持人信息失败")
	}
	if val == nil || val.IsNil() || val.String() == "" {
		return gerror.New("房间暂无主持人，请重新进房后再试")
	}
	if val.String() != requesterIdentity {
		return gerror.New("仅主持人可执行此操作")
	}
	// 续期，避免长会中途过期
	_ = cache.Instance().Set(ctx, key, requesterIdentity, time.Duration(consts.HostCacheTTL)*time.Second)
	return nil
}
