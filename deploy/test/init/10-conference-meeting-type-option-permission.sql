-- 修复：非超管角色点击「新建/编辑会议」时提示「你没有访问权限！」
--
-- 会议编辑弹窗打开时会请求 /conference/meetingType/option 拉取会议类型下拉项，
-- 该接口此前没有任何菜单声明，非超管角色调用即被后台鉴权拦截：
-- 弹窗弹出且会议仍能保存（保存走 /conference/meeting/edit，权限正常），但会弹出报错提示。
-- 这里把选项接口并入「会议列表」菜单权限，随会议列表一起授权。
--
-- 幂等，可重复执行；更新后需重启 HotGo（casbin 在启动时按角色-菜单重新装载），
-- 或在后台「权限管理 → 角色权限」保存一次触发刷新。
SET NAMES utf8mb4;

UPDATE `hg_admin_menu`
SET `permissions` = TRIM(BOTH ',' FROM CONCAT(`permissions`, ',/conference/meetingType/option')),
    `updated_at`  = NOW()
WHERE `name` = 'conferenceMeeting'
  AND `permissions` NOT LIKE '%/conference/meetingType/option%';
