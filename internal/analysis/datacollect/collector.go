// Package datacollect 对标 TradingAgents-AShare 的 DataCollector：并发预取行情/资讯/资金流等并注入各分析师。
//
// 集成测试：go test -tags integration -run TestCollectorLive ./internal/analysis/datacollect/ 且设置 LIVE_DATA=1、AI_DATA_DIR 指向有效数据目录。
// 通达信日线补洞：AIGOSTOCK_TDX_FALLBACK=1（需可连行情服务器）。
package datacollect

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"sync"

	"github.com/easyspace-ai/stock_api/pkg/realtimedata"
	"github.com/easyspace-ai/stock_api/pkg/tsdb"
	"github.com/easyspace-ai/tdx"
)

const (
	fetchLookbackDays = 365
	newsLookbackDays  = 90
)

// Collector 并行拉取分析所需数据。
type Collector struct {
	TSDB *tsdb.UnifiedClient
	RT   *realtimedata.Client
	TDX  DailyBarFetcher
}

// NewCollector 使用已初始化的客户端（tsdb 可与 tools 共享同一实例）。
func NewCollector(tc *tsdb.UnifiedClient, rt *realtimedata.Client, tdxFetch DailyBarFetcher) *Collector {
	return &Collector{TSDB: tc, RT: rt, TDX: tdxFetch}
}

// NewDefaultCollector 从数据目录构造 tsdb + realtimedata（用于测试或未 InitGlobalTools 场景）。
func NewDefaultCollector(dataDir string) (*Collector, error) {
	dataDir = strings.TrimSpace(dataDir)
	if dataDir == "" {
		dataDir = "./data"
	}
	tc, err := tsdb.NewUnifiedClient(tsdb.UnifiedConfig{
		PrimaryDataSource: tsdb.DataSourceStockSDK,
		DataDir:           dataDir,
		CacheMode:         tsdb.CacheModeAuto,
	})
	if err != nil {
		return nil, fmt.Errorf("tsdb: %w", err)
	}
	rtDir := filepath.Join(dataDir, "realtimedata_cache")
	rt, err := realtimedata.NewClient(realtimedata.Config{
		DataDir:       rtDir,
		EnableStorage: true,
		CacheMode:     realtimedata.CacheModeAuto,
	})
	if err != nil {
		return nil, fmt.Errorf("realtimedata: %w", err)
	}
	var tdxF DailyBarFetcher
	if TDXFallbackEnabled() {
		tdxF = &TDXBarFetcher{Dial: func() (*tdx.Client, error) { return tdx.DialDefault() }}
	}
	return NewCollector(tc, rt, tdxF), nil
}

