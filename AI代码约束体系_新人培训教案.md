# AI 代码约束体系 — 新人培训教案

> **目标**：让新人掌握与 AI Coding Agent 协作时的架构纪律，将软件工程原则转化为可执行的 AI 约束指令。
> **课时**：4 周（每周 5 天，每天 2 小时理论 + 2 小时实战）
> **产出**：能够独立编写 System Prompt 约束 AI，产出符合架构规范的代码。

---

## 第一周：分层架构与依赖管理

### Day 1-2: 分层架构核心原则

#### 学习目标
- 理解分层架构的物理边界意义
- 掌握单向依赖原则
- 能够识别和修复跨层调用

#### 核心知识点

**1.1 经典三层架构**
```
┌─────────────────────────────────────┐
│  Presentation Layer (Controller)     │  ← 处理 HTTP 请求/响应
│  职责：参数校验、DTO 转换、调用 Service  │
├─────────────────────────────────────┤
│  Business Layer (Service/Domain)    │  ← 核心业务逻辑
│  职责：业务流程编排、领域规则、事务控制  │
├─────────────────────────────────────┤
│  Data Access Layer (Repository)     │  ← 数据持久化
│  职责：数据库操作、ORM 映射、查询优化   │
└─────────────────────────────────────┘
         ↑ 严格单向依赖，禁止反向
```

**1.2 六边形架构（进阶）**
```
         ┌─────────────┐
         │   外部驱动   │  ← Controller / CLI / Message
         └──────┬──────┘
                │ 通过端口适配器
    ┌───────────▼───────────┐
    │    Application        │  ← 用例层（Use Case）
    │    （编排领域逻辑）    │
    ├───────────────────────┤
    │      Domain           │  ← 核心业务规则（零框架依赖）
    │  （Entity / ValueObj） │
    └───────────┬───────────┘
                │ 通过端口适配器
         ┌──────┴──────┐
         │   外部被驱动  │  ← Repository / API Client / Message Queue
         └─────────────┘
```

**1.3 依赖方向铁律**
```java
// ❌ 错误：Controller 直接调用 Mapper
@RestController
public class OrderController {
    @Autowired
    private OrderMapper orderMapper;  // 跨层调用！
}

// ✅ 正确：Controller → Service → Repository
@RestController
public class OrderController {
    private final OrderService orderService;  // 通过接口依赖

    public OrderController(OrderService orderService) {
        this.orderService = orderService;
    }
}
```

#### 实战任务
1. 审查一段 AI 生成的代码，找出所有跨层调用
2. 使用 ArchUnit 编写规则检查包依赖方向

#### AI 约束指令模板
```
【分层约束】
- 严格遵循 Controller → Service → Repository 单向调用链
- Controller 层禁止直接调用 Mapper/DAO
- Service 层禁止直接操作 HttpServletRequest/Response
- 每层只依赖下一层接口，不依赖实现
```

---

### Day 3-4: 依赖注入与接口设计

#### 学习目标
- 掌握构造器注入
- 理解接口隔离原则（ISP）
- 能够设计高内聚的接口

#### 核心知识点

**2.1 注入方式对比**

| 方式 | 可测试性 | 依赖显式性 | 推荐度 |
|------|---------|-----------|--------|
| 字段注入 (@Autowired) | 差 | 隐藏 | ❌ 禁止 |
| Setter 注入 | 中 | 半显式 | ⚠️ 可选 |
| 构造器注入 | 优 | 完全显式 | ✅ 强制 |

```java
// ❌ 字段注入 - 隐藏依赖，测试困难
@Service
public class OrderService {
    @Autowired
    private OrderRepository orderRepository;
    @Autowired
    private PaymentClient paymentClient;
}

// ✅ 构造器注入 - 依赖显式，不可变
@Service
public class OrderService {
    private final OrderRepository orderRepository;
    private final PaymentClient paymentClient;

    public OrderService(OrderRepository orderRepository, 
                        PaymentClient paymentClient) {
        this.orderRepository = orderRepository;
        this.paymentClient = paymentClient;
    }
}
```

