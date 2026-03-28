// Command datainit 手动拉取 AIGoStock 分析所需的本地全量数据（Parquet + DuckDB 视图依赖）。
// 与 main / API 使用同一 DataDir 与数据源（默认 StockSDK）。
//
// 断点续传：daily / adj_factor / daily_basic 按自然月分块拉取，每成功一月会更新
// data/meta/checkpoints.json。中断后在同一 AI_DATA_DIR 下用相同 -start、-end 再执行即可从上次的下一天继续。
// 若要把 -start 改早补历史，需自行删掉 checkpoints 里对应 dataset 或清理 lake 重复分区，避免重复落盘。
//
// 完整性：StockSDK 源下默认「全市场任意一只失败则该月整段报错」，不会写 parquet、不会推进 checkpoint；
// 修好网络后重跑同一命令即可。若需旧版「尽量有数据就落库」，须在 stockdb Client 上设置 PermitPartialMarketFetch（见 stockdb stocksdk Config）。
//
// 用法示例（在 aigostock 目录下）：
//
//	go run ./cmd/datainit
//	go run ./cmd/datainit -core-only
//	go run ./cmd/datainit -start 20200101 -end 20250301
//	TUSHARE_TOKEN=xxx go run ./cmd/datainit -source tushare
//
// 数据目录仅来自 .env 的 AI_DATA_DIR（经 appenv.Init 加载）。
//
// DuckDB 锁：tusharedb.duckdb 同一时刻只允许一个可写连接。运行 datainit 前请停掉占用同一
// AI_DATA_DIR 的 backend、其它 datainit，或未退出的 go run；排查: lsof <AI_DATA_DIR>/duckdb/tusharedb.duckdb
package main

import (
	"context"
	"flag"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/easyspace-ai/stock_api/pkg/tsdb"

	"github.com/easyspace-ai/yilimi/internal/appenv"
)

func exitNewUnifiedClient(err error, dataDir string) {
	duckPath := filepath.Join(dataDir, "duckdb", "tusharedb.duckdb")
	log.Printf("new client: %v", err)
	low := strings.ToLower(err.Error())
	if strings.Contains(low, "lock") || strings.Contains(low, "conflicting") {
		log.Printf(
			"DuckDB 只允许一个进程以可写方式打开 %s。\n"+
				"请先停止占用该文件的进程（例如 backend、另一次 datainit、卡在后台的 go run），再重试。\n"+
				"排查: lsof %q",
			duckPath, duckPath,
		)
	}
	os.Exit(1)
}

func main() {
	log.SetPrefix("[datainit] ")
	log.SetFlags(log.LstdFlags)

	appenv.Init()

	start := flag.String("start", "20180101", "日线/复权/每日指标 起始日期 YYYYMMDD")
	end := flag.String("end", "", "结束日期 YYYYMMDD，默认今天")
	coreOnly := flag.Bool("core-only", false, "仅同步 stock_basic + trade_cal（较快，可修复缺失 v_stock_basic）")
	skipDaily := flag.Bool("skip-daily", false, "跳过日线全量")
	skipAdjFactor := flag.Bool("skip-adj", false, "跳过复权因子")
	skipDailyBasic := flag.Bool("skip-daily-basic", false, "跳过 daily_basic")
	source := flag.String("source", "stocksdk", "数据源：stocksdk | tushare（tushare 需环境变量 TUSHARE_TOKEN）")
	flag.Parse()

	dir := appenv.DataRootDir()

	endDate := strings.TrimSpace(*end)
	if endDate == "" {
		endDate = time.Now().Format("20060102")
	}

	cfg := tsdb.UnifiedConfig{
		DataDir:      dir,
		CacheMode:    tsdb.CacheModeAuto,
		TushareToken: os.Getenv("TUSHARE_TOKEN"),
	}
	switch strings.ToLower(strings.TrimSpace(*source)) {
	case "stocksdk", "":
		cfg.PrimaryDataSource = tsdb.DataSourceStockSDK
	case "tushare":
		cfg.PrimaryDataSource = tsdb.DataSourceTushare
	default:
		log.Fatalf("unknown -source %q (use stocksdk or tushare)", *source)
	}

	if err := os.MkdirAll(dir, 0o755); err != nil {
		log.Fatalf("mkdir data dir: %v", err)
	}

	client, err := tsdb.NewUnifiedClient(cfg)
	if err != nil {
		exitNewUnifiedClient(err, dir)
	}
	defer func() { _ = client.Close() }()

	ctx := context.Background()

	log.Println("=== SyncCore: trade_cal + stock_basic（上市 L）===")
	if err := client.SyncCore(ctx); err != nil {
		log.Fatalf("SyncCore: %v", err)
	}
	if *coreOnly {
		log.Println("core-only：已结束（未拉日线/复权/daily_basic）。")
		return
	}

	if !*skipDaily {
		if d, ok := client.GetLastSyncDate("daily"); ok {
			log.Printf("checkpoint daily last=%s（将自动续拉至 %s）", d, endDate)
		}
		log.Printf("=== SyncDailyRange: %s ~ %s（全市场，按月 checkpoint，较慢）===", *start, endDate)
		if err := client.SyncDailyRange(ctx, *start, endDate); err != nil {
			log.Fatalf("SyncDailyRange: %v", err)
		}
	} else {
		log.Println("跳过 SyncDailyRange（-skip-daily）")
	}

	if !*skipAdjFactor {
		if d, ok := client.GetLastSyncDate("adj_factor"); ok {
			log.Printf("checkpoint adj_factor last=%s", d)
		}
		log.Printf("=== SyncAdjFactorRange: %s ~ %s（按月 checkpoint）===", *start, endDate)
		if err := client.SyncAdjFactorRange(ctx, *start, endDate); err != nil {
			log.Fatalf("SyncAdjFactorRange: %v", err)
		}
	} else {
		log.Println("跳过 SyncAdjFactorRange（-skip-adj）")
	}

	if !*skipDailyBasic {
		if d, ok := client.GetLastSyncDate("daily_basic"); ok {
			log.Printf("checkpoint daily_basic last=%s", d)
		}
		log.Printf("=== SyncDailyBasicRange: %s ~ %s（全市场，按月 checkpoint，较慢）===", *start, endDate)
		if err := client.SyncDailyBasicRange(ctx, *start, endDate); err != nil {
			log.Fatalf("SyncDailyBasicRange: %v", err)
		}
	} else {
		log.Println("跳过 SyncDailyBasicRange（-skip-daily-basic）")
	}

	log.Printf("=== 完成。请用同一 AI_DATA_DIR（当前 %s）启动 AIGoStock。===\n", dir)
	for _, ds := range []string{"daily", "adj_factor", "daily_basic"} {
		if d, ok := client.GetLastSyncDate(ds); ok {
			log.Printf("checkpoint %s last=%s", ds, d)
		}
	}
}
