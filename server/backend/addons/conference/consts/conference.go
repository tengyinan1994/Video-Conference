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
	// MaxMeetingTypeNameLen 会议类型名称最大长度（会议端标签需完整展示，故限制为 10）
	MaxMeetingTypeNameLen = 10
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
	// MeetingTypeTable 会议类型表
	MeetingTypeTable = "hg_addon_conference_meeting_type"
	// RecordingTable 录制分段表
	RecordingTable = "hg_addon_conference_recording"
	// MinutesTable 会后 AI 纪要表
	MinutesTable = "hg_addon_conference_minutes"

	RecordingStatusStarting = "starting"
	RecordingStatusActive   = "active"
	RecordingStatusStopping = "stopping"
	RecordingStatusComplete = "complete"
	RecordingStatusFailed   = "failed"

	// RecordingPurposePlayback 给人回放的录制
	RecordingPurposePlayback = "playback"
	// RecordingPurposeAI 仅给纪要用的音频采集（不进回放列表）
	RecordingPurposeAI = "ai"

	MinutesStatusPending       = "pending"
	MinutesStatusTranscribing  = "transcribing"
	MinutesStatusSummarizing   = "summarizing"
	MinutesStatusReady         = "ready"
	MinutesStatusFailed        = "failed"
	MinutesStatusSkippedEmpty  = "skipped_empty"
	MinutesStatusUnavailable   = "unavailable"

	// MeetingStatusScheduled 预定
	MeetingStatusScheduled = "scheduled"
	// MeetingStatusOngoing 进行中（已到预定开始时间，或首个真人已进房；到预定结束时间后若仍有人则继续计时）
	MeetingStatusOngoing = "ongoing"
	// MeetingStatusEnded 已结束（手动结束或预定时长到期后无人自动结束）
	MeetingStatusEnded = "ended"
	// MeetingStatusReleased 旧版「已释放」，兼容读库后归一为 ended
	MeetingStatusReleased = "released"

	// MeetingEarlyJoinMinutes 允许提前进入的分钟数（大厅与游客分享页一致）
	MeetingEarlyJoinMinutes = 5

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