// Collect 对标 Python DataCollector.collect：解析交易日并并行取数。
func (c *Collector) Collect(ctx context.Context, tsCode, tradeDateISO string) (*Pool, error) {
	if c == nil || c.TSDB == nil {
		return nil, fmt.Errorf("collector: nil tsdb")
	}
	tsCode = strings.TrimSpace(strings.ToUpper(tsCode))
	reqISO, err := NormDateISO(tradeDateISO)
	if err != nil {
		return nil, err
	}
	resISO, resYMD, err := ResolveTradeDate(ctx, c.TSDB, reqISO)
	if err != nil {
		resISO, resYMD = reqISO, ToYYYYMMDD(reqISO)
	}

	pool := &Pool{
		Meta: PoolMeta{
			RequestedTradeDateISO: reqISO,
			ResolvedTradeDateISO:  resISO,
			ResolvedYYYYMMDD:      resYMD,
			FetchLookbackDays:     fetchLookbackDays,
			NewsLookbackDays:      newsLookbackDays,
		},
		Errors:           make(map[string]string),
		FundamentalsNote: "本地数据源暂未提供合并报表三大表全文；以下基本面仅基于日频估值指标（PE、PB、市值等）与公开资讯摘要，深度财报拆解能力有限。",
	}

	startFetch, endFetch, err := PrevCalendarRangeYYYYMMDD(resISO, fetchLookbackDays)
	if err != nil {
		return nil, err
	}
	startNews, _, err := PrevCalendarRangeYYYYMMDD(resISO, newsLookbackDays)
	if err != nil {
		return nil, err
	}

	var wg sync.WaitGroup
	var mu sync.Mutex

	run := func(key string, fn func() error) {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := fn(); err != nil {
				mu.Lock()
				pool.Errors[key] = err.Error()
				mu.Unlock()
			}
		}()
	}

	run("stock_daily", func() error {
		df, err := c.TSDB.GetStockDaily(ctx, tsCode, startFetch, endFetch, tsdb.AdjustQFQ)
		if err != nil {
			return err
		}
		if df == nil || len(df.Rows) == 0 {
			if c.TDX != nil && TDXFallbackEnabled() {
				txt, tdxErr := c.TDX.FetchDailyBars(ctx, tsCode, 800)
				if tdxErr != nil || strings.TrimSpace(txt) == "" {
					return fmt.Errorf("empty daily and tdx: %v", tdxErr)
				}
				mu.Lock()
				pool.StockDataText = txt + "\n（注：以上为通达信补数，非 DuckDB/tsdb）"
				pool.Indicators = "（通达信补数未走统一指标管线，以下指标可能缺失）\n"
				mu.Unlock()
				return nil
			}
			return fmt.Errorf("empty stock daily rows")
		}
		mu.Lock()
		pool.StockDataText = formatDataFrame(df, 60)
		pool.Indicators = BuildIndicatorsBlock(df)
		mu.Unlock()
		return nil
	})

	run("stock_basic", func() error {
		df, err := c.TSDB.GetStockBasic(ctx, tsdb.StockBasicFilter{TSCode: tsCode})
		if err != nil {
			return err
		}
		mu.Lock()
		pool.StockBasic = formatDataFrame(df, 5)
		mu.Unlock()
		return nil
	})

	run("daily_basic", func() error {
		df, err := c.TSDB.GetDailyBasic(ctx, tsCode, startNews, endFetch)
		if err != nil {
			return err
		}
		mu.Lock()
		pool.DailyBasic = formatDataFrame(df, 30)
		mu.Unlock()
		return nil
	})

	if c.RT != nil {
		run("stock_news", func() error {
			items, err := c.RT.GetStockNews(ctx, tsCode, 20)
			if err != nil {
				return err
			}
			mu.Lock()
			pool.News = formatNewsList(items, 20)
			mu.Unlock()
			return nil
		})
		run("global_news", func() error {
			items, err := c.RT.GetNews(ctx, 30)
			if err != nil {
				return err
			}
			mu.Lock()
			pool.GlobalNews = formatNewsList(items, 30)
			mu.Unlock()
			return nil
		})
		run("sector_flow", func() error {
			items, err := c.RT.GetSectorMoneyFlow(ctx)
			if err != nil {
				return err
			}
			mu.Lock()
			pool.FundFlowBoard = formatSectorMoneyFlows(items, 40)
			mu.Unlock()
			return nil
		})
		run("stock_money", func() error {
			m, err := c.RT.GetMoneyFlow(ctx, tsCode)
			if err != nil {
				return err
			}
			mu.Lock()
			pool.FundFlowStock = formatMoneyFlow(m)
			mu.Unlock()
			return nil
		})
		run("lhb", func() error {
			items, err := c.RT.GetStockDragonTiger(ctx, tsCode)
			if err != nil {
				return err
			}
			mu.Lock()
			pool.LHB = formatDragonTigerDetails(items)
			mu.Unlock()
			return nil
		})
		run("hot_topics", func() error {
			items, err := c.RT.GetHotTopics(ctx)
			if err != nil {
				return err
			}
			mu.Lock()
			pool.HotTopics = formatHotTopics(items, 15)
			mu.Unlock()
			return nil
		})
	}

	wg.Wait()

	head := fmt.Sprintf("标的：%s\n用户请求基准日：%s\n解析用于数据的最近交易日：%s（%s）\n日线拉取区间(YYYYMMDD)：%s ~ %s\n\n",
		tsCode, reqISO, resISO, resYMD, startFetch, endFetch)
	if pool.StockDataText != "" && !strings.HasPrefix(pool.StockDataText, "（通达信") {
		pool.StockDataText = head + "【日线 OHLCV（前复权，节选）】\n" + pool.StockDataText
	} else if pool.StockDataText != "" {
		pool.StockDataText = head + pool.StockDataText
	}

	if strings.TrimSpace(pool.StockDataText) == "" {
		return pool, fmt.Errorf("no_ohlcv: 日线数据为空；errors=%v", pool.Errors)
	}

	return pool, nil
}

// DefaultRealtimeDataDir 与 Collector 一致的副目录名。
func DefaultRealtimeDataDir(dataDir string) string {
	return filepath.Join(dataDir, "realtimedata_cache")
}

// EnsureCollectorForWorkflow 供 server：优先复用已 Init 的全局 tsdb，并挂载 realtimedata。
func EnsureCollectorForWorkflow(dataDir string, globalTSDB *tsdb.UnifiedClient) (*Collector, error) {
	tc := globalTSDB
	var err error
	if tc == nil {
		tc, err = tsdb.NewUnifiedClient(tsdb.UnifiedConfig{
			PrimaryDataSource: tsdb.DataSourceStockSDK,
			DataDir:           dataDir,
			CacheMode:         tsdb.CacheModeAuto,
		})
		if err != nil {
			return nil, err
		}
	}
	rt, err := realtimedata.NewClient(realtimedata.Config{
		DataDir:       DefaultRealtimeDataDir(dataDir),
		EnableStorage: true,
		CacheMode:     realtimedata.CacheModeAuto,
	})
	if err != nil {
		return nil, err
	}
	var tdxF DailyBarFetcher
	if TDXFallbackEnabled() {
		tdxF = &TDXBarFetcher{Dial: func() (*tdx.Client, error) { return tdx.DialDefault() }}
	}
	return NewCollector(tc, rt, tdxF), nil
}
