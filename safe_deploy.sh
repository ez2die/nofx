#!/bin/bash

# ═══════════════════════════════════════════════════════════════
# NOFX 安全部署脚本 - 降低部署失败服务暂停风险
# Usage: ./safe_deploy.sh [--rollback]
# ═══════════════════════════════════════════════════════════════

set -e  # 遇到错误立即退出

# ------------------------------------------------------------------------
# 配置变量
# ------------------------------------------------------------------------
BACKUP_DIR="backups/$(date +%Y%m%d_%H%M%S)"
COMPOSE_FILE="docker-compose-v2.yml"
HEALTH_CHECK_URL="http://localhost:8080/api/health"
TIMEOUT=60
ROLLBACK_FLAG=false

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# ------------------------------------------------------------------------
# 工具函数
# ------------------------------------------------------------------------
print_info() { echo -e "${BLUE}[INFO]${NC} $1"; }
print_success() { echo -e "${GREEN}[SUCCESS]${NC} $1"; }
print_warn() { echo -e "${YELLOW}[WARN]${NC} $1"; }
print_error() { echo -e "${RED}[ERROR]${NC} $1"; }

# ------------------------------------------------------------------------
# 检测Docker Compose命令
# ------------------------------------------------------------------------
detect_compose_cmd() {
    if command -v docker compose &> /dev/null; then
        COMPOSE_CMD="docker compose"
    elif command -v docker-compose &> /dev/null; then
        COMPOSE_CMD="docker-compose"
    else
        print_error "Docker Compose 未安装！"
        exit 1
    fi
    print_info "使用 Docker Compose 命令: $COMPOSE_CMD"
}

# ------------------------------------------------------------------------
# 步骤1: 备份关键数据
# ------------------------------------------------------------------------
backup_data() {
    print_info "步骤1: 备份关键数据..."
    mkdir -p "$BACKUP_DIR"
    
    # 备份数据库
    if [ -f "config.db" ]; then
        cp config.db "$BACKUP_DIR/" && print_info "  ✓ 数据库已备份"
    else
        print_warn "  ⚠️  数据库文件不存在，跳过"
    fi
    
    # 备份配置文件
    if [ -f "config.json" ]; then
        cp config.json "$BACKUP_DIR/" && print_info "  ✓ 配置文件已备份"
    else
        print_warn "  ⚠️  配置文件不存在，跳过"
    fi
    
    # 备份日志
    if [ -d "decision_logs" ]; then
        tar -czf "$BACKUP_DIR/decision_logs.tar.gz" decision_logs/ 2>/dev/null && \
            print_info "  ✓ 日志已备份" || print_warn "  ⚠️  日志备份失败"
    else
        print_warn "  ⚠️  日志目录不存在，跳过"
    fi
    
    # 备份二进制文件（如果存在）
    if [ -f "nofx" ]; then
        cp nofx "$BACKUP_DIR/" 2>/dev/null && print_info "  ✓ 二进制文件已备份" || true
    fi
    
    print_success "备份完成: $BACKUP_DIR"
}

# ------------------------------------------------------------------------
# 步骤2: 验证当前服务状态
# ------------------------------------------------------------------------
verify_current_service() {
    print_info "步骤2: 验证当前服务状态..."
    
    # 检查健康状态
    if curl -f "$HEALTH_CHECK_URL" > /dev/null 2>&1; then
        print_success "  ✅ 当前服务正常"
    else
        print_warn "  ⚠️  当前服务未响应，但继续部署"
    fi
    
    # 检查容器状态
    if $COMPOSE_CMD -f "$COMPOSE_FILE" ps > /dev/null 2>&1; then
        print_info "  ✓ 容器状态检查完成"
    else
        print_warn "  ⚠️  容器状态检查失败"
    fi
}

