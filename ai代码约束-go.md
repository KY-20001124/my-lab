# AI 代码约束体系 — Go 新人培训教案

> **目标**：让新人掌握与 AI Coding Agent 协作时的 Go 工程纪律，将架构原则、测试习惯、错误处理和代码审查规则转化为可执行的 AI 约束指令。
> **课时**：4 周（每周 5 天，每天 2 小时理论 + 2 小时实战）
> **产出**：能够独立编写 System Prompt 约束 AI，产出符合 Go 项目结构、依赖边界、测试规范和错误处理规范的代码。

---

## 第一周：Go 项目分层与依赖管理

### Day 1-2: 分层架构核心原则

#### 学习目标
- 理解 Go 包边界与物理目录边界的意义
- 掌握单向依赖原则，避免业务代码反向依赖基础设施
- 能够识别和修复 handler 直接访问数据库、领域层依赖外部 SDK 等跨层问题

#### 核心知识点

**1.1 推荐目录结构**
```text
.
├── cmd/
│   └── order-api/              # 程序入口，负责组装依赖和启动服务
├── internal/
│   ├── handler/                # 入站适配器：HTTP / RPC / CLI / MQ 消费
│   ├── application/            # 用例层：编排领域对象和端口
│   ├── domain/                 # 领域层：实体、值对象、领域服务、领域错误
│   ├── repository/             # 领域拥有的仓储接口或端口定义
│   └── infrastructure/         # 出站适配器：数据库、第三方 API、消息队列
├── pkg/                        # 可被外部项目复用的稳定公共库，默认不创建
├── test/                       # 跨包测试夹具、契约测试、端到端测试
├── go.mod
└── .golangci.yml
```

**1.2 经典三层架构**
```text
┌────────────────────────────────────────────┐
│  Handler Layer                             │  ← 处理 HTTP / RPC / MQ 输入
│  职责：解析请求、参数校验、DTO 转换、返回响应 │
├────────────────────────────────────────────┤
│  Application Layer                         │  ← 用例编排
│  职责：事务边界、流程控制、调用领域和端口       │
├────────────────────────────────────────────┤
│  Domain Layer                              │  ← 核心业务规则
│  职责：实体、值对象、领域行为、业务不变量       │
├────────────────────────────────────────────┤
│  Infrastructure Layer                      │  ← 外部系统适配
│  职责：数据库、缓存、HTTP 客户端、消息队列      │
└────────────────────────────────────────────┘
         ↑ 严格单向依赖，外层依赖内层，内层不依赖外层实现
```

**1.3 六边形架构（进阶）**
```text
         ┌───────────────┐
         │   外部驱动     │  ← HTTP Handler / CLI / Message Consumer
         └───────┬───────┘
                 │ 调用入站适配器
    ┌────────────▼────────────┐
    │      Application         │  ← Use Case，编排领域逻辑
    ├──────────────────────────┤
    │        Domain            │  ← 零外部框架依赖
    │  Entity / Value Object   │
    └────────────┬────────────┘
                 │ 依赖端口接口
         ┌───────▼───────┐
         │   外部被驱动    │  ← DB / API Client / MQ / Cache
         └───────────────┘
```

**1.4 依赖方向铁律**
```go
// 错误：handler 直接访问数据库，跨过用例层和仓储端口。
package handler

import (
	"database/sql"
	"net/http"
)

type OrderHandler struct {
	db *sql.DB
}

func (h *OrderHandler) Create(w http.ResponseWriter, r *http.Request) {
	_, _ = h.db.ExecContext(r.Context(), "insert into orders(id) values($1)", "ORD-1")
}
```

```go
// 正确：handler 只依赖用例接口，用例依赖仓储端口。
package handler

import (
	"encoding/json"
	"net/http"
)

type CreateOrderUseCase interface {
	CreateOrder(ctx context.Context, cmd CreateOrderCommand) (OrderDTO, error)
}

type OrderHandler struct {
	createOrder CreateOrderUseCase
}

func NewOrderHandler(createOrder CreateOrderUseCase) *OrderHandler {
	return &OrderHandler{createOrder: createOrder}
}

func (h *OrderHandler) Create(w http.ResponseWriter, r *http.Request) {
	var cmd CreateOrderCommand
	if err := json.NewDecoder(r.Body).Decode(&cmd); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	result, err := h.createOrder.CreateOrder(r.Context(), cmd)
	if err != nil {
		writeError(w, err)
		return
	}

	_ = json.NewEncoder(w).Encode(result)
}
```

#### 实战任务
1. 审查一段 AI 生成的 Go 代码，找出 handler 直接访问数据库、domain 引入 SDK、infrastructure 泄漏模型等跨层调用。
2. 使用 `go list` 或自定义测试检查包依赖方向。
3. 为一个小型订单服务画出 `handler -> application -> domain -> repository port -> infrastructure` 的依赖图。

#### AI 约束指令模板
```text
【Go 分层约束】
- 严格遵循 handler -> application -> domain/repository port -> infrastructure adapter 的依赖方向
- handler 层禁止直接访问数据库、缓存、第三方 SDK
- domain 层禁止引入 HTTP、SQL、外部 SDK、日志框架等基础设施依赖
- application 层只能通过接口端口调用外部系统
- infrastructure 层负责实现端口，不得反向污染领域模型
- 默认使用 internal/ 保护项目内部包边界，除非明确需要稳定公共 API，不创建 pkg/
```

---

### Day 3-4: 依赖注入与接口设计

#### 学习目标
- 掌握 Go 中通过构造函数注入依赖
- 理解接口隔离原则，避免为测试而提前抽象过大的接口
- 能够设计由消费者拥有的小接口

#### 核心知识点

**2.1 Go 依赖注入方式对比**

| 方式 | 可测试性 | 依赖显式性 | 推荐度 |
|------|---------|-----------|--------|
| 包级全局变量 | 差 | 隐藏 | 禁止 |
| 在函数内部创建具体依赖 | 差 | 半隐藏 | 禁止用于业务依赖 |
| 构造函数注入 | 优 | 完全显式 | 强制 |
| 函数参数注入 | 优 | 完全显式 | 适合轻量函数 |