**2.2 接口设计原则**
```java
// ❌ 胖接口 - 强迫实现不需要的方法
public interface Worker {
    void work();
    void eat();    // 机器人不需要 eat！
    void sleep();  // AI 不需要 sleep！
}

// ✅ 接口隔离 - 细粒度接口
public interface Workable {
    void work();
}

public interface Feedable {
    void eat();
}

public interface Sleeper {
    void sleep();
}
```

**2.3 依赖倒置实践**
```java
// 高层模块（Service）定义接口
public interface OrderRepository {
    Order findById(OrderId id);
    Order save(Order order);
    List<Order> findByStatus(OrderStatus status);
}

// 低层模块（Infrastructure）实现接口
@Repository
public class OrderRepositoryImpl implements OrderRepository {
    private final OrderJpaRepository jpaRepository;
    private final OrderMapper mapper;

    // 实现细节封装在基础设施层
}
```

#### 实战任务
1. 将一段使用字段注入的代码重构为构造器注入
2. 拆分一个胖接口为多个细粒度接口
3. 编写 ArchUnit 测试验证："Service 层类必须只有构造器注入"

#### AI 约束指令模板
```
【依赖注入约束】
- 禁止使用 @Autowired 字段注入，全部使用构造器注入
- Lombok 的 @RequiredArgsConstructor 优先于手写构造器
- 接口方法数量不超过 5 个，超过则拆分
- 高层模块定义接口，低层模块实现接口
- 禁止在领域层（Domain）使用 Spring 注解
```

---

### Day 5: ArchUnit 架构守护

#### 学习目标
- 使用 ArchUnit 将架构规则代码化
- 配置 CI 自动拦截架构违规

#### 核心知识点

**3.1 ArchUnit 基础规则**
```java
@AnalyzeClasses(packages = "com.example.myapp")
public class ArchitectureTest {

    // 规则1：分层依赖方向
    @ArchTest
    static final ArchRule layer_dependencies_are_respected = 
        layeredArchitecture()
            .layer("Controller").definedBy("..controller..")
            .layer("Service").definedBy("..service..")
            .layer("Repository").definedBy("..repository..")
            .whereLayer("Controller").mayNotBeAccessedByAnyLayer()
            .whereLayer("Service").mayOnlyBeAccessedByLayers("Controller")
            .whereLayer("Repository").mayOnlyBeAccessedByLayers("Service");

    // 规则2：禁止字段注入
    @ArchTest
    static final ArchRule no_field_injection = 
        noFields().should(beAnnotatedWith(Autowired.class));

    // 规则3：领域层零框架依赖
    @ArchTest
    static final ArchRule domain_independent = 
        noClasses().that()
            .resideInAPackage("..domain..")
            .should().dependOnClassesThat()
            .resideInAnyPackage("org.springframework..", "javax.persistence..");

    // 规则4：接口命名规范
    @ArchTest
    static final ArchRule interfaces_should_not_have_impl_suffix = 
        noClasses().that()
            .areInterfaces()
            .should().haveSimpleNameEndingWith("Impl");
}
```

**3.2 CI 集成配置**
```yaml
# .github/workflows/architecture-check.yml
name: Architecture Guard
on: [push, pull_request]
jobs:
  arch-test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - name: Run ArchUnit Tests
        run: ./mvnw test -Dtest=ArchitectureTest
```

#### 实战任务
1. 为项目编写 5 条 ArchUnit 规则
2. 提交 PR 触发 CI 架构检查

---

## 第二周：测试驱动开发（TDD）

### Day 1-2: TDD 基础与 RED-GREEN-REFACTOR

#### 学习目标
- 掌握 TDD 三定律
- 理解测试作为契约的意义
- 能够写出可测试的代码

#### 核心知识点

**4.1 TDD 三定律**
1. **先写测试**：在写生产代码之前，先写一个失败的单元测试
2. **只写刚好够的代码**：只写让测试通过的最少代码
3. **重构**：在测试通过后，清理代码，消除重复

