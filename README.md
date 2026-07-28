# Video Conference

私有化视频会议 Demo：HotGo 业务底座 + LiveKit 媒体 + 独立 Vue3 会议客户端。

## 目录

| 路径 | 说明 |
|---|---|
| `server/` | 服务端整体（后端程序 + 管理后台） |
| `server/backend/` | HotGo Go 后端（GoFrame） |
| `server/backend/addons/conference` | 会议 Token 插件 |
| `server/web/` | 管理后台（Naive UI） |
| `client/` | 会议客户端（后续套 Tauri） |
| `视频会议/` | 设计与学习路线 |

## 本机启动

1. MySQL / Redis：已用 `/Users/chaoming/Middleware/docker-compose.yml`
2. 独立库：`video_conference`（不要用其他项目的 `hotgo` 库）
3. LiveKit：`livekit-server --dev`（或已有 Docker 容器）
4. HotGo：`cd server/backend && air`
5. 客户端：`cd client && pnpm dev` → <http://127.0.0.1:5173>

Token API：`POST /api/conference/token/create`，body：`{"room":"demo","nickname":"张三"}`

## 数据库 MCP

项目级配置：`.cursor/mcp.json`（只读账号）。改完后需**重启 Cursor** 才会加载。DSN 在 `.cursor/dbhub.env`，勿提交。
