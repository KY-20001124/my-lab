# AI 代码约束体系 — 新人培训教案（Go 版本）

> **目标**：让新人掌握与 AI Coding Agent 协作时的架构纪律，将软件工程原则转化为可执行的 AI 约束指令。
> **课时**：4 周（每周 5 天，每天 2 小时理论 + 2 小时实战）
> **技术栈**：Go 1.22+，标准库为主，配合常见框架（Gin/Echo/Fiber + GORM/SQLx）
> **产出**：能够独立编写 System Prompt 约束 AI，产出符合架构规范的 Go 代码。

---

## 第一周：分层架构与依赖管理

### Day 1-2: 分层架构核心原则

#### 学习目标
- 理解分层架构的物理边界意义
- 掌握 Go 项目标准布局（Standard Go Project Layout）
- 能够识别和修复跨层调用

#### 核心知识点

**1.1 Go 项目标准布局**

```
myapp/
├── api/                    # API 定义（OpenAPI/Swagger）
├── cmd/                    # 应用入口
│   └── server/
│       └── main.go          # 唯一 main 包
├── internal/               # 私有代码（Go 编译器保护）
│   ├── controller/         # HTTP 处理层（Gin/Echo Handler）
│   ├── service/            # 业务逻辑接口与实现
│   ├── repository/         # 数据访问接口与实现
│   ├── domain/             # 领域对象（零外部依赖）
│   ├── dto/                # 数据传输对象
│   │   ├── request/        # 请求 DTO
│   │   └── response/       # 响应 DTO
│   ├── infrastructure/     # 基础设施
│   │   ├── client/         # HTTP/RPC 客户端
│   │   ├── acl/            # 防腐层（Anti-Corruption Layer）
│   │   ├── cache/          # 缓存实现
│   │   └── messaging/      # 消息队列实现
│   └── pkg/                # 内部通用库
├── pkg/                    # 公共库（可被外部导入）
├── configs/                # 配置文件
├── scripts/                # 脚本
├── test/                   # 额外测试数据/集成测试
└── go.mod                  # 模块定义
```

**1.2 分层架构与依赖方向**

```
┌─────────────────────────────────────────┐
│  Controller Layer (internal/controller)  │  ← HTTP Handler
│  职责：参数绑定、DTO 转换、调用 Service    │
├─────────────────────────────────────────┤
│  Service Layer (internal/service)        │  ← 业务逻辑
│  职责：流程编排、领域规则、事务控制        │
├─────────────────────────────────────────┤
│  Repository Layer (internal/repository)  │  ← 数据访问
│  职责：数据库操作、ORM、查询优化           │
├─────────────────────────────────────────┤
│  Domain Layer (internal/domain)          │  ← 核心领域
│  职责：实体、值对象、领域事件（零依赖）    │
└─────────────────────────────────────────┘
         ↑ 严格单向依赖，禁止反向
```

**1.3 Go 特有的依赖控制**

```go
// ❌ 错误：Handler 直接调用 GORM
package controller

import "gorm.io/gorm"

type OrderHandler struct {
    db *gorm.DB  // 跨层依赖！直接依赖基础设施
}

func (h *OrderHandler) CreateOrder(c *gin.Context) {
    var order domain.Order
    h.db.Create(&order)  // 业务逻辑泄露到 Handler！
}

// ✅ 正确：通过接口依赖，依赖倒置
package controller

// Handler 只依赖 Service 接口
type OrderHandler struct {
    orderService service.OrderService  // 接口类型
}

func NewOrderHandler(s service.OrderService) *OrderHandler {
    return &OrderHandler{orderService: s}
}

func (h *OrderHandler) CreateOrder(c *gin.Context) {
    var req dto.CreateOrderRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(400, dto.ErrorResponse{Message: err.Error()})
        return
    }

    result, err := h.orderService.CreateOrder(c.Request.Context(), req)
    if err != nil {
        // 错误处理...
        return
    }

    c.JSON(200, result)
}
```

**1.4 internal 包的保护机制**

```go
// internal/domain/order.go
// 这个包下的代码只能被 myapp 模块内引用，外部模块无法导入
package domain

import "time"

// Order 领域实体 - 零框架依赖
type Order struct {
    ID        OrderID
    CustomerID CustomerID
    Items     []OrderItem
    Status    OrderStatus
    Amount    Money
    CreatedAt time.Time
}

func (o *Order) AddItem(item OrderItem) error {
    if item.Quantity <= 0 {
        return ErrInvalidQuantity
    }
    o.Items = append(o.Items, item)
    o.recalculateAmount()
    return nil
}

func (o *Order) recalculateAmount() {
    var total int64
    for _, item := range o.Items {
        total += item.Price * int64(item.Quantity)
    }
    o.Amount = NewMoney(total, CNY)
}
```

#### 实战任务
1. 搭建标准 Go 项目布局，配置 `go.mod` 和目录结构
2. 实现一个 Handler → Service → Repository 的完整链路
3. 验证 `internal` 包的访问限制（尝试从外部导入）

#### AI 约束指令模板

```
【Go 分层约束】
- 严格遵循 internal/controller → service → repository → domain 单向调用
- 使用 internal/ 包机制保护领域层和基础设施实现
- Handler 禁止直接操作数据库（GORM/SQLx）
- Service 禁止直接操作 gin.Context / echo.Context
- 每层只依赖下一层接口，不依赖实现
- domain 包零外部依赖（无 Gin/GORM/Redis 等导入）
```

---

### Day 3-4: 接口设计与依赖注入

#### 学习目标
- 掌握 Go 接口的隐式实现
- 理解构造函数注入模式
- 设计高内聚的接口

#### 核心知识点

**2.1 Go 接口哲学：隐式实现 + 接口由消费者定义**

```go
// ✅ Go 最佳实践：接口定义在使用方（消费者定义接口）
// service/order_service.go
package service

// OrderService 业务逻辑接口 - 由 Service 层定义
type OrderService interface {
    CreateOrder(ctx context.Context, req dto.CreateOrderRequest) (*dto.OrderResponse, error)
    GetOrder(ctx context.Context, id string) (*dto.OrderResponse, error)
    CancelOrder(ctx context.Context, id string) error
}

// 实现放在同包的 impl 子目录或单独文件
// service/order_service_impl.go
type orderServiceImpl struct {
    orderRepo   repository.OrderRepository
    stockClient client.StockClient
    publisher   messaging.EventPublisher
}

// 构造函数注入 - Go 的标准做法
func NewOrderService(
    repo repository.OrderRepository,
    stock client.StockClient,
    pub messaging.EventPublisher,
) OrderService {
    return &orderServiceImpl{
        orderRepo:   repo,
        stockClient: stock,
        publisher:   pub,
    }
}
```

