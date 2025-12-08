# Hyperliquid OI API 分析报告

## 问题描述

Hyperliquid API 在获取 Open Interest (OI) 数据时返回 422 错误（Unprocessable Entity）。

## ✅ 已找到正确的 API 类型

**正确的 API 类型**: `metaAndAssetCtxs`

### API 请求格式

```json
{"type": "metaAndAssetCtxs"}
```

### API 响应格式

响应是一个数组，包含两个元素：
```json
[
  [universe数组],  // 第一个元素：币种信息
  [contexts数组]   // 第二个元素：市场数据
]
```

- **universe**: 包含币种信息，每个元素包含 `name`, `szDecimals`, `maxLeverage` 等字段
- **contexts**: 包含市场数据，每个元素包含 `openInterest`, `markPx`, `funding`, `dayNtlVlm` 等字段

### 数据映射

- `universe[i]` 对应 `contexts[i]`（索引对应）
- 通过 `universe[i].name` 查找币种，然后从 `contexts[i].openInterest` 获取 OI 数据

## 之前的测试结果

### 已测试的 API 类型（都已失败）

1. **`market24hSnapshot`** (之前代码使用)
   ```json
   {"type": "market24hSnapshot"}
   ```
   - 结果: ❌ 422 错误
   - 错误信息: "Failed to deserialize the JSON body into the target type"

2. **`openInterest`** (不带参数)
   ```json
   {"type": "openInterest"}
   ```
   - 结果: ❌ 422 错误
   - 错误信息: "Failed to deserialize the JSON body into the target type"

3. **`openInterest`** (带 req 参数)
   ```json
   {"type": "openInterest", "req": {"coin": "BTC"}}
   ```
   - 结果: ❌ 422 错误
   - 错误信息: "Failed to deserialize the JSON body into the target type"

4. **`userState`** (需要用户地址)
   ```json
   {"type": "userState", "user": "0x..."}
   ```
   - 结果: ❌ 422 错误（需要有效用户地址）

## 已验证可用的 Hyperliquid API 类型

根据代码实现，以下 API 类型已验证可用：

1. ✅ **`meta`** - 获取元信息
   ```json
   {"type": "meta"}
   ```

2. ✅ **`allMids`** - 获取所有币种中间价
   ```json
   {"type": "allMids"}
   ```

3. ✅ **`candleSnapshot`** - 获取K线数据
   ```json
   {
     "type": "candleSnapshot",
     "req": {
       "coin": "BTC",
       "interval": "1h",
       "startTime": 1234567890000,
       "endTime": 1234567890000
     }
   }
   ```

## ✅ 解决方案

### 已实现正确的 API 调用

使用 `metaAndAssetCtxs` API 类型成功获取 OI 数据：

1. ✅ API 调用成功（无 422 错误）
2. ✅ 能够正确解析响应
3. ✅ 能够提取 openInterest 数据
4. ✅ 支持字符串和数字类型的 openInterest 字段

### 实现逻辑

1. 发送请求: `{"type": "metaAndAssetCtxs"}`
2. 解析响应数组: `[universe, contexts]`
3. 查找币种索引: 在 universe 中查找币种的索引位置
4. 获取 OI 数据: 使用相同索引从 contexts 中获取 openInterest 字段
5. 返回 OIData: 包含 Latest 和 Average 值

### 代码位置

- **OI 获取实现**: `market/hyperliquid_client.go` 第 291-356 行
- **容错处理**: `decision/engine.go` 第 189 行
- **数据获取**: `market/data.go` 第 59-62 行

## 建议

### 方案1: 保持当前容错机制（推荐）

如果 OI 数据不是决策的关键因素，保持当前的容错机制即可：
- OI 获取失败时返回默认值 0
- 决策引擎跳过 OI 过滤
- 不影响主要功能

**优点**:
- 简单可靠
- 不影响现有功能
- 系统已实现

### 方案2: 使用其他数据源获取 OI 数据

如果确实需要 OI 数据，可以考虑：
1. 使用 Binance API 获取 OI 数据（如果可用）
2. 使用第三方数据聚合服务
3. 从其他交易所获取 OI 数据

### 方案3: 联系 Hyperliquid 官方支持

如果需要获取 Hyperliquid 的 OI 数据，可以：
1. 查看 Hyperliquid 官方文档: https://hyperliquid.gitbook.io/hyperliquid-docs
2. 联系 Hyperliquid 技术支持
3. 咨询是否有其他方式获取 OI 数据

## 代码修改建议

如果将来找到正确的 API 类型，可以修改 `market/hyperliquid_client.go` 中的 `GetOpenInterest` 方法：

```go
func (c *HyperliquidClient) GetOpenInterest(symbol string) (*OIData, error) {
    coin := c.symbolToCoin(symbol)
    
    // 使用正确的 API 类型（需要验证）
    reqBody := map[string]interface{}{
        "type": "正确的API类型",
        // 可能需要其他参数
    }
    
    resp, err := c.postRequest("/info", reqBody)
    // ... 处理响应
}
```

## 测试命令

```bash
# 测试 market24hSnapshot
curl -s "https://api.hyperliquid.xyz/info" -X POST \
  -H "Content-Type: application/json" \
  -d '{"type":"market24hSnapshot}'

# 测试 openInterest
curl -s "https://api.hyperliquid.xyz/info" -X POST \
  -H "Content-Type: application/json" \
  -d '{"type":"openInterest"}'

# 测试 openInterest with req
curl -s "https://api.hyperliquid.xyz/info" -X POST \
  -H "Content-Type: application/json" \
  -d '{"type":"openInterest","req":{"coin":"BTC"}}'
```

## 相关文件

- `market/hyperliquid_client.go` - Hyperliquid 客户端实现
- `docs/hyperliquid_data_source_research.md` - Hyperliquid 数据源研究文档
- `decision/engine.go` - 决策引擎（包含 OI 过滤逻辑）
- `market/data.go` - 市场数据获取逻辑

---

**创建时间**: 2025-01-04  
**状态**: Hyperliquid 可能不支持公开的 OI API  
**建议**: 保持当前容错机制