```go
// 错误：包级变量隐藏依赖，测试间互相污染。
var orderRepo = NewPostgresOrderRepository()
var paymentClient = NewPaymentClient()

type OrderService struct{}

func (s *OrderService) Create(ctx context.Context, cmd CreateOrderCommand) error {
	order := NewOrder(cmd.CustomerID, cmd.Items)
	if err := orderRepo.Save(ctx, order); err != nil {
		return err
	}
	return paymentClient.Charge(ctx, order.ID, order.Total)
}
```

```go
// 正确：构造函数注入，依赖显式且可替换。
type OrderRepository interface {
	Save(ctx context.Context, order Order) error
	FindByID(ctx context.Context, id OrderID) (Order, error)
}

type PaymentPort interface {
	Charge(ctx context.Context, orderID OrderID, amount Money) error
}

type OrderService struct {
	orders  OrderRepository
	payment PaymentPort
}

func NewOrderService(orders OrderRepository, payment PaymentPort) *OrderService {
	return &OrderService{
		orders:  orders,
		payment: payment,
	}
}
```

**2.2 接口由消费者定义**
```go
// 错误：基础设施层暴露一个胖接口，迫使业务依赖不需要的方法。
type UserStore interface {
	FindByID(ctx context.Context, id string) (User, error)
	FindByEmail(ctx context.Context, email string) (User, error)
	Save(ctx context.Context, user User) error
	Delete(ctx context.Context, id string) error
	RunSQL(ctx context.Context, query string) error
	BeginTx(ctx context.Context) (Tx, error)
}
```

```go
// 正确：用例只声明自己需要的小接口。
type UserReader interface {
	FindByID(ctx context.Context, id UserID) (User, error)
}

type LoadUserUseCase struct {
	users UserReader
}

func NewLoadUserUseCase(users UserReader) *LoadUserUseCase {
	return &LoadUserUseCase{users: users}
}
```

**2.3 依赖倒置实践**
```go
// repository/order.go：领域或用例层拥有端口。
type OrderRepository interface {
	FindByID(ctx context.Context, id OrderID) (Order, error)
	Save(ctx context.Context, order Order) error
	FindByStatus(ctx context.Context, status OrderStatus) ([]Order, error)
}
```

```go
// infrastructure/postgres_order_repository.go：基础设施层实现端口。
type PostgresOrderRepository struct {
	db     *sql.DB
	mapper OrderMapper
}

func NewPostgresOrderRepository(db *sql.DB, mapper OrderMapper) *PostgresOrderRepository {
	return &PostgresOrderRepository{db: db, mapper: mapper}
}

func (r *PostgresOrderRepository) FindByID(ctx context.Context, id OrderID) (Order, error) {
	row := r.db.QueryRowContext(ctx, "select id, status from orders where id = $1", id.String())
	return r.mapper.FromRow(row)
}
```

#### 实战任务
1. 将一段使用包级全局变量的代码重构为 `NewXxx(...)` 构造函数注入。
2. 将一个包含 6 个以上方法的胖接口拆分为 2-3 个消费者拥有的小接口。
3. 编写测试验证业务用例可以通过 fake repository 独立运行。

#### AI 约束指令模板
```text
【Go 依赖注入约束】
- 禁止业务代码使用包级可变全局变量保存依赖
- 依赖必须通过 NewXxx(...) 构造函数或函数参数显式传入
- 接口应由消费者所在包定义，方法数量默认不超过 5 个
- 不为只有一个调用点的简单实现提前抽象接口，除非用于隔离外部系统或测试边界
- 高层模块依赖接口，基础设施模块实现接口
- 构造函数不得执行网络连接、数据库迁移等重副作用操作
```

---

### Day 5: Go 架构守护

#### 学习目标
- 使用工具将架构规则代码化
- 配置 CI 自动拦截跨层 import、循环依赖、静态检查失败
- 理解哪些规则适合自动化，哪些需要代码审查兜底

#### 核心知识点

**3.1 `go list` 依赖边界检查**
```bash
go list -deps ./internal/domain/...
go list -deps ./internal/application/...
```

领域层不应依赖这些包族：
```text
net/http
database/sql
github.com/gin-gonic/gin
github.com/go-redis/redis
github.com/aws/aws-sdk-go
```

**3.2 自定义 import 规则测试**
```go
package architecture_test

import (
	"os/exec"
	"strings"
	"testing"
)

func TestDomainDoesNotDependOnInfrastructure(t *testing.T) {
	cmd := exec.Command("go", "list", "-deps", "./internal/domain/...")
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("go list failed: %v", err)
	}

	forbidden := []string{
		"net/http",
		"database/sql",
		"github.com/gin-gonic/gin",
		"github.com/go-redis/redis",
	}

	deps := string(out)
	for _, pkg := range forbidden {
		if strings.Contains(deps, pkg) {
			t.Fatalf("domain layer must not depend on %s", pkg)
		}
	}
}
```

**3.3 `golangci-lint` 基础配置**
```yaml
run:
  timeout: 5m
  tests: true

linters:
  enable:
    - govet
    - staticcheck
    - errcheck
    - ineffassign
    - revive
    - gosec
    - gocyclo
    - depguard

linters-settings:
  gocyclo:
    min-complexity: 12
  depguard:
    rules:
      domain:
        files:
          - "internal/domain/**/*.go"
        deny:
          - pkg: "net/http"
            desc: "domain must not know transport details"
          - pkg: "database/sql"
            desc: "domain must not know persistence details"
```

**3.4 CI 集成配置**
```yaml
name: Go Quality Guard

on:
  push:
  pull_request:

jobs:
  test-and-lint:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version-file: go.mod
      - name: Download modules
        run: go mod download
      - name: Run tests
        run: go test ./...
      - name: Run lint
        run: golangci-lint run
```

#### 实战任务
1. 为项目编写 5 条 Go 架构规则：领域层纯净、handler 不访问 DB、无循环依赖、无包级可变依赖、infrastructure 不被 domain import。
2. 配置 `.golangci.yml`，启用 `depguard`、`errcheck`、`staticcheck`。
3. 提交一次包含违规 import 的演示分支，观察 CI 拦截结果。

#### AI 约束指令模板
```text
【Go 架构守护约束】
- 每个 PR 必须通过 go test ./... 和 golangci-lint run
- 使用 depguard 或自定义测试检查 internal/domain 不依赖 transport、SQL、缓存、外部 SDK
- 禁止出现 import cycle，发现循环依赖必须重构包边界
- 禁止忽略关键错误返回，除非有注释说明并通过 errcheck 豁免
- CI 中必须包含架构边界检查，不允许只依赖人工审查
```

