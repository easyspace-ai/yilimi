// 统一后端入口：组装限界上下文并暴露 /api/v1/*。
//
// 代码布局见 internal/doc.go：internal/analysis 与 internal/workbench 两个限界上下文（标准 Go 工程 + DDD 分层）。
package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	mdlib "github.com/easyspace-ai/stock_api/pkg/marketdata"
	stocklib "github.com/easyspace-ai/stock_api/pkg/stockapi"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"

	analysisapi "github.com/easyspace-ai/yilimi/internal/analysis/interfaces/http"
	"github.com/easyspace-ai/yilimi/internal/analysis/tools"
	"github.com/easyspace-ai/yilimi/internal/appenv"
	"github.com/easyspace-ai/yilimi/internal/workbench/application/ai"
	"github.com/easyspace-ai/yilimi/internal/workbench/auth"
	"github.com/easyspace-ai/yilimi/internal/workbench/filesystem"
	"github.com/easyspace-ai/yilimi/internal/workbench/infrastructure/cache"
	"github.com/easyspace-ai/yilimi/internal/workbench/infrastructure/clients"
	"github.com/easyspace-ai/yilimi/internal/workbench/infrastructure/marketadapter"
	"github.com/easyspace-ai/yilimi/internal/workbench/infrastructure/persistence/sqlite"
	"github.com/easyspace-ai/yilimi/internal/workbench/infrastructure/persistence/stockdb"
	wbhttp "github.com/easyspace-ai/yilimi/internal/workbench/interfaces/http"
	"github.com/easyspace-ai/yilimi/internal/workbench/plugins"
	"github.com/easyspace-ai/yilimi/internal/workbench/proxy"
	"github.com/easyspace-ai/yilimi/internal/workbench/tags"
	"github.com/easyspace-ai/yilimi/internal/workbench/tdxapi"
	"github.com/easyspace-ai/yilimi/internal/workbench/ws"
)

