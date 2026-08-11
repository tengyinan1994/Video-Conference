-- hotgo 会议管理菜单权限 SQL
-- Date: 2026-08-11
-- 管理员可查看全部会议，并对任意状态会议进行编辑/结束/删除

SET @now := NOW();

-- 避免重复插入
DELETE FROM `hg_admin_role_menu` WHERE `menu_id` IN (
  SELECT id FROM (
    SELECT id FROM `hg_admin_menu` WHERE `name` IN (
      'Conference', 'conferenceMeeting', 'conferenceMeetingView',
      'conferenceMeetingEdit', 'conferenceMeetingDelete', 'conferenceMeetingRelease'
    )
  ) t
);
DELETE FROM `hg_admin_menu` WHERE `name` IN (
  'Conference', 'conferenceMeeting', 'conferenceMeetingView',
  'conferenceMeetingEdit', 'conferenceMeetingDelete', 'conferenceMeetingRelease'
);

-- 一级目录：会议管理
INSERT INTO `hg_admin_menu` (
  `pid`, `title`, `name`, `path`, `icon`, `type`, `redirect`, `permissions`, `permission_name`,
  `component`, `always_show`, `active_menu`, `is_root`, `is_frame`, `frame_src`, `keep_alive`,
  `hidden`, `affix`, `level`, `tree`, `sort`, `remark`, `status`, `created_at`, `updated_at`
) VALUES (
  0, '会议管理', 'Conference', '/conference', 'VideoCameraOutlined', 1, '/conference/meeting', '', '',
  'LAYOUT', 1, '', 0, 0, '', 0,
  0, 0, 1, '', 1, '视频会议管理', 1, @now, @now
);

SET @dirId = LAST_INSERT_ID();
UPDATE `hg_admin_menu` SET `tree` = CONCAT('tr_', @dirId, ' ') WHERE `id` = @dirId;

-- 菜单页面：会议列表
INSERT INTO `hg_admin_menu` (
  `pid`, `title`, `name`, `path`, `icon`, `type`, `redirect`, `permissions`, `permission_name`,
  `component`, `always_show`, `active_menu`, `is_root`, `is_frame`, `frame_src`, `keep_alive`,
  `hidden`, `affix`, `level`, `tree`, `sort`, `remark`, `status`, `created_at`, `updated_at`
) VALUES (
  @dirId, '会议列表', 'conferenceMeeting', 'meeting', '', 2, '', '/conference/meeting/list', '',
  '/addons/conference/meeting/index', 1, 'Conference', 0, 0, '', 1,
  0, 0, 2, CONCAT('tr_', @dirId, ' '), 10, '', 1, @now, @now
);

SET @listId = LAST_INSERT_ID();
UPDATE `hg_admin_menu` SET `tree` = CONCAT('tr_', @dirId, ' tr_', @listId, ' ') WHERE `id` = @listId;

INSERT INTO `hg_admin_menu` (
  `pid`, `title`, `name`, `path`, `icon`, `type`, `redirect`, `permissions`, `permission_name`,
  `component`, `always_show`, `active_menu`, `is_root`, `is_frame`, `frame_src`, `keep_alive`,
  `hidden`, `affix`, `level`, `tree`, `sort`, `remark`, `status`, `created_at`, `updated_at`
) VALUES
(@listId, '会议详情', 'conferenceMeetingView', '', '', 3, '', '/conference/meeting/view', '', '', 1, '', 0, 0, '', 0, 1, 0, 3, CONCAT('tr_', @dirId, ' tr_', @listId, ' '), 10, '', 1, @now, @now),
(@listId, '编辑/新建会议', 'conferenceMeetingEdit', '', '', 3, '', '/conference/meeting/edit', '', '', 1, '', 0, 0, '', 0, 1, 0, 3, CONCAT('tr_', @dirId, ' tr_', @listId, ' '), 20, '', 1, @now, @now),
(@listId, '删除会议', 'conferenceMeetingDelete', '', '', 3, '', '/conference/meeting/delete', '', '', 1, '', 0, 0, '', 0, 1, 0, 3, CONCAT('tr_', @dirId, ' tr_', @listId, ' '), 40, '', 1, @now, @now),
(@listId, '结束会议', 'conferenceMeetingRelease', '', '', 3, '', '/conference/meeting/release', '', '', 1, '', 0, 0, '', 0, 1, 0, 3, CONCAT('tr_', @dirId, ' tr_', @listId, ' '), 50, '', 1, @now, @now);
