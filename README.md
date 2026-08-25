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

> **换网后必做**：本机 IP 会变，先跑 `./deploy/ip.sh -w`（自动检测新 IP 并写入 `deploy/.env` 的 `LIVEKIT_NODE_IP` 和 `deploy/config/config.yaml` 的 `livekit.url`），再重启 LiveKit。

1. MySQL / Redis：已用 `/Users/chaoming/Middleware/docker-compose.yml`，或本仓库 `deploy/docker-compose.yml` 只起中间件
2. 独立库：`video_conference`（不要用其他项目的 `hotgo` 库）
3. LiveKit：`./deploy/dev-livekit.sh`（本机开发用，7880 信令 / 7881 TCP / 7882 UDP，自动用当前局域网 IP 重建 `livekit-dev` 容器，密钥与后端 `manifest/config/config.yaml` 一致）
   - 若要跑 compose 版（17880 端口）：`docker compose -f deploy/docker-compose.yml --env-file deploy/.env up livekit`
   - 若用 brew：`livekit-server --config deploy/livekit.yaml` 或 `--dev` 均可；**参会名单在签发 Token 时写入**，不依赖 webhook
4. HotGo：`cd server/backend && air`
5. 客户端：`cd client && pnpm dev` → <http://127.0.0.1:5173>
6. 已有库补字段：执行 `deploy/init/05-conference-meeting-attendees.sql`

> 开发模式下客户端始终走同源 `/rtc`（Vite 代理 → `127.0.0.1:7880`），因此后端 `livekit.url` 换网时**不用改**；只有 Tauri 安装包 / 局域网对端直连才需要它指向可达地址。

Token API：`POST /api/conference/token/create`，body：`{"room":"demo","nickname":"张三"}`（成功签发后会把昵称写入会议 `attendees`）

Webhook（可选增强）：`POST /api/conference/webhook/livekit`（与 Token 路径去重追加同一字段）

## Docker 部署

```bash
cp deploy/.env.example deploy/.env
cp deploy/config/config.example.yaml deploy/config/config.yaml
# 用 ./deploy/ip.sh -w 自动写入当前局域网 IP（LIVEKIT_NODE_IP + livekit.url）；
# 手动改 ARCH 为 amd64|arm64（与打包脚本一致）
./deploy/ip.sh -w

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
