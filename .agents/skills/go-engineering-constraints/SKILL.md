---
name: go-engineering-constraints
description: Enforce Go engineering best practices for clean architecture, DI, testing, error handling, and code style. Use when asked to "write Go code", "Go service", "Go handler", "refactor Go", "Go project", "Go API", "Go repository", "golang", "Go module", or when generating/modifying any .go file.
---

<role>
你是一位严格遵守 Go 工程规范的资深架构师。生成、修改、重构 Go 代码时，必须遵循以下所有约束。任何违反分层边界、错误处理、测试纪律、依赖注入和 Go 惯例的代码，都应被视为错误。
</role>

<tech_stack>
Go 1.22+, 标准库优先。Web 框架: Gin/Echo/Fiber。持久层: GORM/SQLx/database/sql。消息队列: 按项目选用。
</tech_stack>

## 一、架构分层约束（强制）

### 分层规则

严格遵循单向依赖，禁止反向依赖、禁止跳过层级：

```
cmd/server
  ↓ 启动与组合
internal/controller 或 internal/api/handler
  ↓ HTTP 入参、响应映射
internal/manager 或 internal/service
  ↓ 应用编排与业务逻辑
internal/repository 或 internal/storage
  ↓ 持久化端口
internal/infrastructure/persistence
  ↓ 数据库实现

外部系统 → internal/client 或 internal/infrastructure/acl
internal/domain → 纯领域模型，被 service/repository 使用，不得依赖 Web、DB、Cache、MQ 或第三方 SDK
```

### 跨层禁令

| 层级 | 禁止 | 正确做法 |
|------|------|---------|
| Handler | 直接使用 GORM/SQLx/database/sql | 通过 Service/Manager 接口调用 |
| Handler | import repository/storage 实现包 | 依赖 Service/Manager 抽象 |
| Handler | 承载业务编排、事务、轮询逻辑 | 只做 bind、validate、call、response |
| Service/Biz | 依赖 `gin.Context`/`echo.Context`/`fiber.Ctx` | 使用 `context.Context` + 显式 request struct |
| Service/Biz | 持有或返回 `*gorm.DB`/`*sql.DB`/`sql.Tx` | 依赖 repository/storage port |
| Repository | 向上暴露数据库 handle | 返回领域/内部类型和 `error` |
| Domain | 导入 Gin/GORM/Redis/HTTP client/SDK | 保持纯 Go 类型和领域行为 |
| Infrastructure | 反向调用 Handler | 只实现端口，由 DI 层组装 |

### 推荐包结构

```
myapp/
├── api/                         # OpenAPI/Swagger
├── cmd/server/main.go           # 入口
├── internal/
│   ├── api/ 或 controller/handler/  # HTTP Handler
│   ├── manager/                  # 跨 Service 编排（可选）
│   ├── service/                  # 业务逻辑
│   ├── repository/ 或 storage/    # 持久化端口
│   ├── domain/                   # 领域实体、值对象、领域错误
│   ├── dto/                      # 请求/响应 DTO
│   ├── infrastructure/
│   │   ├── persistence/          # DB 实现
│   │   ├── client/               # HTTP/RPC 客户端
│   │   ├── acl/                  # 防腐层
│   │   ├── cache/                # 缓存实现
│   │   └── messaging/            # 消息队列实现
│   └── di/                       # 依赖组装
├── pkg/                          # 允许外部导入的公共库
├── configs/
├── scripts/
├── test/                         # 集成测试
└── go.mod
```

## 二、接口与依赖注入约束（强制）

```text
1. 构造函数注入所有依赖，禁止业务全局变量和隐式单例。
2. Service/Manager/Handler 依赖接口，不依赖具体实现。
3. 接口由消费方或稳定端口包定义，方法 ≤ 5 个。
4. 重要实现必须添加编译期检查：var _ Interface = (*Impl)(nil)。
5. DI/main/composition root 可以依赖具体实现，业务层不可以。
6. context.Context 必须作为阻塞/I/O 方法的第一个参数，不得存入 struct 字段。
```

### 正确示例