**4.2 测试结构：Given-When-Then**
```java
@Test
@DisplayName("当库存充足时，下单应成功并扣减库存")
void shouldDeductStockWhenOrderCreated() {
    // Given: 准备上下文
    Product product = new Product("P001", "iPhone", 100);
    OrderRequest request = new OrderRequest("P001", 2);
    when(stockService.checkAvailability("P001", 2)).thenReturn(true);

    // When: 执行操作
    OrderResult result = orderService.createOrder(request);

    // Then: 验证结果
    assertThat(result.isSuccess()).isTrue();
    assertThat(result.getOrderId()).isNotNull();
    verify(stockService).deduct("P001", 2);
}
```

**4.3 可测试代码的特征**
```java
// ❌ 难以测试 - 硬编码依赖
public class OrderService {
    private StockService stockService = new StockService();  // 硬编码！
    private PaymentGateway paymentGateway = new AlipayGateway();  // 具体类！
}

// ✅ 易于测试 - 依赖注入 + 接口
public class OrderService {
    private final StockService stockService;
    private final PaymentGateway paymentGateway;

    public OrderService(StockService stockService, 
                       PaymentGateway paymentGateway) {
        this.stockService = stockService;
        this.paymentGateway = paymentGateway;
    }
}
```

#### 实战任务
1. 从需求文档出发，先写测试再写实现（计算器/购物车场景）
2. 识别并重构不可测试的代码

#### AI 约束指令模板
```
【TDD 约束】
- 每个 public 方法必须有对应的单元测试
- 测试必须先于实现编写（RED-GREEN-REFACTOR）
- 测试必须包含：正常路径、边界条件、异常路径
- 外部依赖必须 Mock，数据库使用内存版本
- 测试方法名使用 should_xxx_when_xxx 格式
```

---

### Day 3-4: 测试替身与边界测试

#### 学习目标
- 掌握 Stub / Mock / Spy / Fake 的区别和使用场景
- 编写全面的边界条件测试

#### 核心知识点

**5.1 测试替身类型**

| 类型 | 用途 | 示例 |
|------|------|------|
| **Dummy** | 填充参数，不被使用 | `new Object()` 占位 |
| **Stub** | 返回固定数据 | `when(repo.findById("1")).thenReturn(order)` |
| **Spy** | 包装真实对象，部分 Mock | `spy(new ArrayList<>())` |
| **Mock** | 验证交互行为 | `verify(service).sendEmail(any())` |
| **Fake** | 简化实现（内存数据库） | `FakeUserRepository` |

**5.2 边界条件测试清单**
```java
@Test
@DisplayName("边界条件全面覆盖")
void boundaryTests() {
    // Null 输入
    assertThrows(IllegalArgumentException.class, () -> 
        calculator.add(null, 1));

    // 空集合
    assertThat(statistics.average(Collections.emptyList())).isEqualTo(0.0);

    // 最大值/最小值
    assertThrows(ArithmeticException.class, () -> 
        calculator.add(Integer.MAX_VALUE, 1));

    // 负数
    assertThat(calculator.add(-5, -3)).isEqualTo(-8);

    // 零值
    assertThat(calculator.divide(0, 5)).isEqualTo(0);
    assertThrows(DivideByZeroException.class, () -> 
        calculator.divide(5, 0));

    // 空字符串
    assertThat(stringUtils.trim(null)).isNull();
    assertThat(stringUtils.trim("")).isEqualTo("");

    // 超长输入
    assertThrows(InputTooLongException.class, () -> 
        validator.validate("a".repeat(10001)));
}
```

**5.3 测试金字塔**
```
        /\
       /  \
      / E2E \      ← 少量（10%）验证完整流程
     /─────────\
    / Integration \  ← 中量（20%）验证组件协作
   /─────────────────\
  /    Unit Tests     \ ← 大量（70%）验证业务逻辑
 /─────────────────────\
```

