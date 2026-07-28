package api

import (
	"context"

	"hotgo/addons/conference/api/api/room"
	"hotgo/addons/conference/service"
)

var (
	Room = cRoom{}
)

type cRoom struct{}

func (c *cRoom) Kick(ctx context.Context, req *room.KickReq) (res *room.KickRes, err error) {
	err = service.SysRoom().Kick(ctx, &req.RoomKickInp)
	if err != nil {
		return
	}
	res = new(room.KickRes)
	return
}

func (c *cRoom) MuteAll(ctx context.Context, req *room.MuteAllReq) (res *room.MuteAllRes, err error) {
	data, err := service.SysRoom().MuteAll(ctx, &req.RoomMuteAllInp)
	if err != nil {
		return
	}
	res = new(room.MuteAllRes)
	res.RoomMuteAllModel = data
	return
}

func (c *cRoom) ClaimHost(ctx context.Context, req *room.ClaimHostReq) (res *room.ClaimHostRes, err error) {
	data, err := service.SysRoom().ClaimHost(ctx, &req.RoomClaimHostInp)
	if err != nil {
		return
	}
	res = new(room.ClaimHostRes)
	res.RoomClaimHostModel = data
	return
}