# ------------------------------------------------------------------------
# 步骤3: 检查代码修改
# ------------------------------------------------------------------------
check_code_changes() {
    print_info "步骤3: 检查代码修改..."
    
    if git diff --quiet 2>/dev/null; then
        print_info "  ✓ 没有未提交的修改"
    else
        print_warn "  ⚠️  有未提交的修改"
        git diff --stat
        read -p "继续部署? (y/n): " confirm
        if [ "$confirm" != "y" ] && [ "$confirm" != "Y" ]; then
            print_error "部署已取消"
            exit 1
        fi
    fi
}

# ------------------------------------------------------------------------
# 步骤4: 构建新镜像
# ------------------------------------------------------------------------
build_new_image() {
    print_info "步骤4: 构建新版本镜像..."
    
    # 构建后端镜像
    print_info "  构建后端镜像 (nofx-v2)..."
    if $COMPOSE_CMD -f "$COMPOSE_FILE" build --no-cache nofx-v2; then
        print_success "  ✅ 后端镜像构建完成"
    else
        print_error "  ❌ 后端镜像构建失败"
        exit 1
    fi
    
    # 构建前端镜像
    print_info "  构建前端镜像 (nofx-frontend-v2)..."
    if $COMPOSE_CMD -f "$COMPOSE_FILE" build --no-cache nofx-frontend-v2; then
        print_success "  ✅ 前端镜像构建完成"
    else
        print_error "  ❌ 前端镜像构建失败"
        exit 1
    fi
}

# ------------------------------------------------------------------------
# 步骤5: 停止旧容器（优雅停止）
# ------------------------------------------------------------------------
stop_old_container() {
    print_info "步骤5: 停止旧版本容器..."
    
    # 优雅停止后端容器（等待容器正常关闭）
    print_info "  停止后端容器 (nofx-v2)..."
    if $COMPOSE_CMD -f "$COMPOSE_FILE" stop nofx-v2; then
        print_success "  ✅ 后端容器已停止"
        sleep 2  # 等待容器完全停止
    else
        print_warn "  ⚠️  停止后端容器失败（可能已停止）"
    fi
    
    # 优雅停止前端容器（等待容器正常关闭）
    print_info "  停止前端容器 (nofx-frontend-v2)..."
    if $COMPOSE_CMD -f "$COMPOSE_FILE" stop nofx-frontend-v2; then
        print_success "  ✅ 前端容器已停止"
        sleep 2  # 等待容器完全停止
    else
        print_warn "  ⚠️  停止前端容器失败（可能已停止）"
    fi
}

# ------------------------------------------------------------------------
# 步骤6: 启动新容器
# ------------------------------------------------------------------------
start_new_container() {
    print_info "步骤6: 启动新版本容器..."
    
    # 先启动后端容器
    print_info "  启动后端容器 (nofx-v2)..."
    if $COMPOSE_CMD -f "$COMPOSE_FILE" up -d nofx-v2; then
        print_success "  ✅ 后端容器已启动"
        sleep 3  # 等待后端完全启动
    else
        print_error "  ❌ 后端容器启动失败"
        rollback
        exit 1
    fi
    
    # 再启动前端容器（依赖后端）
    print_info "  启动前端容器 (nofx-frontend-v2)..."
    if $COMPOSE_CMD -f "$COMPOSE_FILE" up -d nofx-frontend-v2; then
        print_success "  ✅ 前端容器已启动"
        sleep 2  # 等待前端完全启动
    else
        print_error "  ❌ 前端容器启动失败"
        rollback
        exit 1
    fi
}