---

## 第二周：测试驱动开发（TDD）

### Day 1-2: TDD 基础与 RED-GREEN-REFACTOR

#### 学习目标
- 掌握 Go 中的 TDD 三定律
- 理解测试作为业务契约和重构护栏的意义
- 能够写出可测试的结构体、函数和包边界

#### 核心知识点

**4.1 TDD 三定律**
1. **先写测试**：在写生产代码之前，先写一个失败的单元测试。
2. **只写刚好够的代码**：只写让测试通过的最少实现。
3. **重构**：在测试通过后清理命名、边界、重复和复杂度。

**4.2 Go 测试结构：Arrange-Act-Assert**
```go
func TestOrderService_CreateOrder_DeductsStockWhenAvailable(t *testing.T) {
	// Arrange
	stock := &fakeStockPort{available: true}
	orders := newFakeOrderRepository()
	service := NewOrderService(orders, stock)

	cmd := CreateOrderCommand{
		ProductID: "P001",
		Quantity:  2,
	}

	// Act
	result, err := service.CreateOrder(context.Background(), cmd)

	// Assert
	if err != nil {
		t.Fatalf("CreateOrder() unexpected error: %v", err)
	}
	if result.OrderID == "" {
		t.Fatalf("CreateOrder() returned empty order id")
	}
	if stock.deductedQuantity != 2 {
		t.Fatalf("stock deducted quantity = %d, want 2", stock.deductedQuantity)
	}
}
```

**4.3 table-driven tests**
```go
func TestMoney_Add(t *testing.T) {
	tests := []struct {
		name    string
		left    Money
		right   Money
		want    Money
		wantErr bool
	}{
		{
			name:  "same currency",
			left:  NewMoney(100, "CNY"),
			right: NewMoney(20, "CNY"),
			want:  NewMoney(120, "CNY"),
		},
		{
			name:    "different currency",
			left:    NewMoney(100, "CNY"),
			right:   NewMoney(20, "USD"),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.left.Add(tt.right)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("Add() expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("Add() unexpected error: %v", err)
			}
			if got != tt.want {
				t.Fatalf("Add() = %v, want %v", got, tt.want)
			}
		})
	}
}
```

**4.4 可测试代码的特征**
```go
// 错误：函数内部创建具体依赖，无法替换。
func CreateOrder(ctx context.Context, cmd CreateOrderCommand) error {
	client := NewPaymentHTTPClient("https://payment.example.com")
	return client.Charge(ctx, cmd.OrderID, cmd.Amount)
}
```

```go
// 正确：依赖端口由外部注入。
type PaymentPort interface {
	Charge(ctx context.Context, orderID OrderID, amount Money) error
}

type CreateOrderUseCase struct {
	payment PaymentPort
}

func NewCreateOrderUseCase(payment PaymentPort) *CreateOrderUseCase {
	return &CreateOrderUseCase{payment: payment}
}
```

#### 实战任务
1. 从购物车需求出发，先写失败测试，再实现 `AddItem`、`RemoveItem`、`Total`。
2. 将一个内部创建 HTTP client 的函数重构为依赖注入。
3. 为 `Money.Add` 编写 table-driven tests，覆盖币种不一致、零金额、负金额。

#### AI 约束指令模板
```text
【Go TDD 约束】
- 新增 public 行为必须先写失败测试，再写实现
- 单元测试优先使用 testing 包和 table-driven tests
- 测试必须覆盖正常路径、边界条件、错误路径
- 外部依赖必须替换为 fake、stub 或 mock，不允许单元测试真实访问网络
- 测试名使用 TestType_Method_Behavior 或 TestFunction_Behavior 格式
- 每个测试必须独立可重复，不依赖执行顺序
```

---

### Day 3-4: 测试替身与边界测试

#### 学习目标
- 掌握 fake、stub、spy、mock 在 Go 中的使用边界
- 能够为业务方法设计全面边界测试
- 知道何时使用 `httptest`、内存 fake、容器化集成测试

#### 核心知识点

**5.1 测试替身类型**

| 类型 | 用途 | Go 示例 |
|------|------|---------|
| Dummy | 填充不被使用的参数 | `context.Background()`、空结构体 |
| Stub | 返回固定数据 | `fakeRepo.findByIDResult = order` |
| Spy | 记录调用参数 | `fakeStockPort.deductedQuantity` |
| Mock | 验证复杂交互 | 使用手写 mock 或生成工具 |
| Fake | 简化可运行实现 | `inMemoryOrderRepository` |

```go
type fakeStockPort struct {
	available        bool
	deductedProduct  ProductID
	deductedQuantity int
}

func (f *fakeStockPort) Check(ctx context.Context, productID ProductID, quantity int) (bool, error) {
	return f.available, nil
}

func (f *fakeStockPort) Deduct(ctx context.Context, productID ProductID, quantity int) error {
	f.deductedProduct = productID
	f.deductedQuantity = quantity
	return nil
}
```

**5.2 边界条件测试清单**
```go
func TestCalculator_Boundaries(t *testing.T) {
	t.Run("zero divisor", func(t *testing.T) {
		_, err := Divide(5, 0)
		if !errors.Is(err, ErrDivideByZero) {
			t.Fatalf("Divide() error = %v, want ErrDivideByZero", err)
		}
	})

	t.Run("empty input", func(t *testing.T) {
		got := Average([]int{})
		if got != 0 {
			t.Fatalf("Average(empty) = %v, want 0", got)
		}
	})

	t.Run("max int overflow", func(t *testing.T) {
		_, err := Add(math.MaxInt, 1)
		if !errors.Is(err, ErrOverflow) {
			t.Fatalf("Add() error = %v, want ErrOverflow", err)
		}
	})

	t.Run("negative numbers", func(t *testing.T) {
		got, err := Add(-5, -3)
		if err != nil {
			t.Fatalf("Add() unexpected error: %v", err)
		}
		if got != -8 {
			t.Fatalf("Add() = %d, want -8", got)
		}
	})
}
```