#### 实战任务
1. 为一个 Service 方法编写完整的边界测试（至少 8 个用例）
2. 使用 @DataJpaTest 编写 Repository 集成测试

#### AI 约束指令模板
```
【测试完整性约束】
- 每个方法必须覆盖：正常输入、null、空集合、最大值、最小值、零值、负数、异常
- 外部 HTTP 调用使用 WireMock，数据库使用 @DataJpaTest
- 禁止在单元测试中使用 Thread.sleep
- 测试执行时间超过 1 秒必须标记为 @Tag("slow")
```

---

### Day 5: 契约测试与 CI 集成

#### 学习目标
- 理解微服务间契约测试
- 配置测试覆盖率门禁

#### 核心知识点

**6.1 PACT 契约测试**
```java
// Consumer 端定义契约
@Pact(consumer = "order-service", provider = "payment-service")
public RequestResponsePact paymentPact(PactDslWithProvider builder) {
    return builder
        .given("payment service is up")
        .uponReceiving("a request to process payment")
        .path("/api/v1/payments")
        .method("POST")
        .body(new PactDslJsonBody()
            .stringType("orderId", "ORD-123")
            .decimalType("amount", 99.99))
        .willRespondWith()
        .status(200)
        .body(new PactDslJsonBody()
            .stringType("transactionId", "TXN-456")
            .stringMatcher("status", "SUCCESS|FAILED", "SUCCESS"))
        .toPact();
}
```

**6.2 覆盖率门禁**
```yaml
# pom.xml 配置
<plugin>
    <groupId>org.jacoco</groupId>
    <artifactId>jacoco-maven-plugin</artifactId>
    <configuration>
        <rules>
            <rule>
                <element>BUNDLE</element>
                <limits>
                    <limit>
                        <counter>LINE</counter>
                        <value>COVEREDRATIO</value>
                        <minimum>0.80</minimum>
                    </limit>
                    <limit>
                        <counter>BRANCH</counter>
                        <value>COVEREDRATIO</value>
                        <minimum>0.70</minimum>
                    </limit>
                </limits>
            </rule>
        </rules>
    </configuration>
</plugin>
```

---

## 第三周：设计模式应用

### Day 1-2: 创建型模式（控制对象创建）

#### 学习目标
- 在 AI 生成代码中识别创建坏味道
- 掌握工厂、Builder、单例的正确使用场景

#### 核心知识点

**7.1 工厂模式**
```java
// ❌ 坏味道：业务逻辑中直接 new 复杂对象
Order order = new Order();
order.setId(UUID.randomUUID().toString());
order.setStatus(OrderStatus.CREATED);
order.setCreatedAt(LocalDateTime.now());
order.setItems(new ArrayList<>());
// ... 更多字段

// ✅ 工厂模式：封装创建逻辑
public class OrderFactory {
    public static Order createNewOrder(CustomerId customerId, 
                                        List<OrderItem> items) {
        return Order.builder()
            .id(OrderId.generate())
            .customerId(customerId)
            .status(OrderStatus.CREATED)
            .createdAt(LocalDateTime.now())
            .items(new ArrayList<>(items))
            .build();
    }
}

// 使用
Order order = OrderFactory.createNewOrder(customerId, items);
```

**7.2 Builder 模式（参数过多时强制使用）**
```java
// ❌ 参数过多，顺序易错
public Order(String id, String customerId, List<OrderItem> items, 
             BigDecimal totalAmount, OrderStatus status, 
             LocalDateTime createdAt, Address shippingAddress) {
    // 构造器参数超过 4 个！
}

// ✅ Builder 模式
Order order = Order.builder()
    .id("ORD-001")
    .customerId("CUST-123")
    .items(items)
    .totalAmount(new BigDecimal("199.99"))
    .status(OrderStatus.CREATED)
    .createdAt(LocalDateTime.now())
    .shippingAddress(address)
    .build();
```

