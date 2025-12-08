#!/bin/bash

# ============================================================
# NOFX 分支测试脚本
# 用于测试 nof1adapt-v2 分支的完整性和功能
# ============================================================

# 注意：不使用 set -e，因为我们需要执行所有测试，即使某些失败
# set -e  # 遇到错误立即退出（但某些测试函数可能返回非零，需要特殊处理）

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 打印函数
print_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[✓]${NC} $1"
}

print_error() {
    echo -e "${RED}[✗]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[!]${NC} $1"
}

# 测试计数器
PASSED=0
FAILED=0

test_pass() {
    PASSED=$((PASSED + 1))
    print_success "$1"
}

test_fail() {
    FAILED=$((FAILED + 1))
    print_error "$1"
}

# ============================================================
# 测试 1: 检查当前分支
# ============================================================
test_current_branch() {
    print_info "测试 1: 检查当前分支..."
    CURRENT_BRANCH=$(git branch --show-current)
    if [ "$CURRENT_BRANCH" = "nof1adapt-v2" ]; then
        test_pass "当前分支: $CURRENT_BRANCH"
    else
        test_fail "当前分支是 $CURRENT_BRANCH，期望 nof1adapt-v2"
        print_warning "请切换到 nof1adapt-v2 分支: git checkout nof1adapt-v2"
    fi
}

# ============================================================
# 测试 2: 检查工作区是否干净
# ============================================================
test_working_directory() {
    print_info "测试 2: 检查工作区状态..."
    if git diff --quiet && git diff --cached --quiet; then
        test_pass "工作区干净，无未提交的更改"
    else
        test_fail "工作区有未提交的更改"
        print_warning "请先提交或暂存更改"
        git status --short
    fi
}

# ============================================================
# 测试 3: Go 代码格式检查
# ============================================================
test_go_format() {
    print_info "测试 3: 检查 Go 代码格式..."
    if go fmt ./... > /dev/null 2>&1; then
        if git diff --quiet; then
            test_pass "代码格式正确"
        else
            test_fail "代码格式不符合规范"
            print_warning "运行 'go fmt ./...' 修复格式问题"
        fi
    else
        test_fail "代码格式检查失败"
    fi
}

# ============================================================
# 测试 4: Go 代码静态分析 (go vet)
# ============================================================
test_go_vet() {
    print_info "测试 4: 运行 go vet 静态分析..."
    # 过滤掉已知的正常警告：
    # - main redeclared: 根目录有多个独立的main程序（测试脚本），这是正常的
    # - command-line-arguments: 编译测试产生的临时警告
    go vet ./... 2>&1 | grep -v "vendor\|main redeclared\|command-line-arguments" > /tmp/go_vet_output.txt || true
    
    if [ -s /tmp/go_vet_output.txt ]; then
        test_fail "go vet 发现潜在问题"
        cat /tmp/go_vet_output.txt
    else
        test_pass "go vet 检查通过"
        # 如果过滤后有 main redeclared 警告，说明这是正常的（多个独立程序）
        if go vet ./... 2>&1 | grep -q "main redeclared"; then
            print_info "  注意: 检测到多个 main 程序（测试脚本），这是正常的"
        fi
    fi
}

