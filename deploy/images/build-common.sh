#!/usr/bin/env bash
# 由 build-amd64.sh / build-arm64.sh source；勿直接执行。

read_public_host() {
  local host="${VC_PUBLIC_HOST:-}"
  if [[ -z "$host" && -f "${ROOT}/deploy/test/.env" ]]; then
    host="$(grep '^PUBLIC_HOST=' "${ROOT}/deploy/test/.env" | head -1 | cut -d= -f2- | tr -d '"' | tr -d "'" | xargs)"
  fi
  echo "$host"
}

ensure_client_certs() {
  local cert_dir="${ROOT}/deploy/images/certs"
  local public_host
  public_host="$(read_public_host)"
  mkdir -p "${cert_dir}"

  local san="DNS:localhost,DNS:video-conference.local,IP:127.0.0.1"
  if [[ -n "$public_host" ]]; then
    san="${san},IP:${public_host}"
  fi

  if [[ "${VC_FORCE_CERT:-0}" == "1" ]] || [[ ! -f "${cert_dir}/server.crt" ]] || [[ ! -f "${cert_dir}/server.key" ]]; then
    openssl req -x509 -nodes -days 3650 -newkey rsa:2048 \
      -keyout "${cert_dir}/server.key" \
      -out "${cert_dir}/server.crt" \
      -subj "/CN=video-conference.local" \
      -addext "subjectAltName=${san}"
    echo "==> 已生成 HTTPS 证书（SAN: ${san}）"
  fi
}

host_prebuild_admin() {
  local arch="$1"
  local art_dir="${ROOT}/deploy/images/artifacts/${arch}"
  echo "==> [${arch}] 本机构建管理后台"
  (cd "${ROOT}/server/web" && pnpm run build)
  rm -rf "${art_dir}/admin"
  mkdir -p "${art_dir}/admin"
  cp -R "${ROOT}/server/web/dist/." "${art_dir}/admin/"
}

# 若误生成了 gf pack 内嵌资源，构建前强制清空，避免覆盖 Docker 内最新 admin 静态文件。
ensure_packed_stub() {
  local packed="${ROOT}/server/backend/internal/packed/packed.go"
  mkdir -p "$(dirname "$packed")"
  cat >"$packed" <<'EOF'
// Package packed 为 GoFrame gres 占位包。
// 不要把管理后台静态资源 gf pack 进二进制（会覆盖 resource/public/admin）。
package packed
EOF
}

host_prebuild_hotgo_binary() {
  local arch="$1"
  local art_dir="${ROOT}/deploy/images/artifacts/${arch}"
  echo "==> [${arch}] 交叉编译 HotGo (linux/${arch})"
  ensure_packed_stub
  mkdir -p "${art_dir}"
  (cd "${ROOT}/server/backend" && \
    CGO_ENABLED=0 GOOS=linux GOARCH="${arch}" \
    go build -ldflags="-s -w" -o "${art_dir}/hotgo" .)
}

host_prebuild_client_dist() {
  local arch="$1"
  local art_dir="${ROOT}/deploy/images/artifacts/${arch}"
  echo "==> [${arch}] 本机构建会议客户端"
  (cd "${ROOT}/client" && \
    printf 'VITE_API_BASE_URL=%s\n' "" > .env.production.local && \
    pnpm run build && \
    rm -f .env.production.local)
  rm -rf "${art_dir}/client"
  mkdir -p "${art_dir}/client"
  cp -R "${ROOT}/client/dist/." "${art_dir}/client/"
  ensure_client_certs
}

host_prebuild() {
  local arch="$1"
  host_prebuild_admin "$arch"
  host_prebuild_hotgo_binary "$arch"
  host_prebuild_client_dist "$arch"
}

docker_build_hotgo_image() {
  local arch="$1"
  local platform="$2"
  local hotgo_tag="video-conference/hotgo:${arch}"
  if [[ "${VC_DOCKER_BUILD:-0}" == "1" ]]; then
    docker buildx build --platform "${platform}" \
      -f "${ROOT}/deploy/images/Dockerfile.hotgo" \
      -t "${hotgo_tag}" --load "${ROOT}"
  else
    docker buildx build --platform "${platform}" \
      --build-arg "TARGETARCH=${arch}" \
      -f "${ROOT}/deploy/images/Dockerfile.hotgo.runtime" \
      -t "${hotgo_tag}" --load "${ROOT}"
  fi
}

docker_build_client_image() {
  local arch="$1"
  local platform="$2"
  local client_tag="video-conference/client:${arch}"
  if [[ "${VC_DOCKER_BUILD:-0}" == "1" ]]; then
    docker buildx build --platform "${platform}" \
      -f "${ROOT}/deploy/images/Dockerfile.client" \
      -t "${client_tag}" --load "${ROOT}"
  else
    docker buildx build --platform "${platform}" \
      --build-arg "TARGETARCH=${arch}" \
      -f "${ROOT}/deploy/images/Dockerfile.client.runtime" \
      -t "${client_tag}" --load "${ROOT}"
  fi
}

docker_build_admin_gateway_image() {
  local arch="$1"
  local platform="$2"
  ensure_client_certs
  echo "==> [${arch}] 打包管理后台 HTTPS 网关"
  docker buildx build --platform "${platform}" \
    -f "${ROOT}/deploy/images/Dockerfile.admin.runtime" \
    -t "video-conference/admin:${arch}" --load "${ROOT}"
}

