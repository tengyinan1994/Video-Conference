# 部署说明

| 目录 | 用途 |
|------|------|
| [`dev/`](dev/) | 本机开发额外中间件（LiveKit、RustFS、Egress）；MySQL/Redis 复用 Middleware |
| [`test/`](test/) | dept 全量 Docker 部署 + 一键脚本 |
| [`images/`](images/) | 镜像构建与 nginx 配置（test 部署消费 tar） |

## 本机开发

见 [`dev/README.md`](dev/README.md)。

## dept 部署

见 [`test/README.md`](test/README.md)。一键入口：

```bash
./deploy/test/deploy.sh full
```

## 持久化

- **dev**：Docker 命名卷（如 RustFS）
- **test**：宿主机目录 bind mount（`${DATA_DIR}`，默认 `/data/video-conference/data`）
