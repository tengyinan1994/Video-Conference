-- 为已有会议表补充参会昵称列表字段（幂等）
SET @exist := (
  SELECT COUNT(*)
  FROM information_schema.COLUMNS
  WHERE TABLE_SCHEMA = DATABASE()
    AND TABLE_NAME = 'hg_addon_conference_meeting'
    AND COLUMN_NAME = 'attendees'
);
SET @sql := IF(
  @exist = 0,
  'ALTER TABLE `hg_addon_conference_meeting` ADD COLUMN `attendees` json DEFAULT NULL COMMENT ''参会显示名去重列表'' AFTER `released_at`',
  'SELECT 1'
);
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;
