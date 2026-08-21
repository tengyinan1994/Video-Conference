#!/bin/sh
# 等待 RustFS 就绪后创建 recordings bucket（S3 兼容 API）
set -e
ENDPOINT="${RUSTFS_ENDPOINT:-http://rustfs:9000}"
ACCESS="${RUSTFS_ACCESS_KEY:-rustfsadmin}"
SECRET="${RUSTFS_SECRET_KEY:-rustfsadmin}"
BUCKET="${RUSTFS_BUCKET:-recordings}"

echo "waiting for rustfs at $ENDPOINT ..."
i=0
while [ "$i" -lt 60 ]; do
  if wget -q -O /dev/null "$ENDPOINT/health" 2>/dev/null || wget -q -O /dev/null "$ENDPOINT/minio/health/live" 2>/dev/null; then
    break
  fi
  i=$((i + 1))
  sleep 2
done

# 使用 AWS CLI 风格的 mc 或 curl PutBucket
# alpine/busybox 无 aws cli；用 Python/curl 发签名请求过重。
# 改用 minio client (mc) 镜像：
if command -v mc >/dev/null 2>&1; then
  mc alias set local "$ENDPOINT" "$ACCESS" "$SECRET"
  mc mb -p "local/$BUCKET" || true
  echo "bucket $BUCKET ready"
  exit 0
fi

# 无 mc 时：尝试匿名创建（多数 S3 兼容会拒）；HotGo 启动时也会 EnsureBucket
echo "mc not found; skip create (HotGo EnsureBucket will create $BUCKET)"
exit 0
