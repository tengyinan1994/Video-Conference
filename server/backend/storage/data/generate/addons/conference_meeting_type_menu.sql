-- hotgo 会议类型菜单权限 SQL
-- Date: 2026-09-09
-- 管理端「会议管理 → 会议类型」：维护会议类型（仅名称），供会议端新建会议时选择
-- 与 deploy/prod/init/08-conference-meeting-type.sql 的菜单段保持一致
SET NAMES utf8mb4;

SET @now := NOW();

-- 避免重复插入
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

-- 父级「会议管理」目录；父级缺失则跳过，避免生成游离的一级菜单
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
