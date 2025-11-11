#!/bin/bash

# NOFX 测试环境初始化脚本
# 用于创建测试环境所需的独立配置文件和目录

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "${SCRIPT_DIR}"

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}NOFX 测试环境初始化${NC}"
echo ""

# 检查源配置文件是否存在
if [ ! -f "config.json" ]; then
    echo -e "${RED}错误: 未找到 config.json 文件${NC}"
    echo "请确保在项目根目录运行此脚本"
    exit 1
fi

# 创建测试配置文件
if [ -f "config.json.test" ]; then
    echo -e "${YELLOW}警告: config.json.test 已存在${NC}"
    read -p "是否覆盖现有文件? (y/N) " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        echo -e "${YELLOW}跳过配置文件创建${NC}"
    else
        cp config.json config.json.test
        echo -e "${GREEN}✓ 已创建 config.json.test${NC}"
    fi
else
    cp config.json config.json.test
    echo -e "${GREEN}✓ 已创建 config.json.test${NC}"
fi

# 创建测试日志目录
if [ ! -d "decision_logs_test" ]; then
    mkdir -p decision_logs_test
    echo -e "${GREEN}✓ 已创建 decision_logs_test/ 目录${NC}"
else
    echo -e "${BLUE}✓ decision_logs_test/ 目录已存在${NC}"
fi

# 检查数据库文件
if [ -f "config.db.test" ]; then
    echo -e "${BLUE}✓ config.db.test 已存在${NC}"
else
    echo -e "${YELLOW}ℹ config.db.test 不存在，首次启动时会自动创建${NC}"
fi

echo ""
echo -e "${GREEN}测试环境初始化完成！${NC}"
echo ""
echo -e "${BLUE}下一步:${NC}"
echo "  1. 根据需要修改 config.json.test 中的配置"
echo "  2. 运行: ./test-docker.sh start"
echo ""