**2.2 接口粒度控制（ISP）**

```go
// ❌ 胖接口 - 强迫实现不需要的方法
type Worker interface {
    Work()
    Eat()    // 机器人不需要
    Sleep()  // AI 不需要
}

// ✅ 接口隔离 - Go 风格的小接口
type Workable interface {
    Work() error
}

type Feedable interface {
    Eat(food string) error
}

type Sleeper interface {
    Sleep(duration time.Duration) error
}

// 组合接口（Go 支持接口嵌入）
type Human interface {
    Workable
    Feedable
    Sleeper
}
```

**2.3 依赖倒置实践**

```go
// repository/order_repository.go
// 接口定义在 repository 包，由消费者（Service）使用
package repository

import (
    "context"
    "myapp/internal/domain"
)

type OrderRepository interface {
    FindByID(ctx context.Context, id domain.OrderID) (*domain.Order, error)
    Save(ctx context.Context, order *domain.Order) error
    FindByStatus(ctx context.Context, status domain.OrderStatus) ([]*domain.Order, error)
}

// infrastructure/persistence/order_repo_gorm.go
// 实现放在 infrastructure 层
package persistence

import (
    "context"
    "gorm.io/gorm"
    "myapp/internal/domain"
    "myapp/internal/repository"
)

// 编译期检查：确保实现了接口
var _ repository.OrderRepository = (*orderRepositoryGorm)(nil)

type orderRepositoryGorm struct {
    db *gorm.DB
}

func NewOrderRepositoryGorm(db *gorm.DB) repository.OrderRepository {
    return &orderRepositoryGorm{db: db}
}

func (r *orderRepositoryGorm) FindByID(ctx context.Context, id domain.OrderID) (*domain.Order, error) {
    var entity OrderEntity
    if err := r.db.WithContext(ctx).First(&entity, "id = ?", id.String()).Error; err != nil {
        return nil, err
    }
    return entity.ToDomain(), nil
}

func (r *orderRepositoryGorm) Save(ctx context.Context, order *domain.Order) error {
    entity := FromDomain(order)
    return r.db.WithContext(ctx).Save(entity).Error
}

func (r *orderRepositoryGorm) FindByStatus(ctx context.Context, status domain.OrderStatus) ([]*domain.Order, error) {
    var entities []OrderEntity
    if err := r.db.WithContext(ctx).Where("status = ?", status.String()).Find(&entities).Error; err != nil {
        return nil, err
    }

    orders := make([]*domain.Order, len(entities))
    for i, e := range entities {
        orders[i] = e.ToDomain()
    }
    return orders, nil
}
```

**2.4 Wire 依赖注入工具（推荐）**

```go
// wire.go
//go:build wireinject
// +build wireinject

package main

import (
    "github.com/google/wire"
    "myapp/internal/config"
    "myapp/internal/controller"
    "myapp/internal/infrastructure/cache"
    "myapp/internal/infrastructure/client"
    "myapp/internal/infrastructure/messaging"
    "myapp/internal/infrastructure/persistence"
    "myapp/internal/service"
)

func InitializeApp(cfg *config.Config) (*App, error) {
    wire.Build(
        // 基础设施层
        persistence.NewOrderRepositoryGorm,
        persistence.NewDB,
        cache.NewRedisCache,
        client.NewStockClient,
        messaging.NewKafkaPublisher,

        // 业务层
        service.NewOrderService,

        // 接口层
        controller.NewOrderHandler,

        // 应用组装
        NewApp,
    )
    return nil, nil
}

// 生成的 wire_gen.go 会自动创建所有依赖
```

#### 实战任务
1. 实现一个完整的接口定义 + 实现 + 编译期检查
2. 使用 Wire 配置依赖注入
3. 编写测试，验证接口隔离效果

#### AI 约束指令模板

```
【Go 接口约束】
- 接口由消费者定义（Service 定义 Repository 接口）
- 实现方用 var _ Interface = (*Impl)(nil) 做编译期检查
- 接口方法不超过 5 个，超过则拆分
- 构造函数注入所有依赖，禁止全局变量
- 使用 Wire 管理依赖关系，禁止手动组装复杂依赖图
- 禁止在 domain 包导入任何第三方库
```

---

### Day 5: 架构守护与静态检查

#### 学习目标
- 使用 `go vet`、`staticcheck` 进行静态分析
- 使用自定义 linter 检查架构规则
- 配置 CI 拦截架构违规

#### 核心知识点

**3.1 Go 内置工具链**

```bash
# 格式化（强制统一风格）
go fmt ./...

# 静态分析
go vet ./...

# 代码复杂度检查
gocyclo -over 10 .

# 静态检查（推荐安装）
staticcheck ./...

# 单元测试覆盖率
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

**3.2 自定义架构检查（使用 AST 分析）**

```go
// tools/archcheck/main.go
// 自定义架构检查工具
package main

import (
    "fmt"
    "go/ast"
    "go/parser"
    "go/token"
    "os"
    "path/filepath"
    "strings"
)

