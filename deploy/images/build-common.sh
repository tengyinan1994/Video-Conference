#!/usr/bin/env bash
# 由 build-amd64.sh / build-arm64.sh source；勿直接执行。
# 默认：本机交叉编译 + 精简运行时镜像（不拉 golang/node 构建基座，适合内网/代理不稳）。
# 可选：VC_DOCKER_BUILD=1 走 Dockerfile.hotgo / Dockerfile.client 全量多阶段构建。

build_and_export() {
  local arch="$1"
  local platform="$2"
  local out_dir="${ROOT}/deploy/images/${arch}"
  local art_dir="${ROOT}/deploy/images/artifacts/${arch}"
  local hotgo_tag="video-conference/hotgo:${arch}"
  local client_tag="video-conference/client:${arch}"
  local livekit_tag="video-conference/livekit:${arch}"

  mkdir -p "${out_dir}" "${art_dir}"

  if [[ "${VC_DOCKER_BUILD:-0}" == "1" ]]; then
    echo "==> [${arch}] Docker 多阶段构建 HotGo"
    docker buildx build --platform "${platform}" \
      -f "${ROOT}/deploy/images/Dockerfile.hotgo" \
      -t "${hotgo_tag}" --load "${ROOT}"
    echo "==> [${arch}] Docker 多阶段构建会议客户端"
    docker buildx build --platform "${platform}" \
      -f "${ROOT}/deploy/images/Dockerfile.client" \
      -t "${client_tag}" --load "${ROOT}"
  else
    host_prebuild "${arch}"
    echo "==> [${arch}] 打包 HotGo 运行时镜像"
    docker buildx build --platform "${platform}" \
      --build-arg "TARGETARCH=${arch}" \
      -f "${ROOT}/deploy/images/Dockerfile.hotgo.runtime" \
      -t "${hotgo_tag}" --load "${ROOT}"
    echo "==> [${arch}] 打包会议客户端运行时镜像"
    docker buildx build --platform "${platform}" \
      --build-arg "TARGETARCH=${arch}" \
      -f "${ROOT}/deploy/images/Dockerfile.client.runtime" \
      -t "${client_tag}" --load "${ROOT}"
  fi

  echo "==> [${arch}] 标记 LiveKit 镜像（优先用本地已有）"
  if ! docker image inspect livekit/livekit-server:latest >/dev/null 2>&1; then
    docker pull --platform "${platform}" livekit/livekit-server:latest
  fi
  docker tag livekit/livekit-server:latest "${livekit_tag}"

  echo "==> [${arch}] 导出 tar 到 ${out_dir}"
  docker save "${hotgo_tag}" -o "${out_dir}/hotgo-${arch}.tar"
  docker save "${client_tag}" -o "${out_dir}/client-${arch}.tar"
  docker save "${livekit_tag}" -o "${out_dir}/livekit-${arch}.tar"

  echo "==> [${arch}] 完成"
  ls -lh "${out_dir}"/*.tar
}

host_prebuild() {
  local arch="$1"
  local art_dir="${ROOT}/deploy/images/artifacts/${arch}"

  echo "==> [${arch}] 本机构建管理后台"
  (cd "${ROOT}/server/web" && pnpm run build)
  rm -rf "${art_dir}/admin"
  mkdir -p "${art_dir}/admin"
  cp -R "${ROOT}/server/web/dist/." "${art_dir}/admin/"

  echo "==> [${arch}] 交叉编译 HotGo (linux/${arch})"
  (cd "${ROOT}/server/backend" && \
    CGO_ENABLED=0 GOOS=linux GOARCH="${arch}" \
    go build -ldflags="-s -w" -o "${art_dir}/hotgo" .)

  echo "==> [${arch}] 本机构建会议客户端"
  # 空 API base：生产走 nginx 同源代理
  (cd "${ROOT}/client" && \
    printf 'VITE_API_BASE_URL=%s\n' "" > .env.production.local && \
    pnpm run build && \
    rm -f .env.production.local)
  rm -rf "${art_dir}/client"
  mkdir -p "${art_dir}/client"
  cp -R "${ROOT}/client/dist/." "${art_dir}/client/"

  echo "==> [${arch}] 确保客户端 HTTPS 自签证书存在"
  local cert_dir="${ROOT}/deploy/images/certs"
  mkdir -p "${cert_dir}"
  if [[ ! -f "${cert_dir}/server.crt" || ! -f "${cert_dir}/server.key" ]]; then
    openssl req -x509 -nodes -days 3650 -newkey rsa:2048 \
      -keyout "${cert_dir}/server.key" \
      -out "${cert_dir}/server.crt" \
      -subj "/CN=video-conference.local" \
      -addext "subjectAltName=DNS:localhost,DNS:video-conference.local,IP:127.0.0.1"
  fi
}