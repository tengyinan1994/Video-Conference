-- 部署初始化：会议类型（管理端「会议管理 → 会议类型」维护）+ 会议表关联字段 + 菜单权限
-- 说明：本文件在全新数据目录由 docker-entrypoint-initdb.d 自动执行；
--       已有库需手动执行（见 deploy/prod/README.md）：
--       docker exec -i vc-mysql mysql -uroot -p$MYSQL_ROOT_PASSWORD $MYSQL_DATABASE < init/08-conference-meeting-type.sql
-- 必须声明客户端字符集，否则 docker initdb 默认 latin1 会把中文二次编码成乱码
SET NAMES utf8mb4;

-- ========== 1. 会议类型表 ==========
CREATE TABLE IF NOT EXISTS `hg_addon_conference_meeting_type` (
  `id` bigint(20) NOT NULL AUTO_INCREMENT COMMENT '主键',
  `name` varchar(32) NOT NULL DEFAULT '' COMMENT '类型名称',
  `created_at` datetime DEFAULT NULL COMMENT '创建时间',
  `updated_at` datetime DEFAULT NULL COMMENT '更新时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_name` (`name`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='视频会议-会议类型';

-- ========== 2. 会议表增加会议类型字段（0=未分类，幂等） ==========
SET @exist := (
  SELECT COUNT(*)
  FROM information_schema.COLUMNS
  WHERE TABLE_SCHEMA = DATABASE()
    AND TABLE_NAME = 'hg_addon_conference_meeting'
    AND COLUMN_NAME = 'type_id'
);
SET @sql := IF(
  @exist = 0,
  'ALTER TABLE `hg_addon_conference_meeting` ADD COLUMN `type_id` bigint(20) NOT NULL DEFAULT 0 COMMENT ''会议类型ID，0=未分类'' AFTER `attendees`',
  'SELECT 1'
);
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

-- ========== 3. 管理后台「会议类型」菜单 ==========
SET @now := NOW();

-- 避免重复插入（可重复执行）
DELETE FROM `hg_admin_role_menu` WHERE `menu_id` IN (
  SELECT id FROM (
    SELECT id FROM `hg_admin_menu` WHERE `name` IN (
      'conferenceMeetingType', 'conferenceMeetingTypeView',
      'conferenceMeetingTypeEdit', 'conferenceMeetingTypeDelete'
    )
  ) t
);
DELETE FROM `hg_admin_menu` WHERE `name` IN (
  'conferenceMeetingType', 'conferenceMeetingTypeView',
  'conferenceMeetingTypeEdit', 'conferenceMeetingTypeDelete'
);

-- 父级「会议管理」目录由 04-conference-menu.sql 创建；此处只挂子菜单，父级缺失则跳过
SET @dirId := (SELECT id FROM `hg_admin_menu` WHERE `name` = 'Conference' AND `pid` = 0 LIMIT 1);
SET @typeListId := 0;

-- 菜单页面：会议类型
INSERT INTO `hg_admin_menu` (
  `pid`, `title`, `name`, `path`, `icon`, `type`, `redirect`, `permissions`, `permission_name`,
  `component`, `always_show`, `active_menu`, `is_root`, `is_frame`, `frame_src`, `keep_alive`,
  `hidden`, `affix`, `level`, `tree`, `sort`, `remark`, `status`, `created_at`, `updated_at`
)
SELECT
  @dirId, '会议类型', 'conferenceMeetingType', 'meetingType', '', 2, '', '/conference/meetingType/list', '',
  '/addons/conference/meetingType/index', 1, 'Conference', 0, 0, '', 1,
  0, 0, 2, CONCAT('tr_', @dirId, ' '), 20, '会议类型维护', 1, @now, @now
FROM DUAL
WHERE @dirId IS NOT NULL;

SET @typeListId := IFNULL((SELECT id FROM `hg_admin_menu` WHERE `name` = 'conferenceMeetingType' LIMIT 1), 0);
UPDATE `hg_admin_menu` SET `tree` = CONCAT('tr_', @dirId, ' tr_', @typeListId, ' ') WHERE `id` = @typeListId;

-- 按钮权限：详情 / 编辑 / 删除
INSERT INTO `hg_admin_menu` (
  `pid`, `title`, `name`, `path`, `icon`, `type`, `redirect`, `permissions`, `permission_name`,
  `component`, `always_show`, `active_menu`, `is_root`, `is_frame`, `frame_src`, `keep_alive`,
  `hidden`, `affix`, `level`, `tree`, `sort`, `remark`, `status`, `created_at`, `updated_at`
)
SELECT
  @typeListId, '会议类型详情', 'conferenceMeetingTypeView', '', '', 3, '', '/conference/meetingType/view', '',
  '', 1, '', 0, 0, '', 0,
  1, 0, 3, CONCAT('tr_', @dirId, ' tr_', @typeListId, ' '), 10, '', 1, @now, @now
FROM DUAL
WHERE @typeListId > 0;

INSERT INTO `hg_admin_menu` (
  `pid`, `title`, `name`, `path`, `icon`, `type`, `redirect`, `permissions`, `permission_name`,
  `component`, `always_show`, `active_menu`, `is_root`, `is_frame`, `frame_src`, `keep_alive`,
  `hidden`, `affix`, `level`, `tree`, `sort`, `remark`, `status`, `created_at`, `updated_at`
)
SELECT
  @typeListId, '编辑/新建会议类型', 'conferenceMeetingTypeEdit', '', '', 3, '', '/conference/meetingType/edit', '',
  '', 1, '', 0, 0, '', 0,
  1, 0, 3, CONCAT('tr_', @dirId, ' tr_', @typeListId, ' '), 20, '', 1, @now, @now
FROM DUAL
WHERE @typeListId > 0;

INSERT INTO `hg_admin_menu` (
  `pid`, `title`, `name`, `path`, `icon`, `type`, `redirect`, `permissions`, `permission_name`,
  `component`, `always_show`, `active_menu`, `is_root`, `is_frame`, `frame_src`, `keep_alive`,
  `hidden`, `affix`, `level`, `tree`, `sort`, `remark`, `status`, `created_at`, `updated_at`
)
SELECT
  @typeListId, '删除会议类型', 'conferenceMeetingTypeDelete', '', '', 3, '', '/conference/meetingType/delete', '',
  '', 1, '', 0, 0, '', 0,
  1, 0, 3, CONCAT('tr_', @dirId, ' tr_', @typeListId, ' '), 30, '', 1, @now, @now
FROM DUAL
WHERE @typeListId > 0;

-- 注：超管角色无需 hg_admin_role_menu 授权即可看到全部菜单；
--     非超管角色需在后台「权限管理 → 角色管理」勾选该菜单。
