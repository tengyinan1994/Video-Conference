#!/usr/bin/env bash
# 远端 rsync / docker 操作

prepare_remote_dirs() {
  remote_ssh "mkdir -p '${REMOTE_DIR}/images' '${REMOTE_DIR}/config' '${REMOTE_DIR}/init' \
    '${DATA_DIR}/mysql' '${DATA_DIR}/redis' '${DATA_DIR}/rustfs' \
    '${DATA_DIR}/hotgo/logs' '${DATA_DIR}/hotgo/storage'"
  remote_ssh "chown 999:999 '${DATA_DIR}/mysql' 2>/dev/null || true"
  remote_ssh "chown 10001:10001 '${DATA_DIR}/rustfs' 2>/dev/null || true"
}

sync_stack_files() {
  echo "==> 同步 compose / init / config / .env 到 ${DEPLOY_HOST}:${REMOTE_DIR}"
  rsync -az "$PROD_DIR/docker-compose.yml" "${DEPLOY_HOST}:${REMOTE_DIR}/"
  rsync -az "$PROD_DIR/.env" "${DEPLOY_HOST}:${REMOTE_DIR}/"
  rsync -az "$PROD_DIR/config/config.yaml" "${DEPLOY_HOST}:${REMOTE_DIR}/config/"
  if [[ -f "$PROD_DIR/config/casbin.conf" ]]; then
    rsync -az "$PROD_DIR/config/casbin.conf" "${DEPLOY_HOST}:${REMOTE_DIR}/config/"
  fi
  rsync -az "$PROD_DIR/init/" "${DEPLOY_HOST}:${REMOTE_DIR}/init/"
}

scp_image_tars() {
  local pattern="${1:-*}"
  echo "==> 上传镜像 tar"
  remote_ssh "mkdir -p '${REMOTE_DIR}/images'"
  if [[ "$pattern" == "all" ]]; then
    scp "${PROJECT_ROOT}/deploy/images/${ARCH}/"*.tar "${DEPLOY_HOST}:${REMOTE_DIR}/images/"
  else
    scp "$(image_tar_path "$pattern")" "${DEPLOY_HOST}:${REMOTE_DIR}/images/"
  fi
}

remote_docker_load() {
  local svc="${1:-all}"
  if [[ "$svc" == "all" ]]; then
    remote_ssh "cd '${REMOTE_DIR}/images' && for f in *.tar; do docker load -i \"\$f\"; done"
  else
    remote_ssh "docker load -i '${REMOTE_DIR}/images/${svc}-${ARCH}.tar'"
  fi
}

remote_compose_pull() {
  echo "==> 远端拉取公共镜像（跳过自定义镜像）"
  remote_ssh "cd '${REMOTE_DIR}' && docker compose --env-file .env pull mysql redis rustfs egress rustfs-init 2>/dev/null || \
    docker compose --env-file .env pull mysql redis rustfs livekit/egress:latest minio/mc:latest 2>/dev/null || true"
}

remote_compose_up() {
  echo "==> 远端启动 compose"
  remote_ssh "cd '${REMOTE_DIR}' && docker compose --env-file .env up -d"
}

remote_compose_up_service() {
  local svc="$1"
  echo "==> 远端重启服务: $svc"
  remote_ssh "cd '${REMOTE_DIR}' && docker compose --env-file .env up -d --no-deps --force-recreate '$svc'"
}

remote_compose_ps() {
  remote_ssh "cd '${REMOTE_DIR}' && docker compose --env-file .env ps"
}

remote_compose_logs() {
  local svc="${1:-}"
  if [[ -n "$svc" ]]; then
    remote_ssh "cd '${REMOTE_DIR}' && docker compose --env-file .env logs -f --tail=100 '$svc'"
  else
    remote_ssh "cd '${REMOTE_DIR}' && docker compose --env-file .env logs -f --tail=50"
  fi
}

remote_compose_down() {
  remote_ssh "cd '${REMOTE_DIR}' && docker compose --env-file .env down"
}

local_build() {
  local svc="$1"
  local platform="linux/${ARCH}"
  ROOT="$PROJECT_ROOT"
  export VC_FORCE_CERT=1
  echo "==> 本机构建 ${svc} (${ARCH})"
  # shellcheck source=/dev/null
  source "${PROJECT_ROOT}/deploy/images/build-common.sh"
  if [[ "$svc" == "all" ]]; then
    build_and_export "$ARCH" "$platform"
  else
    build_service "$ARCH" "$svc" "$platform"
  fi
}
