# 交易历史与Trader关系维护分析

## 问题

当前设计中使用外键约束：
```sql
FOREIGN KEY (trader_id) REFERENCES traders(id) ON DELETE CASCADE
```

**问题**：删除trader时，所有交易历史会被级联删除，可能导致数据丢失。

## 关系维护方案对比

### 方案1：硬外键 + CASCADE（当前设计）

```sql
FOREIGN KEY (trader_id) REFERENCES traders(id) ON DELETE CASCADE
```

**优点**：
- ✅ 数据库层面保证数据一致性
- ✅ 自动维护关系，不需要手动清理
- ✅ 避免孤立记录（orphan records）

**缺点**：
- ❌ 删除trader时，所有交易历史被删除
- ❌ 无法恢复历史数据
- ❌ 影响审计和统计分析
- ❌ 可能违反数据保留要求（合规性）

**适用场景**：
- 测试环境
- 不需要保留历史数据的场景
- 数据量小，重建成本低

---

### 方案2：软外键（推荐）

```sql
-- 不创建外键约束，只维护逻辑关系
-- trader_id TEXT NOT NULL,  -- 仅作为普通字段，无外键约束
-- 添加索引：CREATE INDEX idx_trade_history_trader_id ON trade_history(trader_id);
```

**优点**：
- ✅ 删除trader时，交易历史保留
- ✅ 支持历史数据查询和统计
- ✅ 支持数据恢复（如果trader被误删）
- ✅ 符合审计要求（保留历史记录）
- ✅ 不影响数据完整性查询（通过应用层维护）

**缺点**：
- ⚠️ 需要应用层维护数据一致性
- ⚠️ 可能产生孤立记录（需要手动清理）

**适用场景**：
- 生产环境
- 需要保留历史数据的场景
- 需要审计和合规的场景

---

### 方案3：软删除 + 外键

```sql
-- traders表添加deleted_at字段
ALTER TABLE traders ADD COLUMN deleted_at DATETIME;

-- trade_history表仍使用外键，但通过软删除保留数据
FOREIGN KEY (trader_id) REFERENCES traders(id) ON DELETE RESTRICT
```

**优点**：
- ✅ 保留历史数据
- ✅ 数据库层面维护关系
- ✅ 支持数据恢复

**缺点**：
- ⚠️ 需要修改traders表结构
- ⚠️ 需要修改删除逻辑（改为UPDATE而不是DELETE）
- ⚠️ 查询时需要过滤已删除的trader

**适用场景**：
- 需要完整的数据保留策略
- 需要频繁恢复trader的场景

---

### 方案4：归档表（高级方案）

```sql
-- trade_history表：活跃交易历史
-- trade_history_archive表：归档历史（trader删除后移动）

-- 删除trader时，将交易历史移动到归档表
INSERT INTO trade_history_archive SELECT * FROM trade_history WHERE trader_id = ?;
DELETE FROM trade_history WHERE trader_id = ?;
```

**优点**：
- ✅ 保留所有历史数据
- ✅ 主表保持清洁
- ✅ 支持数据恢复和查询

**缺点**：
- ⚠️ 实现复杂
- ⚠️ 需要维护两个表
- ⚠️ 查询时需要UNION

**适用场景**：
- 数据量大的场景
- 需要长期归档的场景

---

## 推荐方案：软外键（方案2）

### 理由

1. **数据保留**：交易历史是重要的业务数据，不应该因为trader被删除而丢失
2. **审计需求**：即使trader被删除，也需要保留历史记录用于审计和合规
3. **统计分析**：历史数据对于分析和优化非常重要
4. **恢复能力**：如果trader被误删，可以通过历史数据恢复

### 设计调整

```sql
CREATE TABLE IF NOT EXISTS trade_history (
    -- ... 其他字段 ...
    
    trader_id TEXT NOT NULL,  -- 仅作为普通字段，无外键约束
    
    -- ... 其他字段 ...
    
    -- 索引优化查询
    CREATE INDEX IF NOT EXISTS idx_trade_history_trader_id ON trade_history(trader_id);
    
    -- 不创建外键约束
    -- FOREIGN KEY (trader_id) REFERENCES traders(id) ON DELETE CASCADE
);
```

### 应用层维护

```go
// 在Repository层添加验证方法
func (r *repository) ValidateTraderExists(ctx context.Context, traderID string) (bool, error) {
    var exists bool
    err := r.db.QueryRow(`
        SELECT EXISTS(SELECT 1 FROM traders WHERE id = ?)
    `, traderID).Scan(&exists)
    return exists, err
}

// 在Service层保存前验证
func (s *service) RecordTrade(ctx context.Context, record *TradeRecord) error {
    // 验证trader是否存在（可选，不影响保存）
    exists, _ := s.repo.ValidateTraderExists(ctx, record.TraderID)
    if !exists {
        log.Printf("⚠️ 警告：trader %s 不存在，但交易历史仍会保存", record.TraderID)
    }
    
    // 保存交易记录
    return s.repo.Save(ctx, record)
}
```

### 查询优化

