-- 业务会议室表（不随空房销毁；主持人结束或 end_at+2h 后标记 ended；建错用硬删）
CREATE TABLE IF NOT EXISTS `hg_addon_conference_meeting` (
  `id` bigint(20) NOT NULL AUTO_INCREMENT COMMENT '主键',
  `title` varchar(128) NOT NULL DEFAULT '' COMMENT '会议名称',
  `room_name` varchar(64) NOT NULL DEFAULT '' COMMENT 'LiveKit 房间名',
  `host_id` bigint(20) NOT NULL DEFAULT 0 COMMENT '主持人用户ID',
  `host_name` varchar(64) NOT NULL DEFAULT '' COMMENT '主持人显示名',
  `start_at` datetime DEFAULT NULL COMMENT '开始时间',
  `end_at` datetime DEFAULT NULL COMMENT '结束时间',
  `status` varchar(32) NOT NULL DEFAULT 'scheduled' COMMENT 'scheduled/ongoing/ended',
  `share_code` varchar(32) NOT NULL DEFAULT '' COMMENT '分享短码',
  `created_by` bigint(20) NOT NULL DEFAULT 0 COMMENT '创建人',
  `created_at` datetime DEFAULT NULL COMMENT '创建时间',
  `updated_at` datetime DEFAULT NULL COMMENT '更新时间',
  `released_at` datetime DEFAULT NULL COMMENT '结束时间（手动或自动）',
  `attendees` json DEFAULT NULL COMMENT '参会显示名去重列表，如 ["张三","李四"]',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_room_name` (`room_name`),
  UNIQUE KEY `uk_share_code` (`share_code`),
  KEY `idx_status_end` (`status`, `end_at`),
  KEY `idx_host_id` (`host_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='视频会议-业务会议室';