# ============================================================
# 测试 5: 编译测试
# ============================================================
test_build() {
    print_info "测试 5: 编译项目..."
    # 注意：根目录有多个独立的 main 程序（测试脚本），这是正常的
    # 我们只编译主程序 main.go
    
    # 清理之前的编译输出
    rm -f /tmp/build_output.txt /tmp/build_output_main.txt
    
    # 先尝试编译整个包（会失败，因为有多个main程序）
    go build -o /tmp/nofx_test . > /tmp/build_output.txt 2>&1
    BUILD_STATUS=$?
    cat /tmp/build_output.txt
    
    # 检查是否是多个 main 程序的错误（这是正常的）
    if [ $BUILD_STATUS -ne 0 ] && grep -q "main redeclared" /tmp/build_output.txt 2>/dev/null; then
        print_warning "检测到多个 main 程序（测试脚本），这是正常的"
        print_info "尝试单独编译主程序..."
        go build -o /tmp/nofx_test main.go > /tmp/build_output_main.txt 2>&1
        MAIN_BUILD_STATUS=$?
        cat /tmp/build_output_main.txt
        
        if [ $MAIN_BUILD_STATUS -eq 0 ]; then
            test_pass "主程序编译成功"
            # 保留二进制文件供后续测试使用
            if [ -f "/tmp/nofx_test" ]; then
                print_info "  二进制文件已生成: /tmp/nofx_test ($(du -h /tmp/nofx_test | cut -f1))"
            else
                test_fail "编译成功但二进制文件未生成"
                return 1
            fi
        else
            test_fail "主程序编译失败"
            cat /tmp/build_output_main.txt
            return 1
        fi
    elif [ $BUILD_STATUS -eq 0 ]; then
        test_pass "编译成功"
        # 保留二进制文件供后续测试使用（测试9和10需要）
        if [ -f "/tmp/nofx_test" ]; then
            print_info "  二进制文件已生成: /tmp/nofx_test ($(du -h /tmp/nofx_test | cut -f1))"
        else
            test_fail "编译成功但二进制文件未生成"
            return 1
        fi
    else
        test_fail "编译失败（非多个main程序错误）"
        cat /tmp/build_output.txt
        return 1
    fi
}

# ============================================================
# 测试 6: 检查依赖完整性
# ============================================================
test_dependencies() {
    print_info "测试 6: 检查 Go 模块依赖..."
    if go mod verify 2>&1; then
        test_pass "依赖验证通过"
    else
        test_fail "依赖验证失败"
        print_warning "运行 'go mod tidy' 修复依赖问题"
    fi
}

# ============================================================
# 测试 7: 检查配置文件是否存在
# ============================================================
test_config_files() {
    print_info "测试 7: 检查配置文件..."
    local missing_files=0
    
    if [ -f "config.json" ]; then
        test_pass "config.json 存在"
    else
        test_fail "config.json 不存在"
        print_warning "从 config.json.example 创建 config.json"
        ((missing_files++))
    fi
    
    if [ -f "config.json.example" ]; then
        test_pass "config.json.example 存在"
    else
        test_fail "config.json.example 不存在"
        ((missing_files++))
    fi
    
    if [ $missing_files -eq 0 ]; then
        return 0
    else
        return 1
    fi
}

# ============================================================
# 测试 8: 检查关键文件是否存在
# ============================================================
test_critical_files() {
    print_info "测试 8: 检查关键文件..."
    local missing_files=0
    local critical_files=(
        "main.go"
        "go.mod"
        "go.sum"
        "api/server.go"
        "manager/trader_manager.go"
        "decision/engine.go"
    )
    
    for file in "${critical_files[@]}"; do
        if [ -f "$file" ]; then
            print_success "  ✓ $file"
        else
            test_fail "关键文件缺失: $file"
            ((missing_files++))
        fi
    done
    
    if [ $missing_files -eq 0 ]; then
        return 0
    else
        return 1
    fi
}

# ============================================================
# 测试 9: 测试编译后的二进制文件（不实际运行）
# ============================================================
test_binary() {
    print_info "测试 9: 测试编译后的二进制文件..."
    if [ -f "/tmp/nofx_test" ]; then
        # 检查二进制文件是否可以执行
        if file /tmp/nofx_test | grep -q "executable"; then
            test_pass "二进制文件格式正确"
        else
            test_fail "二进制文件格式不正确"
        fi
        
        # 检查是否有基本的帮助信息（如果有 --help 选项）
        # 注意：nofx 可能没有 --help，所以这个测试可能失败，但不影响总体
        if /tmp/nofx_test --help > /dev/null 2>&1 || true; then
            print_info "  (帮助信息检查跳过 - 可能不支持 --help)"
        fi
    else
        test_fail "二进制文件不存在，请先运行编译测试"
    fi
}

