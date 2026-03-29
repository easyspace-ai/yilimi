#!/usr/bin/env bash
# AI 分析自动化闭环：L1（数据闸门）→ L2（RunTradingWorkflow）→ L3（e2e 机审）。
#
# 用法（在 backend 目录）:
#   ./scripts/analysis-e2e.sh
#   SYMBOL=000001.SZ TRADE_DATE=2025-03-01 ./scripts/analysis-e2e.sh
#
# CI / 无密钥环境：仅跑 L1（计划中的「非 E2E_FULL」快捷路径可用本变量）
#   E2E_LITE=1 ./scripts/analysis-e2e.sh
#   E2E_FULL=0 ./scripts/analysis-e2e.sh   # 与 E2E_LITE=1 等效
#
# L1 失败 → 常见行动：datainit 补 lake；停掉占用 DuckDB 的进程；检查 TDX / AIGOSTOCK_TDX_FALLBACK。
# L2 失败（超时 / job.failed）→ OPENAI_*、网络、或调大 AIGOSTOCK_ANALYSIS_TIMEOUT / -timeout。
# L3 失败（e2e-verdict.json）→ 提示词或 datacollect 注入为空；对照 e2e-events.jsonl 与 pool.data_gaps。
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"
export CC="${CC:-/usr/bin/clang}"
export CGO_ENABLED="${CGO_ENABLED:-1}"

SYMBOL="${SYMBOL:-600820.SH}"
TRADE_DATE="${TRADE_DATE:-}"
OUT_BASE="${OUT_BASE:-artifacts/analysis-e2e}"
STAMP="$(date +%Y%m%d-%H%M%S)"
ART_DIR="${ART_DIR:-$OUT_BASE/$STAMP}"
mkdir -p "$ART_DIR"

echo "== L1: analysischeck (无多智能体图) =="
go run ./cmd/analysischeck -symbol "$SYMBOL" | tee "$ART_DIR/l1-analysischeck.log"

if [[ "${E2E_LITE:-}" == "1" || "${E2E_FULL:-}" == "0" ]]; then
  echo "E2E_LITE=1 或 E2E_FULL=0 → 跳过 L2/L3（全链路需 LLM 与较长时间；完整闭环请勿设置二者）"
  echo "artifacts: $ART_DIR"
  exit 0
fi

echo ""
echo "== L2+L3: analysise2e (RunTradingWorkflow + gates) =="
OPTS=(-symbol "$SYMBOL" -out "$ART_DIR")
if [[ -n "$TRADE_DATE" ]]; then
  OPTS+=(-date "$TRADE_DATE")
fi
go run ./cmd/analysise2e "${OPTS[@]}"

echo "artifacts: $ART_DIR"
echo "  e2e-events.jsonl  e2e-verdict.json  result.json"