**7.3 单例模式（严格限制场景）**
```java
// ✅ 无状态工具类可用单例
public class IdGenerator {
    private static final IdGenerator INSTANCE = new IdGenerator();
    private final AtomicLong counter = new AtomicLong(0);

    private IdGenerator() {}

    public static IdGenerator getInstance() {
        return INSTANCE;
    }

    public String generate() {
        return "ID-" + counter.incrementAndGet();
    }
}

// ❌ 禁止：状态类单例（线程安全问题）
public class OrderService {  // 单例但持有状态！
    private List<Order> cache = new ArrayList<>();  // 危险！
}
```

#### AI 约束指令模板
```
【创建型模式约束】
- 构造器参数超过 4 个必须使用 Builder 模式
- 复杂对象创建必须走工厂方法，禁止在业务逻辑中直接 new
- 单例仅限无状态工具类，禁止在单例中持有业务状态
- 禁止使用双重检查锁（DCL），使用枚举或静态内部类实现单例
```

---

### Day 3-4: 结构型与行为型模式

#### 学习目标
- 掌握 if-else 重构为策略/责任链
- 理解装饰器模式处理横切关注点

#### 核心知识点

**8.1 策略模式（if-else > 3 层必须重构）**
```java
// ❌ 坏味道：if-else 膨胀
public BigDecimal calculateDiscount(Order order) {
    if (order.getCustomerType() == CustomerType.VIP) {
        return order.getAmount().multiply(new BigDecimal("0.8"));
    } else if (order.getCustomerType() == CustomerType.GOLD) {
        return order.getAmount().multiply(new BigDecimal("0.9"));
    } else if (order.getCustomerType() == CustomerType.SILVER) {
        return order.getAmount().multiply(new BigDecimal("0.95"));
    } else {
        return order.getAmount();
    }
}

// ✅ 策略模式
public interface DiscountStrategy {
    BigDecimal applyDiscount(BigDecimal amount);
}

@Component
public class VipDiscountStrategy implements DiscountStrategy {
    @Override
    public BigDecimal applyDiscount(BigDecimal amount) {
        return amount.multiply(new BigDecimal("0.8"));
    }
}

@Component
public class DiscountContext {
    private final Map<CustomerType, DiscountStrategy> strategies;

    public DiscountContext(List<DiscountStrategy> strategyList) {
        this.strategies = strategyList.stream()
            .collect(Collectors.toMap(
                s -> s.getType(),  // 每个策略声明自己支持的类型
                Function.identity()
            ));
    }

    public BigDecimal calculate(Order order) {
        return strategies.getOrDefault(
            order.getCustomerType(),
            new NoDiscountStrategy()
        ).applyDiscount(order.getAmount());
    }
}
```

**8.2 责任链模式（多条件判断）**
```java
// ❌ 嵌套 if 难以维护
public void process(Order order) {
    if (order.getAmount().compareTo(new BigDecimal("10000")) > 0) {
        if (order.getCustomerType() == CustomerType.VIP) {
            if (order.getItems().size() > 10) {
                // 复杂审批逻辑
            }
        }
    }
}

// ✅ 责任链模式
public interface OrderApprover {
    void setNext(OrderApprover next);
    ApprovalResult approve(Order order);
}

@Component
public class AmountCheckApprover implements OrderApprover {
    private OrderApprover next;
    private static final BigDecimal THRESHOLD = new BigDecimal("10000");

    @Override
    public ApprovalResult approve(Order order) {
        if (order.getAmount().compareTo(THRESHOLD) <= 0) {
            return ApprovalResult.approved();
        }
        return next != null ? next.approve(order) : ApprovalResult.rejected("金额超限");
    }
}

@Component
public class ApprovalChain {
    private final OrderApprover head;

    public ApprovalChain(List<OrderApprover> approvers) {
        // 按优先级排序并链接
        this.head = chain(approvers);
    }
}
```

