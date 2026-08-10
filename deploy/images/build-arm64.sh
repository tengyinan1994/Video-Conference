#!/usr/bin/env bash
# 构建并导出 linux/arm64 镜像到 deploy/images/arm64/
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
# shellcheck source=build-common.sh
source "$(dirname "$0")/build-common.sh"
build_and_export arm64 linux/arm64
