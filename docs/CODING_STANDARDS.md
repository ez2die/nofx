# 📋 NOFX 代码规范与命名规则

本文档总结了 NOFX 项目的命名规则和开发规范，用于指导 AI 和开发者编写符合项目标准的代码。

---

## 📦 目录

- [Go 代码规范](#go-代码规范)
- [TypeScript/React 代码规范](#typescriptreact-代码规范)
- [文件命名规范](#文件命名规范)
- [提交信息规范](#提交信息规范)
- [分支命名规范](#分支命名规范)
- [代码组织规范](#代码组织规范)
- [错误处理规范](#错误处理规范)
- [注释规范](#注释规范)

---

## 🔷 Go 代码规范

### 包命名 (Package Naming)

- **规则**: 小写字母，单数形式，简短且有意义
- **示例**:
  ```go
  package trader
  package market
  package config
  package decision
  package trade_history
  ```

### 类型命名 (Type Naming)

- **规则**: PascalCase（首字母大写的驼峰命名）
- **示例**:
  ```go
  type TraderConfig struct { ... }
  type AutoTrader struct { ... }
  type PositionInfo struct { ... }
  type AccountInfo struct { ... }
  type Decision struct { ... }
  type FullDecision struct { ... }
  ```

### 接口命名 (Interface Naming)

- **规则**: PascalCase，通常以 `er` 结尾表示行为
- **示例**:
  ```go
  type Trader interface { ... }
  type MarketDataClient interface { ... }
  ```

### 函数命名 (Function Naming)

- **公开函数**: PascalCase
  ```go
  func NewServer(...) *Server
  func GetFullDecision(...) (*FullDecision, error)
  func LoadConfig(filename string) (*Config, error)
  ```

- **私有函数**: camelCase（小写开头）
  ```go
  func buildSystemPrompt(...) string
  func fetchMarketDataForContext(...) error
  func validateDecision(...) error
  ```

### 变量命名 (Variable Naming)

- **规则**: camelCase（小写开头的驼峰命名）
- **示例**:
  ```go
  traderManager := manager.NewTraderManager()
  apiServer := api.NewServer(...)
  database, err := config.NewDatabase(dbPath)
  ctx := &Context{...}
  ```

### 常量命名 (Constant Naming)

- **规则**: PascalCase
- **示例**:
  ```go
  const DataSourceBinance = "binance"
  const DataSourceHyperliquid = "hyperliquid"
  ```

### 结构体字段命名 (Struct Field Naming)

- **规则**: PascalCase，JSON 标签使用 snake_case
- **示例**:
  ```go
  type TradeRecord struct {
      ID              int64     `json:"id" db:"id"`
      TraderID        string    `json:"trader_id" db:"trader_id"`
      Symbol          string    `json:"symbol" db:"symbol"`
      ExecutionPrice  float64   `json:"execution_price" db:"execution_price"`
      Timestamp       time.Time `json:"timestamp" db:"timestamp"`
  }
  ```

### 错误处理

- **规则**: 必须显式处理所有错误，使用 `fmt.Errorf` 包装错误并添加上下文
- **示例**:
  ```go
  if err != nil {
      return nil, fmt.Errorf("读取配置文件失败: %w", err)
  }
  
  if err != nil {
      return fmt.Errorf("获取 %s 市场数据失败: %w", symbol, err)
  }
  ```

---

## 🔷 TypeScript/React 代码规范

### 接口/类型命名 (Interface/Type Naming)

- **规则**: PascalCase
- **示例**:
  ```typescript
  interface TraderConfig {
    id: string;
    name: string;
    aiModel: string;
  }
  
  interface SystemStatus {
    trader_id: string;
    trader_name: string;
    is_running: boolean;
  }
  
  type DecisionAction = {
    action: string;
    symbol: string;
  };
  ```

### 组件命名 (Component Naming)

- **规则**: PascalCase，文件名与组件名一致
- **示例**:
  ```typescript
  // Header.tsx
  export function Header({ simple = false }: HeaderProps) { ... }
  
  // TraderCard.tsx
  export const TraderCard: React.FC<{ trader: TraderConfig }> = ({ trader }) => { ... }
  ```

### 函数命名 (Function Naming)

- **规则**: camelCase
- **示例**:
  ```typescript
  async function getTraders(): Promise<TraderInfo[]> { ... }
  async function createTrader(request: CreateTraderRequest): Promise<TraderInfo> { ... }
  function handleStart() { ... }
  function handleStop() { ... }
  ```

### 变量命名 (Variable Naming)

- **规则**: camelCase
- **示例**:
  ```typescript
  const isRunning = useState(false);
  const traderId = "trader_001";
  const apiBase = '/api';
  ```

### 常量命名 (Constant Naming)

- **规则**: UPPER_SNAKE_CASE 或 camelCase（根据使用场景）
- **示例**:
  ```typescript
  const API_BASE = '/api';
  const MAX_RETRIES = 3;
  ```

### Props 接口命名

- **规则**: 组件名 + `Props`
- **示例**:
  ```typescript
  interface HeaderProps {
    simple?: boolean;
  }
  
  interface TraderCardProps {
    trader: TraderConfig;
    onStart: (id: string) => void;
  }
  ```

### API 函数命名

- **规则**: 动词开头，camelCase，使用 async/await
- **示例**:
  ```typescript
  export const api = {
    async getTraders(): Promise<TraderInfo[]> { ... },
    async createTrader(request: CreateTraderRequest): Promise<TraderInfo> { ... },
    async startTrader(traderId: string): Promise<void> { ... },
    async stopTrader(traderId: string): Promise<void> { ... },
  };
  ```

---

## 📁 文件命名规范

### Go 文件

- **规则**: snake_case（下划线分隔）
- **示例**:
  ```
  auto_trader.go
  client_interface.go
  database_trade_history_migration.go
  sync_hyperliquid.go
  ```

### TypeScript/React 文件

- **组件文件**: PascalCase
  ```
  Header.tsx
  TraderCard.tsx
  CompetitionPage.tsx
  ```

- **工具/库文件**: camelCase
  ```
  api.ts
  utils.ts
  config.ts
  ```

- **类型定义文件**: camelCase 或 PascalCase
  ```
  types.ts
  types/index.ts
  ```

### 文档文件

- **规则**: UPPER_SNAKE_CASE.md
- **示例**:
  ```
  CONTRIBUTING.md
  TRADE_HISTORY_ANALYTICS_REQUIREMENTS.md
  TRADE_PAIRING_STRATEGY.md
  HYPERLIQUID_SIZE_SIGN_RULE.md
  ```

### 配置文件

- **规则**: kebab-case 或 camelCase
- **示例**:
  ```
  config.json
  docker-compose.yml
  package.json
  tsconfig.json
  ```

---

## 📝 提交信息规范

### 格式

遵循 [Conventional Commits](https://www.conventionalcommits.org/) 格式：

```
<type>(<scope>): <subject>

<body>

<footer>
```

### 类型 (Types)

- `feat` - 新功能
- `fix` - Bug 修复
- `docs` - 文档更新
- `style` - 代码格式（不影响代码运行的变动）
- `refactor` - 重构（既不是新增功能，也不是修复 bug）
- `perf` - 性能优化
- `test` - 测试相关
- `chore` - 构建过程或辅助工具的变动
- `ci` - CI/CD 相关
- `security` - 安全相关

### 示例

```
feat(exchange): add OKX exchange integration

- Implement order placement and cancellation
- Add balance and position retrieval
- Support leverage configuration

Closes #123
```

```
fix(trader): prevent duplicate position opening

The trader was opening multiple positions in the same direction
for the same symbol. Added check to prevent this behavior.

Fixes #456
```

```
docs: update Docker deployment guide

- Add troubleshooting section
- Update environment variables
- Add examples for common scenarios
```

### 规则

- 使用现在时态（"add" 而不是 "added"）
- 使用祈使语气（"move" 而不是 "moves"）
- 第一行 ≤ 72 个字符
- 引用相关的 issue 和 PR
- 解释 "what" 和 "why"，而不是 "how"

---

## 🌿 分支命名规范

### 格式

```
<type>/<description>
```

### 类型前缀

- `feature/` - 新功能
- `fix/` - Bug 修复
- `docs/` - 文档更新
- `refactor/` - 代码重构
- `perf/` - 性能优化
- `test/` - 测试相关
- `chore/` - 构建/配置更改

### 示例

```
feature/okx-exchange-integration
fix/position-tracking-bug
docs/update-deployment-guide
refactor/extract-common-interface
```

---

## 📂 代码组织规范

### Go 项目结构

```
nofx/
├── api/              # API 服务器
├── auth/             # 认证模块
├── config/           # 配置管理
├── decision/         # AI 决策引擎
├── logger/           # 日志记录
├── manager/          # 交易员管理
├── market/           # 市场数据
├── mcp/              # MCP 客户端
├── pool/             # 币种池
├── prompts/          # AI 提示词模板
├── trade_history/    # 交易历史
├── trader/           # 交易器实现
├── validation/       # 验证工具
└── web/              # 前端代码
```

### 前端项目结构

```
web/
├── src/
│   ├── components/   # React 组件
│   ├── pages/        # 页面组件
│   ├── hooks/        # 自定义 Hooks
│   ├── contexts/     # React Context
│   ├── lib/          # 工具库
│   ├── types/        # TypeScript 类型定义
│   ├── i18n/         # 国际化
│   └── utils/        # 工具函数
└── public/           # 静态资源
```

---

## ⚠️ 错误处理规范

### Go 错误处理

1. **必须显式处理所有错误**
   ```go
   // ✅ Good
   if err != nil {
       return fmt.Errorf("操作失败: %w", err)
   }
   
   // ❌ Bad
   _ = someFunction() // 忽略错误
   ```

2. **使用 `fmt.Errorf` 包装错误并添加上下文**
   ```go
   if err != nil {
       return fmt.Errorf("读取配置文件失败: %w", err)
   }
   ```

3. **错误信息使用中文（面向用户）**
   ```go
   return fmt.Errorf("获取 %s 市场数据失败: %w", symbol, err)
   ```

### TypeScript 错误处理

1. **使用 try-catch 处理异步错误**
   ```typescript
   try {
     await startTrader(trader.id);
     setIsRunning(true);
   } catch (error) {
     console.error('Failed to start trader:', error);
     // 显示错误提示
   }
   ```

2. **检查 HTTP 响应状态**
   ```typescript
   const res = await fetch(`${API_BASE}/traders`);
   if (!res.ok) throw new Error('获取trader列表失败');
   return res.json();
   ```

---

## 💬 注释规范

### Go 注释

1. **包注释**: 每个包都应该有包级别的注释
   ```go
   // Package trader 提供交易器接口和实现
   package trader
   ```

2. **公开函数/类型注释**: 使用完整的句子，以被注释的内容开头
   ```go
   // Trader 交易器统一接口
   // 支持多个交易平台（币安、Hyperliquid等）
   type Trader interface { ... }
   
   // GetFullDecision 获取AI的完整交易决策（批量分析所有币种和持仓）
   func GetFullDecision(ctx *Context, mcpClient *mcp.Client) (*FullDecision, error) { ... }
   ```

3. **复杂逻辑注释**: 使用中文解释复杂逻辑
   ```go
   // ⚠️ 流动性过滤：持仓价值低于15M USD的币种不做（多空都不做）
   // 持仓价值 = 持仓量 × 当前价格
   // 但现有持仓必须保留（需要决策是否平仓）
   ```

### TypeScript 注释

1. **函数注释**: 使用 JSDoc 格式
   ```typescript
   /**
    * 获取所有交易员列表
    * @returns Promise<TraderInfo[]> 交易员信息数组
    */
   async function getTraders(): Promise<TraderInfo[]> { ... }
   ```

2. **复杂逻辑注释**: 使用中文解释
   ```typescript
   // 根据账户状态动态调整候选币种数量
   const maxCandidates = calculateMaxCandidates(ctx);
   ```

---

## ✅ 代码质量检查清单

### 提交前检查

- [ ] 代码编译成功（`go build` 或 `npm run build`）
- [ ] 所有测试通过（`go test ./...` 或 `npm test`）
- [ ] 无 linting 错误（`go fmt`, `go vet`, `npm run lint`）
- [ ] 遵循命名规范
- [ ] 错误处理完整
- [ ] 注释充分（特别是复杂逻辑）
- [ ] 提交信息符合规范

### Go 代码检查

```bash
# 格式化代码
go fmt ./...

# 静态分析
go vet ./...

# 运行测试
go test ./...
```

### TypeScript 代码检查

```bash
# 格式化代码
npm run format

# Linting
npm run lint

# 类型检查
npm run type-check
```

---

## 📚 参考资源

- [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments)
- [Conventional Commits](https://www.conventionalcommits.org/)
- [TypeScript Style Guide](https://google.github.io/styleguide/tsguide.html)
- [React Best Practices](https://react.dev/learn)

---

## 🔄 更新记录

- **2024-11-XX**: 初始版本，总结项目现有命名规则和开发规范

---

**注意**: 本文档基于项目现有代码总结，如有疑问或需要补充，请提交 Issue 或 PR。

