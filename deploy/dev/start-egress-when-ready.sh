#!/usr/bin/env bash
# 等 egress 镜像就绪后启动开发版录制旁路（需 RustFS 已起，宿主 17886）
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
cd "$ROOT"

echo "waiting for image livekit/egress:latest ..."
for i in $(seq 1 180); do
  if docker image inspect livekit/egress:latest >/dev/null 2>&1; then
    echo "image ready"
    break
  fi
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

docker rm -f vc-egress >/dev/null 2>&1 || true

docker compose -f deploy/dev/docker-compose.yml --env-file deploy/dev/.env up -d egress

sleep 2
docker compose -f deploy/dev/docker-compose.yml ps
docker logs vc-egress-dev 2>&1 | tail -20
echo "egress started"