```go
package service

import (
    "context"
    "errors"
    "fmt"
    "myapp/internal/domain"
)

var ErrOrderNotFound = errors.New("order not found")

type OrderRepository interface {
    FindByID(ctx context.Context, id domain.OrderID) (*domain.Order, error)
    Save(ctx context.Context, order *domain.Order) error
}

type OrderService interface {
    GetOrder(ctx context.Context, id domain.OrderID) (*domain.Order, error)
}

type orderService struct {
    repo OrderRepository
}

func NewOrderService(repo OrderRepository) OrderService {
    if repo == nil {
        panic("order repository is nil")
    }
    return &orderService{repo: repo}
}

func (s *orderService) GetOrder(ctx context.Context, id domain.OrderID) (*domain.Order, error) {
    order, err := s.repo.FindByID(ctx, id)
    if err != nil {
        if errors.Is(err, domain.ErrNotFound) {
            return nil, fmt.Errorf("%w: id=%s", ErrOrderNotFound, id)
        }
        return nil, fmt.Errorf("find order: %w", err)
    }
    return order, nil
}

var _ OrderService = (*orderService)(nil)
```

### 错误示例（绝对禁止）

```go
package handler

import (
    "github.com/gin-gonic/gin"
    "gorm.io/gorm"
)

type OrderHandler struct {
    db *gorm.DB // 错误：Handler 直接依赖数据库
}

func (h *OrderHandler) Create(c *gin.Context) {
    var order Order
    _ = h.db.Create(&order).Error // 错误：Handler 承载持久化逻辑
}
```

## 三、TDD 与测试约束（强制）

```text
1. 新增 public 函数或重要分支必须补充 *_test.go。
2. 默认使用 table-driven 测试覆盖正常路径、零值、nil、空集合、边界值和错误路径。
3. 外部依赖必须使用 fake/mock/stub，不访问真实外部服务。
4. 单元测试禁止 time.Sleep，使用 fake clock、channel、context 或条件等待。
5. Repository 集成测试使用内存数据库或 testcontainers，用 build tag 隔离：
   //go:build integration
6. 并发敏感代码必须运行 go test -race ./...。
7. 测试必须可重复，不依赖执行顺序或共享脏数据。

HTTP client 测试使用 httptest.Server。
数据库测试每个用例后清理数据。
```

### 测试模板

```go
func TestOrderService_GetOrder(t *testing.T) {
    tests := []struct {
        name    string
        id      domain.OrderID
        setup   func(*fakeOrderRepository)
        wantErr error
    }{
        {
            name: "returns order when found",
            id:   domain.OrderID("O-001"),
            setup: func(repo *fakeOrderRepository) {
                repo.orders["O-001"] = &domain.Order{ID: domain.OrderID("O-001")}
            },
        },
        {
            name:    "returns not found when repository misses",
            id:      domain.OrderID("missing"),
            setup:   func(repo *fakeOrderRepository) {},
            wantErr: ErrOrderNotFound,
        },
    }
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            repo := newFakeOrderRepository()
            tt.setup(repo)
            svc := NewOrderService(repo)
            got, err := svc.GetOrder(context.Background(), tt.id)
            if tt.wantErr != nil {
                if !errors.Is(err, tt.wantErr) {
                    t.Fatalf("want error %v, got %v", tt.wantErr, err)
                }
                return
            }
            if err != nil {
                t.Fatalf("unexpected error: %v", err)
            }
            if got == nil {
                t.Fatal("expected order, got nil")
            }
        })
    }
}
```

## 四、设计模式约束（强制）

### 分支复杂度

- if-else 嵌套超过 3 层 → 提前返回、策略模式或责任链模式。
- switch 超过 2 个 case → 策略表、map 分发或接口多态。
- 同一条件判断重复出现 → 提取策略或领域方法。

### 创建型

- 复杂对象使用 `NewXxx(...)` 构造函数封装不变量。
- 参数超过 4 个 → request struct、options 或 Builder。
- Builder 返回 `*Builder`（链式），`Build()` 返回对象 + error。
- 单例仅限无状态配置或只读资源，使用 `sync.Once`。
- 禁止业务状态单例；禁止在业务逻辑中 `&Struct{}` 后手动填充多字段。

### 结构型与行为型

- 横切关注点（日志/缓存/链路追踪/指标）→ 装饰器或 middleware。
- 装饰器优先用 `WithLogging(service, logger)` 这种函数包装。
- 外部系统调用必须经过 Client/ACL，不得把第三方模型传入 service/domain。
- 状态流转 → 领域方法或状态模式，禁止散落在 Service 分支中。
- 算法族替换 → 函数类型策略或小接口策略。

## 五、错误处理约束（强制）

```text
1. 禁止忽略 error。有意忽略必须在调用点说明原因。
2. 保留错误链使用 %w 包装。
3. 调用方判断错误类型必须用 errors.Is 或 errors.As，禁止字符串比较 error。
4. 业务逻辑禁止 panic。panic 仅允许在启动、构造期不变量失败或不可恢复的程序员错误中使用。
```

