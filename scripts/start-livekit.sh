#!/bin/zsh
set -euo pipefail

# 本机 LiveKit：优先复用已有 livekit-dev 容器；没有则按本机可用配置创建
NAME="livekit-dev"

if docker inspect "$NAME" >/dev/null 2>&1; then
  docker start "$NAME" >/dev/null
  echo "LiveKit 容器已启动: $NAME -> ws://127.0.0.1:7880"
else
  docker run -d --name "$NAME" \
    -p 7880:7880 -p 7881:7881 -p 7882:7882/udp \
    -e LIVEKIT_CONFIG='
port: 7880
rtc:
  tcp_port: 7881
  udp_port: 7882
  use_external_ip: false
  node_ip: 127.0.0.1
keys:
  devkey: secret
' livekit/livekit-server --bind 0.0.0.0 >/dev/null
  echo "LiveKit 容器已创建并启动: $NAME -> ws://127.0.0.1:7880"
fi

# 前台跟日志，方便在调试面板里看到输出；停止调试即停跟日志（容器仍可继续跑）
exec docker logs -f "$NAME"
