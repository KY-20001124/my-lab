# middleware-ops-agent Reference Shape

Use `/Users/yao.ke/go/src/middleware-ops-agent` as a local reference when available. It is a pattern source, not a dependency.

## Dependency Model

The reference project documents this layer flow:

```text
server/main
  -> internal/api + internal/api/handler
  -> internal/manager
  -> internal/service
  -> internal/storage + internal/client
```

Allowed shortcuts:

- `api/handler -> service` for simple single-service CRUD.
- `service -> client` for outbound ports.
- `job -> service` for scheduled work.
- `internal/di` can wire all concrete implementations.

Enforced forbidden imports in the reference:

- `internal/api/handler` must not import `internal/storage`.
- `internal/job` must not import `internal/api`.
- `internal/manager` must not import `internal/api` or `internal/storage`.
- `internal/service` must not import `internal/api/handler`.
- `internal/service` must not import `internal/manager`.
- `internal/api` contracts must not import `internal/service`.

The enforcement script is `scripts/check-architecture.sh`.

## Engineering Conventions

- `interface.go`: package contracts and signature-related imports only.
- `types.go`: DTOs, enums, aliases, and package contract types.
- `helpers.go`: small helpers only, no domain logic.
- Handlers stay thin: bind, validate, call manager/service, map response.
- Routes live in `internal/api/routes.go`.
- Domain/application logic lives in `internal/service`.
- Cross-service orchestration lives in `internal/manager`.
- Service/store methods pass `context.Context` first.
- Feature changes generally update service, manager if needed, API interface/handler, routes, DI, swagger, and checks.

## Storage/DAO Reference

The reference project's MySQL implementation lives in `internal/storage/mysql/`.

Preferred shape:

```go
type XxxTable struct {
    // gorm column mapping
}

type XxxDao interface {
    // deterministic table-level methods
}

type XxxDaoImpl struct {
    db        *gorm.DB
    tableName string
}

func NewXxxDao(db *gorm.DB) XxxDao
```

Rules:

- `storage.I*Store` interfaces are stable upward-facing ports.
- `NewXxxStore(...)` is the upper-layer entrypoint.
- One table usually has one `<table_name>_dao.go`.
- Cross-table orchestration belongs in `xxx_store.go`.
- DAO impls may hold `*gorm.DB`, but service/manager must not.
- Avoid `DBHelper`, `baseDAO`, `gormRow`, `gormRows`, and `gormExec` in new code.
- Default to GORM chain APIs rather than large SQL strings.
- Avoid joins by default; perform multiple queries and aggregate in memory unless a join is clearly justified.
- DAO files should not open transactions; transaction owners pass `tx *gorm.DB` into table-level helpers.

## Good Examples to Inspect

- `internal/service/session/service.go`: service depends on `storage.ISessionStore` and implements `service.ISessionService`.
- `internal/storage/interface.go`: upward-facing storage ports.
- `internal/storage/mysql/session_store.go`: GORM implementation hidden behind storage interface.
- `internal/di/container.go`: composition root wires concrete stores, services, managers, and handlers.
- `scripts/check-architecture.sh`: simple import-boundary guard.

## Known Imperfections

The reference is close to target but not perfect. Do not copy these patterns blindly:

- Some service interfaces are broad and should be split when adding new features.
- Some storage types/errors are re-exported upward to preserve handler boundaries; prefer dedicated service-level DTOs/errors in new code.
- Handler tests may instantiate storage/mysql for integration-style coverage; keep production handlers dependent on service/manager ports.
