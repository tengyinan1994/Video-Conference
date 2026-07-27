package consts

const (
	// DefaultTokenTTL 默认 Token 有效期（秒）
	DefaultTokenTTL = 900
	// DefaultRateLimitPerMinute 同一 IP 每分钟最多签发次数
	DefaultRateLimitPerMinute = 30
	// MaxRoomNameLen 房间名最大长度
	MaxRoomNameLen = 64
	// MaxNicknameLen 昵称最大长度
	MaxNicknameLen = 32
	// RateLimitCachePrefix 限流缓存 key 前缀
	RateLimitCachePrefix = "conference:token:rate:"
)
