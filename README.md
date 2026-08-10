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
| `deploy/` | Docker 部署（compose + 镜像打包） |

## 本机开发启动

1. MySQL / Redis：已用 `/Users/chaoming/Middleware/docker-compose.yml`，或本仓库 `deploy/docker-compose.yml` 只起中间件
2. 独立库：`video_conference`（不要用其他项目的 `hotgo` 库）
3. LiveKit：`docker compose -f deploy/docker-compose.yml --env-file deploy/.env up livekit`
4. HotGo：`cd server/backend && air`
5. 客户端：`cd client && pnpm dev` → <http://127.0.0.1:5173>

Token API：`POST /api/conference/token/create`，body：`{"room":"demo","nickname":"张三"}`

## Docker 部署

```bash
cp deploy/.env.example deploy/.env
cp deploy/config/config.example.yaml deploy/config/config.yaml
# 编辑 .env 的 LIVEKIT_NODE_IP、ARCH；编辑 config.yaml 的 livekit.url

# 打包镜像（按目标架构二选一）
./deploy/images/build-amd64.sh
# ./deploy/images/build-arm64.sh

docker compose -f deploy/docker-compose.yml --env-file deploy/.env up -d
```

- 会议客户端：<https://宿主机:17885>（自签证书，需浏览器点「继续访问」；摄像头/麦克风依赖 HTTPS）
- 管理后台：<http://宿主机:17883/admin>
- LiveKit 信令：`ws://宿主机:17880`（HTTPS 会议页会改走同源 `wss://宿主机:17885`，由 nginx 反代 `/rtc`）
- 宿主机对外端口：`17880–17883`、`17885`（LiveKit / HotGo / 客户端；mysql·redis 不映射）

离线机：将 `deploy/images/amd64/*.tar`（或 `arm64`）拷过去后 `docker load -i ...`，再 `compose up`。

## 数据库 MCP

项目级配置：`.cursor/mcp.json`（只读账号）。改完后需**重启 Cursor** 才会加载。DSN 在 `.cursor/dbhub.env`，勿提交。