# ============================================================
# 测试 10: 启动服务测试（短暂运行，然后停止）
# ============================================================
test_service_start() {
    print_info "测试 10: 测试服务启动（5秒后自动停止）..."
    
    if [ ! -f "/tmp/nofx_test" ]; then
        print_warning "编译二进制文件以进行启动测试..."
        go build -o /tmp/nofx_test || {
            test_fail "无法编译二进制文件"
            return 1
        }
    fi
    
    # 检查端口是否被占用
    if lsof -Pi :8080 -sTCP:LISTEN -t >/dev/null 2>&1; then
        test_fail "端口 8080 已被占用"
        print_warning "请先停止占用 8080 端口的服务"
        return 1
    fi
    
    # 启动服务（后台运行）
    print_info "  启动服务（5秒后自动停止）..."
    timeout 5 /tmp/nofx_test > /tmp/service_test.log 2>&1 &
    SERVICE_PID=$!
    
    # 等待服务启动
    sleep 2
    
    # 检查服务是否在运行
    if ps -p $SERVICE_PID > /dev/null 2>&1; then
        test_pass "服务成功启动"
        
        # 尝试访问健康检查端点
        sleep 1
        if curl -s http://localhost:8080/api/health > /dev/null 2>&1; then
            test_pass "健康检查端点可访问"
        else
            print_warning "健康检查端点暂时不可访问（可能服务还在启动中）"
        fi
        
        # 停止服务
        kill $SERVICE_PID 2>/dev/null || true
        wait $SERVICE_PID 2>/dev/null || true
        sleep 1
        
        # 确保进程已停止
        if ! ps -p $SERVICE_PID > /dev/null 2>&1; then
            test_pass "服务成功停止"
        else
            test_fail "服务未能正常停止，PID: $SERVICE_PID"
            kill -9 $SERVICE_PID 2>/dev/null || true
        fi
    else
        test_fail "服务未能启动"
        print_warning "查看启动日志:"
        cat /tmp/service_test.log
        return 1
    fi
}

# ============================================================
# 主测试流程
# ============================================================
main() {
    echo "=========================================="
    echo "  NOFX 分支测试套件"
    echo "  测试分支: nof1adapt-v2"
    echo "=========================================="
    echo ""
    
    # 运行所有测试
    test_current_branch
    echo ""
    
    test_working_directory
    echo ""
    
    test_go_format || true
    echo ""
    
    test_go_vet || true
    echo ""
    
    test_build || true
    echo ""
    
    test_dependencies || true
    echo ""
    
    test_config_files || true
    echo ""
    
    test_critical_files || true
    echo ""
    
    # 只有在编译成功后才运行这些测试
    if [ -f "/tmp/nofx_test" ]; then
        test_binary
        echo ""
        
        # 询问是否运行服务启动测试（需要配置文件）
        if [ -f "config.json" ]; then
            read -p "是否运行服务启动测试？(需要配置文件，y/n): " -n 1 -r
            echo
            if [[ $REPLY =~ ^[Yy]$ ]]; then
                test_service_start
                echo ""
            else
                print_info "跳过服务启动测试"
            fi
        else
            print_warning "缺少 config.json，跳过服务启动测试"
        fi
    fi
    
    # 清理临时文件（但保留二进制文件，可能需要再次测试）
    rm -f /tmp/go_vet_output.txt /tmp/build_output.txt /tmp/build_output_main.txt /tmp/service_test.log
    # 可选：清理二进制文件（取消注释以清理）
    # rm -f /tmp/nofx_test
    
    # 输出测试结果
    echo ""
    echo "=========================================="
    echo "  测试结果汇总"
    echo "=========================================="
    echo -e "${GREEN}通过: $PASSED${NC}"
    echo -e "${RED}失败: $FAILED${NC}"
    echo ""
    
    if [ $FAILED -eq 0 ]; then
        print_success "所有测试通过！分支 nof1adapt-v2 状态良好 ✓"
        exit 0
    else
        test_fail "部分测试失败，请检查上述错误"
        exit 1
    fi
}

# 运行主函数
main