**5.3 HTTP 边界测试**
```go
func TestOrderHandler_Create_InvalidJSON(t *testing.T) {
	handler := NewOrderHandler(&fakeCreateOrderUseCase{})
	req := httptest.NewRequest(http.MethodPost, "/orders", strings.NewReader("{bad-json"))
	rec := httptest.NewRecorder()

	handler.Create(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}
```

**5.4 测试金字塔**
```text
        /\
       /  \
      / E2E \          ← 少量，验证完整流程
     /────────\
    / Contract \       ← 少量，验证外部协议
   /────────────\
  / Integration  \     ← 中量，验证数据库和组件协作
 /────────────────\
/   Unit Tests     \   ← 大量，验证业务逻辑
```

#### 实战任务
1. 为一个用例方法编写完整边界测试，至少覆盖：空输入、零值、负数、最大值、重复请求、外部依赖失败、上下文取消、成功路径。
2. 使用 `httptest` 测试 handler 的状态码、响应体、错误返回。
3. 使用内存 fake 测试 application 层，避免单元测试访问真实数据库。

#### AI 约束指令模板
```text
【Go 测试完整性约束】
- 每个核心业务方法必须覆盖正常输入、零值、空集合、最大值、最小值、负数、错误路径
- handler 测试必须使用 httptest 验证状态码和响应体
- 单元测试禁止真实访问网络、真实数据库、真实消息队列
- 数据库集成测试使用 testcontainers-go、临时数据库或事务回滚策略
- 禁止在单元测试中使用 time.Sleep；需要等待时使用 fake clock 或可控同步原语
- 慢测试必须用 build tag 或测试命名与常规单元测试区分
```

---

### Day 5: 契约测试与 CI 集成

#### 学习目标
- 理解服务间契约测试的价值
- 配置测试覆盖率和静态检查门禁
- 知道 Go 项目如何在 CI 中稳定运行测试

#### 核心知识点

**6.1 HTTP 契约测试示例**
```go
func TestPaymentProviderContract(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("method = %s, want POST", r.Method)
		}
		if r.URL.Path != "/api/v1/payments" {
			t.Fatalf("path = %s, want /api/v1/payments", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"transactionID":"TXN-456","status":"SUCCESS"}`))
	}))
	defer server.Close()

	client := NewPaymentClient(server.URL)
	result, err := client.Pay(context.Background(), PaymentRequest{
		OrderID: "ORD-123",
		Amount:  NewMoney(9999, "CNY"),
	})

	if err != nil {
		t.Fatalf("Pay() unexpected error: %v", err)
	}
	if result.TransactionID != "TXN-456" {
		t.Fatalf("transaction id = %s, want TXN-456", result.TransactionID)
	}
}
```

**6.2 覆盖率门禁**
```bash
go test ./... -coverprofile=coverage.out
go tool cover -func=coverage.out
```

```bash
coverage=$(go tool cover -func=coverage.out | awk '/total:/ {print substr($3, 1, length($3)-1)}')
awk -v coverage="$coverage" 'BEGIN { if (coverage < 80) exit 1 }'
```

**6.3 CI 推荐流程**
```yaml
name: Go Test Guard

on:
  push:
  pull_request:

jobs:
  quality:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version-file: go.mod
      - run: go mod download
      - run: go test ./... -race -coverprofile=coverage.out
      - run: go tool cover -func=coverage.out
      - run: golangci-lint run
```

#### 实战任务
1. 为一个外部支付客户端编写契约测试，固定请求路径、方法、字段、响应解析。
2. 配置 CI 执行 `go test ./... -race`、覆盖率检查和 `golangci-lint run`。
3. 让 AI 生成一段缺少错误处理的代码，观察 `errcheck` 是否能拦截。

#### AI 约束指令模板
```text
【Go CI 测试约束】
- CI 必须执行 go test ./...，关键服务必须额外执行 go test ./... -race
- 核心业务包行覆盖率不得低于 80%，分支复杂逻辑必须有边界测试
- 外部 HTTP 客户端必须有契约测试或 httptest 测试
- 测试失败、静态检查失败、覆盖率不足均不得合入
- CI 脚本必须可在本地复现，不允许只依赖远端环境
```

---

## 第三周：设计模式应用

### Day 1-2: 创建型模式（控制对象创建）

#### 学习目标
- 在 AI 生成的 Go 代码中识别对象创建坏味道
- 掌握工厂函数、参数对象、Builder 的使用边界
- 理解单例和全局状态在 Go 中的风险

#### 核心知识点

**7.1 工厂函数**
```go
// 错误：业务流程中散落复杂创建逻辑。
order := Order{
	ID:        NewOrderID(),
	Customer: customerID,
	Status:    OrderStatusCreated,
	Items:     append([]OrderItem(nil), items...),
	CreatedAt: clock.Now(),
}
if len(order.Items) == 0 {
	return Order{}, ErrEmptyOrder
}
```

```go
// 正确：工厂函数集中保护业务不变量。
func NewOrder(customerID CustomerID, items []OrderItem, now time.Time) (Order, error) {
	if customerID == "" {
		return Order{}, ErrInvalidCustomer
	}
	if len(items) == 0 {
		return Order{}, ErrEmptyOrder
	}

	copiedItems := append([]OrderItem(nil), items...)
	return Order{
		ID:        NewOrderID(),
		Customer: customerID,
		Status:    OrderStatusCreated,
		Items:     copiedItems,
		CreatedAt: now,
	}, nil
}
```

**7.2 参数对象替代长参数列表**
```go
// 错误：参数过多，顺序易错。
func CreateOrder(customerID string, productID string, quantity int, couponCode string, address string, note string) error {
	return nil
}
```

```go
// 正确：使用命令对象表达意图。
type CreateOrderCommand struct {
	CustomerID CustomerID
	ProductID  ProductID
	Quantity   int
	CouponCode string
	Address    ShippingAddress
	Note       string
}

func (uc *CreateOrderUseCase) CreateOrder(ctx context.Context, cmd CreateOrderCommand) (OrderDTO, error) {
	if cmd.Quantity <= 0 {
		return OrderDTO{}, ErrInvalidQuantity
	}
	return uc.create(ctx, cmd)
}
```

**7.3 Builder 模式（仅在配置复杂时使用）**
```go
type ServerConfig struct {
	Host            string
	Port            int
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	ShutdownTimeout time.Duration
}

type ServerConfigBuilder struct {
	config ServerConfig
}

