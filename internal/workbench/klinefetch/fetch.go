// Package klinefetch 聚合东财仓储与通达信 K 线，供 /api/v1/kline 与 /api/v1/analysis/market/kline 共用。
//
// 环境变量 AISTOCK_KLINE_SOURCE：
//   - auto（默认）：先东财，无数据或失败再试通达信
//   - eastmoney：仅东财
//   - tdx：优先 TDX，失败或未就绪则东财
package klinefetch

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/easyspace-ai/yilimi/internal/workbench/domain/stock"
	"github.com/easyspace-ai/yilimi/internal/workbench/klinecompat"
	"github.com/easyspace-ai/yilimi/internal/workbench/ports"
	"github.com/easyspace-ai/yilimi/internal/workbench/tdxapi"
)

// DailyBars 按 AISTOCK_KLINE_SOURCE 拉取日线区间数据（已与 klinecompat 区间语义对齐）。
func DailyBars(repo ports.StockRepository, rawSymbol string, startTime, endTime time.Time) ([]stock.KLineItem, error) {
	src := strings.ToLower(strings.TrimSpace(os.Getenv("AISTOCK_KLINE_SOURCE")))
	if src == "" {
		src = "auto"
	}

	tryEM := func() ([]stock.KLineItem, error) {
		if repo == nil {
			return nil, fmt.Errorf("stock repository unavailable")
		}
		return klinecompat.DailyBarsInRange(repo, rawSymbol, startTime, endTime)
	}
	tryTDX := func() ([]stock.KLineItem, error) {
		svc := tdxapi.ActiveService()
		if svc == nil {
			return nil, fmt.Errorf("tdx service not ready")
		}
		norm := klinecompat.NormalizeSymbol(rawSymbol)
		return svc.DailyKLineItemsForNormalizedSymbol(norm, startTime, endTime)
	}

	switch src {
	case "tdx":
		items, err := tryTDX()
		if err == nil && len(items) > 0 {
			return items, nil
		}
		return tryEM()
	case "auto":
		items, err := tryEM()
		if err == nil && len(items) > 0 {
			return items, nil
		}
		return tryTDX()
	case "eastmoney", "em":
		return tryEM()
	default:
		return tryEM()
	}
}
