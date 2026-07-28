package service

import (
	"context"

	"hotgo/addons/conference/model/input/sysin"
)

type (
	ISysToken interface {
		Create(ctx context.Context, in *sysin.TokenCreateInp) (res *sysin.TokenCreateModel, err error)
	}
	ISysRoom interface {
		Kick(ctx context.Context, in *sysin.RoomKickInp) (err error)
		MuteAll(ctx context.Context, in *sysin.RoomMuteAllInp) (res *sysin.RoomMuteAllModel, err error)
		ClaimHost(ctx context.Context, in *sysin.RoomClaimHostInp) (res *sysin.RoomClaimHostModel, err error)
	}
)

var (
	localSysToken ISysToken
	localSysRoom  ISysRoom
)

func SysToken() ISysToken {
	if localSysToken == nil {
		panic("implement not found for interface ISysToken, forgot register?")
	}
	return localSysToken
}

func RegisterSysToken(i ISysToken) {
	localSysToken = i
}

func SysRoom() ISysRoom {
	if localSysRoom == nil {
		panic("implement not found for interface ISysRoom, forgot register?")
	}
	return localSysRoom
}

func RegisterSysRoom(i ISysRoom) {
	localSysRoom = i
}