**8.3 装饰器模式（横切关注点）**
```java
// 基础接口
public interface OrderService {
    Order createOrder(OrderRequest request);
}

// 核心实现
@Service
public class OrderServiceImpl implements OrderService {
    @Override
    public Order createOrder(OrderRequest request) {
        // 纯业务逻辑
    }
}

// 日志装饰器（不修改原代码）
@Component
@Primary
public class LoggingOrderService implements OrderService {
    private final OrderService delegate;
    private final Logger logger;

    public LoggingOrderService(OrderService delegate, Logger logger) {
        this.delegate = delegate;
        this.logger = logger;
    }

    @Override
    public Order createOrder(OrderRequest request) {
        logger.info("Creating order: {}", request);
        Order result = delegate.createOrder(request);
        logger.info("Order created: {}", result.getId());
        return result;
    }
}

// 缓存装饰器
@Component
public class CachingOrderService implements OrderService {
    private final OrderService delegate;
    private final CacheManager cache;

    @Override
    public Order createOrder(OrderRequest request) {
        // 先查缓存，再调用 delegate
    }
}
```

#### AI 约束指令模板
```
【行为型模式约束】
- if-else 超过 3 层必须重构为策略模式或责任链模式
- switch 语句超过 2 个 case 必须考虑策略模式
- 横切关注点（日志、缓存、权限）必须使用装饰器模式，禁止修改原方法
- 状态流转使用状态模式，禁止在 Service 中写状态转换 if-else
```

---

### Day 5: 代码坏味道识别与重构

#### 学习目标
- 使用《重构》目录识别坏味道
- 掌握自动化重构工具

#### 核心坏味道清单

| 坏味道 | 识别特征 | 重构手法 | AI 约束 |
|--------|---------|---------|---------|
| **过长方法** | > 30 行 | 提取方法 | 方法不超过 30 行 |
| **过大类** | > 200 行 | 提取类 | 类不超过 200 行 |
| **过多参数** | > 4 个参数 | 引入参数对象 / Builder | 参数超 4 个用 Builder |
| **发散式变化** | 一个类因多种原因修改 | 拆分职责 | 单一职责原则 |
| **霰弹式修改** | 修改分散在多个类 | 搬移方法/内联类 | 内聚性检查 |
| **依恋情结** | 方法大量调用其他类 | 搬移方法 | 方法应操作自身数据 |
| **数据泥团** | 相同数据字段反复出现 | 提取 Value Object | 重复字段提取对象 |
| **基本类型偏执** | 使用 String/int 表示概念 | 提取 Value Object | 金额用 Money 类 |
| **重复代码** | 相同/相似代码多处 | 提取方法/父类 | DRY 原则 |
| **临时字段** | 某些实例变量仅部分方法使用 | 提取类 | 字段应在所有方法有意义 |

---

## 第四周：防御性编程与 AI 协作规范

### Day 1-2: 防御性编程

#### 学习目标
- 消除空指针异常
- 设计健壮的异常体系
- 掌握不可变性

#### 核心知识点

**9.1 Optional 与空值处理**
```java
// ❌ 返回 null
public User findById(String id) {
    return userRepository.findById(id);  // 可能返回 null
}

// ✅ 返回 Optional
public Optional<User> findById(String id) {
    return Optional.ofNullable(userRepository.findById(id));
}

// 调用方处理
User user = userService.findById(id)
    .orElseThrow(() -> new UserNotFoundException(id));

// 链式操作
String city = userService.findById(id)
    .flatMap(User::getAddress)
    .map(Address::getCity)
    .orElse("Unknown");
```

**9.2 异常分层体系**
```java
// 领域异常 - 业务规则违反
public class InsufficientStockException extends DomainException {
    public InsufficientStockException(ProductId productId, int requested, int available) {
        super(String.format("Product %s: requested %d, available %d", 
            productId, requested, available));
    }
}

// 应用异常 - 用例执行失败
public class OrderCreationFailedException extends ApplicationException {
    public OrderCreationFailedException(String reason, Throwable cause) {
        super("ORDER_CREATION_FAILED", reason, cause);
    }
}

// 基础设施异常 - 技术问题
public class DatabaseConnectionException extends InfrastructureException {
    // ...
}

// 全局异常处理器（仅在 Presentation 层）
@RestControllerAdvice
public class GlobalExceptionHandler {
    @ExceptionHandler(DomainException.class)
    public ResponseEntity<ErrorResponse> handleDomain(DomainException e) {
        return ResponseEntity.badRequest()
            .body(new ErrorResponse("BUSINESS_ERROR", e.getMessage()));
    }

    @ExceptionHandler(InfrastructureException.class)
    public ResponseEntity<ErrorResponse> handleInfra(InfrastructureException e) {
        // 记录详细日志，返回通用错误
        log.error("Infrastructure error", e);
        return ResponseEntity.status(503)
            .body(new ErrorResponse("SERVICE_UNAVAILABLE", "请稍后重试"));
    }
}
```

