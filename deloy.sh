#!/bin/bash

# 学习助手后端 Docker 一键部署脚本

set -e

echo "🚀 开始部署学习助手后端服务..."

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 检查 Docker 是否安装
if ! command -v docker &> /dev/null; then
    echo -e "${RED}❌ Docker 未安装，请先安装 Docker${NC}"
    exit 1
fi

# 检查 Docker Compose 是否安装
if ! command -v docker-compose &> /dev/null && ! docker compose version &> /dev/null; then
    echo -e "${RED}❌ Docker Compose 未安装，请先安装 Docker Compose${NC}"
    exit 1
fi

# 检查环境变量文件
if [ ! -f ".env" ]; then
    echo -e "${YELLOW}⚠️  未找到 .env 文件${NC}"
    if [ -f ".env.docker" ]; then
        echo -e "${BLUE}📋 复制 .env.docker 模板...${NC}"
        cp .env.docker .env
        echo -e "${YELLOW}⚠️  请编辑 .env 文件，填入真实的 ARK_API_KEY${NC}"
        echo -e "${YELLOW}⚠️  编辑完成后请重新运行此脚本${NC}"
        exit 1
    else
        echo -e "${RED}❌ 未找到环境变量模板文件${NC}"
        exit 1
    fi
fi

# 检查 ARK_API_KEY 是否设置
if ! grep -q "^ARK_API_KEY=.*[^=]" .env; then
    echo -e "${RED}❌ ARK_API_KEY 未设置，请编辑 .env 文件${NC}"
    exit 1
fi

echo -e "${GREEN}✅ 环境检查通过${NC}"

# 停止现有容器
echo -e "${BLUE}🛑 停止现有容器...${NC}"
docker-compose down 2>/dev/null || docker compose down 2>/dev/null || true

# 清理旧镜像（可选）
read -p "是否清理旧的 Docker 镜像？(y/N): " -n 1 -r
echo
if [[ $REPLY =~ ^[Yy]$ ]]; then
    echo -e "${BLUE}🧹 清理旧镜像...${NC}"
    docker image prune -f
    docker rmi $(docker images "everyday-study-backend*" -q) 2>/dev/null || true
fi

# 构建并启动服务
echo -e "${BLUE}🔨 构建并启动服务...${NC}"
if command -v docker-compose &> /dev/null; then
    docker-compose up -d --build
else
    docker compose up -d --build
fi

# 等待服务启动
echo -e "${BLUE}⏳ 等待服务启动...${NC}"
sleep 10

# 检查服务状态
echo -e "${BLUE}🔍 检查服务状态...${NC}"
if curl -f -s http://localhost:91/api/health > /dev/null; then
    echo -e "${GREEN}✅ 服务启动成功！${NC}"
    echo -e "${GREEN}📡 服务地址: http://localhost:91${NC}"
    echo -e "${GREEN}🔍 健康检查: http://localhost:91/api/health${NC}"
    echo ""
    echo -e "${BLUE}📊 可用的 API 接口：${NC}"
    echo "   GET  http://localhost:91/api/health"
    echo "   GET  http://localhost:91/api/today-learning/english"
    echo "   GET  http://localhost:91/api/today-learning/chinese"
    echo "   GET  http://localhost:91/api/today-learning/tcm"
    echo "   GET  http://localhost:91/api/learning-history"
    echo "   GET  http://localhost:91/api/stats"
    echo ""
    echo -e "${BLUE}🔧 管理命令：${NC}"
    echo "   查看日志: docker-compose logs -f"
    echo "   停止服务: docker-compose down"
    echo "   重启服务: docker-compose restart"
    echo "   查看状态: docker-compose ps"
else
    echo -e "${RED}❌ 服务启动失败${NC}"
    echo -e "${YELLOW}📋 查看日志:${NC}"
    if command -v docker-compose &> /dev/null; then
        docker-compose logs --tail=20
    else
        docker compose logs --tail=20
    fi
    exit 1
fi

echo -e "${GREEN}🎉 部署完成！${NC}"