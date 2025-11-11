#!/bin/bash

# NOFX 测试 Docker 环境管理脚本

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
COMPOSE_FILE="${SCRIPT_DIR}/docker-compose-test.yml"

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# 帮助信息
show_help() {
    cat << EOF
NOFX 测试 Docker 环境管理脚本

用法: $0 [命令]

命令:
    start       启动测试环境（构建并启动）
    stop        停止测试环境
    restart     重启测试环境
    logs        查看日志（所有服务）
    logs-backend 查看后端日志
    logs-frontend 查看前端日志
    status      查看容器状态
    build       仅构建镜像（不启动）
    clean       停止并清理容器
    help        显示此帮助信息

示例:
    $0 start          # 启动测试环境
    $0 logs           # 查看所有日志
    $0 stop           # 停止测试环境
EOF
}

# 检查 Docker 和 Docker Compose
check_dependencies() {
    if ! command -v docker &> /dev/null; then
        echo -e "${RED}错误: 未找到 Docker${NC}"
        exit 1
    fi
    
    if ! docker compose version &> /dev/null; then
        echo -e "${RED}错误: 未找到 Docker Compose${NC}"
        exit 1
    fi
}

# 检查测试环境配置
check_test_config() {
    if [ ! -f "${SCRIPT_DIR}/config.json.test" ]; then
        echo -e "${YELLOW}警告: 未找到 config.json.test 配置文件${NC}"
        echo -e "${YELLOW}测试环境需要独立的配置文件以避免与线上环境冲突${NC}"
        echo ""
        echo -e "${GREEN}请运行以下命令初始化测试环境:${NC}"
        echo "  ./init-test-env.sh"
        echo ""
        read -p "是否现在初始化测试环境? (Y/n) " -n 1 -r
        echo
        if [[ ! $REPLY =~ ^[Nn]$ ]]; then
            if [ -f "${SCRIPT_DIR}/init-test-env.sh" ]; then
                "${SCRIPT_DIR}/init-test-env.sh"
            else
                echo -e "${YELLOW}未找到初始化脚本，手动创建配置文件...${NC}"
                if [ -f "${SCRIPT_DIR}/config.json" ]; then
                    cp "${SCRIPT_DIR}/config.json" "${SCRIPT_DIR}/config.json.test"
                    mkdir -p "${SCRIPT_DIR}/decision_logs_test"
                    echo -e "${GREEN}✓ 已创建测试配置文件${NC}"
                else
                    echo -e "${RED}错误: 未找到 config.json 文件${NC}"
                    exit 1
                fi
            fi
        else
            echo -e "${RED}错误: 缺少测试配置文件，无法启动${NC}"
            exit 1
        fi
    fi
    
    # 确保日志目录存在
    if [ ! -d "${SCRIPT_DIR}/decision_logs_test" ]; then
        mkdir -p "${SCRIPT_DIR}/decision_logs_test"
        echo -e "${GREEN}✓ 已创建测试日志目录${NC}"
    fi
}

# 启动服务
start_services() {
    check_test_config
    echo -e "${GREEN}正在启动测试 Docker 环境...${NC}"
    docker compose -f "${COMPOSE_FILE}" up -d --build
    echo -e "${GREEN}测试环境已启动${NC}"
    echo ""
    show_status
}

# 停止服务
stop_services() {
    echo -e "${YELLOW}正在停止测试 Docker 环境...${NC}"
    docker compose -f "${COMPOSE_FILE}" down
    echo -e "${GREEN}测试环境已停止${NC}"
}

# 重启服务
restart_services() {
    echo -e "${YELLOW}正在重启测试 Docker 环境...${NC}"
    docker compose -f "${COMPOSE_FILE}" restart
    echo -e "${GREEN}测试环境已重启${NC}"
    show_status
}

# 查看日志
show_logs() {
    docker compose -f "${COMPOSE_FILE}" logs -f
}

# 查看后端日志
show_backend_logs() {
    docker compose -f "${COMPOSE_FILE}" logs -f nofx-test
}

# 查看前端日志
show_frontend_logs() {
    docker compose -f "${COMPOSE_FILE}" logs -f nofx-frontend-test
}

# 查看状态
show_status() {
    echo -e "${GREEN}容器状态:${NC}"
    docker compose -f "${COMPOSE_FILE}" ps
    echo ""
    echo -e "${GREEN}访问地址:${NC}"
    echo "  前端 HTTP:  http://localhost:8082"
    echo "  前端 HTTPS: https://localhost:8443"
    echo "  后端 API:   http://localhost:8081/api/"
    echo ""
    echo -e "${GREEN}健康检查:${NC}"
    echo "  前端: http://localhost:8082/health"
    echo "  后端: http://localhost:8081/api/health"
}

# 构建镜像
build_images() {
    echo -e "${GREEN}正在构建镜像...${NC}"
    docker compose -f "${COMPOSE_FILE}" build
    echo -e "${GREEN}镜像构建完成${NC}"
}

# 清理
clean_services() {
    echo -e "${YELLOW}正在清理测试 Docker 环境...${NC}"
    read -p "确定要停止并删除测试容器吗? (y/N) " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        docker compose -f "${COMPOSE_FILE}" down
        echo -e "${GREEN}清理完成${NC}"
    else
        echo -e "${YELLOW}已取消${NC}"
    fi
}

# 主逻辑
main() {
    check_dependencies
    
    case "${1:-help}" in
        start)
            start_services
            ;;
        stop)
            stop_services
            ;;
        restart)
            restart_services
            ;;
        logs)
            show_logs
            ;;
        logs-backend)
            show_backend_logs
            ;;
        logs-frontend)
            show_frontend_logs
            ;;
        status)
            show_status
            ;;
        build)
            build_images
            ;;
        clean)
            clean_services
            ;;
        help|--help|-h)
            show_help
            ;;
        *)
            echo -e "${RED}错误: 未知命令 '${1}'${NC}"
            echo ""
            show_help
            exit 1
            ;;
    esac
}

main "$@"

