#!/usr/bin/env bash
set -euo pipefail

fail() {
  echo "[check-architecture] ERROR: $1" >&2
  exit 1
}

hits="$(rg -n --glob 'internal/api/handler/**/*.go' --glob '!**/*_test.go' 'internal/(storage|repository|repo)' . || true)"
if [[ -n "$hits" ]]; then
  echo "$hits" >&2
  fail "handlers must not import storage/repository"
fi

hits="$(rg -n --glob 'internal/service/**/*.go' --glob '!**/*_test.go' '(gin\.Context|\*gin\.Context|echo\.Context|fiber\.Ctx|\*gorm\.DB|DBHelper)' . || true)"
if [[ -n "$hits" ]]; then
  echo "$hits" >&2
  fail "service must not depend on web contexts or database implementation types"
fi

echo "[check-architecture] OK"
