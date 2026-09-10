# dept 全量部署（test）

独立 MySQL / Redis / RustFS / LiveKit / Egress / HotGo / Client，数据 bind mount 到 `${DATA_DIR}`（默认 `/data/video-conference/data`）。

会议页与管理后台 **分端口 HTTPS**：

- 会议：`https://PUBLIC_HOST:CLIENT_PORT/`（默认 17885）
- 管理后台：`https://PUBLIC_HOST:ADMIN_PORT/admin`（默认 17883）

## 首次部署

```bash
cp deploy/test/.env.example deploy/test/.env
cp deploy/test/config/config.example.yaml deploy/test/config/config.yaml
# 按需改 PUBLIC_HOST、密钥、密码

./deploy/test/deploy.sh full
```

`full` 会：本机交叉编译 amd64 镜像 → rsync 到 dept → `docker load` → 启动 compose。

## 单服务更新

```bash
./deploy/test/deploy.sh update hotgo
./deploy/test/deploy.sh update client
./deploy/test/deploy.sh update admin
./deploy/test/deploy.sh update livekit
./deploy/test/deploy.sh update minutes-worker
```

会后 AI 纪要依赖 `minutes-worker`（见 `services/minutes-worker/README.md`）。真实 LLM：在 `.env` 填 `OPENAI_API_KEY` 并设 `MOCK_LLM=0`。


## 运维

```bash
./deploy/test/deploy.sh sync      # 只同步 compose/init/config
./deploy/test/deploy.sh pull      # 拉公共镜像
./deploy/test/deploy.sh status
./deploy/test/deploy.sh logs hotgo
./deploy/test/deploy.sh down
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
| 17886/17887 | RustFS（管理后台回放直链依赖 17886；可按安全策略加防火墙） |

MySQL / Redis **不映射**宿主机，仅 compose 内网访问。

## 录制回放

- `recording.s3.endpoint`：HotGo 容器内访问 RustFS（`http://rustfs:9000`）
- 会议客户端和管理后台回放/下载都走 HotGo HTTPS 代理，浏览器不直连 RustFS
- 管理后台：`/admin/conference/recording/play|download`；会议客户端：`/api/conference/recording/play|download`
- `recording.publicEndpoint` 仅作备用直链；可留空。若仍暴露 17886，HTTPS 后台点 HTTP 直链会被 Edge/Chrome 报「无法安全下载」

## 验收

- 会议：`https://125.211.217.19:17885/`
- 管理后台：`https://125.211.217.19:17883/admin`（默认 admin / 123456，上线后改密）
- 自签证书需在浏览器手动信任
