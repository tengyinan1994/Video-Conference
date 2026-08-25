#!/usr/bin/env bash
# 检测本机当前局域网 IP（换网后 IP 会变，所有依赖 IP 的配置都要跟随）。
#
# 用法：
#   ./deploy/ip.sh          只打印检测到的 IP
#   ./deploy/ip.sh -w       打印 IP 并同步改写：
#                             - deploy/.env           的 LIVEKIT_NODE_IP
#                             - deploy/config/config.yaml 的 livekit.url
#   ./deploy/ip.sh --check  与已写入的配置比对，不一致时提示（并给出修复命令）
#
# 换网后执行一次：./deploy/ip.sh -w
# 然后重启 LiveKit：./deploy/dev-livekit.sh（本机开发）或 docker compose ... up livekit
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
ENV_FILE="$ROOT/deploy/.env"
CONFIG_FILE="$ROOT/deploy/config/config.yaml"

# 检测主局域网 IP：优先取默认路由所在网卡（macOS/Linux 通用）
detect_ip() {
  local ip="" iface=""

  # macOS
  if command -v route >/dev/null 2>&1 && command -v ipconfig >/dev/null 2>&1; then
    iface="$(route -n get default 2>/dev/null | awk '/interface:/{print $2; exit}')"
    if [ -n "$iface" ]; then
      ip="$(ipconfig getifaddr "$iface" 2>/dev/null || true)"
    fi
  fi

  # Linux
  if [ -z "$ip" ] && command -v ip >/dev/null 2>&1; then
    iface="$(ip route get 1.1.1.1 2>/dev/null | sed -n 's/.*dev \([^ ]*\).*/\1/p' | head -1)"
    if [ -n "$iface" ]; then
      ip="$(ip -4 addr show dev "$iface" 2>/dev/null | awk '/inet /{print $2; exit}' | cut -d/ -f1)"
    fi
  fi

  # Linux 兜底
  if [ -z "$ip" ] && command -v hostname >/dev/null 2>&1; then
    ip="$(hostname -I 2>/dev/null | awk '{print $1}')"
  fi

  # macOS 兜底：常见网卡
  if [ -z "$ip" ] && command -v ipconfig >/dev/null 2>&1; then
    for iface in en0 en1; do
      ip="$(ipconfig getifaddr "$iface" 2>/dev/null || true)"
      [ -n "$ip" ] && break
    done
  fi

  # 只接受 IPv4 且排除回环
  case "$ip" in
    127.*|"") return 1 ;;
  esac
  echo "$ip"
}

current_ip() {
  grep '^LIVEKIT_NODE_IP=' "$ENV_FILE" 2>/dev/null | head -1 | cut -d= -f2 || true
}

write_all() {
  local ip="$1"

  if [ -f "$ENV_FILE" ]; then
    if grep -q '^LIVEKIT_NODE_IP=' "$ENV_FILE"; then
      sed -i.bak "s|^LIVEKIT_NODE_IP=.*|LIVEKIT_NODE_IP=$ip|" "$ENV_FILE" && rm -f "$ENV_FILE.bak"
    else
      echo "LIVEKIT_NODE_IP=$ip" >> "$ENV_FILE"
    fi
    echo "已更新 $ENV_FILE -> LIVEKIT_NODE_IP=$ip" >&2
  fi

  if [ -f "$CONFIG_FILE" ] && grep -q '^  url: "ws://' "$CONFIG_FILE"; then
    sed -i.bak "s|^\(  url: \"ws://\)[^\"]*\(:17880\"\)|\1$ip\2|" "$CONFIG_FILE" && rm -f "$CONFIG_FILE.bak"
    echo "已更新 $CONFIG_FILE -> livekit.url=ws://$ip:17880" >&2
  fi
}

if ! IP="$(detect_ip)"; then
  echo "未检测到局域网 IP（可能未联网）。请手动编辑 $ENV_FILE 的 LIVEKIT_NODE_IP。" >&2
  exit 1
fi

case "${1:-}" in
  -w|--write)
    echo "检测到局域网 IP: $IP" >&2
    write_all "$IP"
    echo "$IP"   # stdout 只输出 IP，供脚本捕获（如 dev-livekit.sh）
    ;;
  --check)
    OLD="$(current_ip)"
    echo "检测到局域网 IP: $IP"
    if [ -n "$OLD" ] && [ "$OLD" != "$IP" ]; then
      echo "配置仍是旧 IP: ${OLD}（可能已过期）"
      echo "修复：./deploy/ip.sh -w && ./deploy/dev-livekit.sh"
      exit 2
    fi
    echo "配置与当前 IP 一致（${IP}）"
    ;;
  *)
    echo "$IP"
    ;;
esac