func main() {
    violations := 0

    // 检查规则1：domain 包不能导入 gorm
    filepath.Walk("internal/domain", func(path string, info os.FileInfo, err error) error {
        if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
            return nil
        }

        fset := token.NewFileSet()
        node, err := parser.ParseFile(fset, path, nil, parser.ImportsOnly)
        if err != nil {
            return err
        }

        for _, imp := range node.Imports {
            path := strings.Trim(imp.Path.Value, `"`)
            if strings.Contains(path, "gorm") {
                fmt.Printf("VIOLATION: %s imports %s\n", path, path)
                violations++
            }
        }
        return nil
    })

    // 检查规则2：controller 不能导入 repository
    // ... 类似逻辑

    if violations > 0 {
        os.Exit(1)
    }
}
```

**3.3 CI 配置**

```yaml
# .github/workflows/go-ci.yml
name: Go CI
on: [push, pull_request]
jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Set up Go
        uses: actions/setup-go@v5
        with:
          go-version: '1.22'

      - name: Format Check
        run: |
          if [ -n "$(go fmt ./...)" ]; then
            echo "Code is not formatted. Run 'go fmt ./...'"
            exit 1
          fi

      - name: Vet
        run: go vet ./...

      - name: Staticcheck
        uses: dominikh/staticcheck-action@v1
        with:
          version: "2023.1.6"

      - name: Architecture Check
        run: go run tools/archcheck/main.go

      - name: Test
        run: go test -race -coverprofile=coverage.out ./...

      - name: Coverage Check
        run: |
          coverage=$(go tool cover -func=coverage.out | grep total | awk '{print $3}')
          if (( $(echo "$coverage < 80.0" | bc -l) )); then
            echo "Coverage $coverage is below 80%"
            exit 1
          fi
```

#### 实战任务
1. 编写自定义架构检查工具（检查 domain 包纯净性）
2. 配置完整的 CI 流水线
3. 提交违规代码验证 CI 拦截效果

---

## 第二周：测试驱动开发（TDD）

### Day 1-2: TDD 基础与 Go 测试文化

#### 学习目标
- 掌握 Go 的测试框架和标准库
- 理解 TDD 三定律
- 能够写出可测试的代码

#### 核心知识点

**4.1 Go 测试标准库**

```go
// service/order_service_test.go
package service

import (
    "context"
    "testing"
    "time"

    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/mock"
    "myapp/internal/domain"
    "myapp/internal/dto"
)

// 4.2 Mock 实现（Go 标准做法：手写 Mock）
// 或使用 mockery 自动生成

type mockOrderRepository struct {
    mock.Mock
}

func (m *mockOrderRepository) FindByID(ctx context.Context, id domain.OrderID) (*domain.Order, error) {
    args := m.Called(ctx, id)
    if args.Get(0) == nil {
        return nil, args.Error(1)
    }
    return args.Get(0).(*domain.Order), args.Error(1)
}

func (m *mockOrderRepository) Save(ctx context.Context, order *domain.Order) error {
    args := m.Called(ctx, order)
    return args.Error(0)
}

func (m *mockOrderRepository) FindByStatus(ctx context.Context, status domain.OrderStatus) ([]*domain.Order, error) {
    args := m.Called(ctx, status)
    if args.Get(0) == nil {
        return nil, args.Error(1)
    }
    return args.Get(0).([]*domain.Order), args.Error(1)
}

// 4.3 TDD 测试示例
func TestOrderService_CreateOrder(t *testing.T) {
    // Given
    mockRepo := new(mockOrderRepository)
    mockStock := new(mockStockClient)
    mockPub := new(mockEventPublisher)

    svc := NewOrderService(mockRepo, mockStock, mockPub)

    req := dto.CreateOrderRequest{
        ProductID: "P001",
        Quantity:  2,
        CustomerID: "C001",
    }

    mockStock.On("CheckAvailability", mock.Anything, "P001", 2).
        Return(true, nil)
    mockRepo.On("Save", mock.Anything, mock.AnythingOfType("*domain.Order")).
        Return(nil)
    mockPub.On("Publish", mock.Anything, mock.AnythingOfType("domain.OrderCreatedEvent")).
        Return(nil)

    // When
    result, err := svc.CreateOrder(context.Background(), req)

    // Then
    assert.NoError(t, err)
    assert.NotNil(t, result)
    assert.NotEmpty(t, result.ID)
    assert.Equal(t, domain.OrderStatusCreated, result.Status)

    mockStock.AssertExpectations(t)
    mockRepo.AssertExpectations(t)
    mockPub.AssertExpectations(t)
}

func TestOrderService_CreateOrder_InsufficientStock(t *testing.T) {
    // Given
    mockRepo := new(mockOrderRepository)
    mockStock := new(mockStockClient)
    mockPub := new(mockEventPublisher)

    svc := NewOrderService(mockRepo, mockStock, mockPub)

    req := dto.CreateOrderRequest{
        ProductID: "P001",
        Quantity:  100,
    }

    mockStock.On("CheckAvailability", mock.Anything, "P001", 100).
        Return(false, nil)

    // When
    result, err := svc.CreateOrder(context.Background(), req)

    // Then
    assert.Error(t, err)
    assert.Nil(t, result)
    assert.IsType(t, domain.ErrInsufficientStock, err)

    mockRepo.AssertNotCalled(t, "Save", mock.Anything, mock.Anything)
    mockPub.AssertNotCalled(t, "Publish", mock.Anything, mock.Anything)
}
```

**4.4 表驱动测试（Go 惯用法）**

```go
func TestOrder_CalculateTotal(t *testing.T) {
    tests := []struct {
        name     string
        items    []domain.OrderItem
        expected domain.Money
        wantErr  bool
    }{
        {
            name: "正常计算",
            items: []domain.OrderItem{
                {ProductID: "P001", Price: 1000, Quantity: 2}, // 2000
                {ProductID: "P002", Price: 500, Quantity: 1},  // 500
            },
            expected: domain.NewMoney(2500, domain.CNY),
            wantErr:  false,
        },
        {
            name:     "空订单",
            items:    []domain.OrderItem{},
            expected: domain.NewMoney(0, domain.CNY),
            wantErr:  false,
        },
        {
            name: "负数数量",
            items: []domain.OrderItem{
                {ProductID: "P001", Price: 1000, Quantity: -1},
            },
            expected: domain.Money{},
            wantErr:  true,
        },
        {
            name: "零价格",
            items: []domain.OrderItem{
                {ProductID: "P001", Price: 0, Quantity: 5},
            },
            expected: domain.NewMoney(0, domain.CNY),
            wantErr:  false,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            order := domain.NewOrder("C001")
            for _, item := range tt.items {
                err := order.AddItem(item)
                if tt.wantErr {
                    assert.Error(t, err)
                    return
                }
            }

            if !tt.wantErr {
                assert.Equal(t, tt.expected, order.Amount())
            }
        })
    }
}
```

#### 实战任务
1. 从需求出发，先写测试再写实现（计算器/订单场景）
2. 使用 testify/mock 编写 Mock 测试
3. 使用表驱动测试覆盖边界条件

#### AI 约束指令模板

```
【Go TDD 约束】
- 每个 public 方法/函数必须有对应的 _test.go
- 测试文件名：xxx_test.go，与被测文件同包
- 使用表驱动测试（[]struct + for range）
- 外部依赖必须 Mock（手写或使用 mockery 生成）
- 测试覆盖：正常路径、null/零值、空集合、最大值、负数、错误路径
- 使用 testify/assert 断言，禁止 t.Fatal 滥用
- 单元测试覆盖率不低于 80%
```

---

### Day 3-4: 集成测试与测试容器

#### 学习目标
- 编写数据库集成测试
- 使用 testcontainers-go
- 掌握测试生命周期管理

#### 核心知识点

**5.1 数据库集成测试**

```go
// infrastructure/persistence/order_repo_test.go
package persistence

