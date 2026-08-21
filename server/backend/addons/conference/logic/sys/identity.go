package sys

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"
)

// newMemberIdentity 每次进房生成唯一 identity，避免同账号多端/分享链接再进时顶掉已在房会话。
// 格式：u_{memberId}_{hex}
func newMemberIdentity(memberId int64) (string, error) {
	buf := make([]byte, 4)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return fmt.Sprintf("u_%d_%s", memberId, hex.EncodeToString(buf)), nil
}

// memberIdFromIdentity 从 u_{id} 或 u_{id}_{suffix} 解析用户 ID
func memberIdFromIdentity(identity string) int64 {
	if !strings.HasPrefix(identity, "u_") {
		return 0
	}
	rest := strings.TrimPrefix(identity, "u_")
	idPart := rest
	if i := strings.IndexByte(rest, '_'); i >= 0 {
		idPart = rest[:i]
	}
	id, err := strconv.ParseInt(idPart, 10, 64)
	if err != nil || id <= 0 {
		return 0
	}
	return id
}

// isEgressIdentity LiveKit Egress 进房 identity/显示名形如 EG_xxxx，不是真人参会者
func isEgressIdentity(identity string) bool {
	return strings.HasPrefix(strings.TrimSpace(identity), "EG_")
}

func isMeetingHostIdentity(identity string, hostId int64) bool {
	if hostId <= 0 || identity == "" {
		return false
	}
	return memberIdFromIdentity(identity) == hostId
}
