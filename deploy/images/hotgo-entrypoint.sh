#!/bin/sh
set -eu
cd /app
# 前台运行，便于 docker stop 发信号退出
exec ./hotgo
