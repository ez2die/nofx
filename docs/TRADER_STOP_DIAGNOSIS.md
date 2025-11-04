# Test Trader 停止问题诊断报告

## 问题描述
Test trader 多次被停止，需要定位停止的原因。

## 诊断结果

### 停止时间点
根据日志分析，test trader 在以下时间点被停止：

1. **02:09:11** - `📛 收到退出信号，正在停止所有trader...`
2. **02:22:25** - `📛 收到退出信号，正在停止所有trader...`
   - 02:30:00 - `⏹  交易员 test 已停止`
3. **02:32:12** - `📛 收到退出信号，正在停止所有trader...`
4. **02:41:03** - `📛 收到退出信号，正在停止所有trader...`
5. **02:43:18** - `📛 收到退出信号，正在停止所有trader...`

### 根本原因分析

#### 1. 退出信号来源
从代码分析（`main.go:349-360`）：
```go
// 设置优雅退出
sigChan := make(chan os.Signal, 1)
signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

// 等待退出信号
<-sigChan
log.Println("📛 收到退出信号，正在停止所有trader...")
traderManager.StopAll()
```

**结论：** 系统接收到 `SIGTERM` 或 `SIGINT` 信号时，会优雅退出并停止所有trader。

#### 2. 容器重启
从容器状态检查：
- 容器状态：`running`
- 创建时间：47分钟前
- 启动时间：约1分钟前
- **状态变化：** 容器在多个时间点重启

#### 3. 重启原因推测

可能的原因：
1. **手动重启容器** - 用户执行 `docker compose restart` 命令
2. **容器健康检查失败** - 如果配置了健康检查，失败可能导致重启
3. **资源限制** - 内存或CPU超限可能导致容器被杀死并重启
4. **Docker Compose配置** - restart策略可能设置为always，自动重启

#### 4. 停止流程

正常停止流程：
```
收到 SIGTERM/SIGINT 信号
  ↓
打印 "📛 收到退出信号，正在停止所有trader..."
  ↓
调用 traderManager.StopAll()
  ↓
遍历所有trader，调用 trader.Stop()
  ↓
打印 "⏹  停止所有Trader..."
  ↓
打印 "👋 感谢使用AI交易系统！"
  ↓
程序退出
```

### 观察到的现象

1. **频繁重启** - 容器在短时间内多次重启（02:09, 02:22, 02:32, 02:41, 02:43）
2. **正常停止流程** - 每次停止都遵循优雅退出流程
3. **trader状态丢失** - 容器重启后，trader需要重新启动

### 影响

1. **交易中断** - 容器重启会导致所有正在运行的trader停止
2. **决策周期中断** - 正在执行的决策周期可能被中断
3. **状态丢失** - 内存中的状态会丢失，需要重新加载配置

### 建议解决方案

#### 1. 检查容器重启策略
```bash
docker inspect nofx-trading --format='{{.HostConfig.RestartPolicy.Name}}'
```

#### 2. 检查健康检查配置
```bash
docker inspect nofx-trading --format='{{json .Config.Healthcheck}}'
```

#### 3. 监控容器资源使用
```bash
docker stats nofx-trading --no-stream
```

#### 4. 检查是否有手动重启操作
- 查看是否有定时任务或脚本执行重启
- 检查Docker Compose配置中的restart策略

#### 5. 添加持久化状态
- 将trader运行状态保存到数据库
- 容器重启后自动恢复运行中的trader

### 验证步骤

1. **检查当前容器状态**
   ```bash
   docker ps | grep nofx-trading
   docker inspect nofx-trading --format='{{.State.Status}} {{.State.RestartCount}}'
   ```

2. **检查最近的重启事件**
   ```bash
   docker events --since 1h --filter container=nofx-trading
   ```

3. **监控容器资源**
   ```bash
   docker stats nofx-trading
   ```

4. **检查Docker Compose配置**
   ```bash
   cat docker-compose.yml | grep -A 5 "restart\|healthcheck"
   ```

## 结论

**问题原因：** 容器多次重启，每次重启时收到 `SIGTERM` 信号，触发优雅退出流程，导致所有trader被停止。

**建议：** 
1. 调查容器重启的根本原因（健康检查、资源限制、手动操作等）
2. 实现trader状态的持久化，容器重启后自动恢复
3. 优化容器稳定性，减少不必要的重启

