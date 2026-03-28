//go:build integration

package datacollect

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/easyspace-ai/stock_api/pkg/realtimedata"
	"github.com/easyspace-ai/stock_api/pkg/tsdb"
	"github.com/easyspace-ai/yilimi/internal/appenv"
)

// LIVE_DATA=1 且 AI_DATA_DIR 已用 datainit 等公司级同步过时运行。
// 使用 CacheModeReadOnly，避免 Auto 模式触发「5492 只股票 daily_basic 全量同步」导致测试极慢。
func TestCollectorLive_600820(t *testing.T) {
	if os.Getenv("LIVE_DATA") != "1" {
		t.Skip("set LIVE_DATA=1 to run network/local data integration")
	}
	appenv.EnsureUnifiedDataDir()
	dir := appenv.DataRootDir()

	tc, err := tsdb.NewUnifiedClient(tsdb.UnifiedConfig{
		PrimaryDataSource: tsdb.DataSourceStockSDK,
		DataDir:           dir,
		CacheMode:         tsdb.CacheModeReadOnly,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tc.Close() }()

	rt, err := realtimedata.NewClient(realtimedata.Config{
		DataDir:       DefaultRealtimeDataDir(dir),
		EnableStorage: true,
		CacheMode:     realtimedata.CacheModeAuto,
	})
	if err != nil {
		t.Fatal(err)
	}

	col := NewCollector(tc, rt, nil)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	pool, err := col.Collect(ctx, "600820.SH", "2024-06-28")
	if err != nil {
		t.Skipf("collect failed (need lake under AI_DATA_DIR or non-readonly sync): %v", err)
	}
	body := strings.ToLower(pool.StockDataText + pool.Indicators)
	if !strings.Contains(body, "close") && !strings.Contains(body, "收") {
		t.Logf("stock text: %s", pool.StockDataText[:min(200, len(pool.StockDataText))])
		t.Fatal("expected OHLCV-like content")
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
