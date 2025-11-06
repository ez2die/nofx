# Dashboard 最近决策查询逻辑问题分析

## 问题描述

- **现象**：在日志文件夹中只能看到cycle15（11月6日），但前端显示有cycle16
- **原因**：`GetLatestRecords` 按文件修改时间排序，而不是按 `cycle_number` 或 `timestamp` 排序

## 问题分析

### 1. 文件情况

- **11月5日 cycle16**: `decision_20251105_232052_cycle16.json`
  - 时间戳: 2025-11-05 23:20:52
  - Cycle Number: 16
  
- **11月6日 cycle15**: `decision_20251106_111402_cycle15.json`
  - 时间戳: 2025-11-06 11:14:02
  - Cycle Number: 15

### 2. 当前实现

```go
// GetLatestRecords 获取最近N条记录（按时间正序：从旧到新）
func (l *DecisionLogger) GetLatestRecords(n int) ([]*DecisionRecord, error) {
    files, err := ioutil.ReadDir(l.logDir)
    if err != nil {
        return nil, fmt.Errorf("读取日志目录失败: %w", err)
    }

    // 先按修改时间倒序收集（最新的在前）
    var records []*DecisionRecord
    count := 0
    for i := len(files) - 1; i >= 0 && count < n; i-- {
        file := files[i]
        // ... 读取文件并解析 ...
        records = append(records, &record)
        count++
    }

    // 反转数组，让时间从旧到新排列（用于图表显示）
    for i, j := 0, len(records)-1; i < j; i, j = i+1, j-1 {
        records[i], records[j] = records[j], records[i]
    }

    return records, nil
}
```

### 3. 问题根源

1. **`ReadDir` 返回顺序不确定**：
   - `ioutil.ReadDir` 返回的文件顺序可能不是按修改时间排序
   - 即使假设是按名称排序，也不能保证是最新的文件

2. **按文件修改时间排序不可靠**：
   - 如果文件被手动修改或复制，修改时间会改变
   - 文件系统的修改时间可能不准确

3. **应该按 `timestamp` 或 `cycle_number` 排序**：
   - 决策记录中有 `timestamp` 字段（JSON中的时间戳）
   - 决策记录中有 `cycle_number` 字段（周期编号）
   - 应该读取文件内容，解析JSON，然后按这些字段排序

## 解决方案

### 方案1：按 `timestamp` 排序（推荐）

修改 `GetLatestRecords`，读取所有文件，解析JSON，按 `timestamp` 排序：

```go
func (l *DecisionLogger) GetLatestRecords(n int) ([]*DecisionRecord, error) {
    files, err := ioutil.ReadDir(l.logDir)
    if err != nil {
        return nil, fmt.Errorf("读取日志目录失败: %w", err)
    }

    // 读取所有文件并解析
    var records []*DecisionRecord
    for _, file := range files {
        if file.IsDir() {
            continue
        }

        filepath := filepath.Join(l.logDir, file.Name())
        data, err := ioutil.ReadFile(filepath)
        if err != nil {
            continue
        }

        var record DecisionRecord
        if err := json.Unmarshal(data, &record); err != nil {
            continue
        }

        records = append(records, &record)
    }

    // 按 timestamp 排序（从旧到新）
    sort.Slice(records, func(i, j int) bool {
        return records[i].Timestamp.Before(records[j].Timestamp)
    })

    // 返回最后N条（最新的）
    if len(records) > n {
        records = records[len(records)-n:]
    }

    return records, nil
}
```

### 方案2：按 `cycle_number` 排序

如果cycle_number是连续的，可以按cycle_number排序：

```go
// 按 cycle_number 排序（从小到大）
sort.Slice(records, func(i, j int) bool {
    return records[i].CycleNumber < records[j].CycleNumber
})
```

### 方案3：混合排序（最可靠）

如果timestamp和cycle_number都可用，优先按timestamp，如果timestamp相同则按cycle_number：

```go
sort.Slice(records, func(i, j int) bool {
    if records[i].Timestamp.Equal(records[j].Timestamp) {
        return records[i].CycleNumber < records[j].CycleNumber
    }
    return records[i].Timestamp.Before(records[j].Timestamp)
})
```

## API端点影响

- **`/api/decisions/latest`**：调用 `GetLatestRecords(5)`
  - 前端使用：`web/src/App.tsx:144-154`
  - 轮询间隔：30秒
  - 影响：前端显示的"最近决策"可能不是最新的

## 建议

1. **立即修复**：修改 `GetLatestRecords`，按 `timestamp` 排序
2. **测试验证**：确保返回的记录是按时间排序的
3. **文档更新**：更新API文档，说明排序逻辑

