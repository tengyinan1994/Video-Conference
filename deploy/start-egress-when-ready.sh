#!/usr/bin/env bash
# 本机开发：等 livekit/egress 镜像就绪后启动录制旁路（RustFS 需已在跑，宿主口 17886）
# 由 deploy/docker-compose.dev.yml 管理开发版 Egress（vc-egress-dev）
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

echo "waiting for image livekit/egress:latest ..."
for i in $(seq 1 180); do
  if docker image inspect livekit/egress:latest >/dev/null 2>&1; then
    echo "image ready"
    break
  fi
  # 也接受 dao cloud 拉下来的 tag
  if docker image inspect docker.m.daocloud.io/livekit/egress:latest >/dev/null 2>&1; then
    docker tag docker.m.daocloud.io/livekit/egress:latest livekit/egress:latest
    echo "tagged from daocloud"
    break
  fi
  sleep 10
  if [ "$i" = "180" ]; then
    echo "timeout waiting for egress image" >&2
    exit 1
  fi
done

# 开发版 Egress 由 deploy/docker-compose.dev.yml 管理（容器 vc-egress-dev），
# 连宿主 7880 的 livekit-dev；与主 compose 里部署版 egress 相互独立。
# 旧版用 docker run 起的 vc-egress（若还在）先清掉，避免与新容器并存。
docker rm -f vc-egress >/dev/null 2>&1 || true

docker compose -f deploy/docker-compose.dev.yml up -d egress

sleep 2
docker compose -f deploy/docker-compose.dev.yml ps
docker logs vc-egress-dev 2>&1 | tail -20
echo "egress started"
