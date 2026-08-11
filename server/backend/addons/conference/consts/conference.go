package consts

import (
	"hotgo/internal/library/dict"
	"hotgo/internal/model"
)

func init() {
	dict.RegisterEnums("MeetingStatusOptions", "会议状态选项", MeetingStatusOptions)
}

const (
	// DefaultTokenTTL 默认 Token 有效期（秒）
	DefaultTokenTTL = 900
	// DefaultRateLimitPerMinute 同一 IP 每分钟最多签发次数
	DefaultRateLimitPerMinute = 30
	// MaxRoomNameLen 房间名最大长度
	MaxRoomNameLen = 64
	// MaxNicknameLen 昵称最大长度
	MaxNicknameLen = 32
	// MaxMeetingTitleLen 会议名称最大长度
	MaxMeetingTitleLen = 64
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

	// MeetingTable 业务会议室表
	MeetingTable = "hg_addon_conference_meeting"

	// MeetingStatusScheduled 预定
	MeetingStatusScheduled = "scheduled"
	// MeetingStatusOngoing 进行中（含结束后宽限期）
	MeetingStatusOngoing = "ongoing"
	// MeetingStatusEnded 已结束（手动结束或 end_at+宽限期）
	MeetingStatusEnded = "ended"
	// MeetingStatusReleased 旧版「已释放」，兼容读库后归一为 ended
	MeetingStatusReleased = "released"

	// MeetingReleaseGraceHours 结束后自动结束宽限（小时）
	MeetingReleaseGraceHours = 2

	// MeetingListTabOngoing 进行中
	MeetingListTabOngoing = "ongoing"
	// MeetingListTabScheduled 预定
	MeetingListTabScheduled = "scheduled"
	// MeetingListTabAll 全部（含历史已结束；已删除的不在表中）
	MeetingListTabAll = "all"
	// MeetingListTabEnded 历史（已结束）
	MeetingListTabEnded = "ended"
)

// MeetingStatusOptions 会议状态选项（管理端展示用有效状态）
var MeetingStatusOptions = []*model.Option{
	dict.GenInfoOption(MeetingStatusScheduled, "预定"),
	dict.GenSuccessOption(MeetingStatusOngoing, "进行中"),
	dict.GenDefaultOption(MeetingStatusEnded, "已结束"),
}
