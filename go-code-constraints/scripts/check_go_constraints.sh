#!/usr/bin/env bash
set -euo pipefail

ROOT="${1:-.}"
if [[ ! -d "$ROOT" ]]; then
  echo "usage: $0 /path/to/go/repo" >&2
  exit 2
fi

if ! command -v rg >/dev/null 2>&1; then
  echo "ERROR: rg is required" >&2
  exit 2
fi

cd "$ROOT"

if [[ ! -f go.mod ]]; then
  echo "ERROR: $ROOT does not look like a Go module (go.mod missing)" >&2
  exit 2
fi

violations=0

report() {
  local rule="$1"
  local text="$2"
  if [[ -n "$text" ]]; then
    violations=$((violations + 1))
    echo
    echo "[$rule]"
    echo "$text"
  fi
}

scan() {
  local globs="$1"
  local pattern="$2"
  local extra_args=()
  IFS=' ' read -r -a extra_args <<< "$globs"
  rg -n "${extra_args[@]}" --glob '!**/*_test.go' "$pattern" . || true
}

scan_tests() {
  local pattern="$1"
  rg -n --glob '**/*_test.go' "$pattern" . || true
}

handler_storage="$(scan "--glob internal/**/handler/**/*.go --glob internal/handler/**/*.go --glob internal/api/handler/**/*.go" '".*internal/(storage|repo|repository)(/|")')"
report "GO-LAYER-001 handler-must-not-import-storage" "$handler_storage"

service_web_context="$(scan "--glob internal/**/service/**/*.go --glob internal/service/**/*.go --glob internal/**/biz/**/*.go --glob internal/biz/**/*.go" '(gin\.Context|\*gin\.Context|echo\.Context|fiber\.Ctx)')"
report "GO-LAYER-002 service-biz-must-not-use-web-context" "$service_web_context"

service_db_leak="$(scan "--glob internal/**/service/**/*.go --glob internal/service/**/*.go --glob internal/**/biz/**/*.go --glob internal/biz/**/*.go" '(\*gorm\.DB|\*sql\.DB|\bsql\.DB\b|DBHelper|database/sql|gorm\.io/)')"
report "GO-LAYER-003 service-biz-must-not-expose-db" "$service_db_leak"

service_impl_import="$(scan "--glob internal/**/service/**/*.go --glob internal/service/**/*.go --glob internal/**/biz/**/*.go --glob internal/biz/**/*.go --glob internal/**/handler/**/*.go --glob internal/handler/**/*.go --glob internal/api/handler/**/*.go" '".*internal/client/impl(/|")')"
report "GO-LAYER-004 depend-on-client-port-not-impl" "$service_impl_import"

service_handler_import="$(scan "--glob internal/**/service/**/*.go --glob internal/service/**/*.go --glob internal/**/biz/**/*.go --glob internal/biz/**/*.go" '".*internal/(api/handler|handler|router|middleware)(/|")')"
report "GO-LAYER-005 service-biz-must-not-import-entry-layer" "$service_handler_import"

repo_gorm_return="$(scan "--glob internal/**/repo/**/*.go --glob internal/repo/**/*.go --glob internal/**/repository/**/*.go --glob internal/repository/**/*.go --glob internal/**/storage/**/*.go --glob internal/storage/**/*.go" '^[[:space:]]*[A-Za-z0-9_]+\([^)]*\)[[:space:]]+\*gorm\.DB|[A-Za-z0-9_]+[[:space:]]*\([^)]*\)[[:space:]]*\*gorm\.DB')"
report "GO-LAYER-006 repository-interface-must-not-return-gorm" "$repo_gorm_return"

sleep_tests="$(scan_tests 'time\.Sleep\(')"
report "GO-TEST-001 unit-tests-should-not-sleep" "$sleep_tests"

if [[ "$violations" -eq 0 ]]; then
  echo "[check-go-constraints] OK: no advisory violations found"
else
  echo
  echo "[check-go-constraints] Found $violations rule group(s) with advisory violations"
fi

exit 0
