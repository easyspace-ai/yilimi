#!/usr/bin/env bash
# 本地分析链路快速检测（需 CGO + DuckDB；若 CC 未设则改用 Apple clang）。
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"
export CC="${CC:-/usr/bin/clang}"
export CGO_ENABLED=1
echo "== go test internal/analysis/datacollect =="
go test ./internal/analysis/datacollect/... -count=1 -v
echo "== analysischeck =="
go run ./cmd/analysischeck -symbol "${1:-600820.SH}"
