#!/usr/bin/env bash
# deploy/test 公共变量与检查

TEST_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
PROJECT_ROOT="$(cd "$TEST_DIR/../.." && pwd)"

load_env() {
  if [[ -f "$TEST_DIR/.env" ]]; then
    set -a
    # shellcheck source=/dev/null
    source "$TEST_DIR/.env"
    set +a
  else
    echo "缺少 $TEST_DIR/.env，请先：cp deploy/test/.env.example deploy/test/.env" >&2
    exit 1
  fi
  DEPLOY_HOST="${DEPLOY_HOST:-dept}"
  REMOTE_DIR="${REMOTE_DIR:-/data/video-conference}"
  DATA_DIR="${DATA_DIR:-${REMOTE_DIR}/data}"
  ARCH="${ARCH:-amd64}"
  PUBLIC_HOST="${PUBLIC_HOST:-}"
}

require_config() {
  if [[ ! -f "$TEST_DIR/config/config.yaml" ]]; then
    echo "缺少 $TEST_DIR/config/config.yaml" >&2
    echo "请先：cp deploy/test/config/config.example.yaml deploy/test/config/config.yaml" >&2
    exit 1
  fi
}

remote_ssh() {
  ssh -o BatchMode=yes "$DEPLOY_HOST" "$@"
}

image_tar_path() {
  local svc="$1"
  echo "${PROJECT_ROOT}/deploy/images/${ARCH}/${svc}-${ARCH}.tar"
}

compose_service_name() {
  local svc="$1"
  case "$svc" in
    hotgo|client|admin|livekit|minutes-worker) echo "$svc" ;;
    *) echo "未知服务: $svc（可选 hotgo|client|admin|livekit|minutes-worker）" >&2; exit 1 ;;
  esac
}