**9.3 不可变性**
```java
// ✅ 不可变 Value Object
@Value  // Lombok
public class Money {
    private final BigDecimal amount;
    private final Currency currency;

    public Money add(Money other) {
        if (!this.currency.equals(other.currency)) {
            throw new IllegalArgumentException("Currency mismatch");
        }
        return new Money(this.amount.add(other.amount), this.currency);
    }
}

// ✅ 不可变集合防御
public class Order {
    private final List<OrderItem> items;

    public Order(List<OrderItem> items) {
        this.items = Collections.unmodifiableList(new ArrayList<>(items));
    }

    public List<OrderItem> getItems() {
        return items;  // 已经是不可变的
    }
}
```

#### AI 约束指令模板
```
【防御性编程约束】
- 方法返回可能为空时必须使用 Optional，禁止返回 null
- public 方法入口必须使用 Assert/Preconditions 校验参数
- 异常必须分层：Domain / Application / Infrastructure
- 禁止在领域层抛出 HTTP 状态码异常
- Value Object 必须不可变（final 字段 + 无 setter）
- 集合返回前使用 Collections.unmodifiableList 包装
```

---

### Day 3-4: 防腐层与外部系统隔离

#### 学习目标
- 设计防腐层（ACL）
- 隔离第三方依赖

#### 核心知识点

**10.1 防腐层设计**
```java
// ❌ 直接透传外部模型
@RestController
public class OrderController {
    @PostMapping
    public ResponseEntity<AlipayResponse> pay(@RequestBody AlipayRequest request) {
        // 直接暴露第三方模型！
    }
}

// ✅ 防腐层转换
// 1. 定义领域接口（由领域层拥有）
public interface PaymentGateway {
    PaymentResult process(PaymentRequest request);
}

// 2. 基础设施层实现 + 转换
@Component
public class AlipayPaymentGateway implements PaymentGateway {
    private final AlipayClient alipayClient;
    private final AlipayTranslator translator;

    @Override
    public PaymentResult process(PaymentRequest request) {
        // 转换：领域对象 → 第三方对象
        AlipayRequest alipayRequest = translator.toAlipayRequest(request);

        // 调用第三方
        AlipayResponse response = alipayClient.pay(alipayRequest);

        // 转换：第三方对象 → 领域对象
        return translator.toPaymentResult(response);
    }
}

// 3. 转换器
@Component
public class AlipayTranslator {
    public AlipayRequest toAlipayRequest(PaymentRequest request) {
        return AlipayRequest.builder()
            .outTradeNo(request.getOrderId().getValue())
            .totalAmount(request.getAmount().getValue().toString())
            .subject(request.getDescription())
            .build();
    }

    public PaymentResult toPaymentResult(AlipayResponse response) {
        return PaymentResult.builder()
            .success(response.isSuccess())
            .transactionId(new TransactionId(response.getTradeNo()))
            .errorCode(response.getSubCode())
            .build();
    }
}
```

**10.2 外部 API 客户端隔离**
```java
// 领域层定义端口
public interface InventoryPort {
    boolean checkAvailability(ProductId productId, int quantity);
    void reserve(ProductId productId, int quantity);
}

// 基础设施层适配
@Component
public class InventoryServiceAdapter implements InventoryPort {
    private final InventoryServiceClient client;
    private final CircuitBreaker circuitBreaker;

    @Override
    public boolean checkAvailability(ProductId productId, int quantity) {
        return circuitBreaker.execute(() -> 
            client.checkStock(productId.getValue(), quantity)
        );
    }
}
```

