#!/usr/bin/env bash
# 检测本机局域网 IP，仅更新 deploy/dev/.env 的 LIVEKIT_NODE_IP。
#
# 用法：
#   ./deploy/dev/ip.sh          只打印 IP
#   ./deploy/dev/ip.sh -w       写入 deploy/dev/.env
#   ./deploy/dev/ip.sh --check  比对配置与当前 IP
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
ENV_FILE="$ROOT/deploy/dev/.env"

detect_ip() {
  local ip="" iface=""

  if command -v route >/dev/null 2>&1 && command -v ipconfig >/dev/null 2>&1; then
    iface="$(route -n get default 2>/dev/null | awk '/interface:/{print $2; exit}')"
    if [ -n "$iface" ]; then
      ip="$(ipconfig getifaddr "$iface" 2>/dev/null || true)"
    fi
  fi

  if [ -z "$ip" ] && command -v ip >/dev/null 2>&1; then
    iface="$(ip route get 1.1.1.1 2>/dev/null | sed -n 's/.*dev \([^ ]*\).*/\1/p' | head -1)"
    if [ -n "$iface" ]; then
      ip="$(ip -4 addr show dev "$iface" 2>/dev/null | awk '/inet /{print $2; exit}' | cut -d/ -f1)"
    fi
  fi

  if [ -z "$ip" ] && command -v hostname >/dev/null 2>&1; then
    ip="$(hostname -I 2>/dev/null | awk '{print $1}')"
  fi

  if [ -z "$ip" ] && command -v ipconfig >/dev/null 2>&1; then
    for iface in en0 en1; do
      ip="$(ipconfig getifaddr "$iface" 2>/dev/null || true)"
      [ -n "$ip" ] && break
    done
  fi

  case "$ip" in
    127.*|"") return 1 ;;
  esac
  echo "$ip"
}

current_ip() {
  grep '^LIVEKIT_NODE_IP=' "$ENV_FILE" 2>/dev/null | head -1 | cut -d= -f2 || true
}

write_env() {
  local ip="$1"
  if [ -f "$ENV_FILE" ]; then
    if grep -q '^LIVEKIT_NODE_IP=' "$ENV_FILE"; then
      sed -i.bak "s|^LIVEKIT_NODE_IP=.*|LIVEKIT_NODE_IP=$ip|" "$ENV_FILE" && rm -f "$ENV_FILE.bak"
    else
      echo "LIVEKIT_NODE_IP=$ip" >> "$ENV_FILE"
    fi
    echo "已更新 $ENV_FILE -> LIVEKIT_NODE_IP=$ip" >&2
  else
    echo "请先 cp deploy/dev/.env.example deploy/dev/.env" >&2
    exit 1
  fi
}

if ! IP="$(detect_ip)"; then
  echo "未检测到局域网 IP。请手动编辑 $ENV_FILE 的 LIVEKIT_NODE_IP。" >&2
  exit 1
fi

case "${1:-}" in
  -w|--write)
    echo "检测到局域网 IP: $IP" >&2
    write_env "$IP"
    echo "$IP"
    ;;
  --check)
    OLD="$(current_ip)"
    echo "检测到局域网 IP: $IP"
    if [ -n "$OLD" ] && [ "$OLD" != "$IP" ]; then
      echo "配置仍是旧 IP: ${OLD}"
      echo "修复：./deploy/dev/ip.sh -w && ./deploy/dev/dev-livekit.sh"
      exit 2
    fi
    echo "配置与当前 IP 一致（${IP}）"
    ;;
  *)
    echo "$IP"
    ;;
esac
