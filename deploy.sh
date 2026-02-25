#!/bin/bash

# ================= 配置 =================
PROJECT_DIR="/opt/market-sentinel"
# =======================================

set -e

echo "🚀 开始部署 MarketSentinel..."

cd "$PROJECT_DIR"

# 1. 拉取最新代码
echo "📥 拉取最新代码..."
git pull

# 2. 构建并重启
echo "🐳 构建并重启服务..."
docker compose up -d --build

# 3. 清理旧镜像
echo "🧹 清理旧镜像..."
docker image prune -f

echo "🎉 部署完成！"