# ------------------------------------------------------------------------
# 步骤7: 等待健康检查通过
# ------------------------------------------------------------------------
wait_for_health_check() {
    print_info "步骤7: 等待健康检查通过（最多${TIMEOUT}秒）..."
    
    # 检查后端健康状态
    print_info "  检查后端健康状态..."
    elapsed=0
    backend_healthy=false
    while [ $elapsed -lt $TIMEOUT ]; do
        if curl -f "$HEALTH_CHECK_URL" > /dev/null 2>&1; then
            print_success "  ✅ 后端健康检查通过（耗时${elapsed}秒）"
            backend_healthy=true
            break
        fi
        sleep 2
        elapsed=$((elapsed + 2))
        echo -n "."
    done
    echo ""
    
    if [ "$backend_healthy" = false ]; then
        print_error "  ❌ 后端健康检查失败（超过${TIMEOUT}秒）"
        return 1
    fi
    
    # 检查前端健康状态
    print_info "  检查前端健康状态..."
    elapsed=0
    frontend_healthy=false
    while [ $elapsed -lt 30 ]; do  # 前端健康检查超时30秒
        if curl -f "http://localhost/health" > /dev/null 2>&1; then
            print_success "  ✅ 前端健康检查通过（耗时${elapsed}秒）"
            frontend_healthy=true
            break
        fi
        sleep 2
        elapsed=$((elapsed + 2))
        echo -n "."
    done
    echo ""
    
    if [ "$frontend_healthy" = false ]; then
        print_warn "  ⚠️  前端健康检查失败（超过30秒），但继续验证功能"
    fi
    
    return 0
}

# ------------------------------------------------------------------------
# 步骤8: 验证功能
# ------------------------------------------------------------------------
verify_functionality() {
    print_info "步骤8: 验证功能..."
    
    # 验证后端API可用性
    print_info "  验证后端API..."
    if curl -f "http://localhost:8080/api/traders" > /dev/null 2>&1; then
        print_success "  ✅ 后端API功能正常"
    else
        print_error "  ❌ 后端API功能异常"
        return 1
    fi
    
    # 验证前端访问
    print_info "  验证前端访问..."
    if curl -f "http://localhost/" > /dev/null 2>&1; then
        print_success "  ✅ 前端访问正常"
    else
        print_warn "  ⚠️  前端访问异常，但后端正常"
    fi
    
    # 检查后端容器状态
    if $COMPOSE_CMD -f "$COMPOSE_FILE" ps nofx-v2 | grep -q "Up"; then
        print_success "  ✅ 后端容器运行正常"
    else
        print_error "  ❌ 后端容器状态异常"
        return 1
    fi
    
    # 检查前端容器状态
    if $COMPOSE_CMD -f "$COMPOSE_FILE" ps nofx-frontend-v2 | grep -q "Up"; then
        print_success "  ✅ 前端容器运行正常"
    else
        print_warn "  ⚠️  前端容器状态异常，但后端正常"
    fi
    
    return 0
}

# ------------------------------------------------------------------------
# 步骤9: 检查错误日志
# ------------------------------------------------------------------------
check_error_logs() {
    print_info "步骤9: 检查错误日志..."
    
    # 等待日志生成
    sleep 5
    
    # 检查后端是否有panic或fatal错误
    print_info "  检查后端日志..."
    if $COMPOSE_CMD -f "$COMPOSE_FILE" logs --tail=50 nofx-v2 2>&1 | grep -i "panic\|fatal" > /dev/null; then
        print_error "  ❌ 后端发现严重错误（panic/fatal）"
        $COMPOSE_CMD -f "$COMPOSE_FILE" logs --tail=50 nofx-v2 | grep -i "panic\|fatal"
        return 1
    fi
    
    # 检查后端是否有错误
    backend_error_count=$($COMPOSE_CMD -f "$COMPOSE_FILE" logs --tail=50 nofx-v2 2>&1 | grep -i "error" | wc -l)
    if [ "$backend_error_count" -gt 0 ]; then
        print_warn "  ⚠️  后端发现 $backend_error_count 个错误日志，请检查"
        $COMPOSE_CMD -f "$COMPOSE_FILE" logs --tail=50 nofx-v2 | grep -i "error" | head -5
    else
        print_success "  ✅ 后端未发现错误日志"
    fi
    
    # 检查前端日志（nginx错误）
    print_info "  检查前端日志..."
    frontend_error_count=$($COMPOSE_CMD -f "$COMPOSE_FILE" logs --tail=50 nofx-frontend-v2 2>&1 | grep -iE "error|emerg|crit" | wc -l)
    if [ "$frontend_error_count" -gt 0 ]; then
        print_warn "  ⚠️  前端发现 $frontend_error_count 个错误日志，请检查"
        $COMPOSE_CMD -f "$COMPOSE_FILE" logs --tail=50 nofx-frontend-v2 | grep -iE "error|emerg|crit" | head -5
    else
        print_success "  ✅ 前端未发现错误日志"
    fi
    
    return 0
}

