package room

import (
	"hotgo/addons/conference/model/input/sysin"

	"github.com/gogf/gf/v2/frame/g"
)

// KickReq 踢出参与者
type KickReq struct {
	g.Meta `path:"/room/kick" method:"post" tags:"视频会议" summary:"主持人踢人"`
	sysin.RoomKickInp
}

type KickRes struct{}

// MuteAllReq 全员静音（麦克风）
type MuteAllReq struct {
	g.Meta `path:"/room/muteAll" method:"post" tags:"视频会议" summary:"主持人全员静音"`
	sysin.RoomMuteAllInp
}

type MuteAllRes struct {
	*sysin.RoomMuteAllModel
}

// ClaimHostReq 进房后认领/同步主持人（空房或原主持已离会时可接任）
type ClaimHostReq struct {
	g.Meta `path:"/room/claimHost" method:"post" tags:"视频会议" summary:"认领主持人"`
	sysin.RoomClaimHostInp
}

type ClaimHostRes struct {
	*sysin.RoomClaimHostModel
}