import (
    "context"
    "testing"

    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/suite"
    "gorm.io/driver/sqlite"
    "gorm.io/gorm"
    "myapp/internal/domain"
)

type OrderRepositoryTestSuite struct {
    suite.Suite
    db   *gorm.DB
    repo *orderRepositoryGorm
}

func (s *OrderRepositoryTestSuite) SetupSuite() {
    // 使用内存 SQLite 进行集成测试
    db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
    assert.NoError(s.T(), err)

    // 自动迁移
    db.AutoMigrate(&OrderEntity{})

    s.db = db
    s.repo = NewOrderRepositoryGorm(db).(*orderRepositoryGorm)
}

func (s *OrderRepositoryTestSuite) TearDownTest() {
    // 每个测试后清理数据
    s.db.Exec("DELETE FROM orders")
}

func (s *OrderRepositoryTestSuite) TestSaveAndFind() {
    // Given
    order := domain.NewOrder("C001")
    order.AddItem(domain.OrderItem{
        ProductID: "P001",
        Price:     1000,
        Quantity:  2,
    })

    // When
    err := s.repo.Save(context.Background(), order)
    assert.NoError(s.T(), err)

    // Then
    found, err := s.repo.FindByID(context.Background(), order.ID)
    assert.NoError(s.T(), err)
    assert.NotNil(s.T(), found)
    assert.Equal(s.T(), order.ID, found.ID)
    assert.Equal(s.T(), 1, len(found.Items))
}

func TestOrderRepositorySuite(t *testing.T) {
    suite.Run(t, new(OrderRepositoryTestSuite))
}
```

**5.2 Testcontainers（真实数据库测试）**

```go
// test/integration/order_integration_test.go
package integration

import (
    "context"
    "fmt"
    "testing"

    "github.com/stretchr/testify/assert"
    "github.com/testcontainers/testcontainers-go"
    "github.com/testcontainers/testcontainers-go/modules/postgres"
    "gorm.io/driver/postgres"
    "gorm.io/gorm"
)

func TestOrderRepository_WithPostgres(t *testing.T) {
    ctx := context.Background()

    // 启动 PostgreSQL 容器
    pgContainer, err := postgres.Run(ctx,
        "postgres:16-alpine",
        postgres.WithDatabase("testdb"),
        postgres.WithUsername("test"),
        postgres.WithPassword("test"),
    )
    assert.NoError(t, err)
    defer pgContainer.Terminate(ctx)

    // 获取连接信息
    connStr, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
    assert.NoError(t, err)

    // 连接数据库
    db, err := gorm.Open(postgres.Open(connStr), &gorm.Config{})
    assert.NoError(t, err)

    // 执行测试...
    fmt.Println("Connected to test PostgreSQL")
}
```

#### AI 约束指令模板

```
【Go 集成测试约束】
- Repository 测试使用 SQLite 内存数据库或 testcontainers
- 每个测试后清理数据，禁止测试间状态依赖
- HTTP 客户端测试使用 httptest.Server
- 禁止在单元测试中使用 time.Sleep
- 集成测试标记 //go:build integration，常规测试跳过
```

---

### Day 5: 测试覆盖率与性能测试

#### 学习目标
- 配置覆盖率门禁
- 编写 Benchmark 测试

#### 核心知识点

**6.1 覆盖率配置**

```bash
# 生成覆盖率报告
go test -coverprofile=coverage.out ./...
go tool cover -func=coverage.out

# HTML 可视化
go tool cover -html=coverage.out -o coverage.html
```

**6.2 Benchmark 测试**

```go
func BenchmarkOrder_CalculateTotal(b *testing.B) {
    order := domain.NewOrder("C001")
    for i := 0; i < 100; i++ {
        order.AddItem(domain.OrderItem{
            ProductID: fmt.Sprintf("P%03d", i),
            Price:     int64(i * 100),
            Quantity:  i + 1,
        })
    }

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        order.TotalAmount()
    }
}
```

---

## 第三周：设计模式应用

### Day 1-2: 创建型模式

#### 学习目标
- 掌握 Go 的创建模式惯用法
- 控制对象生命周期

#### 核心知识点

**7.1 构造函数模式（Go 标准）**

```go
// ❌ 坏味道：零值初始化后手动设置字段
order := &domain.Order{}
order.ID = domain.GenerateOrderID()
order.CustomerID = "C001"
order.Status = domain.OrderStatusCreated
order.CreatedAt = time.Now()

// ✅ 构造函数封装创建逻辑
func NewOrder(customerID string) *Order {
    return &Order{
        ID:        GenerateOrderID(),
        CustomerID:  CustomerID(customerID),
        Status:     OrderStatusCreated,
        CreatedAt:  time.Now(),
        Items:      make([]OrderItem, 0),
    }
}

// 使用
order := domain.NewOrder("C001")
```

**7.2 Builder 模式（参数过多时）**

```go
// ❌ 参数过多
func NewOrder(id, customerID string, items []OrderItem, 
              amount Money, status OrderStatus, createdAt time.Time,
              shippingAddress Address) *Order {
    // 参数超过 4 个！
}

// ✅ Builder 模式
type OrderBuilder struct {
    customerID      string
    items           []OrderItem
    shippingAddress *Address
    couponCode      *string
}

func NewOrderBuilder(customerID string) *OrderBuilder {
    return &OrderBuilder{
        customerID: customerID,
        items:      make([]OrderItem, 0),
    }
}

