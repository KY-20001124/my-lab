---
name: go-code-constraints
description: Use when working on Go service scaffolding, Gin/GORM code generation, layered architecture reviews, legacy Go refactors, handler-service-repository boundaries, DBHelper or gorm leakage, client/ACL isolation, Go TDD, or internal AI coding guardrails.
---

# Go Code Constraints

Use this skill to keep Go service code generation and refactoring aligned with the team's layered architecture rules. Start from observed repository shape, then choose one of three modes: scaffold, implement, or refactor/review.

Treat `/Users/yao.ke/go/src/middleware-ops-agent` as the near-target reference architecture when it is locally available. Use it as a pattern source only; never add it as a dependency.

## Workflow

1. Inspect the repo before changing code: read `go.mod`, top-level directories, `internal/`, entrypoints, Makefile/CI, and relevant imports.
2. Classify the task:
   - **Scaffold**: create a new Go service skeleton from `assets/go-service-template/`.
   - **Implement**: add features inside the existing layer model.
   - **Refactor/review**: find boundary leaks and propose or apply staged changes.
3. Load references as needed:
   - Read `references/go-constraints.md` for the compact rule set.
   - Read `references/middleware-ops-agent-reference.md` when choosing target layout or comparing a legacy project to the reference shape.
4. Keep dependency direction explicit:
   - HTTP entrypoints call handlers.
   - Handlers bind, validate, and map HTTP responses.
   - Managers coordinate multiple services.
   - Services own application/domain logic and depend on ports.
   - Storage/repository packages hide GORM/SQL details.
   - Client/ACL packages isolate external systems.
   - DI/composition roots may wire concrete implementations.
5. Add tests appropriate to the risk: table-driven unit tests by default, integration tests behind build tags or environment gates, and no `time.Sleep` in unit tests.
6. Run or recommend verification. Prefer existing project targets such as `make check`, `go test ./...`, `go vet ./...`, and this skill's `scripts/check_go_constraints.sh`.

## Hard Rules

- Do not put `gin.Context`, `echo.Context`, or `fiber.Ctx` in service/biz APIs.
- Do not expose `*gorm.DB`, `DBHelper`, `database/sql`, or SQL driver types above storage/repository.
- Do not import concrete client implementations from handler/service/biz; depend on interfaces and wire implementations in DI/main.
- Do not let handlers import storage/repository packages.
- Do not let service/biz import handlers, routers, or web middleware.
- Do not pass third-party SDK models into domain/service contracts unless they are boundary DTOs deliberately owned by a client/ACL layer.
- Keep interfaces small. Split ports that become broad "god interfaces".

## Scanner

Run the advisory scanner from this skill root:

```bash
bash scripts/check_go_constraints.sh /path/to/go/repo
```

The scanner is read-only. It reports likely boundary violations with rule codes and file paths; it is not a replacement for code review or project-specific architecture checks.

## Scaffold Template

For a new service, copy `assets/go-service-template/` and replace the placeholder module names and sample `Thing` feature with the real domain language. Keep the first feature small enough to demonstrate the full route -> handler -> manager/service -> storage path.

## Legacy Refactor Strategy

For existing projects, do not start with a directory-wide rewrite. Stabilize one boundary at a time:

1. Stop DB leakage: hide `*gorm.DB`/`DBHelper` behind storage interfaces.
2. Stop framework leakage: replace service/biz web contexts with `context.Context` and explicit request structs.
3. Stop implementation leakage: depend on client/storage ports, then wire concrete impls in DI/main.
4. Split large interfaces around caller needs.
5. Add characterization tests before changing unclear behavior.
