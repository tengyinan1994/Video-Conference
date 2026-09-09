-- 会议实际开始时间（首个参会者早于预定时间进房时记录，作为会议计时的起点；幂等）
SET NAMES utf8mb4;

SET @exist := (
  SELECT COUNT(*)
  FROM information_schema.COLUMNS
  WHERE TABLE_SCHEMA = DATABASE()
    AND TABLE_NAME = 'hg_addon_conference_meeting'
    AND COLUMN_NAME = 'started_at'
);
SET @sql := IF(
  @exist = 0,
  'ALTER TABLE `hg_addon_conference_meeting` ADD COLUMN `started_at` datetime DEFAULT NULL COMMENT ''实际开始时间（首个参会者提前入会时记录；无提前入会为空，计时以 start_at 为准）'' AFTER `record_enabled`',
  'SELECT 1'
);
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;