func (b *OrderBuilder) AddItem(item OrderItem) *OrderBuilder {
    b.items = append(b.items, item)
    return b
}

func (b *OrderBuilder) WithShippingAddress(addr Address) *OrderBuilder {
    b.shippingAddress = &addr
    return b
}

func (b *OrderBuilder) WithCoupon(code string) *OrderBuilder {
    b.couponCode = &code
    return b
}

func (b *OrderBuilder) Build() (*Order, error) {
    if b.customerID == "" {
        return nil, ErrMissingCustomerID
    }
    if len(b.items) == 0 {
        return nil, ErrEmptyOrder
    }

    order := NewOrder(b.customerID)
    for _, item := range b.items {
        if err := order.AddItem(item); err != nil {
            return nil, err
        }
    }

    if b.shippingAddress != nil {
        order.SetShippingAddress(*b.shippingAddress)
    }

    if b.couponCode != nil {
        if err := order.ApplyCoupon(*b.couponCode); err != nil {
            return nil, err
        }
    }

    return order, nil
}

// 使用
order, err := domain.NewOrderBuilder("C001").
    AddItem(domain.OrderItem{ProductID: "P001", Price: 1000, Quantity: 2}).
    AddItem(domain.OrderItem{ProductID: "P002", Price: 500, Quantity: 1}).
    WithShippingAddress(addr).
    WithCoupon("SAVE20").
    Build()
```

**7.3 单例模式（Go 惯用：sync.Once）**

```go
// ✅ 无状态配置类可用单例
type AppConfig struct {
    DatabaseURL string
    RedisAddr   string
    LogLevel    string
}

var (
    configInstance *AppConfig
    configOnce     sync.Once
)

func GetConfig() *AppConfig {
    configOnce.Do(func() {
        configInstance = loadConfigFromEnv()
    })
    return configInstance
}

// ❌ 禁止：业务状态单例
type OrderCache struct {
    mu     sync.RWMutex
    orders map[string]*domain.Order  // 危险！全局状态
}

var cacheInstance *OrderCache
var cacheOnce sync.Once

func GetOrderCache() *OrderCache {
    cacheOnce.Do(func() {
        cacheInstance = &OrderCache{orders: make(map[string]*domain.Order)}
    })
    return cacheInstance
}
```

#### AI 约束指令模板

```
【Go 创建型模式约束】
- 复杂对象使用构造函数 NewXxx() 封装创建逻辑
- 参数超过 4 个必须使用 Builder 模式
- Builder 方法返回 *Builder 支持链式调用
- 单例仅限无状态配置类，使用 sync.Once 实现
- 禁止业务状态单例，状态必须显式传递
- 禁止在业务逻辑中直接 &Struct{} 后手动设置多个字段
```

---

### Day 3-4: 结构型与行为型模式

#### 学习目标
- 掌握 Go 的接口组合实现装饰器
- 使用函数类型实现策略模式

#### 核心知识点

**8.1 策略模式（Go 函数式实现）**

```go
// ❌ if-else 膨胀
func CalculateDiscount(customerType string, amount int64) int64 {
    if customerType == "VIP" {
        return amount * 80 / 100
    } else if customerType == "GOLD" {
        return amount * 90 / 100
    } else if customerType == "SILVER" {
        return amount * 95 / 100
    }
    return amount
}

// ✅ 策略模式 - Go 函数式风格
type DiscountStrategy func(amount int64) int64

var (
    VIPDiscount    DiscountStrategy = func(amount int64) int64 { return amount * 80 / 100 }
    GoldDiscount   DiscountStrategy = func(amount int64) int64 { return amount * 90 / 100 }
    SilverDiscount DiscountStrategy = func(amount int64) int64 { return amount * 95 / 100 }
    NoDiscount     DiscountStrategy = func(amount int64) int64 { return amount }
)

var strategies = map[string]DiscountStrategy{
    "VIP":    VIPDiscount,
    "GOLD":   GoldDiscount,
    "SILVER": SilverDiscount,
}

func GetDiscountStrategy(customerType string) DiscountStrategy {
    if strategy, ok := strategies[customerType]; ok {
        return strategy
    }
    return NoDiscount
}

// 使用
strategy := GetDiscountStrategy("VIP")
discounted := strategy(originalAmount)
```

**8.2 装饰器模式（Go 函数式 + 接口）**

```go
// 基础接口
type OrderService interface {
    CreateOrder(ctx context.Context, req dto.CreateOrderRequest) (*dto.OrderResponse, error)
}

// 核心实现
type orderServiceImpl struct{...}

// 日志装饰器
func WithLogging(service OrderService, logger *slog.Logger) OrderService {
    return &loggingOrderService{
        base:   service,
        logger: logger,
    }
}

type loggingOrderService struct {
    base   OrderService
    logger *slog.Logger
}

func (s *loggingOrderService) CreateOrder(ctx context.Context, req dto.CreateOrderRequest) (*dto.OrderResponse, error) {
    s.logger.InfoContext(ctx, "creating order", "request", req)

    start := time.Now()
    result, err := s.base.CreateOrder(ctx, req)
    duration := time.Since(start)

    if err != nil {
        s.logger.ErrorContext(ctx, "order creation failed", 
            "error", err, 
            "duration", duration)
    } else {
        s.logger.InfoContext(ctx, "order created", 
            "order_id", result.ID, 
            "duration", duration)
    }

    return result, err
}

// 缓存装饰器
func WithCaching(service OrderService, cache cache.Cache) OrderService {
    return &cachingOrderService{base: service, cache: cache}
}

// 链路追踪装饰器
func WithTracing(service OrderService, tracer trace.Tracer) OrderService {
    return &tracingOrderService{base: service, tracer: tracer}
}

// 组合使用（洋葱式）
service := NewOrderService(repo, stock, pub)
service = WithLogging(service, logger)
service = WithCaching(service, redisCache)
service = WithTracing(service, tracer)
service = WithMetrics(service, meter)  // 监控装饰器
```

**8.3 责任链模式**

```go
type OrderApprover interface {
    SetNext(next OrderApprover)
    Approve(ctx context.Context, order *domain.Order) (bool, error)
}

type baseApprover struct {
    next OrderApprover
}

func (a *baseApprover) SetNext(next OrderApprover) {
    a.next = next
}

