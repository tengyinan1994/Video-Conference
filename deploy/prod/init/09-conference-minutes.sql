-- AI 纪要：录制分段 purpose + 纪要表（幂等）
SET NAMES utf8mb4;

SET @exist := (
  SELECT COUNT(*)
  FROM information_schema.COLUMNS
  WHERE TABLE_SCHEMA = DATABASE()
    AND TABLE_NAME = 'hg_addon_conference_recording'
    AND COLUMN_NAME = 'purpose'
);
SET @sql := IF(
  @exist = 0,
  'ALTER TABLE `hg_addon_conference_recording` ADD COLUMN `purpose` varchar(16) NOT NULL DEFAULT ''playback'' COMMENT ''playback=回放录制 ai=纪要音源'' AFTER `seq`',
  'SELECT 1'
);
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @idx := (
  SELECT COUNT(*)
  FROM information_schema.STATISTICS
  WHERE TABLE_SCHEMA = DATABASE()
    AND TABLE_NAME = 'hg_addon_conference_recording'
    AND INDEX_NAME = 'idx_meeting_purpose_status'
);
SET @sql := IF(
  @idx = 0,
  'ALTER TABLE `hg_addon_conference_recording` ADD KEY `idx_meeting_purpose_status` (`meeting_id`, `purpose`, `status`)',
  'SELECT 1'
);
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

CREATE TABLE IF NOT EXISTS `hg_addon_conference_minutes` (
  `id` bigint(20) NOT NULL AUTO_INCREMENT COMMENT '主键',
  `meeting_id` bigint(20) NOT NULL DEFAULT 0 COMMENT '会议ID',
  `status` varchar(32) NOT NULL DEFAULT 'pending' COMMENT 'pending/transcribing/summarizing/ready/failed/skipped_empty/unavailable',
  `transcript` mediumtext COMMENT '完整转写',
  `summary` text COMMENT '纪要正文',
  `structured` json DEFAULT NULL COMMENT 'todos/decisions/risks',
  `source_recording_ids` json DEFAULT NULL COMMENT '引用的 ai 分段 id',
  `error_msg` varchar(512) NOT NULL DEFAULT '' COMMENT '失败原因',
  `model` varchar(128) NOT NULL DEFAULT '' COMMENT 'LLM 模型名',
  `generated_at` datetime DEFAULT NULL COMMENT '生成完成时间',
  `created_at` datetime DEFAULT NULL COMMENT '创建时间',
  `updated_at` datetime DEFAULT NULL COMMENT '更新时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_meeting_id` (`meeting_id`),
  KEY `idx_status` (`status`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='视频会议-会后AI纪要';
