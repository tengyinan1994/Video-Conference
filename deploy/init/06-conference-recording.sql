-- 会议录制开关 + 多段录制表（幂等）

SET @exist := (
  SELECT COUNT(*)
  FROM information_schema.COLUMNS
  WHERE TABLE_SCHEMA = DATABASE()
    AND TABLE_NAME = 'hg_addon_conference_meeting'
    AND COLUMN_NAME = 'record_enabled'
);
SET @sql := IF(
  @exist = 0,
  'ALTER TABLE `hg_addon_conference_meeting` ADD COLUMN `record_enabled` tinyint(1) NOT NULL DEFAULT 0 COMMENT ''创建时是否自动开录（主持人进房起第一段）'' AFTER `attendees`',
  'SELECT 1'
);
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

CREATE TABLE IF NOT EXISTS `hg_addon_conference_recording` (
  `id` bigint(20) NOT NULL AUTO_INCREMENT COMMENT '主键',
  `meeting_id` bigint(20) NOT NULL DEFAULT 0 COMMENT '会议ID',
  `room_name` varchar(64) NOT NULL DEFAULT '' COMMENT 'LiveKit 房间名',
  `egress_id` varchar(128) NOT NULL DEFAULT '' COMMENT 'LiveKit Egress ID',
  `seq` int(11) NOT NULL DEFAULT 1 COMMENT '本场第几段',
  `status` varchar(32) NOT NULL DEFAULT 'starting' COMMENT 'starting/active/stopping/complete/failed',
  `object_key` varchar(512) NOT NULL DEFAULT '' COMMENT 'RustFS 对象路径',
  `file_size` bigint(20) NOT NULL DEFAULT 0 COMMENT '文件字节数',
  `started_at` datetime DEFAULT NULL COMMENT '开始时间',
  `ended_at` datetime DEFAULT NULL COMMENT '结束时间',
  `started_by` bigint(20) NOT NULL DEFAULT 0 COMMENT '发起人用户ID',
  `error_msg` varchar(512) NOT NULL DEFAULT '' COMMENT '失败原因',
  `created_at` datetime DEFAULT NULL COMMENT '创建时间',
  `updated_at` datetime DEFAULT NULL COMMENT '更新时间',
  PRIMARY KEY (`id`),
  KEY `idx_meeting_id` (`meeting_id`),
  KEY `idx_room_name` (`room_name`),
  KEY `idx_egress_id` (`egress_id`),
  KEY `idx_meeting_status` (`meeting_id`, `status`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='视频会议-录制分段';
