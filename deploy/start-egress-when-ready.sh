#!/usr/bin/env bash
# 本机开发：等 livekit/egress 镜像就绪后启动录制旁路（RustFS 需已在跑）
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

docker network create vc-dev 2>/dev/null || true
docker rm -f vc-egress 2>/dev/null || true

docker run -d --name vc-egress --network vc-dev --restart unless-stopped \
  --add-host=host.docker.internal:host-gateway \
  --cap-add=SYS_ADMIN \
  -e AWS_REQUEST_CHECKSUM_CALCULATION=when_required \
  -e AWS_RESPONSE_CHECKSUM_VALIDATION=when_required \
  -e "EGRESS_CONFIG_BODY=
api_key: devkey
api_secret: secret
ws_url: ws://host.docker.internal:7880
redis:
  address: host.docker.internal:6379
  db: 1
insecure: true
logging:
  level: info
s3:
  access_key: rustfsadmin
  secret: rustfsadmin
  region: us-east-1
  endpoint: http://vc-rustfs:9000
  bucket: recordings
  force_path_style: true
" \
  livekit/egress:latest

sleep 2
docker ps --filter name=vc-egress --format 'table {{.Names}}\t{{.Status}}'
docker logs vc-egress 2>&1 | tail -20
echo "egress started"