# ------------------------------------------------------------------------
# 回滚函数
# ------------------------------------------------------------------------
rollback() {
    print_error "🔄 开始回滚..."
    
    # 停止新版本容器
    print_info "  停止新版本容器..."
    $COMPOSE_CMD -f "$COMPOSE_FILE" stop nofx-v2 2>/dev/null || true
    $COMPOSE_CMD -f "$COMPOSE_FILE" stop nofx-frontend-v2 2>/dev/null || true
    
    # 重启旧版本（如果有）
    if [ -f "docker-compose.yml" ]; then
        print_info "  使用旧版本配置重启..."
        docker compose restart nofx-v2 2>/dev/null || true
        docker compose restart nofx-frontend-v2 2>/dev/null || true
    fi
    
    # 等待服务恢复
    sleep 5
    if curl -f "$HEALTH_CHECK_URL" > /dev/null 2>&1; then
        print_success "  ✅ 后端回滚成功，服务已恢复"
    else
        print_error "  ❌ 后端回滚后服务仍未恢复，需要手动检查"
    fi
    
    if curl -f "http://localhost/health" > /dev/null 2>&1; then
        print_success "  ✅ 前端回滚成功，服务已恢复"
    else
        print_warn "  ⚠️  前端回滚后服务仍未恢复，需要手动检查"
    fi
}

# ------------------------------------------------------------------------
# 主部署流程
# ------------------------------------------------------------------------
main_deploy() {
    print_info "═══════════════════════════════════════════════════════"
    print_info "开始安全部署 - $(date '+%Y-%m-%d %H:%M:%S')"
    print_info "═══════════════════════════════════════════════════════"
    echo ""
    
    # 检测Docker Compose
    detect_compose_cmd
    
    # 执行部署步骤
    backup_data
    verify_current_service
    check_code_changes
    build_new_image
    stop_old_container
    start_new_container
    
    # 等待健康检查
    if ! wait_for_health_check; then
        print_error "健康检查失败，开始回滚..."
        rollback
        exit 1
    fi
    
    # 验证功能
    if ! verify_functionality; then
        print_error "功能验证失败，开始回滚..."
        rollback
        exit 1
    fi
    
    # 检查错误日志
    if ! check_error_logs; then
        print_error "发现严重错误，开始回滚..."
        rollback
        exit 1
    fi
    
    # 部署成功
    echo ""
    print_success "═══════════════════════════════════════════════════════"
    print_success "部署成功！"
    print_success "═══════════════════════════════════════════════════════"
    echo ""
    print_info "📊 部署信息："
    print_info "  - 备份位置: $BACKUP_DIR"
    print_info "  - 后端健康检查: $HEALTH_CHECK_URL"
    print_info "  - 前端健康检查: http://localhost/health"
    print_info "  - 查看后端日志: $COMPOSE_CMD -f $COMPOSE_FILE logs -f nofx-v2"
    print_info "  - 查看前端日志: $COMPOSE_CMD -f $COMPOSE_FILE logs -f nofx-frontend-v2"
    print_info "  - 查看状态: $COMPOSE_CMD -f $COMPOSE_FILE ps"
    echo ""
    print_warn "⚠️  建议在部署后30分钟内持续监控日志和功能"
}

# ------------------------------------------------------------------------
# 主函数
# ------------------------------------------------------------------------
main() {
    # 检查参数
    if [ "$1" == "--rollback" ]; then
        detect_compose_cmd
        rollback
        exit 0
    fi
    
    # 执行部署
    main_deploy
}

# 执行主函数
main "$@"

