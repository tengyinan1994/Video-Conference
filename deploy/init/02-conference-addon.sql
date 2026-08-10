-- 部署初始化：启用视频会议插件（否则 /api/conference/* 不会注册）
INSERT INTO `hg_sys_addons_install` (`name`, `version`, `status`, `created_at`, `updated_at`)
SELECT 'conference', 'v1.0.0', 1, NOW(), NOW()
FROM DUAL
WHERE NOT EXISTS (
  SELECT 1 FROM `hg_sys_addons_install` WHERE `name` = 'conference'
);
