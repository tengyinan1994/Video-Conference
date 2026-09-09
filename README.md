# Video Conference

私有化视频会议 Demo：HotGo 业务底座 + LiveKit 媒体 + 独立 Vue3 会议客户端。

## 目录

| 路径 | 说明 |
|---|---|
| `server/` | 服务端整体（后端程序 + 管理后台） |
| `server/backend/` | HotGo Go 后端（GoFrame） |
| `server/backend/addons/conference` | 会议 Token 插件 |
| `server/web/` | 管理后台（Naive UI） |
| `client/` | 会议客户端（Tauri 可选） |
| `deploy/dev/` | 本机开发中间件（LiveKit、RustFS、Egress） |
| `deploy/test/` | dept 全量部署与一键脚本 |
| `deploy/images/` | 镜像打包 |

## 本机开发启动

> **换网后**：`./deploy/dev/ip.sh -w` 后重跑 `./deploy/dev/dev-livekit.sh`。

1. MySQL / Redis：复用本机 Middleware（如 `/Users/chaoming/Middleware/docker-compose.yml`）
2. 独立库：`video_conference`
3. LiveKit：`./deploy/dev/dev-livekit.sh`（7880/7881/7882）
   - 录制：`docker compose -f deploy/dev/docker-compose.yml --env-file deploy/dev/.env up -d rustfs`，再 `./deploy/dev/start-egress-when-ready.sh`
4. HotGo：`cd server/backend && air`
5. 客户端：`cd client && pnpm dev` → <https://127.0.0.1:5173>
6. 已有库补字段/菜单：按序执行 `deploy/test/init/` 下缺的增量脚本
   （如 `05-conference-meeting-attendees.sql`、`07-conference-meeting-actual-start.sql`、`08-conference-meeting-type.sql`）

开发模式客户端走 Vite 代理 `/rtc` → `127.0.0.1:7880`，后端 `livekit.url` 换网时通常不用改。

Token API：`POST /api/conference/token/create`，body：`{"room":"demo","nickname":"张三"}`

## dept 部署（test）

```bash
cp deploy/test/.env.example deploy/test/.env
cp deploy/test/config/config.example.yaml deploy/test/config/config.yaml
./deploy/test/deploy.sh full
```

详见 [`deploy/test/README.md`](deploy/test/README.md)。

- 会议客户端：`https://公网IP:17885/`
- 管理后台：`https://公网IP:17883/admin`
- LiveKit 信令：`ws://公网IP:17880`（会议页内走 `wss` 同源 `/rtc`）

单服务更新：`./deploy/test/deploy.sh update hotgo|client|admin|livekit`

## 数据库 MCP

项目级配置：`.cursor/mcp.json`。DSN 在 `.cursor/dbhub.env`，勿提交。