type AmountApprover struct {
    baseApprover
    threshold int64
}

func (a *AmountApprover) Approve(ctx context.Context, order *domain.Order) (bool, error) {
    if order.TotalAmount().Value() <= a.threshold {
        return true, nil  // 小额直接通过
    }
    if a.next != nil {
        return a.next.Approve(ctx, order)
    }
    return false, nil
}

type VIPApprover struct {
    baseApprover
}

func (a *VIPApprover) Approve(ctx context.Context, order *domain.Order) (bool, error) {
    if order.CustomerType == domain.CustomerTypeVIP {
        return true, nil  // VIP 免审
    }
    if a.next != nil {
        return a.next.Approve(ctx, order)
    }
    return false, nil
}

// 组装责任链
func BuildApprovalChain() OrderApprover {
    vip := &VIPApprover{}
    amount := &AmountApprover{threshold: 10000}
    manager := &ManagerApprover{}

    vip.SetNext(amount)
    amount.SetNext(manager)

    return vip
}
```

#### AI 约束指令模板

```
【Go 行为型模式约束】
- if-else 超过 3 层 → 策略模式（函数式或接口式）
- switch 超过 2 个 case → 考虑策略模式
- 横切关注点（日志、缓存、链路追踪）→ 装饰器模式
- 装饰器使用函数包装：WithLogging(service, logger)
- 状态流转 → 状态模式，禁止在 Service 中写状态转换 if-else
- 函数参数超过 4 个 → 使用结构体参数或 Builder
```

---

### Day 5: 代码坏味道识别与重构

#### 核心坏味道清单（Go 特化版）

| 坏味道 | 识别特征 | 重构手法 | AI 约束 |
|--------|---------|---------|---------|
| **过长函数** | > 30 行 | 提取函数 | 函数不超过 30 行 |
| **过大文件** | > 500 行 | 拆分到多个文件 | 文件不超过 500 行 |
| **过多参数** | > 4 个 | 结构体参数 / Builder | 参数超 4 个用结构体 |
| **全局变量** | package 级别 var | 依赖注入 | 禁止业务全局变量 |
| **空接口滥用** | interface{} 传参 | 泛型或具体类型 | 优先使用泛型 |
| **重复代码** | 相同逻辑多处 | 提取函数 | DRY 原则 |
| **深层嵌套** | 多层 if/for 嵌套 | 提前返回 / 提取函数 | 嵌套不超过 3 层 |
| **魔法数字** | 未命名常量 | 提取为 const | 禁止魔法数字 |
| **裸返回** | 命名返回值裸 return | 显式返回 | 禁止裸返回 |
| **panic 滥用** | 业务逻辑中 panic | 返回 error | 禁止业务 panic |

---

## 第四周：防御性编程与 AI 协作规范

### Day 1-2: 防御性编程

#### 学习目标
- Go 的 error 处理最佳实践
- 零值与 nil 的安全处理
- 不可变设计

#### 核心知识点

**9.1 Error 处理（Go 核心哲学）**

```go
// ❌ 错误处理反模式
func ProcessOrder(orderID string) {
    order, _ := repo.FindByID(orderID)  // 忽略错误！
    _ = service.Process(order)          // 忽略错误！
}

// ❌ 过度包装
func ProcessOrder(orderID string) error {
    order, err := repo.FindByID(orderID)
    if err != nil {
        return fmt.Errorf("failed to find order: %w", err)
    }

    err = service.Process(order)
    if err != nil {
        return fmt.Errorf("failed to process order: %w", err)
    }

    err = notify.Send(order)
    if err != nil {
        return fmt.Errorf("failed to send notification: %w", err)
    }

    return nil
}

// ✅ 清晰错误处理 + 错误类型判断
var (
    ErrOrderNotFound    = errors.New("order not found")
    ErrInsufficientStock = errors.New("insufficient stock")
    ErrPaymentFailed    = errors.New("payment failed")
)

func ProcessOrder(orderID string) error {
    order, err := repo.FindByID(orderID)
    if err != nil {
        if errors.Is(err, sql.ErrNoRows) {
            return fmt.Errorf("%w: id=%s", ErrOrderNotFound, orderID)
        }
        return fmt.Errorf("database error: %w", err)
    }

    if err := service.Process(order); err != nil {
        return fmt.Errorf("process order %s: %w", orderID, err)
    }

    // 通知失败不阻断主流程
    if err := notify.Send(order); err != nil {
        slog.Error("notification failed", "order_id", orderID, "error", err)
    }

    return nil
}

// 调用方判断错误类型
if err := ProcessOrder(id); err != nil {
    if errors.Is(err, ErrOrderNotFound) {
        return c.JSON(404, dto.ErrorResponse{Code: "ORDER_NOT_FOUND"})
    }
    if errors.Is(err, ErrInsufficientStock) {
        return c.JSON(409, dto.ErrorResponse{Code: "STOCK_INSUFFICIENT"})
    }
    return c.JSON(500, dto.ErrorResponse{Code: "INTERNAL_ERROR"})
}
```

**9.2 Nil 安全处理**

```go
// ✅ 构造函数保证非 nil
func NewOrderService(...) OrderService {
    if repo == nil {
        panic("repository is nil")  // 仅在构造函数中允许 panic
    }
    return &orderServiceImpl{...}
}

// ✅ 防御性检查
func (s *orderServiceImpl) CreateOrder(ctx context.Context, req dto.CreateOrderRequest) (*dto.OrderResponse, error) {
    if req.CustomerID == "" {
        return nil, fmt.Errorf("%w: customer_id is required", ErrInvalidRequest)
    }
    if req.Quantity <= 0 {
        return nil, fmt.Errorf("%w: quantity must be positive", ErrInvalidRequest)
    }

    // 业务逻辑...
}

// ✅ 使用 option 模式处理可选配置
type ServerOption func(*Server)

func WithPort(port int) ServerOption {
    return func(s *Server) {
        s.port = port
    }
}

func WithLogger(logger *slog.Logger) ServerOption {
    return func(s *Server) {
        s.logger = logger
    }
}

func NewServer(opts ...ServerOption) *Server {
    s := &Server{
        port:   8080,
        logger: slog.Default(),
    }
    for _, opt := range opts {
        opt(s)
    }
    return s
}

