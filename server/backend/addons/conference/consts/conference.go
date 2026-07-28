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
	// HostCachePrefix 房间主持人缓存 key 前缀
	HostCachePrefix = "conference:room:host:"
	// HostCacheTTL 主持人标记缓存时长
	HostCacheTTL = 2 * 60 * 60
	// RoleHost 参与者 metadata 角色：主持人
	RoleHost = "host"
	// RoleMember 参与者 metadata 角色：普通成员
	RoleMember = "member"
)