func NewServerConfigBuilder() *ServerConfigBuilder {
	return &ServerConfigBuilder{
		config: ServerConfig{
			Host:            "0.0.0.0",
			Port:            8080,
			ReadTimeout:     5 * time.Second,
			WriteTimeout:    5 * time.Second,
			ShutdownTimeout: 10 * time.Second,
		},
	}
}

func (b *ServerConfigBuilder) Port(port int) *ServerConfigBuilder {
	b.config.Port = port
	return b
}

func (b *ServerConfigBuilder) Build() (ServerConfig, error) {
	if b.config.Port <= 0 {
		return ServerConfig{}, ErrInvalidPort
	}
	return b.config, nil
}
```

**7.4 单例与全局状态限制**
```go
// 可接受：无状态纯函数或只读配置。
func NormalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}
```

```go
// 风险：可变全局状态导致测试污染和并发问题。
var currentUserID string

func SetCurrentUser(id string) {
	currentUserID = id
}
```

#### 实战任务
1. 将散落在用例中的复杂 `Order` 创建逻辑提取为 `NewOrder(...)`。
2. 将超过 4 个参数的函数重构为命令对象或配置对象。
3. 识别项目中的可变全局状态，并改为显式依赖注入。

#### AI 约束指令模板
```text
【Go 创建型模式约束】
- 复杂领域对象创建必须通过 NewXxx(...) 工厂函数保护不变量
- 函数参数超过 4 个时优先引入参数对象或命令对象
- Builder 仅用于复杂配置或可选项较多的对象，不作为默认模式
- 禁止在业务代码中使用可变全局状态保存请求上下文、用户状态、外部依赖
- 构造函数只做依赖赋值和轻量校验，不做网络请求、数据库访问、后台 goroutine 启动
```

---

### Day 3-4: 结构型与行为型模式

#### 学习目标
- 掌握用策略接口替换膨胀条件判断
- 理解责任链在校验、审批、过滤场景中的应用
- 使用装饰器包装横切关注点，而不是污染核心业务

#### 核心知识点

**8.1 策略模式**
```go
// 错误：条件判断持续膨胀。
func CalculateDiscount(customerType CustomerType, amount Money) Money {
	if customerType == CustomerTypeVIP {
		return amount.Multiply(80).Divide(100)
	}
	if customerType == CustomerTypeGold {
		return amount.Multiply(90).Divide(100)
	}
	if customerType == CustomerTypeSilver {
		return amount.Multiply(95).Divide(100)
	}
	return amount
}
```

```go
// 正确：策略接口 + map 注册。
type DiscountStrategy interface {
	Apply(amount Money) Money
}

type DiscountCalculator struct {
	strategies map[CustomerType]DiscountStrategy
	defaultOne DiscountStrategy
}

func NewDiscountCalculator(strategies map[CustomerType]DiscountStrategy, defaultOne DiscountStrategy) *DiscountCalculator {
	return &DiscountCalculator{strategies: strategies, defaultOne: defaultOne}
}

func (c *DiscountCalculator) Calculate(customerType CustomerType, amount Money) Money {
	strategy, ok := c.strategies[customerType]
	if !ok {
		strategy = c.defaultOne
	}
	return strategy.Apply(amount)
}
```

**8.2 责任链模式**
```go
type OrderValidator interface {
	Validate(ctx context.Context, order Order) error
}

type OrderValidationChain struct {
	validators []OrderValidator
}

func NewOrderValidationChain(validators ...OrderValidator) *OrderValidationChain {
	return &OrderValidationChain{validators: validators}
}

func (c *OrderValidationChain) Validate(ctx context.Context, order Order) error {
	for _, validator := range c.validators {
		if err := validator.Validate(ctx, order); err != nil {
			return err
		}
	}
	return nil
}
```

**8.3 装饰器模式**
```go
type OrderCreator interface {
	CreateOrder(ctx context.Context, cmd CreateOrderCommand) (OrderDTO, error)
}

type LoggingOrderCreator struct {
	next   OrderCreator
	logger Logger
}

func NewLoggingOrderCreator(next OrderCreator, logger Logger) *LoggingOrderCreator {
	return &LoggingOrderCreator{next: next, logger: logger}
}

func (d *LoggingOrderCreator) CreateOrder(ctx context.Context, cmd CreateOrderCommand) (OrderDTO, error) {
	d.logger.Info("creating order", "customerID", cmd.CustomerID)
	result, err := d.next.CreateOrder(ctx, cmd)
	if err != nil {
		d.logger.Error("create order failed", "error", err)
		return OrderDTO{}, err
	}
	d.logger.Info("order created", "orderID", result.OrderID)
	return result, nil
}
```

**8.4 函数式装饰器**
```go
type HandlerFunc func(ctx context.Context, cmd CreateOrderCommand) (OrderDTO, error)