### 正确示例

```go
var ErrInvalidRequest = errors.New("invalid request")

func (s *orderService) CreateOrder(ctx context.Context, req CreateOrderRequest) (*OrderResponse, error) {
    if req.CustomerID == "" {
        return nil, fmt.Errorf("%w: customer_id is required", ErrInvalidRequest)
    }
    if req.Quantity <= 0 {
        return nil, fmt.Errorf("%w: quantity must be positive", ErrInvalidRequest)
    }
    order, err := domain.NewOrder(req.CustomerID, req.Quantity)
    if err != nil {
        return nil, fmt.Errorf("build order: %w", err)
    }
    if err := s.repo.Save(ctx, order); err != nil {
        return nil, fmt.Errorf("save order: %w", err)
    }
    return orderResponseFromDomain(order), nil
}
```

## 六、防御性编程约束（强制）

### Nil 与零值

- 构造函数必须校验必需依赖是否为 nil。
- public 函数入口必须校验关键参数。
- 返回可能为 nil 的指针时，函数注释或命名必须表达清楚。
- 优先让类型零值可用；零值不可用则必须通过构造函数创建。
- 切片和 map 作为内部状态暴露时，必须返回副本。

### 不可变性

```go
type Money struct {
    value    int64
    currency Currency
}

func NewMoney(value int64, currency Currency) Money {
    return Money{value: value, currency: currency}
}

func (m Money) Add(other Money) (Money, error) {
    if m.currency != other.currency {
        return Money{}, ErrCurrencyMismatch
    }
    return NewMoney(m.value+other.value, m.currency), nil
}
```

值对象应使用私有字段、无 setter、修改方法返回新值。

## 七、代码风格约束（强制）

### 硬性指标

| 指标 | 限制 |
|------|------|
| 函数长度 | ≤ 30 行 |
| 文件长度 | ≤ 500 行 |
| 参数数量 | ≤ 4 个 |
| 圈复杂度 | ≤ 10 |
| 嵌套深度 | ≤ 3 层 |

### 命名规范

- package 名：短小小写，不使用下划线和复数。
- 导出标识符：清晰语义，避免滥用 `Manager`/`Util`。
- 接口按能力命名：Reader、Writer、Store、Repository、Publisher。禁止 `IUserService` 前缀。
- 构造函数：`NewXxx`。错误变量：`ErrXxx`。context 参数：`ctx`，第一个参数。
- receiver 名短小一致，例如 `s *orderService`、`r *orderRepository`。

### 注释规范

- 导出类型/函数/常量/变量必须有注释，注释以标识符开头。
- 注释解释"为什么"，不复述"做什么"。
- 复杂并发、事务、错误兼容逻辑必须说明约束。
- 禁止无意义注释和魔法数字（必须提取为 const 并说明业务含义）。

### 工具链

```bash
go fmt ./...
go vet ./...
go test ./... -count=1
go test -race ./...
golangci-lint run ./...
```

## 八、日志与监控约束（强制）

```text
1. 服务端禁止 fmt.Println 输出业务日志。
2. 使用 log/slog、zap 或 logrus，项目内保持统一。
3. 日志必须带关键上下文：request_id、user_id、order_id、cluster_id 等。
4. error 日志必须包含 error 对象和定位字段。
5. 密码、token、手机号、身份证、密钥等敏感信息必须脱敏。
6. 禁止在高频循环中逐条打印 info/error 日志。
7. 有 context 的路径优先使用带 Context 后缀的日志方法。
```

```go
logger.InfoContext(ctx, "order created",
    "order_id", order.ID,
    "customer_id", order.CustomerID,
    "amount", order.Amount,
)
logger.ErrorContext(ctx, "payment failed",
    "order_id", order.ID,
    "error", err,
)
```

## 九、数据库与持久层约束（强制）

```text
1. GORM/SQLx/database/sql 只能出现在 persistence/storage/repository 实现层。
2. Repository/Storage 接口不得返回 *gorm.DB、*sql.DB、sql.Tx 或 driver 专属类型。
3. Table/Entity 与 domain/dto 必须分离，通过转换函数映射。
4. 所有数据库调用必须使用 WithContext(ctx) 或等价 context 传递。
5. 事务边界必须显式，事务 handle 不得泄漏到 service/biz API。
6. N+1 查询必须识别并处理。
7. 批量操作使用 batch，不得在大集合上循环单条写入。
8. 迁移、索引、唯一约束和乐观锁策略必须与业务一致。
```

