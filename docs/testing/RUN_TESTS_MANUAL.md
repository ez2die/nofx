# 手动运行测试 9 和 10 的步骤

## 📍 执行目录
所有命令都在项目根目录执行：
```bash
cd /root/nofx
```

## 🔧 前置条件

### 1. 确保 Go 环境已配置
```bash
export PATH=$PATH:/usr/local/go/bin
go version
# 应该显示: go version go1.25.0 linux/amd64
```

### 2. 确保配置文件存在
```bash
ls -la config.json
# 如果不存在，从示例创建:
# cp config.json.example config.json
```

## 🚀 手动运行测试步骤

### 步骤 1: 编译生成二进制文件
```bash
cd /root/nofx
export PATH=$PATH:/usr/local/go/bin
go build -o /tmp/nofx_test main.go
```

**验证编译成功：**
```bash
ls -lh /tmp/nofx_test
# 应该看到二进制文件，大小约 40M
```

### 步骤 2: 运行测试 9（二进制文件检查）
```bash
cd /root/nofx
file /tmp/nofx_test
# 应该显示: ELF 64-bit LSB executable, x86-64...
```

### 步骤 3: 运行测试 10（服务启动测试）

#### 选项 A: 使用测试脚本（推荐）
```bash
cd /root/nofx
export PATH=$PATH:/usr/local/go/bin
./test_branch.sh
```

当脚本运行到这一步时，会提示：
```
是否运行服务启动测试？(需要配置文件，y/n):
```
**输入 `y` 然后按回车**，脚本会自动执行测试 10。

#### 选项 B: 手动运行测试 10
```bash
cd /root/nofx
export PATH=$PATH:/usr/local/go/bin

# 1. 确保二进制文件存在
if [ ! -f "/tmp/nofx_test" ]; then
    go build -o /tmp/nofx_test main.go
fi

# 2. 检查端口是否被占用
lsof -Pi :8080 -sTCP:LISTEN || echo "端口 8080 可用"

# 3. 启动服务（5秒后自动停止）
timeout 5 /tmp/nofx_test > /tmp/service_test.log 2>&1 &
SERVICE_PID=$!

# 4. 等待服务启动
sleep 2

# 5. 检查服务是否在运行
if ps -p $SERVICE_PID > /dev/null 2>&1; then
    echo "✓ 服务成功启动"
    
    # 6. 测试健康检查端点
    sleep 1
    curl -s http://localhost:8080/api/health && echo "" || echo "健康检查端点暂时不可访问"
    
    # 7. 停止服务
    kill $SERVICE_PID 2>/dev/null || true
    wait $SERVICE_PID 2>/dev/null || true
    sleep 1
    
    # 8. 验证服务已停止
    if ! ps -p $SERVICE_PID > /dev/null 2>&1; then
        echo "✓ 服务成功停止"
    else
        echo "✗ 服务未能正常停止，强制停止"
        kill -9 $SERVICE_PID 2>/dev/null || true
    fi
else
    echo "✗ 服务未能启动"
    echo "查看启动日志:"
    cat /tmp/service_test.log
fi
```

## 📋 完整测试流程（一次性执行）

```bash
# 1. 切换到项目目录
cd /root/nofx

# 2. 设置 Go 环境
export PATH=$PATH:/usr/local/go/bin

# 3. 编译生成二进制文件
go build -o /tmp/nofx_test main.go

# 4. 验证二进制文件
file /tmp/nofx_test

# 5. 运行完整测试脚本（包含测试 9 和 10）
./test_branch.sh
# 当提示时输入 'y' 来执行测试 10
```

## ⚠️ 注意事项

1. **端口占用**：如果 8080 端口被占用，测试 10 会失败
   - 检查端口占用：`lsof -Pi :8080 -sTCP:LISTEN`
   - 停止占用端口的服务或使用其他端口

2. **配置文件**：测试 10 需要 `config.json` 文件
   - 如果不存在，从示例创建：`cp config.json.example config.json`

3. **二进制文件**：测试 9 和 10 需要 `/tmp/nofx_test` 存在
   - 如果不存在，先运行编译命令

4. **服务启动**：测试 10 会实际启动服务，但会在 5 秒后自动停止
   - 不会影响其他正在运行的服务

## 🔍 验证测试结果

### 测试 9 成功标志：
- ✓ 二进制文件存在：`/tmp/nofx_test`
- ✓ 文件格式正确：`file /tmp/nofx_test` 显示 "executable"

### 测试 10 成功标志：
- ✓ 服务成功启动：进程在运行
- ✓ 健康检查端点可访问：`curl http://localhost:8080/api/health` 返回 JSON
- ✓ 服务成功停止：进程已终止

## 📝 示例输出

### 测试 9 成功示例：
```
[INFO] 测试 9: 测试编译后的二进制文件...
  ✓ 二进制文件格式正确
```

### 测试 10 成功示例：
```
[INFO] 测试 10: 测试服务启动（5秒后自动停止）...
  启动服务（5秒后自动停止）...
  ✓ 服务成功启动
  ✓ 健康检查端点可访问
  ✓ 服务成功停止
```

## 🆘 常见问题

### Q: 编译失败怎么办？
A: 检查 Go 环境是否正确配置：
```bash
export PATH=$PATH:/usr/local/go/bin
go version
```

### Q: 端口 8080 被占用怎么办？
A: 停止占用端口的服务，或修改配置文件中的端口号。

### Q: 服务启动失败怎么办？
A: 查看启动日志：
```bash
cat /tmp/service_test.log
```

### Q: 如何跳过交互式确认？
A: 使用选项 B 手动运行测试 10，或者修改脚本移除 `read` 命令。