// 使用
server := NewServer(
    WithPort(9090),
    WithLogger(customLogger),
)
```

**9.3 不可变设计**

```go
// ✅ 不可变值对象
type Money struct {
    value    int64
    currency Currency
}

func NewMoney(value int64, currency Currency) Money {
    return Money{value: value, currency: currency}
}

func (m Money) Value() int64 {
    return m.value
}

func (m Money) Currency() Currency {
    return m.currency
}

func (m Money) Add(other Money) (Money, error) {
    if m.currency != other.currency {
        return Money{}, ErrCurrencyMismatch
    }
    return NewMoney(m.value+other.value, m.currency), nil
}

// Money 不可变：没有 setter，Add 返回新值

// ✅ 不可变集合防御
type Order struct {
    id     OrderID
    items  []OrderItem  // 私有字段
}

func (o *Order) Items() []OrderItem {
    // 返回副本，防止外部修改
    result := make([]OrderItem, len(o.items))
    copy(result, o.items)
    return result
}
```

#### AI 约束指令模板

```
【Go 防御性编程约束】
- 禁止忽略 error（_ = someFunc()），必须处理或显式注释原因
- 使用 errors.Is 判断错误类型，支持错误链
- 构造函数对 nil 依赖使用 panic（快速失败）
- public 函数入口验证所有参数
- 返回可能为 nil 的指针时，文档中明确标注
- 值对象设计为不可变（无 setter，修改方法返回新值）
- 切片/映射返回时考虑返回副本防止外部修改
- 禁止业务逻辑中使用 panic，panic 仅限程序启动和构造函数
```

---

### Day 3-4: 防腐层与外部系统隔离

#### 学习目标
- 设计 Go 风格的防腐层
- 隔离第三方依赖

#### 核心知识点

**10.1 防腐层设计**

```go
// ❌ 直接透传外部模型
package controller

func (h *OrderHandler) AlipayCallback(c *gin.Context) {
    var req alipay.TradeNotification  // 直接使用第三方结构体！
    c.ShouldBindJSON(&req)

    if req.TradeStatus == "TRADE_SUCCESS" {
        // 业务逻辑...
    }
}

// ✅ 防腐层：领域层定义接口
// domain/payment_gateway.go
package domain

type PaymentGateway interface {
    ProcessPayment(ctx context.Context, req PaymentRequest) (*PaymentResult, error)
    VerifyCallback(ctx context.Context, callbackData map[string]string) (*PaymentNotification, error)
}

type PaymentRequest struct {
    OrderID string
    Amount  Money
    Channel PaymentChannel
}

type PaymentResult struct {
    TransactionID string
    Status        PaymentStatus
    PaidAt        time.Time
}

// infrastructure/acl/alipay_adapter.go
package acl

import (
    "context"
    "fmt"
    "myapp/internal/domain"
)

type AlipayAdapter struct {
    client AlipayClient
}

var _ domain.PaymentGateway = (*AlipayAdapter)(nil)  // 编译期检查

func NewAlipayAdapter(client AlipayClient) *AlipayAdapter {
    return &AlipayAdapter{client: client}
}

func (a *AlipayAdapter) ProcessPayment(ctx context.Context, req domain.PaymentRequest) (*domain.PaymentResult, error) {
    // 转换：领域对象 → 支付宝对象
    alipayReq := AlipayTradeRequest{
        OutTradeNo: req.OrderID,
        TotalAmount: fmt.Sprintf("%.2f", float64(req.Amount.Value())/100),
        Subject:    fmt.Sprintf("Order %s", req.OrderID),
    }

    // 调用第三方
    resp, err := a.client.TradePagePay(ctx, alipayReq)
    if err != nil {
        return nil, fmt.Errorf("alipay trade failed: %w", err)
    }

    // 转换：支付宝对象 → 领域对象
    return &domain.PaymentResult{
        TransactionID: resp.TradeNo,
        Status:        domain.PaymentStatusPending,
    }, nil
}

func (a *AlipayAdapter) VerifyCallback(ctx context.Context, data map[string]string) (*domain.PaymentNotification, error) {
    // 验证签名
    if !a.client.VerifySign(data) {
        return nil, domain.ErrInvalidSignature
    }

    // 转换为领域对象
    return &domain.PaymentNotification{
        OrderID:       data["out_trade_no"],
        TransactionID: data["trade_no"],
        Status:        parseAlipayStatus(data["trade_status"]),
        Amount:        parseAmount(data["total_amount"]),
        PaidAt:        parseTime(data["gmt_payment"]),
    }, nil
}

func parseAlipayStatus(status string) domain.PaymentStatus {
    switch status {
    case "TRADE_SUCCESS", "TRADE_FINISHED":
        return domain.PaymentStatusSuccess
    case "WAIT_BUYER_PAY":
        return domain.PaymentStatusPending
    default:
        return domain.PaymentStatusFailed
    }
}
```

**10.2 外部 API 客户端隔离**

```go
// domain/ports.go
package domain

type InventoryPort interface {
    CheckAvailability(ctx context.Context, productID string, quantity int) (bool, error)
    Reserve(ctx context.Context, productID string, quantity int) error
    Release(ctx context.Context, productID string, quantity int) error
}

// infrastructure/client/inventory_client.go
package client

type InventoryClient struct {
    baseURL string
    client  *http.Client
    breaker *gobreaker.CircuitBreaker
}

func NewInventoryClient(baseURL string) *InventoryClient {
    return &InventoryClient{
        baseURL: baseURL,
        client:  &http.Client{Timeout: 5 * time.Second},
        breaker: gobreaker.NewCircuitBreaker(gobreaker.Settings{
            Name:        "inventory",
            MaxRequests: 100,
            Interval:    10 * time.Second,
            Timeout:     30 * time.Second,
            ReadyToTrip: func(counts gobreaker.Counts) bool {
                failureRatio := float64(counts.TotalFailures) / float64(counts.Requests)
                return counts.Requests >= 10 && failureRatio >= 0.5
            },
        }),
    }
}

