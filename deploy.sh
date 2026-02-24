#!/bin/bash

# ================= 配置 =================
PROJECT_DIR="/opt/market-sentinel"
# =======================================

set -e

echo "🚀 开始部署 MarketSentinel..."

cd "$PROJECT_DIR"

# 1. 拉取最新镜像
echo "📥 正在从 GHCR 拉取最新镜像..."
docker compose pull

# 2. 重启服务（保留 grafana 不动）
echo "🐳 正在平滑重启服务..."
docker compose up -d

# 3. 清理旧镜像
echo "🧹 清理旧镜像..."
docker image prune -f

echo "🎉 部署完成！"
