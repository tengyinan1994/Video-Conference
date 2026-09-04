#!/usr/bin/env bash
# 本机开发 LiveKit（livekit-dev：7880 信令 / 7881 TCP / 7882 UDP）。
# MySQL/Redis 复用本机 Middleware；密钥 devkey:secret 与 backend manifest 一致。
#
# 换网后：./deploy/dev/ip.sh -w && ./deploy/dev/dev-livekit.sh
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
NAME="${LIVEKIT_DEV_NAME:-livekit-dev}"
IMAGE="${LIVEKIT_IMAGE:-livekit/livekit-server:latest}"

IP="$("$ROOT/deploy/dev/ip.sh" -w)"
echo "使用局域网 IP: ${IP}"

docker rm -f "$NAME" >/dev/null 2>&1 || true

LIVEKIT_CONFIG=$(cat <<EOF
port: 7880
rtc:
  tcp_port: 7881
  udp_port: 7882
  use_external_ip: false
redis:
  address: host.docker.internal:6379
  db: 1
keys:
  devkey: secret
webhook:
  api_key: devkey
  urls:
    - http://host.docker.internal:8000/api/conference/webhook/livekit
logging:
  level: info
EOF
)

docker run -d --name "$NAME" --restart unless-stopped \
  -p 7880:7880 \
  -p 7881:7881 \
  -p 7882:7882/udp \
  -e "LIVEKIT_CONFIG=$LIVEKIT_CONFIG" \
  "$IMAGE" \
  --bind 0.0.0.0 --node-ip "$IP"

echo "livekit-dev：ws://${IP}:7880（信令）、tcp ${IP}:7881、udp ${IP}:7882"
echo "录制旁路：./deploy/dev/start-egress-when-ready.sh"
docker ps --filter "name=$NAME" --format 'table {{.Names}}\t{{.Status}}\t{{.Ports}}'
