package httpapi

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/easyspace-ai/yilimi/internal/appenv"
	"github.com/easyspace-ai/yilimi/internal/webui"
)

// Server 独立运行模式下的 API 服务器（仍可用于仅启动 AI 分析服务）。
type Server struct {
	router      *gin.Engine
	handler     *Handler
	reportStore *ReportStore
}

// BootstrapAnalysis 初始化 AI 投研限界上下文：任务管理、报告库、HTTP Handler。
func BootstrapAnalysis(ctx context.Context) (*Handler, *ReportStore, error) {
	jobManager := NewJobManager()

	// 可选覆盖；否则 AI_DATA_DIR/aigostock.db（进程入口须 appenv.Init）。
	dbPath := os.Getenv("AIGOSTOCK_SQLITE_PATH")
	if dbPath == "" {
		dbPath = filepath.Join(appenv.DataRootDir(), "aigostock.db")
	}
	reportStore, err := OpenReportStore(dbPath)
	if err != nil {
		return nil, nil, err
	}
	jobManager.SetReportStore(reportStore)
	handler := NewHandler(ctx, jobManager, reportStore)
	return handler, reportStore, nil
}

// AnalysisCORSMiddleware 与 OpenUI 前端对齐的 CORS（合并服务时由根路由统一挂载一次即可）。
func AnalysisCORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, PATCH, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")
		c.Header("Access-Control-Allow-Credentials", "true")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	}
}

// RegisterAnalysisRoutes 注册 AI 投研 HTTP API，统一挂在 /api/v1/analysis（OpenUI 契约）。
func RegisterAnalysisRoutes(engine *gin.Engine, handler *Handler) {
	engine.GET("/healthz", handler.Healthz)

	g := engine.Group("/api/v1/analysis")
	{
		g.POST("/analyze", handler.StartAnalysis)
		g.GET("/jobs/:job_id", handler.GetJobStatus)
		g.GET("/jobs/:job_id/result", handler.GetJobResult)
		g.GET("/jobs/:job_id/events", handler.GetJobEvents)

		g.POST("/chat/completions", handler.ChatCompletions)

		g.GET("/market/kline", handler.GetKline)
		g.GET("/market/minute", handler.GetMinute)
		g.GET("/market/stock-search", handler.SearchStocks)

		g.POST("/auth/request-code", handler.RequestLoginCode)
		g.POST("/auth/verify-code", handler.VerifyLoginCode)
		g.GET("/auth/me", handler.GetMe)

		g.GET("/config", handler.GetConfig)
		g.PATCH("/config", handler.UpdateConfig)

		g.GET("/reports", handler.GetReports)
		g.POST("/reports", handler.CreateReport)
		g.GET("/reports/:report_id", handler.GetReport)
		g.DELETE("/reports/:report_id", handler.DeleteReport)

		g.GET("/announcements/latest", handler.GetLatestAnnouncement)

		g.GET("/watchlist", handler.GetWatchlist)
		g.POST("/watchlist", handler.AddToWatchlist)
		g.DELETE("/watchlist/:id", handler.RemoveFromWatchlist)

		g.GET("/scheduled", handler.GetScheduled)
		g.POST("/scheduled", handler.CreateScheduled)
		g.PATCH("/scheduled/:id", handler.UpdateScheduled)
		g.DELETE("/scheduled/:id", handler.DeleteScheduled)

		g.GET("/tokens", handler.GetTokens)
		g.POST("/tokens", handler.CreateToken)
		g.DELETE("/tokens/:token_id", handler.DeleteToken)
	}
}

// RegisterAnalysisStatic 托管主前端靜態資源（可執行檔旁 web/ 或 AISTOCK_WEB_DIR）。AISTOCK_SERVE_WEB=0 可禁用。
func RegisterAnalysisStatic(engine *gin.Engine) {
	webui.Mount(engine)
}

// NewServer 创建独立运行的 API 服务器（仅分析上下文 + 可选静态资源）。
func NewServer(ctx context.Context) *Server {
	appenv.Init()
	handler, reportStore, err := BootstrapAnalysis(ctx)
	if err != nil {
		log.Fatalf("打开报告库失败: %v", err)
	}

	gin.SetMode(gin.ReleaseMode)
	router := gin.Default()
	router.Use(AnalysisCORSMiddleware())

	server := &Server{
		router:      router,
		handler:     handler,
		reportStore: reportStore,
	}
	RegisterAnalysisRoutes(router, handler)
	RegisterAnalysisStatic(router)
	return server
}

// Start 启动服务器
func (s *Server) Start(addr string) error {
	log.Printf("🚀 服务器启动在 %s", addr)

	srv := &http.Server{
		Addr:    addr,
		Handler: s.router,
	}

	go func() {
		quit := make(chan os.Signal, 1)
		signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
		<-quit
		log.Println("🛑 正在关闭服务器...")

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := srv.Shutdown(ctx); err != nil {
			log.Fatal("服务器关闭错误:", err)
		}
		if s.reportStore != nil {
			_ = s.reportStore.Close()
		}
		log.Println("✅ 服务器已关闭")
	}()

	return srv.ListenAndServe()
}