func WithMetrics(next HandlerFunc, metrics Metrics) HandlerFunc {
	return func(ctx context.Context, cmd CreateOrderCommand) (OrderDTO, error) {
		started := time.Now()
		result, err := next(ctx, cmd)
		metrics.Observe("create_order_duration", time.Since(started))
		return result, err
	}
}
```

#### 实战任务
1. 将超过 3 个分支的折扣计算逻辑重构为策略模式。
2. 将订单校验逻辑重构为责任链，每个校验器只负责一条规则。
3. 为一个用例添加日志装饰器和指标装饰器，不修改核心实现。

#### AI 约束指令模板
```text
【Go 行为型模式约束】
- if/else 或 switch 分支超过 3 个且后续会扩展时，优先考虑策略模式或查表
- 多步骤校验、审批、过滤逻辑优先使用责任链，避免深层嵌套
- 日志、指标、重试、缓存、权限等横切关注点使用装饰器包装，禁止污染核心业务方法
- 状态流转必须集中在领域对象或状态机中，不允许散落在多个 handler 或 use case 中
- 模式服务于可读性和变化点，不为展示模式而引入多余抽象
```

---

### Day 5: 代码坏味道识别与重构

#### 学习目标
- 识别 Go 项目中的常见坏味道
- 掌握 Go 工具链和编辑器重构能力
- 能够将坏味道转化为 AI 约束指令

#### 核心坏味道清单

| 坏味道 | 识别特征 | 重构手法 | AI 约束 |
|--------|---------|---------|---------|
| 过长函数 | > 40 行或认知复杂度高 | 提取函数、拆分用例 | 函数保持单一职责 |
| 过大包 | 一个包承担多种业务含义 | 按业务边界拆包 | 包名表达业务能力 |
| 过多参数 | > 4 个参数 | 引入命令对象 / 配置对象 | 参数超 4 个必须解释 |
| 包级可变状态 | `var` 保存业务状态或依赖 | 构造函数注入 | 禁止隐藏依赖 |
| 错误被吞掉 | `_ = fn()` 或只打印不返回 | 返回并包装错误 | 关键错误必须处理 |
| 深层嵌套 | 多层 `if` 包裹主流程 | guard clause | 优先提前返回 |
| 基本类型偏执 | 用 `string` 表达金额、订单号 | 提取 Value Object | 业务概念显式建模 |
| 重复数据转换 | 多处手写 DTO 转换 | 提取 mapper/translator | 转换逻辑集中 |
| 领域模型贫血 | 所有规则都在用例里 | 移动行为到实体/值对象 | 领域对象保护不变量 |
| 过度抽象 | 每个实现都有无意义接口 | 删除接口或移动到消费者 | 只有变化点才抽象 |

#### 实战任务
1. 对一段 AI 生成的 Go 代码做坏味道标注。
2. 用 `gofmt`、`go vet`、`golangci-lint` 和人工审查结合完成重构。
3. 将重构前后的测试结果和复杂度变化写入代码审查说明。

#### AI 约束指令模板
```text
【Go 重构约束】
- 修改行为前必须有测试保护；无测试时先补最小关键路径测试
- 函数超过 40 行、分支超过 3 层、认知复杂度超过阈值时必须说明或重构
- 优先使用 guard clause 降低嵌套
- 禁止为每个结构体机械生成接口
- 业务概念不得长期停留在 string/int/float64，必要时提取值对象
- 重构必须保持 go test ./... 通过
```

---

## 第四周：防御性编程与 AI 协作规范

### Day 1-2: 防御性编程

#### 学习目标
- 消除隐式 nil 风险和错误吞噬
- 设计清晰的错误体系
- 掌握不可变值对象和防御性 copy

#### 核心知识点

**9.1 nil 与错误返回**
```go
// 错误：失败时返回零值但没有错误，调用方无法判断。
func FindUser(ctx context.Context, id UserID) User {
	user, _ := repo.FindByID(ctx, id)
	return user
}
```

```go
// 正确：显式返回错误。
var ErrUserNotFound = errors.New("user not found")

func FindUser(ctx context.Context, id UserID) (User, error) {
	if id == "" {
		return User{}, ErrInvalidUserID
	}

	user, err := repo.FindByID(ctx, id)
	if err != nil {
		return User{}, fmt.Errorf("find user %s: %w", id, err)
	}
	return user, nil
}
```

**9.2 `errors.Is` 与 `errors.As`**
```go
if errors.Is(err, ErrUserNotFound) {
	http.Error(w, "user not found", http.StatusNotFound)
	return
}

var validationErr ValidationError
if errors.As(err, &validationErr) {
	http.Error(w, validationErr.Message, http.StatusBadRequest)
	return
}
```

**9.3 typed error**
```go
type ValidationError struct {
	Field   string
	Message string
}

func (e ValidationError) Error() string {
	return e.Field + ": " + e.Message
}

func NewQuantity(quantity int) (Quantity, error) {
	if quantity <= 0 {
		return 0, ValidationError{
			Field:   "quantity",
			Message: "must be greater than zero",
		}
	}
	return Quantity(quantity), nil
}
```

**9.4 不可变值对象**
```go
type Money struct {
	amount   int64
	currency string
}

func NewMoney(amount int64, currency string) (Money, error) {
	if currency == "" {
		return Money{}, ErrInvalidCurrency
	}
	return Money{amount: amount, currency: currency}, nil
}

func (m Money) Amount() int64 {
	return m.amount
}

func (m Money) Currency() string {
	return m.currency
}

func (m Money) Add(other Money) (Money, error) {
	if m.currency != other.currency {
		return Money{}, ErrCurrencyMismatch
	}
	return Money{amount: m.amount + other.amount, currency: m.currency}, nil
}
```

**9.5 防御性 copy**
```go
type Order struct {
	items []OrderItem
}

func NewOrder(items []OrderItem) (Order, error) {
	if len(items) == 0 {
		return Order{}, ErrEmptyOrder
	}
	return Order{items: append([]OrderItem(nil), items...)}, nil
}

func (o Order) Items() []OrderItem {
	return append([]OrderItem(nil), o.items...)
}
```

#### 实战任务
1. 将一个返回零值且吞掉错误的函数改为显式错误返回。
2. 为领域错误设计 sentinel error 和 typed error，并用 `errors.Is` / `errors.As` 测试。
3. 将可变切片字段改为私有字段加防御性 copy。

#### AI 约束指令模板
```text
【Go 防御性编程约束】
- 业务失败必须通过 error 显式返回，禁止用 panic 表达可恢复业务错误
- 关键错误不得被 `_ =` 忽略；忽略错误必须说明原因
- 跨层返回错误时使用 fmt.Errorf("context: %w", err) 保留错误链
- 对外暴露错误判断时使用 sentinel error 或 typed error，并支持 errors.Is / errors.As
- Value Object 字段默认私有，通过构造函数保护不变量
- 暴露切片、map 等可变集合时必须做防御性 copy
```

---

### Day 3-4: 防腐层与外部系统隔离

#### 学习目标
- 设计防腐层（ACL）隔离第三方模型
- 隔离外部 SDK、HTTP client、消息队列和数据库模型
- 让外部系统变化不影响领域层

#### 核心知识点

**10.1 防腐层设计**
```go
// 错误：handler 直接透传外部支付模型。
type OrderHandler struct {
	payment *AlipayClient
}

func (h *OrderHandler) Pay(w http.ResponseWriter, r *http.Request) {
	var req AlipayRequest
	_ = json.NewDecoder(r.Body).Decode(&req)
	resp, _ := h.payment.Pay(r.Context(), req)
	_ = json.NewEncoder(w).Encode(resp)
}
```

```go
// 正确：领域端口 + 基础设施适配器 + translator。
type PaymentPort interface {
	Pay(ctx context.Context, request PaymentRequest) (PaymentResult, error)
}

