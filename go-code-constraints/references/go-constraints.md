# Go Code Constraints

## Scope

These rules apply to Go 1.22+ service code generated or refactored by AI agents. They are strict for new code and should be applied incrementally to legacy code.

## Layering

Preferred dependency direction:

```text
entrypoint -> api/handler -> manager -> service/biz -> storage/repository
                                      -> client/acl
```

- `cmd/` or server entrypoints start the process and call DI/bootstrap code.
- `internal/api` owns route contracts, HTTP envelope helpers, and handler interfaces.
- `internal/api/handler` owns Gin/Echo/Fiber handlers.
- `internal/manager` coordinates multi-service workflows.
- `internal/service` or `internal/biz` owns application and domain logic.
- `internal/storage` or `internal/repository` owns persistence ports.
- `internal/storage/mysql` or `internal/repository/*` owns GORM/SQL implementation.
- `internal/client` owns outbound clients.
- `internal/infrastructure/acl` or client adapters translate third-party models into internal models.
- `internal/di` or `main.go` wires concrete implementations.

## Handler/API Rules

- Handlers bind request input, validate transport-level parameters, call one manager/service method, and map errors to HTTP responses.
- Handlers may use `gin.Context`, `echo.Context`, or `fiber.Ctx`.
- Handlers must not import storage/repository packages.
- Handlers must not contain database transactions, SQL/GORM calls, polling loops, or multi-service business workflows.
- Keep route registration in a small routing package or entrypoint.

## Service/Biz Rules

- Service/biz methods accept `context.Context` as the first parameter for operations that can block or call I/O.
- Service/biz must not depend on Web framework contexts.
- Service/biz must not hold or return `*gorm.DB`, `DBHelper`, `*sql.DB`, `sql.Tx`, or driver-specific types.
- Service/biz depends on storage/client interfaces, not concrete implementations.
- Cross-service orchestration belongs in manager/application coordinator packages rather than handlers.
- Business logic should use internal request/value types instead of raw third-party SDK structs.
- Public service methods validate important inputs and return typed or wrapped errors.

## Storage/Repository Rules

- Persistence implementations hide GORM/SQL details below the storage/repository boundary.
- Repository/store interfaces return domain/internal types and `error`, not GORM handles.
- Concrete GORM implementations may hold `*gorm.DB`.
- Transaction ownership should be explicit. Higher-level store/facade code may open a transaction and pass `tx *gorm.DB` to table-level helpers, but the transaction handle must not escape to service/biz.
- Prefer one table struct per table and local conversion helpers.
- Avoid broad generic DB helpers such as `DBHelper`, `baseDAO`, `gormRow`, `gormRows`, or `sqlRunner` in new code.

## Client and ACL Rules

- External systems must be reached through `internal/client` or an ACL adapter.
- Third-party SDK structs and errors should be translated before they reach service/biz contracts.
- External HTTP/RPC clients must set timeouts and should expose cancellation through `context.Context`.
- Add circuit breaking or equivalent resilience when an external dependency can affect request latency or availability.
- Do not import `internal/client/impl` from handler/service/biz; wire impls only in DI/main.

## Interface and DI Rules

- Define interfaces at the consuming boundary or shared port package.
- Keep interfaces small. As a guide, more than 5 methods usually means the port should be split.
- Add compile-time implementation checks for important ports:

```go
var _ service.SessionService = (*SessionService)(nil)
```

- Use constructor injection. Avoid business global variables and stateful singletons.
- DI/main/composition roots are allowed to depend on concrete implementations.

## Testing Rules

- Put unit tests next to code in `*_test.go`.
- Prefer table-driven tests for validation and branch-heavy logic.
- Mock or fake external clients and storage for unit tests.
- Repository integration tests should use SQLite/testcontainers/local env gates as appropriate.
- Do not use `time.Sleep` in unit tests; prefer fake clocks, channels, contexts, or condition-based waits.
- Cover normal paths, zero/nil values, empty collections, bounds, and error paths.
- Use `go test -race` for concurrency-sensitive code.

## Defensive Programming Rules

- Do not ignore errors. If an error is intentionally ignored, document why at the call site.
- Wrap errors with `%w` when callers need `errors.Is` or `errors.As`.
- Use sentinel/domain errors for expected business outcomes.
- Panic only at startup, construction-time invariant failures, or truly unrecoverable programmer errors.
- Return copies of slices/maps when exposing mutable internal state.
- Prefer immutable value objects for concepts such as money, IDs, addresses, and status values.

## Legacy Refactor Priorities

1. Prevent framework leakage from service/biz APIs.
2. Prevent GORM/DBHelper leakage above repository/storage.
3. Move concrete client implementation imports into DI/main.
4. Extract ports around one feature at a time.
5. Split large interfaces by caller use case.
6. Add characterization tests before changing high-risk legacy behavior.
