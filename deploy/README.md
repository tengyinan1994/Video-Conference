# 部署说明

| 目录 | 用途 |
|------|------|
| [`dev/`](dev/) | 本机开发额外中间件（LiveKit、RustFS、Egress）；MySQL/Redis 复用 Middleware |
| [`prod/`](prod/) | ⚠️ **生产环境**（dept，已上线）：全量 Docker 部署 + 一键脚本 |
| [`images/`](images/) | 镜像构建与 nginx 配置（生产部署消费 tar） |

> ⚠️ **`prod/` 是生产环境。** 不要执行 `deploy.sh full` / `down`；
> 发布只允许 `update <单个服务>`，且需用户明确授权。详见仓库根 [`AGENTS.md`](../AGENTS.md) 文首。

## 本机开发

见 [`dev/README.md`](dev/README.md)。

## 生产部署（dept）

见 [`prod/README.md`](prod/README.md)。⚠️ 首次配置与全量部署只适用于**全新机器**；
现有生产环境**禁止**重复执行：

```bash
# 常规发布（只动一个服务，需用户授权）
./deploy/prod/deploy.sh update <hotgo|client|admin|livekit|minutes-worker>
```

## 持久化

- **dev**：Docker 命名卷（如 RustFS）
- **test**：宿主机目录 bind mount（`${DATA_DIR}`，默认 `/data/video-conference/data`）

## 会后 AI 纪要

- SQL：`deploy/prod/init/09-conference-minutes.sql`（`purpose` + `hg_addon_conference_minutes`）
- Worker：`services/minutes-worker/`（FunASR/mock + LangChain OpenAI 兼容）
- HotGo `minutes` 配置：`enabled` / `workerUrl` / `callbackSecret`
- **与回放录制解耦**：每场会默默开 AI audio-only Egress；用户未开「录制」也可出纪要
- 验收：空发言 → `skipped_empty`；AI 音源失败 → `unavailable`；Worker 未起不影响开会

详见 [`services/minutes-worker/README.md`](../services/minutes-worker/README.md)。