docker_build_livekit_image() {
  local arch="$1"
  local platform="$2"
  local livekit_tag="video-conference/livekit:${arch}"
  docker pull --platform "${platform}" livekit/livekit-server:latest
  docker tag livekit/livekit-server:latest "${livekit_tag}"
}

export_hotgo_tar() {
  local arch="$1"
  local out_dir="${ROOT}/deploy/images/${arch}"
  docker save "video-conference/hotgo:${arch}" -o "${out_dir}/hotgo-${arch}.tar"
}

export_client_tar() {
  local arch="$1"
  local out_dir="${ROOT}/deploy/images/${arch}"
  docker save "video-conference/client:${arch}" -o "${out_dir}/client-${arch}.tar"
}

export_admin_tar() {
  local arch="$1"
  local out_dir="${ROOT}/deploy/images/${arch}"
  docker save "video-conference/admin:${arch}" -o "${out_dir}/admin-${arch}.tar"
}

export_livekit_tar() {
  local arch="$1"
  local out_dir="${ROOT}/deploy/images/${arch}"
  docker save "video-conference/livekit:${arch}" -o "${out_dir}/livekit-${arch}.tar"
}

docker_build_minutes_worker_image() {
  local arch="$1"
  local platform="$2"
  local tag="video-conference/minutes-worker:${arch}"
  echo "==> [${arch}] 打包 minutes-worker 镜像"
  docker buildx build --platform "${platform}" \
    -f "${ROOT}/services/minutes-worker/Dockerfile" \
    -t "${tag}" --load "${ROOT}/services/minutes-worker"
}

export_minutes_worker_tar() {
  local arch="$1"
  local out_dir="${ROOT}/deploy/images/${arch}"
  docker save "video-conference/minutes-worker:${arch}" -o "${out_dir}/minutes-worker-${arch}.tar"
}

build_service() {
  local arch="$1"
  local service="$2"
  local platform="$3"
  local out_dir="${ROOT}/deploy/images/${arch}"
  local art_dir="${ROOT}/deploy/images/artifacts/${arch}"
  mkdir -p "${out_dir}" "${art_dir}"

  case "$service" in
    hotgo)
      host_prebuild_admin "$arch"
      host_prebuild_hotgo_binary "$arch"
      echo "==> [${arch}] 打包 HotGo 镜像"
      docker_build_hotgo_image "$arch" "$platform"
      export_hotgo_tar "$arch"
      ;;
    client)
      host_prebuild_client_dist "$arch"
      echo "==> [${arch}] 打包会议客户端镜像"
      docker_build_client_image "$arch" "$platform"
      export_client_tar "$arch"
      ;;
    admin)
      docker_build_admin_gateway_image "$arch" "$platform"
      export_admin_tar "$arch"
      ;;
    livekit)
      echo "==> [${arch}] 标记 LiveKit 镜像"
      docker_build_livekit_image "$arch" "$platform"
      export_livekit_tar "$arch"
      ;;
    minutes-worker)
      docker_build_minutes_worker_image "$arch" "$platform"
      export_minutes_worker_tar "$arch"
      ;;
    all)
      build_and_export "$arch" "$platform"
      return
      ;;
    *)
      echo "未知服务: $service（可选 hotgo|client|admin|livekit|minutes-worker|all）" >&2
      exit 1
      ;;
  esac
  echo "==> [${arch}] ${service} 完成"
  ls -lh "${out_dir}/${service}-${arch}.tar" 2>/dev/null || ls -lh "${out_dir}"/*.tar
}

build_and_export() {
  local arch="$1"
  local platform="$2"
  local out_dir="${ROOT}/deploy/images/${arch}"
  local art_dir="${ROOT}/deploy/images/artifacts/${arch}"

  mkdir -p "${out_dir}" "${art_dir}"

  if [[ "${VC_DOCKER_BUILD:-0}" == "1" ]]; then
    echo "==> [${arch}] Docker 多阶段构建 HotGo"
    docker buildx build --platform "${platform}" \
      -f "${ROOT}/deploy/images/Dockerfile.hotgo" \
      -t "video-conference/hotgo:${arch}" --load "${ROOT}"
    echo "==> [${arch}] Docker 多阶段构建会议客户端"
    docker buildx build --platform "${platform}" \
      -f "${ROOT}/deploy/images/Dockerfile.client" \
      -t "video-conference/client:${arch}" --load "${ROOT}"
  else
    host_prebuild "$arch"
    echo "==> [${arch}] 打包 HotGo 运行时镜像"
    docker_build_hotgo_image "$arch" "$platform"
    echo "==> [${arch}] 打包会议客户端运行时镜像"
    docker_build_client_image "$arch" "$platform"
  fi

  docker_build_admin_gateway_image "$arch" "$platform"

  echo "==> [${arch}] 标记 LiveKit 镜像"
  docker_build_livekit_image "$arch" "$platform"

  echo "==> [${arch}] 导出 tar 到 ${out_dir}"
  export_hotgo_tar "$arch"
  export_client_tar "$arch"
  export_admin_tar "$arch"
  export_livekit_tar "$arch"
  docker_build_minutes_worker_image "$arch" "$platform"
  export_minutes_worker_tar "$arch"

  echo "==> [${arch}] 完成"
  ls -lh "${out_dir}"/*.tar
}
