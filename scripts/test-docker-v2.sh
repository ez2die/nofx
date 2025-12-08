#!/bin/bash
# 快速启动测试 Docker 环境 V2 的脚本

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

# 检测 Docker Compose 命令（支持 V1 和 V2）
if command -v docker >/dev/null 2>&1 && docker compose version >/dev/null 2>&1; then
    DOCKER_COMPOSE="docker compose"
    echo "✅ 检测到 Docker Compose V2"
elif command -v docker-compose >/dev/null 2>&1; then
    DOCKER_COMPOSE="docker-compose"
    echo "✅ 检测到 Docker Compose V1"
else
    echo "❌ 错误: 未找到 Docker Compose 命令"
    echo "   请安装 Docker Compose 或确保 docker 命令可用"
    exit 1
fi

echo "🚀 启动 NOFX 测试 Docker 环境 V2"
echo "=================================="
echo ""

# 检查配置文件
if [ ! -f "config.json.test" ]; then
    echo "⚠️  警告: config.json.test 不存在"
    if [ -f "config.json" ]; then
        echo "📋 从 config.json 复制创建 config.json.test..."
        cp config.json config.json.test
        echo "✅ 已创建 config.json.test"
    else
        echo "❌ 错误: config.json 也不存在，请先创建配置文件"
        exit 1
    fi
fi

# 检查端口占用
check_port() {
    local port=$1
    if command -v netstat >/dev/null 2>&1; then
        if netstat -tuln 2>/dev/null | grep -q ":$port "; then
            return 1
        fi
    elif command -v ss >/dev/null 2>&1; then
        if ss -tuln 2>/dev/null | grep -q ":$port "; then
            return 1
        fi
    fi
    return 0
}

echo "🔍 检查端口占用..."
PORTS=(8083 8084 8444)
PORT_NAMES=("后端API" "前端HTTP" "前端HTTPS")
PORT_CONFLICTS=()

for i in "${!PORTS[@]}"; do
    if ! check_port "${PORTS[$i]}"; then
        PORT_CONFLICTS+=("${PORT_NAMES[$i]} (${PORTS[$i]})")
    fi
done

if [ ${#PORT_CONFLICTS[@]} -gt 0 ]; then
    echo "⚠️  警告: 以下端口已被占用:"
    for conflict in "${PORT_CONFLICTS[@]}"; do
        echo "   - $conflict"
    done
    echo ""
    echo "💡 提示: 可以通过环境变量修改端口:"
    echo "   export NOFX_BACKEND_PORT_TEST_V2=8085"
    echo "   export NOFX_FRONTEND_PORT_TEST_V2=8086"
    echo "   export NOFX_FRONTEND_HTTPS_PORT_TEST_V2=8445"
    echo ""
    read -p "是否继续? (y/N): " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        exit 1
    fi
else
    echo "✅ 端口检查通过"
fi

echo ""
echo "🔨 构建并启动 Docker 容器..."
echo ""

# 构建并启动
$DOCKER_COMPOSE -f docker-compose-test-v2.yml up -d --build

echo ""
echo "⏳ 等待服务启动..."
sleep 5

# 检查服务状态
echo ""
echo "📊 服务状态:"
$DOCKER_COMPOSE -f docker-compose-test-v2.yml ps

echo ""
echo "✅ 测试环境 V2 已启动！"
echo ""
echo "📍 访问地址:"
echo "   - 前端 HTTP:  http://localhost:${NOFX_FRONTEND_PORT_TEST_V2:-8084}"
echo "   - 前端 HTTPS: https://localhost:${NOFX_FRONTEND_HTTPS_PORT_TEST_V2:-8444}"
echo "   - 后端 API:   http://localhost:${NOFX_BACKEND_PORT_TEST_V2:-8083}/api/"
echo ""
echo "🔍 查看日志:"
echo "   $DOCKER_COMPOSE -f docker-compose-test-v2.yml logs -f"
echo ""
echo "🛑 停止服务:"
echo "   $DOCKER_COMPOSE -f docker-compose-test-v2.yml down"
echo ""