#### AI 约束指令模板
```
【防腐层约束】
- 调用第三方 API 必须经过 ACL 转换，禁止透传外部模型
- 外部系统接口变更不得影响领域层代码
- 第三方客户端必须包装在基础设施层，领域层仅依赖接口
- 外部调用必须加熔断（CircuitBreaker）和重试策略
```

---

### Day 5: 完整 System Prompt 编写与实战

#### 学习目标
- 整合所有约束为可执行的 System Prompt
- 在真实项目中验证约束效果

---

## 附录 A：完整 AI 约束指令集（System Prompt）

见独立文档：《AI System Prompt 完整约束模板》

---

## 附录 B：ArchUnit 规则库

```java
public class CompleteArchitectureRules {

    // 1. 分层依赖
    @ArchTest
    static final ArchRule layer_dependencies = layeredArchitecture()
        .layer("Controller").definedBy("..controller..")
        .layer("Service").definedBy("..service..")
        .layer("Repository").definedBy("..repository..")
        .layer("Domain").definedBy("..domain..")
        .layer("Infrastructure").definedBy("..infrastructure..")
        .whereLayer("Controller").mayNotBeAccessedByAnyLayer()
        .whereLayer("Service").mayOnlyBeAccessedByLayers("Controller")
        .whereLayer("Repository").mayOnlyBeAccessedByLayers("Service")
        .whereLayer("Domain").mayOnlyBeAccessedByLayers("Service", "Controller")
        .whereLayer("Infrastructure").mayNotBeAccessedByAnyLayer();

    // 2. 命名规范
    @ArchTest
    static final ArchRule naming_conventions = CompositeArchRule.of(
        classes().that().resideInAPackage("..controller..")
            .should().haveSimpleNameEndingWith("Controller"),
        classes().that().resideInAPackage("..service..")
            .should().haveSimpleNameEndingWith("Service").orShould().haveSimpleNameEndingWith("ServiceImpl"),
        classes().that().resideInAPackage("..repository..")
            .should().haveSimpleNameEndingWith("Repository")
    );

    // 3. 禁止规则
    @ArchTest
    static final ArchRule forbidden_practices = CompositeArchRule.of(
        noClasses().should().beAnnotatedWith("org.springframework.stereotype.Service")
            .andShould().resideOutsideOfPackage("..service.."),
        noFields().should(beAnnotatedWith(Autowired.class)),
        noClasses().should().callMethodWhere(targetOwner(is(assignableFrom(System.class)))
            .and(targetNameMatching("(out|err)\.println")))
    );

    // 4. 领域层纯净性
    @ArchTest
    static final ArchRule domain_purity = noClasses()
        .that().resideInAPackage("..domain..")
        .should().dependOnClassesThat()
        .resideInAnyPackage(
            "org.springframework..",
            "javax.persistence..",
            "lombok.."
        );
}
```

---

## 附录 C：代码审查 Checklist

### 架构层面
- [ ] 是否存在跨层调用？
- [ ] 领域层是否依赖框架？
- [ ] 包依赖方向是否正确？
- [ ] 是否有循环依赖？

### 设计模式
- [ ] if-else 是否超过 3 层？
- [ ] 参数是否超过 4 个？
- [ ] 是否存在重复代码？
- [ ] 横切关注点是否使用装饰器？

### 测试
- [ ] 每个 public 方法是否有测试？
- [ ] 是否覆盖边界条件？
- [ ] 外部依赖是否 Mock？
- [ ] 测试是否独立可重复？

### 防御性
- [ ] 是否返回 Optional 替代 null？
- [ ] 参数是否有前置校验？
- [ ] 异常是否分层？
- [ ] Value Object 是否不可变？

---

*文档版本: 1.0 | 适用语言: Java / Kotlin | 更新时间: 2026-05*