### 正确示例

```go
type OrderStore interface {
    FindByID(ctx context.Context, id domain.OrderID) (*domain.Order, error)
    Save(ctx context.Context, order *domain.Order) error
}

type orderStoreGorm struct {
    db *gorm.DB
}

func NewOrderStoreGorm(db *gorm.DB) OrderStore {
    if db == nil {
        panic("gorm db is nil")
    }
    return &orderStoreGorm{db: db}
}

func (s *orderStoreGorm) FindByID(ctx context.Context, id domain.OrderID) (*domain.Order, error) {
    var row OrderTable
    if err := s.db.WithContext(ctx).First(&row, "id = ?", id).Error; err != nil {
        return nil, mapGormError(err)
    }
    return row.ToDomain(), nil
}
```

## 十、外部系统与防腐层约束（强制）

```text
1. 调用第三方 API 必须经过 Client 或 ACL adapter。
2. 第三方 SDK struct 不得出现在 domain/service contract 中。
3. 外部请求必须设置 timeout，通过 context.Context 支持取消。
4. 外部依赖影响主链路可用性时，必须考虑 retry、熔断、限流或降级。
5. 回调接口必须先验签，再转换为内部模型。
6. 第三方错误必须转换为领域错误或基础设施错误后向上传递。
```

### 正确示例

```go
type PaymentGateway interface {
    ProcessPayment(ctx context.Context, req PaymentRequest) (*PaymentResult, error)
}

type paymentAdapter struct {
    client ThirdPartyPaymentClient
}

func (a *paymentAdapter) ProcessPayment(ctx context.Context, req PaymentRequest) (*PaymentResult, error) {
    thirdReq := toThirdPartyPaymentRequest(req)
    resp, err := a.client.Pay(ctx, thirdReq)
    if err != nil {
        return nil, fmt.Errorf("payment provider failed: %w", err)
    }
    return paymentResultFromProvider(resp), nil
}
```

## 架构检查清单

生成 Go 代码后逐项自检：

```text
□ Handler 是否直接访问数据库？
□ Handler 是否 import repository/storage 实现？
□ Service/Biz 是否依赖 gin.Context/echo.Context/fiber.Ctx？
□ Service/Biz 是否暴露 *gorm.DB/*sql.DB/sql.Tx？
□ Domain 是否导入 Web、数据库、缓存、消息队列或第三方 SDK？
□ 是否使用 context.Context 作为 I/O 方法第一个参数？
□ 接口是否超过 5 个方法？
□ 是否使用构造函数注入？
□ 是否添加编译期接口检查？
□ 是否忽略 error？
□ 是否使用 %w 包装需要保留链路的 error？
□ 是否使用 errors.Is/errors.As 判断错误类型？
□ 是否写了 table-driven 测试？
□ 单元测试是否使用了 time.Sleep？
□ 是否运行 go fmt、go vet、go test？
□ 并发敏感代码是否运行 go test -race？
□ 函数是否过长、参数是否过多、嵌套是否过深？
□ 第三方 API 是否经过 Client/ACL？
□ 日志是否包含上下文并脱敏？
□ 数据库事务是否显式且未泄漏到底层实现外？
```

## 常用 AI 指令模板

以下模板可用于快速发起 Go 代码生成/重构任务：

```text
生成 Go Service：
生成 OrderService 接口及实现，包含创建订单、查询订单、取消订单。
要求：构造函数注入、context.Context 作为第一个参数、依赖 repository 接口、显式 error 处理、补充表驱动单元测试。

生成 Handler：
生成 Gin Handler，只负责参数绑定、传输层校验、调用 Service、映射 HTTP 响应。
禁止 Handler 直接访问 GORM/SQLx/database/sql 或 repository 实现。

重构跨层依赖：
将 Handler 直接访问数据库的代码重构为 Handler → Service → Repository/Storage。
保持行为不变，先补 characterization test，再移动数据库逻辑。

添加防腐层：
为调用第三方支付 API 添加 Client/ACL。
要求：内部 request/result 类型、第三方模型转换、context 超时、错误转换、回调验签、单元测试。

重构复杂分支：
将超过 3 层 if-else 或超过 2 个 case 的 switch 重构为策略模式、map 分发或责任链。
保持原有行为，添加表驱动测试覆盖所有分支。

生成 Repository：
生成 OrderStore 接口和 GORM 实现。
要求：接口不暴露 *gorm.DB，所有数据库调用使用 WithContext(ctx)，Table 与 Domain 分离，错误转换支持 errors.Is。
```
