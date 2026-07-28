#!/bin/zsh
set -euo pipefail

# 本机 LiveKit：优先复用已有 livekit-dev 容器；没有则按本机可用配置创建
# 局域网第二台设备开会时，ICE 候选必须宣告「别的机器能连上的 IP」，
# 不能再用 127.0.0.1（那台 Windows 会当成它自己）。
NAME="livekit-dev"

detect_lan_ip() {
  local ip=""
  for iface in en0 en1 en2; do
    ip=$(ipconfig getifaddr "$iface" 2>/dev/null || true)
    if [[ -n "$ip" ]]; then
      echo "$ip"
      return 0
    fi
  done
  # 兜底：走默认路由的网卡
  ip=$(route -n get default 2>/dev/null | awk '/interface:/{print $2}' | xargs -I{} ipconfig getifaddr {} 2>/dev/null || true)
  if [[ -n "$ip" ]]; then
    echo "$ip"
    return 0
  fi
  echo "127.0.0.1"
}

NODE_IP=$(detect_lan_ip)
echo "LiveKit ICE node_ip -> $NODE_IP"

# 已有容器但 node_ip 可能仍是旧的 127.0.0.1：用期望 IP 比对，不一致则重建
need_recreate=0
if docker inspect "$NAME" >/dev/null 2>&1; then
  existing=$(docker inspect -f '{{range .Config.Env}}{{println .}}{{end}}' "$NAME" 2>/dev/null | grep '^LIVEKIT_CONFIG=' || true)
  if [[ "$existing" != *"node_ip: $NODE_IP"* ]]; then
    echo "已有容器的 node_ip 与当前局域网 IP 不一致，将重建 $NAME"
    need_recreate=1
    docker rm -f "$NAME" >/dev/null
  fi
fi

if docker inspect "$NAME" >/dev/null 2>&1 && [[ "$need_recreate" -eq 0 ]]; then
  docker start "$NAME" >/dev/null
  echo "LiveKit 容器已启动: $NAME"
  echo "  信令: ws://127.0.0.1:7880（本机） / 局域网请经 Vite https://$NODE_IP:5173 进房"
  echo "  媒体 ICE: $NODE_IP:7881(TCP) / $NODE_IP:7882(UDP) — 防火墙需放行"
else
  docker run -d --name "$NAME" \
    -p 7880:7880 -p 7881:7881 -p 7882:7882/udp \
    -e LIVEKIT_CONFIG="
port: 7880
rtc:
  tcp_port: 7881
  udp_port: 7882
  use_external_ip: false
  node_ip: $NODE_IP
keys:
  devkey: secret
" livekit/livekit-server --bind 0.0.0.0 >/dev/null
  echo "LiveKit 容器已创建并启动: $NAME"
  echo "  信令: ws://127.0.0.1:7880"
  echo "  媒体 ICE node_ip: $NODE_IP （局域网第二台设备必需）"
fi

# 前台跟日志，方便在调试面板里看到输出；停止调试即停跟日志（容器仍可继续跑）
exec docker logs -f "$NAME"