func (c *InventoryClient) CheckAvailability(ctx context.Context, productID string, quantity int) (bool, error) {
    result, err := c.breaker.Execute(func() (interface{}, error) {
        req, _ := http.NewRequestWithContext(ctx, "GET",
            fmt.Sprintf("%s/api/v1/stock/%s?quantity=%d", c.baseURL, productID, quantity), nil)

        resp, err := c.client.Do(req)
        if err != nil {
            return nil, err
        }
        defer resp.Body.Close()

        if resp.StatusCode != 200 {
            return nil, fmt.Errorf("inventory service returned %d", resp.StatusCode)
        }

        var result struct{ Available bool }
        if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
            return nil, err
        }
        return result.Available, nil
    })

    if err != nil {
        return false, err
    }
    return result.(bool), nil
}
```

#### AI 约束指令模板

```
【Go 防腐层约束】
- 调用第三方 API 必须经过 ACL 转换，禁止透传外部模型到 domain/service
- 第三方 SDK 结构体禁止出现在 domain 包
- 外部 HTTP 调用必须加熔断（gobreaker）和超时控制
- 回调接口必须验证签名后再转换为领域对象
- 第三方错误必须转换为领域错误后向上传递
```

---

### Day 5: 完整 System Prompt 编写与实战

#### 学习目标
- 整合所有约束为可执行的 System Prompt
- 在真实 Go 项目中验证约束效果

---

## 附录 A：完整 AI 约束指令集（System Prompt）

见独立文档：《Go AI System Prompt 完整约束模板》

---

## 附录 B：Go 架构检查工具配置

### golangci-lint 配置

```yaml
# .golangci.yml
run:
  timeout: 5m
  go: "1.22"

linters:
  enable:
    - errcheck      # 检查未处理的 error
    - gosimple      # 简化代码建议
    - govet         # 标准 vet
    - ineffassign   # 无效赋值
    - staticcheck   # 静态分析
    - unused        # 未使用代码
    - cyclop        # 圈复杂度
    - gocognit      # 认知复杂度
    - gocyclo       # 循环复杂度
    - funlen        # 函数长度
    - lll           # 行长度限制
    - revive        # 代码规范
    - gocritic      # 代码审查
    - gosec         # 安全检查
    - noctx         # context 检查
    - nilerr        # nil error 检查
    - wrapcheck     # error 包装检查

linters-settings:
  cyclop:
    max-complexity: 10
    package-average: 5.0
  funlen:
    lines: 30
    statements: 20
  lll:
    line-length: 120
  gocognit:
    min-complexity: 10
  wrapcheck:
    ignoreSigs:
      - .Errorf(
      - errors.New(
      - errors.Unwrap(
      - .Wrap(
      - .Wrapf(
      - .WithMessage(
      - .WithMessagef(
      - .WithStack(

issues:
  exclude-rules:
    # 测试文件放宽限制
    - path: _test\.go
      linters:
        - funlen
        - gocognit
        - cyclop
```

### Makefile 集成

```makefile
.PHONY: lint test coverage arch-check

lint:
	golangci-lint run ./...

test:
	go test -race -count=1 ./...

coverage:
	go test -coverprofile=coverage.out ./...
	go tool cover -func=coverage.out | grep total

arch-check:
	go run tools/archcheck/main.go

ci: lint arch-check test coverage
```

---

## 附录 C：代码审查 Checklist（Go 特化版）

### 架构层面
- [ ] 是否存在跨层调用（Handler 直接调用 Repository）？
- [ ] domain 包是否导入了第三方库（Gin/GORM/Redis）？
- [ ] 是否使用了 internal 包保护私有代码？
- [ ] 是否存在循环依赖（go mod 可检测）？
- [ ] 是否使用了接口进行依赖倒置？

### Error 处理
- [ ] 是否有被忽略的 error（`_ = xxx()`）？
- [ ] error 是否使用了 `%w` 包装以支持 errors.Is？
- [ ] 业务逻辑中是否使用了 panic？
- [ ] 错误类型是否定义清晰（变量而非字符串比较）？

### 设计模式
- [ ] if-else 是否超过 3 层？
- [ ] switch 是否超过 2 个 case？
- [ ] 参数是否超过 4 个？
- [ ] 是否存在重复代码？
- [ ] 横切关注点是否使用装饰器？

### 测试
- [ ] 每个 public 函数是否有测试？
- [ ] 是否使用了表驱动测试？
- [ ] 是否覆盖边界条件（nil、零值、空集合）？
- [ ] 外部依赖是否 Mock？
- [ ] 是否有并发测试（-race）？

### 防御性
- [ ] 函数入口是否验证参数？
- [ ] 切片/映射操作是否检查边界？
- [ ] 返回指针时是否可能为 nil？
- [ ] 资源是否正确释放（defer Close）？
- [ ] 时间操作是否使用 time.Now() 而非固定值？

### 代码风格
- [ ] 是否运行过 go fmt？
- [ ] 函数是否超过 30 行？
- [ ] 文件是否超过 500 行？
- [ ] 是否有魔法数字？
- [ ] 命名是否符合 Go 惯例（驼峰/首字母大写导出）？

---

## 附录 D：Go 与 Java 约束对照表

| 约束维度 | Java | Go |
|---------|------|-----|
| **分层保护** | ArchUnit + 包命名 | internal/ 包 + 自定义检查 |
| **依赖注入** | Spring @Autowired 构造器 | 手动构造函数 + Wire |
| **接口定义** | 显式 implements | 隐式实现 + 编译期检查 |
| **测试框架** | JUnit + Mockito | testing + testify |
| **Mock 生成** | Mockito / Spring Boot Test | 手写 / mockery |
| **错误处理** | Exception 体系 | error 返回值 + errors.Is |
| **空值处理** | Optional<T> | 返回值标注 + 防御性检查 |
| **不可变性** | final + @Value | struct 值类型 + 无 setter |
| **装饰器** | Spring AOP / 接口包装 | 函数式包装 + 接口嵌入 |
| **单例** | 枚举 / 静态内部类 | sync.Once |
| **Builder** | Lombok @Builder | 手写 Builder（链式） |
| **静态检查** | ArchUnit + Checkstyle | golangci-lint + 自定义 |
| **日志** | SLF4J + Logback | slog / zap / logrus |
| **ORM** | JPA / MyBatis | GORM / SQLx |
| **Web 框架** | Spring Boot | Gin / Echo / Fiber |

---

*文档版本: 1.0 | 适用语言: Go 1.22+ | 更新时间: 2026-05*
