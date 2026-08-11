package sysin

import (
	"context"
	"strings"
	"unicode/utf8"

	"hotgo/addons/conference/consts"

	"github.com/gogf/gf/v2/errors/gerror"
)

// RoomKickInp 踢人入参
type RoomKickInp struct {
	Room               string `json:"room" v:"required#房间名不能为空" dc:"房间名"`
	TargetIdentity     string `json:"targetIdentity" v:"required#被踢身份不能为空" dc:"被踢参与者 identity"`
	RequesterIdentity  string `json:"requesterIdentity" v:"required#操作者身份不能为空" dc:"主持人 identity"`
}

func (in *RoomKickInp) Filter(ctx context.Context) (err error) {
	in.Room = strings.TrimSpace(in.Room)
	in.TargetIdentity = strings.TrimSpace(in.TargetIdentity)
	in.RequesterIdentity = strings.TrimSpace(in.RequesterIdentity)
	if in.Room == "" {
		return gerror.New("房间名不能为空")
	}
	if utf8.RuneCountInString(in.Room) > consts.MaxRoomNameLen {
		return gerror.Newf("房间名最长 %d 个字符", consts.MaxRoomNameLen)
	}
	if !roomNamePattern.MatchString(in.Room) {
		return gerror.New("房间名仅支持字母、数字、下划线和中划线")
	}
	if in.TargetIdentity == "" {
		return gerror.New("被踢身份不能为空")
	}
	if in.RequesterIdentity == "" {
		return gerror.New("操作者身份不能为空")
	}
	if in.TargetIdentity == in.RequesterIdentity {
		return gerror.New("不能踢出自己")
	}
	return
}

// RoomMuteAllInp 全员静音入参
type RoomMuteAllInp struct {
	Room              string `json:"room" v:"required#房间名不能为空" dc:"房间名"`
	RequesterIdentity string `json:"requesterIdentity" v:"required#操作者身份不能为空" dc:"主持人 identity"`
}

func (in *RoomMuteAllInp) Filter(ctx context.Context) (err error) {
	in.Room = strings.TrimSpace(in.Room)
	in.RequesterIdentity = strings.TrimSpace(in.RequesterIdentity)
	if in.Room == "" {
		return gerror.New("房间名不能为空")
	}
	if utf8.RuneCountInString(in.Room) > consts.MaxRoomNameLen {
		return gerror.Newf("房间名最长 %d 个字符", consts.MaxRoomNameLen)
	}
	if !roomNamePattern.MatchString(in.Room) {
		return gerror.New("房间名仅支持字母、数字、下划线和中划线")
	}
	if in.RequesterIdentity == "" {
		return gerror.New("操作者身份不能为空")
	}
	return
}

// RoomMuteAllModel 全员静音结果
type RoomMuteAllModel struct {
	MutedCount int `json:"mutedCount" dc:"成功静音的麦克风轨数量"`
}

// RoomClaimHostInp 进房后认领/同步主持人
type RoomClaimHostInp struct {
	Room              string `json:"room" v:"required#房间名不能为空" dc:"房间名"`
	RequesterIdentity string `json:"requesterIdentity" v:"required#操作者身份不能为空" dc:"当前参与者 identity"`
}

func (in *RoomClaimHostInp) Filter(ctx context.Context) (err error) {
	in.Room = strings.TrimSpace(in.Room)
	in.RequesterIdentity = strings.TrimSpace(in.RequesterIdentity)
	if in.Room == "" {
		return gerror.New("房间名不能为空")
	}
	if utf8.RuneCountInString(in.Room) > consts.MaxRoomNameLen {
		return gerror.Newf("房间名最长 %d 个字符", consts.MaxRoomNameLen)
	}
	if !roomNamePattern.MatchString(in.Room) {
		return gerror.New("房间名仅支持字母、数字、下划线和中划线")
	}
	if in.RequesterIdentity == "" {
		return gerror.New("操作者身份不能为空")
	}
	return
}

// RoomClaimHostModel 认领主持结果
type RoomClaimHostModel struct {
	IsHost bool `json:"isHost" dc:"是否为预定主持人"`
}