type AlipayPaymentAdapter struct {
	client     AlipayClient
	translator AlipayTranslator
}

func NewAlipayPaymentAdapter(client AlipayClient, translator AlipayTranslator) *AlipayPaymentAdapter {
	return &AlipayPaymentAdapter{client: client, translator: translator}
}

func (a *AlipayPaymentAdapter) Pay(ctx context.Context, request PaymentRequest) (PaymentResult, error) {
	externalReq := a.translator.ToExternalRequest(request)
	externalResp, err := a.client.Pay(ctx, externalReq)
	if err != nil {
		return PaymentResult{}, fmt.Errorf("alipay pay: %w", err)
	}
	return a.translator.ToDomainResult(externalResp), nil
}
```

**10.2 外部 API 客户端隔离**
```go
type InventoryPort interface {
	CheckAvailability(ctx context.Context, productID ProductID, quantity int) (bool, error)
	Reserve(ctx context.Context, productID ProductID, quantity int) error
}

type InventoryHTTPAdapter struct {
	client  *http.Client
	baseURL string
}

func NewInventoryHTTPAdapter(client *http.Client, baseURL string) *InventoryHTTPAdapter {
	return &InventoryHTTPAdapter{client: client, baseURL: baseURL}
}

func (a *InventoryHTTPAdapter) CheckAvailability(ctx context.Context, productID ProductID, quantity int) (bool, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, a.baseURL+"/stock", nil)
	if err != nil {
		return false, fmt.Errorf("build inventory request: %w", err)
	}

	resp, err := a.client.Do(req)
	if err != nil {
		return false, fmt.Errorf("call inventory service: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 500 {
		return false, ErrInventoryUnavailable
	}

	return decodeAvailability(resp.Body)
}
```

**10.3 重试、超时与熔断边界**
```go
type RetryingInventoryPort struct {
	next    InventoryPort
	retries int
}

func NewRetryingInventoryPort(next InventoryPort, retries int) *RetryingInventoryPort {
	return &RetryingInventoryPort{next: next, retries: retries}
}

func (p *RetryingInventoryPort) Reserve(ctx context.Context, productID ProductID, quantity int) error {
	var lastErr error
	for attempt := 0; attempt <= p.retries; attempt++ {
		err := p.next.Reserve(ctx, productID, quantity)
		if err == nil {
			return nil
		}
		if errors.Is(err, context.Canceled) {
			return err
		}
		lastErr = err
	}
	return fmt.Errorf("reserve inventory after retries: %w", lastErr)
}
```

#### 实战任务
1. 将直接暴露第三方请求/响应模型的 handler 重构为领域 DTO + translator。
2. 为外部库存服务定义 `InventoryPort`，并实现 HTTP adapter。
3. 为外部调用增加超时、错误包装和契约测试。

#### AI 约束指令模板
```text
【Go 防腐层约束】
- 调用第三方 API 必须经过 adapter 和 translator，禁止在 handler 或 domain 中透传外部模型
- 外部系统接口变更不得影响 domain 和 application 包
- 第三方 SDK 只能出现在 infrastructure 包中
- HTTP client 必须设置超时，外部调用必须传递 context.Context
- 外部调用失败必须包装错误上下文，并在边界层转换为领域可理解的错误
- 重试、熔断、缓存等策略使用装饰器或 adapter 包装，不污染核心用例
```

---

### Day 5: 完整 System Prompt 编写与实战

#### 学习目标
- 整合所有约束为可执行的 System Prompt
- 在真实 Go 项目中验证约束效果
- 学会让 AI 先写测试、再写实现、最后自检

#### 核心知识点

**11.1 Prompt 结构**
```text
角色：你是资深 Go 工程师，遵循清晰包边界、显式错误处理和测试优先。
目标：实现用户需求，同时保持架构边界、测试覆盖和可维护性。
约束：列出不可违反的工程规则。
流程：先理解现有代码，再写测试，再实现，再运行验证。
输出：说明改动、测试结果、风险和后续建议。
```

**11.2 AI 协作流程**
1. 让 AI 先读取项目结构、`go.mod`、已有测试和 lint 配置。
2. 让 AI 明确将改哪些包，为什么这些包是正确边界。
3. 让 AI 先补或更新测试。
4. 让 AI 实现最小变更，避免顺手重构无关代码。
5. 让 AI 运行 `gofmt`、`go test ./...`、`golangci-lint run`。
6. 让 AI 输出剩余风险，不允许只说“已完成”。

#### 实战任务
1. 编写一份适用于当前 Go 项目的 System Prompt。
2. 让 AI 基于该 Prompt 实现一个小需求。
3. 用附录 Checklist 审查 AI 输出，记录违规项和修正指令。

---

## 附录 A：完整 AI 约束指令集（System Prompt）

```text
你是资深 Go 工程师。你必须在理解现有代码结构后再修改代码，遵循以下约束。

【项目结构】
- 默认使用 cmd/ 放程序入口，internal/ 放项目内部实现。
- handler 负责协议适配，application 负责用例编排，domain 负责业务规则，infrastructure 负责外部系统适配。
- 不创建无必要的 pkg/；只有稳定、可被外部复用的公共库才能放入 pkg/。

【依赖方向】
- handler -> application -> domain/repository port -> infrastructure adapter。
- domain 禁止依赖 HTTP、SQL、缓存、外部 SDK、日志框架。
- application 只能通过接口端口访问外部系统。
- infrastructure 实现端口，不得让外部模型泄漏到 domain。

【依赖注入】
- 禁止使用包级可变全局变量保存业务依赖。
- 所有依赖通过 NewXxx(...) 构造函数或函数参数显式传入。
- 接口由消费者定义，默认保持小接口。
- 不为无变化点的简单实现机械创建接口。

【测试】
- 新增 public 行为先写失败测试，再写实现。
- 单元测试使用 testing 包和 table-driven tests。
- handler 测试使用 httptest。
- 外部依赖用 fake、stub、mock 或契约测试替代真实访问。
- 每个核心业务方法覆盖正常路径、边界条件和错误路径。
- 修改完成后运行 go test ./...，关键并发或服务代码运行 go test ./... -race。

【错误处理】
- 业务失败通过 error 返回，禁止用 panic 表达可恢复业务错误。
- 关键错误不得忽略；忽略错误必须说明原因。
- 跨层错误使用 fmt.Errorf("context: %w", err) 保留错误链。
- 对外错误判断使用 errors.Is 或 errors.As。
- handler 负责将领域错误转换为协议状态码或响应。

