// Command analysischeck 诊断「AI 分析」工作流依赖的数据与模型是否可用（不跑完整多智能体图）。
//
// 数据侧：tusharedb-go UnifiedClient（与 tools.StockTools 同源，StockSDK + 本地 lake/duckdb）。
// 模型侧：与 agents 相同的环境变量 OPENAI_API_KEY / OPENAI_BASE_URL / OPENAI_MODEL。
//
// 用法（在 backend 目录）:
//
//	go run ./cmd/analysischeck
//	go run ./cmd/analysischeck -symbol 000001.SZ
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/easyspace-ai/stock_api/pkg/tsdb"

	"github.com/easyspace-ai/yilimi/internal/analysis/agents/common"
	"github.com/easyspace-ai/yilimi/internal/appenv"
)

func main() {
	log.SetPrefix("[analysischeck] ")
	log.SetFlags(0)

	symbol := flag.String("symbol", "600519.SH", "ts_code，用于探测日线/基本面")
	flag.Parse()

	appenv.Init()
	dataDir := appenv.DataRootDir()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	fmt.Println("=== AIGoStock 分析依赖自检 ===")
	fmt.Printf("AI_DATA_DIR=%q\n", dataDir)
	fmt.Printf("标的=%s\n\n", *symbol)

	okAll := true

	// 1) LLM
	fmt.Println("--- 大模型（与分析师相同配置）---")
	if _, err := common.TryChatModel(ctx); err != nil {
		okAll = false
		fmt.Printf("FAIL  %v\n", err)
	} else {
		fmt.Println("PASS  TryChatModel（密钥与 BaseURL 可初始化客户端）")
	}

	// 2) DuckDB / StockSDK 数据客户端
	fmt.Println("\n--- 本地数据管道（UnifiedClient / DuckDB）---")
	client, err := tsdb.NewUnifiedClient(tsdb.UnifiedConfig{
		PrimaryDataSource: tsdb.DataSourceStockSDK,
		DataDir:           dataDir,
		CacheMode:         tsdb.CacheModeAuto,
		TushareToken:      os.Getenv("TUSHARE_TOKEN"),
	})
	if err != nil {
		okAll = false
		fmt.Printf("FAIL  NewUnifiedClient: %v\n", err)
		fmt.Println("      若报 DuckDB lock，请先停掉 backend / 其它 datainit / analysischeck。")
		os.Exit(1)
	}
	defer func() { _ = client.Close() }()

	end := time.Now()
	start := end.AddDate(0, 0, -120)
	sdt := start.Format("20060102")
	edt := end.Format("20060102")
	calStart := end.AddDate(0, -6, 0).Format("20060102")

	run := func(name string, fn func() error) {
		if e := fn(); e != nil {
			okAll = false
			fmt.Printf("FAIL  %s: %v\n", name, e)
			return
		}
		fmt.Printf("PASS  %s\n", name)
	}

	run("GetStockBasic", func() error {
		df, err := client.GetStockBasic(ctx, tsdb.StockBasicFilter{TSCode: *symbol})
		if err != nil {
			return err
		}
		if df == nil || len(df.Rows) == 0 {
			return fmt.Errorf("empty stock_basic")
		}
		return nil
	})

	run("GetTradeCalendar", func() error {
		df, err := client.GetTradeCalendar(ctx, tsdb.TradeCalendarFilter{StartDate: calStart, EndDate: edt})
		if err != nil {
			return err
		}
		if df == nil || len(df.Rows) == 0 {
			return fmt.Errorf("empty trade_cal")
		}
		return nil
	})

	run("GetStockDaily(QFQ)", func() error {
		df, err := client.GetStockDaily(ctx, *symbol, sdt, edt, tsdb.AdjustQFQ)
		if err != nil {
			return err
		}
		if df == nil || len(df.Rows) == 0 {
			return fmt.Errorf("empty daily (检查 lake 是否已 datainit)")
		}
		return nil
	})

	run("GetDailyBasic", func() error {
		df, err := client.GetDailyBasic(ctx, *symbol, sdt, edt)
		if err != nil {
			return err
		}
		if df == nil || len(df.Rows) == 0 {
			return fmt.Errorf("empty daily_basic（可选：补跑 datainit -skip-daily -skip-adj）")
		}
		return nil
	})

	fmt.Println("\n--- 说明 ---")
	fmt.Println("• 市场分析师 / 基本面分析师：使用上述 StockTools（工具失败会转成模型可读错误，不中断整次分析）。")
	fmt.Println("• 舆情 / 新闻 / 宏观 / 主力资金：当前无独立数据工具，结论主要依赖模型；缺少外部数据源时易偏主观。")
	fmt.Println("详细清单见仓库 docs/ai-analysis-data-and-apis.md")

	if !okAll {
		os.Exit(1)
	}
	fmt.Println("\n全部通过。")
}
