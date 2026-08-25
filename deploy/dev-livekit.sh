#!/usr/bin/env bash
# 本机开发用 LiveKit（livekit-dev 容器，7880 信令 / 7881 TCP / 7882 UDP）。
# 密钥 devkey:secret 与 server/backend/manifest/config/config.yaml 对齐；
# 客户端开发模式经 Vite /rtc 代理到 127.0.0.1:7880（见 client/vite.config.ts）。
#
# 换网后重跑本脚本即可：自动检测新局域网 IP（并同步 deploy/.env）后重建容器。
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
NAME="${LIVEKIT_DEV_NAME:-livekit-dev}"
IMAGE="${LIVEKIT_IMAGE:-livekit/livekit-server:latest}"

IP="$("$ROOT/deploy/ip.sh" -w)"
echo "使用局域网 IP: ${IP}（已同步 deploy/.env）"

# 重建容器（LiveKit 无状态，直接删了重跑）
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

echo "livekit-dev 已启动：ws://${IP}:7880（信令）、tcp ${IP}:7881、udp ${IP}:7882"
echo "如需录制旁路，再运行：./deploy/start-egress-when-ready.sh"
docker ps --filter "name=$NAME" --format 'table {{.Names}}\t{{.Status}}\t{{.Ports}}'