```go
// 查询时，如果trader不存在，仍然可以查询交易历史
func (s *service) GetTradeHistory(ctx context.Context, filter *TradeRecordFilter) ([]*TradeRecord, error) {
    // 即使trader已被删除，仍然可以查询历史
    records, err := s.repo.FindByFilter(ctx, filter)
    
    // 可选：标记trader是否仍然存在
    for _, record := range records {
        exists, _ := s.repo.ValidateTraderExists(ctx, record.TraderID)
        if !exists {
            // 可以添加标记，但不影响查询结果
            // record.TraderDeleted = true
        }
    }
    
    return records, err
}
```

---

## 数据一致性维护

### 1. 写入时验证（可选）

```go
// 在保存交易记录时，验证trader是否存在
func (s *service) RecordTrade(ctx context.Context, record *TradeRecord) error {
    // 验证trader是否存在
    exists, err := s.repo.ValidateTraderExists(ctx, record.TraderID)
    if err != nil {
        log.Printf("⚠️ 验证trader存在性失败: %v", err)
        // 继续保存，不阻止
    }
    
    if !exists {
        log.Printf("⚠️ 警告：保存交易记录时，trader %s 不存在", record.TraderID)
        // 可以选择记录警告日志，但不阻止保存
    }
    
    // 保存交易记录
    return s.repo.Save(ctx, record)
}
```

### 2. 定期清理（可选）

```go
// 定期清理孤立记录（如果trader被删除超过30天）
func (s *service) CleanupOrphanRecords(ctx context.Context, days int) error {
    cutoffTime := time.Now().AddDate(0, 0, -days)
    
    // 查找trader不存在且时间超过阈值的记录
    orphanRecords, err := s.repo.FindOrphanRecords(ctx, cutoffTime)
    if err != nil {
        return err
    }
    
    // 可以选择归档或删除
    // 1. 归档到trade_history_archive表
    // 2. 或者直接删除（如果确认不需要保留）
    
    return nil
}
```

### 3. 查询时标记

```go
// 查询时，标记trader是否仍然存在
type TradeRecordWithStatus struct {
    *TradeRecord
    TraderExists bool `json:"trader_exists"`
    TraderName   string `json:"trader_name,omitempty"`
}

func (s *service) GetTradeHistoryWithStatus(ctx context.Context, filter *TradeRecordFilter) ([]*TradeRecordWithStatus, error) {
    records, err := s.repo.FindByFilter(ctx, filter)
    if err != nil {
        return nil, err
    }
    
    result := make([]*TradeRecordWithStatus, len(records))
    for i, record := range records {
        // 查询trader信息
        trader, _ := s.traderRepo.GetTrader(record.TraderID)
        
        result[i] = &TradeRecordWithStatus{
            TradeRecord: record,
            TraderExists: trader != nil,
            TraderName:   trader != nil ? trader.Name : "",
        }
    }
    
    return result, nil
}
```

---

## 最终推荐

### 推荐：软外键（方案2）

**表结构**：
```sql
CREATE TABLE IF NOT EXISTS trade_history (
    -- ... 其他字段 ...
    trader_id TEXT NOT NULL,  -- 无外键约束
    -- ... 其他字段 ...
);

-- 仅创建索引，不创建外键
CREATE INDEX IF NOT EXISTS idx_trade_history_trader_id ON trade_history(trader_id);
```

**优点**：
1. ✅ 保留历史数据，即使trader被删除
2. ✅ 支持审计和合规要求
3. ✅ 支持数据恢复
4. ✅ 不影响查询性能（通过索引优化）
5. ✅ 实现简单，不需要修改现有trader删除逻辑

**注意事项**：
1. ⚠️ 应用层需要维护数据一致性（可选验证）
2. ⚠️ 可能产生孤立记录（可通过定期清理解决）
3. ⚠️ 查询时需要验证trader是否存在（可选）

---

## 实施建议

### 1. 修改表结构设计

```sql
-- 移除外键约束，只保留索引
CREATE TABLE IF NOT EXISTS trade_history (
    -- ... 其他字段 ...
    trader_id TEXT NOT NULL,  -- 无外键约束
    -- ... 其他字段 ...
);

-- 仅创建索引
CREATE INDEX IF NOT EXISTS idx_trade_history_trader_id ON trade_history(trader_id);
```

### 2. 添加验证方法（可选）

```go
// 在Repository层添加验证方法
func (r *repository) ValidateTraderExists(ctx context.Context, traderID string) (bool, error) {
    // ...
}
```

### 3. 在Service层添加警告日志（可选）

```go
// 保存时验证并记录警告
func (s *service) RecordTrade(ctx context.Context, record *TradeRecord) error {
    // 验证trader是否存在（可选）
    // 记录警告日志（如果不存在）
    // 继续保存（不阻止）
}
```

### 4. 添加清理功能（可选）

```go
// 定期清理孤立记录（可选）
func (s *service) CleanupOrphanRecords(ctx context.Context, days int) error {
    // ...
}
```

---

## 总结

**推荐使用软外键（方案2）**：
- 保留历史数据，符合业务需求
- 实现简单，不需要修改现有逻辑
- 通过索引优化查询性能
- 应用层可选择性维护一致性

**关键点**：
- 不创建外键约束
- 只创建索引优化查询
- 应用层可选验证（不阻止保存）
- 查询时可选标记trader状态

