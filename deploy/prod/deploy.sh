#!/usr/bin/env bash
# dept 一键部署（本机执行，SSH 到 DEPLOY_HOST）
#
# 用法：
#   ./deploy/prod/deploy.sh full
#   ./deploy/prod/deploy.sh update hotgo|client|admin|livekit|minutes-worker
#   ./deploy/prod/deploy.sh sync
#   ./deploy/prod/deploy.sh pull
#   ./deploy/prod/deploy.sh status
#   ./deploy/prod/deploy.sh logs [service]
#   ./deploy/prod/deploy.sh down
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
# shellcheck source=lib/common.sh
source "$SCRIPT_DIR/lib/common.sh"
# shellcheck source=lib/remote.sh
source "$SCRIPT_DIR/lib/remote.sh"

usage() {
  cat <<EOF
用法: $0 <command> [args]

  full              全量：构建镜像 → 同步 → 上传 tar → 远端启动
  update <service>  单服务更新（hotgo|client|admin|livekit）
  sync              仅同步 compose/init/config（不构建镜像）
  pull              远端 docker compose pull 公共镜像
  status            远端 compose ps
  logs [service]    远端 compose logs
  down              远端 compose down
EOF
}

cmd_full() {
  load_env
  require_config
  local_build all
  prepare_remote_dirs
  sync_stack_files
  scp_image_tars all
  remote_docker_load all
  remote_compose_pull
  remote_compose_up
  remote_compose_ps
  echo ""
  echo "部署完成。会议: https://${PUBLIC_HOST}:${CLIENT_PORT:-17885}/"
  echo "管理后台: https://${PUBLIC_HOST}:${ADMIN_PORT:-17883}/admin"
}

cmd_update() {
  local svc="${1:-}"
  [[ -z "$svc" ]] && { echo "请指定服务: hotgo|client|admin|livekit|minutes-worker" >&2; exit 1; }
  compose_service_name "$svc"
  load_env
  require_config
  local_build "$svc"
  prepare_remote_dirs
  sync_stack_files
  scp_image_tars "$svc"
  remote_docker_load "$svc"
  remote_compose_up_service "$svc"
  remote_compose_ps
}

cmd_sync() {
  load_env
  require_config
  prepare_remote_dirs
  sync_stack_files
  echo "同步完成"
}

cmd_pull() {
  load_env
  remote_compose_pull
}

cmd_status() {
  load_env
  remote_compose_ps
}

cmd_logs() {
  load_env
  remote_compose_logs "${1:-}"
}

cmd_down() {
  load_env
  remote_compose_down
}

main() {
  local cmd="${1:-}"
  case "$cmd" in
    full) cmd_full ;;
    update) cmd_update "${2:-}" ;;
    sync) cmd_sync ;;
    pull) cmd_pull ;;
    status) cmd_status ;;
    logs) cmd_logs "${2:-}" ;;
    down) cmd_down ;;
    -h|--help|help|"") usage ;;
    *) echo "未知命令: $cmd" >&2; usage; exit 1 ;;
  esac
}

main "$@"