func main() {
	appenv.Init()
	ctx := context.Background()

	root := appenv.WorkspaceRoot()
	fmt.Println("root", root)

	if err := os.MkdirAll(root, 0o755); err != nil {
		log.Fatalf("failed to ensure data root: %v", err)
	}
	docPath := filepath.Join(root, "data")
	stockDbDir := appenv.StockDatabaseDir()
	log.Printf("AI_DATA_DIR=%s", stockDbDir)
	if err := os.MkdirAll(stockDbDir, 0o755); err != nil {
		log.Fatalf("stock database dir (%s): %v", stockDbDir, err)
	}

	// ========== AI 分析：数据工具（Parquet / DuckDB 等）==========
	analysisDataDir := appenv.DataRootDir()
	if err := os.MkdirAll(analysisDataDir, 0o755); err != nil {
		log.Fatalf("analysis data dir: %v", err)
	}
	if err := tools.InitGlobalTools(analysisDataDir); err != nil {
		log.Fatalf("init analysis tools: %v", err)
	}

	analysisHandler, analysisReports, err := analysisapi.BootstrapAnalysis(ctx)
	if err != nil {
		log.Fatalf("bootstrap analysis: %v", err)
	}
	defer func() {
		if analysisReports != nil {
			_ = analysisReports.Close()
		}
	}()

	// ========== 工作区 / 股票服务（原 server）==========
	fsService, err := filesystem.NewService(docPath)
	if err != nil {
		log.Fatalf("failed to init filesystem service: %v", err)
	}

	indexer := tags.NewIndexer(root)
	if err := indexer.ReindexAll(); err != nil {
		log.Printf("tag indexer initial build failed: %v", err)
	}

	hub := ws.NewHub()
	go hub.Run()

	pluginsDir := filepath.Join(root, ".plugins")
	pluginDbPath := filepath.Join(root, ".plugins.db")
	pluginService, err := plugins.NewService(pluginDbPath, pluginsDir)
	if err != nil {
		log.Printf("plugin system disabled: %v", err)
	}
	defer func() {
		if pluginService != nil {
			_ = pluginService.Close()
		}
	}()

	config := clients.DefaultConfig()
	if token := os.Getenv("TUSHARE_TOKEN"); token != "" {
		config.TushareToken = token
	}

	cacheLayer, err := cache.NewMultiLayerCache(stockDbDir)
	if err != nil {
		log.Printf("cache layer disabled: %v", err)
	}
	defer func() {
		if cacheLayer != nil {
			_ = cacheLayer.Close()
		}
	}()

	log.Printf("stock.db path: %s", filepath.Join(stockDbDir, "stock.db"))
	stockDB, err := stockdb.InitStockDatabase(stockDbDir)
	if err != nil {
		log.Printf("stock database disabled: %v", err)
	}

	stockRepo := sqlite.NewStockRepository(stockDB, cacheLayer, config)
	analysisHandler.SetStockRepository(stockRepo)
	mdCfg := mdlib.DefaultConfig()
	if config.CrawlTimeOut > 0 {
		mdCfg.Timeout = time.Duration(config.CrawlTimeOut) * time.Second
	}
	marketRepo := marketadapter.NewRepository(mdlib.NewClient(mdCfg))
	aiService := ai.NewAIService()

	stockHandler := wbhttp.NewStockHandler(stockRepo)
	stockV1Client, err := stocklib.NewClientWithConfig(buildStockAPIConfig())
	if err != nil {
		log.Fatalf("failed to init stockapi client: %v", err)
	}
	stockV1Handler := wbhttp.NewStockV1Handler(stockV1Client)
	marketHandler := wbhttp.NewMarketHandler(marketRepo)
	legacyAIHandler := wbhttp.NewAIHandler(aiService)

	r := gin.Default()

	r.Use(cors.New(cors.Config{
		AllowAllOrigins:  true,
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS", "HEAD"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization", "X-Request-Id", "X-User-Id", "X-User-Email", "X-User-Name", "X-User-Role"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: false,
		MaxAge:           12 * time.Hour,
	}))

	// 与 stock.db 共用 AI_DATA_DIR，避免打包部署只配数据根时落到默认 WorkspaceRoot 下的空 auth.db 导致登录 401。
	authDBPath := filepath.Join(stockDbDir, "auth.db")
	if p := strings.TrimSpace(os.Getenv("AUTH_DB_PATH")); p != "" {
		authDBPath = filepath.Clean(p)
	}
	if err := os.MkdirAll(filepath.Dir(authDBPath), 0o755); err != nil {
		log.Fatalf("auth database dir (%s): %v", filepath.Dir(authDBPath), err)
	}
	log.Printf("auth.db path: %s", authDBPath)
	jwtSecret := strings.TrimSpace(os.Getenv("AUTH_JWT_SECRET"))
	if jwtSecret == "" {
		jwtSecret = "dev-insecure-jwt-secret-change-me"
		log.Println("WARNING: AUTH_JWT_SECRET is empty; set it for production")
	}
	authSvc, err := auth.NewService(authDBPath, jwtSecret)
	if err != nil {
		log.Fatalf("auth: %v", err)
	}
	defer func() { _ = authSvc.Close() }()

	// ---------- 对外统一版本化 API：/api/v1/* ----------
	v1 := r.Group("/api/v1")
	v1.GET("/health", wbhttp.Health)
	v1.GET("/vip-status", wbhttp.GetVipStatus)
	auth.RegisterRoutes(v1.Group("/auth"), authSvc)

	wbhttp.RegisterWorkspaceRoutes(v1.Group("/workspace"), fsService, hub, indexer, root)
	stockHandler.RegisterRoutes(v1)
	marketHandler.RegisterRoutes(v1)
	legacyAIHandler.RegisterRoutes(v1)
	// tusharedb StockSDK：实时/历史行情与分时（供前端 market-data）
	stockV1Handler.RegisterRoutes(v1.Group("/market-data"))
	if pluginService != nil {
		plugins.RegisterPluginRoutes(v1, pluginService)
	}

	v1BacktestPrefix := "/api/v1/backtest"
	r.Any(v1BacktestPrefix, proxy.BacktestReverseProxyHandlerWithPrefix(v1BacktestPrefix))
	r.Any(v1BacktestPrefix+"/*path", proxy.BacktestReverseProxyHandlerWithPrefix(v1BacktestPrefix))

	// 前端唯一定义：同源 /daily-api → 本进程转发至 DailyAPI（见 DAILYAPI_PORT / DAILYAPI_URL）。
	dailyAPIPrefix := "/daily-api"
	r.Any(dailyAPIPrefix, proxy.DailyAPIReverseProxyHandlerWithPrefix(dailyAPIPrefix))
	r.Any(dailyAPIPrefix+"/*path", proxy.DailyAPIReverseProxyHandlerWithPrefix(dailyAPIPrefix))

	if strings.TrimSpace(os.Getenv("TDX_ENABLED")) != "0" {
		// 后台初始化：避免代码表与多路连接拖慢整进程启动；就绪前 /api/v1/tdx/* 返回 503 + Retry-After
		tdxLazy := tdxapi.NewLazyHandler(stockDbDir)
		mountTDX := func(prefix string) {
			h := http.StripPrefix(prefix, tdxLazy)
			r.Any(prefix, gin.WrapH(h))
			r.Any(prefix+"/*path", gin.WrapH(h))
		}
		mountTDX("/api/v1/tdx")
	}

	r.GET("/ws", func(c *gin.Context) {
		ws.ServeWS(hub, c.Writer, c.Request)
	})

	// /healthz 仅在 RegisterAnalysisRoutes 内注册一份，避免重复 panic
	analysisapi.RegisterAnalysisRoutes(r, analysisHandler)
	analysisapi.RegisterAnalysisStatic(r)

	addr := ":8787"
	if env := os.Getenv("PORT"); env != "" {
		addr = ":" + env
	}
	log.Printf("aistock backend listening on %s, root=%s", addr, root)
	log.Printf("HTTP API: /api/v1/* only (analysis, market-data, tdx, workspace, …); /ws, /healthz")
	if err := r.Run(addr); err != nil {
		log.Fatal(err)
	}
}

