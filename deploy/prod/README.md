# 生产部署（dept）

> ⚠️ **本目录是生产环境（PROD），已上线，有真实用户在使用。**（原名 `deploy/test/`，已更名为 `deploy/prod/`）
>
> - 只允许发布单个服务：`./deploy/prod/deploy.sh update <service>`
> - **禁止** `full`（重建全部并重启）与 `down`（用户断会）
> - 数据库结构变更必须先备份再生产手写执行
> - 任何部署都需**用户明确授权**后才可执行
>
> 详见仓库根 [`AGENTS.md`](../../AGENTS.md) 文首「生产环境状态」。

独立 MySQL / Redis / RustFS / LiveKit / Egress / HotGo / Client，数据 bind mount 到 `${DATA_DIR}`（默认 `/data/video-conference/data`）。

> ⚠️ **不要改动 `DATA_DIR` / `REMOTE_DIR`，不要清空远端 data 目录。**
> compose 把 `init/*.sql` 挂到 MySQL 的 `/docker-entrypoint-initdb.d/`，该目录只在数据目录为**空**时执行 ——
> 数据目录一旦为空/配错，MySQL 会重新初始化，生产数据受损。

会议页与管理后台 **分端口 HTTPS**：

- 会议：`https://PUBLIC_HOST:CLIENT_PORT/`（默认 17885）
- 管理后台：`https://PUBLIC_HOST:ADMIN_PORT/admin`（默认 17883）

## 从零部署（新环境才用，现有生产勿执行）

> 首次配置已在这些生产机上完成。以下命令**仅适用于全新机器**。

```bash
cp deploy/prod/.env.example deploy/prod/.env
cp deploy/prod/config/config.example.yaml deploy/prod/config/config.yaml
# 按需改 PUBLIC_HOST、密钥、密码

./deploy/prod/deploy.sh full
```

`full` 会：本机交叉编译 amd64 镜像 → rsync 到 dept → `docker load` → 启动 compose。

## 单服务更新

```bash
./deploy/prod/deploy.sh update hotgo
./deploy/prod/deploy.sh update client
./deploy/prod/deploy.sh update admin
./deploy/prod/deploy.sh update livekit
./deploy/prod/deploy.sh update minutes-worker
```

会后 AI 纪要依赖 `minutes-worker`（见 `services/minutes-worker/README.md`）。真实 LLM：在 `.env` 填 `OPENAI_API_KEY` 并设 `MOCK_LLM=0`。


## 运维

> ⚠️ `down` 会停掉全部生产服务（用户断会），**生产禁用**；`sync` / `pull` 会覆盖配置或拉新镜像，执行前先确认。

```bash
./deploy/prod/deploy.sh status
./deploy/prod/deploy.sh logs hotgo
./deploy/prod/deploy.sh sync      # ⚠️ 会覆盖远端 compose/init/config
./deploy/prod/deploy.sh pull      # ⚠️ 会拉公共镜像新版本
```

## 数据目录（远端）

| 路径 | 内容 |
|------|------|
| `${DATA_DIR}/mysql` | MySQL 数据 |
| `${DATA_DIR}/redis` | Redis |
| `${DATA_DIR}/rustfs` | 录制文件 |
| `${DATA_DIR}/hotgo/logs` | HotGo 日志 |
| `${DATA_DIR}/hotgo/storage` | HotGo 存储 |

备份：`rsync -a dept:/data/video-conference/data/ ./backup/`

## 端口（默认）

| 端口 | 用途 |
|------|------|
| 17883 | 管理后台 HTTPS |
| 17884 | 管理后台 HTTP → 301 HTTPS |
| 17885 | 会议客户端 HTTPS |
| 17888 | 会议 HTTP → 301 HTTPS |
| 17880 | LiveKit 信令 WS |
| 17881/17882 | WebRTC（UDP 17882 必放行） |

MySQL / Redis / HotGo / minutes-worker / Egress / **RustFS 均不映射宿主机**，仅 compose 内网（`vc_net`）访问。

## 录制回放

- `recording.s3.endpoint`：HotGo 容器内访问 RustFS（`http://rustfs:9000`）
- 会议客户端和管理后台回放/下载都走 HotGo HTTPS 代理，浏览器不直连 RustFS
- 管理后台：`/admin/conference/recording/play|download`；会议客户端：`/api/conference/recording/play|download`
- RustFS 的 S3 API（9000）与控制台（9001）**不对宿主机暴露端口**，Egress / minutes-worker / HotGo 全部走 `vc_net` 内网
- `recording.publicEndpoint` 保持为空：它只用于生成浏览器直链，当前前端未消费该字段；若将来需要直链，应给 RustFS 配域名 + TLS，而不是裸露 S3 端口

## 验收

- 会议：`https://125.211.217.19:17885/`
- 管理后台：`https://125.211.217.19:17883/admin`（默认 admin / 123456，上线后改密）
- 自签证书需在浏览器手动信任
