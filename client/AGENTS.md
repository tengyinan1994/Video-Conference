# 本文件约束 client/ 会议客户端。
# server/AGENTS.md 仅约束 HotGo 管理后台，不适用于本工程。
# UI 使用 Ant Design Vue；HTTP 自行封装，不复用 server/web 的 axios。
#
# 形态：Vue 3 前端（src/）+ Tauri 2 壳（src-tauri/），一套代码两种分发。
# 网页：pnpm dev / pnpm build
# 桌面：pnpm tauri:dev / pnpm tauri:build
# 会议业务只改 src/；系统能力（托盘、权限、updater）才动 src-tauri/。
# 不要拆成 client/web 与 client/desktop 两套前端。
