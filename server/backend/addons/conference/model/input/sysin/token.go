package sysin

import (
	"context"
	"regexp"
	"strings"
	"unicode/utf8"

	"hotgo/addons/conference/consts"

	"github.com/gogf/gf/v2/errors/gerror"
)

var roomNamePattern = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)

// TokenCreateInp 创建会议 Token 入参
type TokenCreateInp struct {
	Room      string `json:"room" dc:"房间名（大厅进房必填；分享进房可省略）"`
	Nickname  string `json:"nickname" v:"required#昵称不能为空" dc:"显示昵称"`
	ShareCode string `json:"shareCode" dc:"分享短码（分享链接进房）"`
}

func (in *TokenCreateInp) Filter(ctx context.Context) (err error) {
	in.Room = strings.TrimSpace(in.Room)
	in.Nickname = strings.TrimSpace(in.Nickname)
	in.ShareCode = strings.TrimSpace(in.ShareCode)

	if in.ShareCode == "" && in.Room == "" {
		return gerror.New("请指定会议房间或分享码")
	}
	if in.Room != "" {
		if utf8.RuneCountInString(in.Room) > consts.MaxRoomNameLen {
			return gerror.Newf("房间名最长 %d 个字符", consts.MaxRoomNameLen)
		}
		if strings.Contains(in.Room, "..") || strings.Contains(in.Room, "/") {
			return gerror.New("房间名包含非法字符")
		}
		if !roomNamePattern.MatchString(in.Room) {
			return gerror.New("房间名仅支持字母、数字、下划线和中划线")
		}
	}

	if in.Nickname == "" {
		return gerror.New("昵称不能为空")
	}
	if utf8.RuneCountInString(in.Nickname) > consts.MaxNicknameLen {
		return gerror.Newf("昵称最长 %d 个字符", consts.MaxNicknameLen)
	}
	return
}

// TokenCreateModel 创建会议 Token 出参
type TokenCreateModel struct {
	ServerUrl string `json:"serverUrl" dc:"LiveKit 服务地址"`
	Room      string `json:"room" dc:"房间名"`
	Title     string `json:"title" dc:"会议名称"`
	Identity  string `json:"identity" dc:"参与者身份（服务端生成）"`
	Nickname  string `json:"nickname" dc:"显示昵称"`
	Token     string `json:"token" dc:"进房 JWT"`
	ExpiresAt int64  `json:"expiresAt" dc:"过期时间戳（秒）"`
	IsHost    bool   `json:"isHost" dc:"是否为本房间主持人"`
}
