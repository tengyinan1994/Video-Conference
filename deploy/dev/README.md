# 本机开发中间件

MySQL / Redis **不在此目录**，复用本机 Middleware（如 `/Users/chaoming/Middleware/docker-compose.yml`）。

HotGo（`air`）、会议客户端（`pnpm dev`）、管理后台（`pnpm dev`）在源码目录启动，不属于 deploy。

## 组件

| 组件 | 启动方式 | 持久化 |
|------|----------|--------|
| LiveKit | `./deploy/dev/dev-livekit.sh` | 无 |
| RustFS | `docker compose -f deploy/dev/docker-compose.yml --env-file deploy/dev/.env up -d rustfs` | 命名卷 `vc_rustfs_dev_data` |
| Egress | `./deploy/dev/start-egress-when-ready.sh` | 无 |

## 首次配置

```bash
cp deploy/dev/.env.example deploy/dev/.env
./deploy/dev/ip.sh -w    # 换网后重跑
./deploy/dev/dev-livekit.sh
```

录制开发需先起 RustFS，再 `start-egress-when-ready.sh`。

## 会后 AI 纪要（本机）

1. 执行 SQL：`deploy/test/init/09-conference-minutes.sql`
2. HotGo `config.yaml` 增加 `minutes` 段（见 `manifest/config/config.example.yaml`）
3. 起 Worker：见 `services/minutes-worker/README.md`（开发可用 `ASR_BACKEND=mock`）
4. 需 Egress + RustFS 可用（与录制相同），AI 音源会另占一路 audio-only Egress