【领域建模】
- 业务概念优先使用 Value Object，避免长期使用 string/int/float64 表达订单号、金额、数量。
- Value Object 字段默认私有，通过构造函数保护不变量。
- 切片、map 等可变集合对外暴露时必须防御性 copy。
- 领域对象负责维护自身不变量，不把所有规则堆到 application。

【设计模式】
- 复杂对象创建使用 NewXxx(...) 工厂函数。
- 参数超过 4 个时优先使用命令对象或配置对象。
- 分支超过 3 个且存在扩展趋势时，考虑策略模式、查表或责任链。
- 日志、指标、重试、缓存、权限等横切关注点使用装饰器或 adapter 包装。
- 不为展示模式引入多余抽象。

【防腐层】
- 第三方 API、SDK、数据库模型必须隔离在 infrastructure。
- 外部请求/响应模型不得透传到 handler、application、domain。
- 外部调用必须传递 context.Context，并设置超时。
- 重试、熔断和缓存策略不得污染核心业务逻辑。

【工具与验证】
- 所有 Go 文件必须通过 gofmt。
- 每次修改后运行 go test ./...。
- 项目配置了 golangci-lint 时必须运行 golangci-lint run。
- 使用 depguard 或自定义测试检查架构依赖边界。
- 最终回复必须说明改动内容、验证命令、失败项或剩余风险。
```

---

## 附录 B：Go 架构规则库

**规则 1：领域层纯净性**
```go
func TestDomainLayerHasNoForbiddenDependencies(t *testing.T) {
	out, err := exec.Command("go", "list", "-deps", "./internal/domain/...").Output()
	if err != nil {
		t.Fatalf("go list domain deps: %v", err)
	}

	forbidden := []string{
		"net/http",
		"database/sql",
		"github.com/gin-gonic/gin",
		"github.com/go-redis/redis",
		"github.com/aws/aws-sdk-go",
	}

	deps := string(out)
	for _, pkg := range forbidden {
		if strings.Contains(deps, pkg) {
			t.Fatalf("domain layer depends on forbidden package %s", pkg)
		}
	}
}
```

**规则 2：handler 不访问数据库**
```go
func TestHandlerLayerDoesNotDependOnSQL(t *testing.T) {
	out, err := exec.Command("go", "list", "-deps", "./internal/handler/...").Output()
	if err != nil {
		t.Fatalf("go list handler deps: %v", err)
	}

	if strings.Contains(string(out), "database/sql") {
		t.Fatalf("handler layer must not depend on database/sql")
	}
}
```

**规则 3：无循环依赖**
```bash
go list ./...
```

`go list` 一旦发现 import cycle 会直接失败，应作为 CI 必跑命令。

**规则 4：lint 规则配置**
```yaml
run:
  timeout: 5m
  tests: true

linters:
  enable:
    - govet
    - staticcheck
    - errcheck
    - ineffassign
    - revive
    - gosec
    - gocyclo
    - depguard

linters-settings:
  gocyclo:
    min-complexity: 12
  depguard:
    rules:
      domain:
        files:
          - "internal/domain/**/*.go"
        deny:
          - pkg: "net/http"
            desc: "domain must not depend on transport"
          - pkg: "database/sql"
            desc: "domain must not depend on persistence"
      handler:
        files:
          - "internal/handler/**/*.go"
        deny:
          - pkg: "database/sql"
            desc: "handler must call application use cases instead of database"
```

**规则 5：错误处理检查**
```text
- errcheck：检查未处理错误。
- staticcheck：检查无效代码、错误 API 使用和潜在 bug。
- gosec：检查常见安全问题。
- govet：检查格式化参数、copy lock 等问题。
- revive：检查命名、复杂度、包注释等风格问题。
```

---

## 附录 C：代码审查 Checklist

### 架构层面
- [ ] 是否存在 handler 直接访问数据库、缓存或第三方 SDK？
- [ ] domain 是否依赖 HTTP、SQL、日志框架或外部 SDK？
- [ ] application 是否只通过接口端口访问外部系统？
- [ ] infrastructure 是否泄漏外部模型到 domain？
- [ ] 是否存在 import cycle？
- [ ] 是否滥用 pkg/ 暴露内部实现？

### 接口与依赖
- [ ] 依赖是否通过 `NewXxx(...)` 构造函数或函数参数显式传入？
- [ ] 是否存在包级可变全局状态？
- [ ] 接口是否由消费者定义？
- [ ] 接口方法是否过多？
- [ ] 是否为无变化点的实现创建了无意义接口？

### 设计模式
- [ ] 复杂对象创建是否集中在工厂函数？
- [ ] 参数超过 4 个时是否使用命令对象或配置对象？
- [ ] 分支超过 3 个时是否考虑策略模式、查表或责任链？
- [ ] 日志、指标、重试、缓存等横切关注点是否使用装饰器或 adapter？
- [ ] 是否存在为模式而模式的过度抽象？

### 测试
- [ ] 新增 public 行为是否有测试？
- [ ] 是否使用 table-driven tests 覆盖多场景？
- [ ] 是否覆盖正常路径、边界条件和错误路径？
- [ ] handler 是否使用 httptest？
- [ ] 外部依赖是否用 fake、stub、mock 或契约测试隔离？
- [ ] 测试是否独立可重复？
- [ ] 是否运行 `go test ./...`？

### 错误处理
- [ ] 关键错误是否被处理或向上返回？
- [ ] 是否使用 `%w` 保留错误链？
- [ ] 调用方是否使用 `errors.Is` / `errors.As` 判断错误？
- [ ] handler 是否负责协议层错误转换？
- [ ] 是否错误地使用 panic 处理可恢复业务失败？

### 防御性
- [ ] Value Object 是否通过构造函数保护不变量？
- [ ] 业务概念是否避免长期停留在 string/int/float64？
- [ ] 切片、map 对外暴露时是否做防御性 copy？
- [ ] 外部调用是否传递 context.Context？
- [ ] HTTP client 是否设置超时？
- [ ] 第三方模型是否被防腐层隔离？

---

*文档版本: 1.0 | 适用语言: Go / Golang | 更新时间: 2026-05*