func buildStockAPIConfig() stocklib.Config {
	cfg := stocklib.DefaultConfig()

	if mode := strings.TrimSpace(os.Getenv("STOCKAPI_CACHE_MODE")); mode != "" {
		switch strings.ToLower(mode) {
		case "disabled":
			cfg.CacheMode = stocklib.CacheModeDisabled
		case "readonly":
			cfg.CacheMode = stocklib.CacheModeReadOnly
		default:
			cfg.CacheMode = stocklib.CacheModeAuto
		}
	}

	cfg.DataDir = appenv.DataRootDir()

	if v := parseDurationEnv("STOCKAPI_QUOTES_TTL"); v > 0 {
		cfg.QuotesTTL = v
	}
	if v := parseDurationEnv("STOCKAPI_HISTORY_TTL"); v > 0 {
		cfg.HistoryTTL = v
	}
	if v := parseDurationEnv("STOCKAPI_TIMELINE_TTL"); v > 0 {
		cfg.TimelineTTL = v
	}
	if v := parseIntEnv("STOCKAPI_BATCH_SIZE"); v > 0 {
		cfg.BatchSize = v
	}
	if v := parseIntEnv("STOCKAPI_CONCURRENCY"); v > 0 {
		cfg.Concurrency = v
	}

	return cfg
}

func parseDurationEnv(key string) time.Duration {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return 0
	}
	v, err := time.ParseDuration(raw)
	if err != nil {
		return 0
	}
	return v
}

func parseIntEnv(key string) int {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return 0
	}
	v, err := strconv.Atoi(raw)
	if err != nil {
		return 0
	}
	return v
}
