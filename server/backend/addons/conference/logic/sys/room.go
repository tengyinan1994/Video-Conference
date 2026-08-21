package sys

import (
	"context"
	"encoding/json"

	"hotgo/addons/conference/consts"
	"hotgo/addons/conference/model/input/sysin"
	"hotgo/addons/conference/service"

	"github.com/gogf/gf/v2/errors/gerror"
	"github.com/gogf/gf/v2/frame/g"
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
	if isEgressIdentity(in.TargetIdentity) {
		return gerror.New("不能对录制服务执行此操作")
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
		if isEgressIdentity(p.Identity) || isEgressIdentity(p.Name) {
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

// ClaimHost 仅同步「预定主持人」的 metadata，不再支持空房接任/转让。
func (s *sSysRoom) ClaimHost(ctx context.Context, in *sysin.RoomClaimHostInp) (res *sysin.RoomClaimHostModel, err error) {
	if err = in.Filter(ctx); err != nil {
		return
	}
	res = &sysin.RoomClaimHostModel{IsHost: false}

	meeting, err := service.SysMeeting().GetByRoomName(ctx, in.Room)
	if err != nil {
		return nil, err
	}
	if meeting == nil {
		return res, nil
	}
	if !isMeetingHostIdentity(in.RequesterIdentity, meeting.HostId) {
		return res, nil
	}

	client, _, err := newRoomServiceClient(ctx)
	if err != nil {
		return
	}
	participants, err := listRoomParticipants(ctx, client, in.Room)
	if err != nil {
		return
	}
	inRoom := false
	for _, p := range participants {
		if p != nil && p.Identity == in.RequesterIdentity {
			inRoom = true
			break
		}
	}
	if !inRoom {
		return nil, gerror.New("你不在该房间内")
	}

	meta, _ := json.Marshal(map[string]string{"role": consts.RoleHost})
	_, _ = client.UpdateParticipant(ctx, &livekit.UpdateParticipantRequest{
		Room:     in.Room,
		Identity: in.RequesterIdentity,
		Metadata: string(meta),
	})
	res.IsHost = true
	if meeting.RecordEnabled != 0 {
		if autoErr := service.SysRecording().TryAutoStart(ctx, meeting, meeting.HostId); autoErr != nil {
			g.Log().Warningf(ctx, "conference auto-start recording on claimHost failed meeting=%d err=%+v", meeting.Id, autoErr)
		}
	}
	return
}

// assertHost 以业务会议室 host_id 为准，不依赖 Redis 抢占/接任。
func (s *sSysRoom) assertHost(ctx context.Context, room, requesterIdentity string) error {
	meeting, err := service.SysMeeting().GetByRoomName(ctx, room)
	if err != nil {
		return err
	}
	if meeting == nil {
		return gerror.New("会议室不存在")
	}
	if !isMeetingHostIdentity(requesterIdentity, meeting.HostId) {
		return gerror.New("仅预定主持人可执行此操作")
	}
	return nil
}
