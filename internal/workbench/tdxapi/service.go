package tdxapi

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/easyspace-ai/tdx"

	"github.com/easyspace-ai/yilimi/internal/appenv"
)

// NewService 使用 dataRoot（一般为 AI_DATA_DIR）将 TDX 库目录设为 {dataRoot}/tdx/database，然后拨号并初始化代码表与连接池（同 tdx-api/web init）。
// dataRoot 为空时使用 appenv.DataRootDir()。
func NewService(dataRoot string) (*Service, error) {
	root := strings.TrimSpace(dataRoot)
	if root == "" {
		root = appenv.DataRootDir()
	}
	tdx.SetDefaultDatabaseRoot(root)
	log.Printf("tdx: database dir: %s", tdx.DefaultDatabaseDir)

	client, err := tdx.DialDefault(tdx.WithDebug(false))
	if err != nil {
		return nil, fmt.Errorf("tdx dial: %w", err)
	}

	if err := os.MkdirAll(tdx.DefaultDatabaseDir, 0755); err != nil {
		log.Printf("tdx: create data dir: %v", err)
	}
	if codes, err := tdx.NewCodesSqlite(client); err != nil {
		log.Printf("tdx: codes sqlite: %v", err)
	} else {
		tdx.DefaultCodes = codes
		if err := tdx.DefaultCodes.Update(); err != nil {
			log.Printf("tdx: update codes: %v", err)
		} else {
			log.Printf("tdx: loaded code map (%d entries)", len(tdx.DefaultCodes.Map))
		}
	}

	manager, err := tdx.NewManage(&tdx.ManageConfig{Number: 4})
	if err != nil {
		return nil, fmt.Errorf("tdx manage: %w", err)
	}
	if err := manager.Codes.Update(); err != nil {
		log.Printf("tdx: manager codes update: %v", err)
	}
	if err := manager.Workday.Update(); err != nil {
		log.Printf("tdx: workday update: %v", err)
	}
	manager.Cron.Start()

	return &Service{
		client:  client,
		manager: manager,
		tasks:   NewTaskManager(),
	}, nil
}

// ServeHTTP serves TDX API paths relative to the mount prefix（例如 /api/v1/tdx/quote）。
func (s *Service) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mux().ServeHTTP(w, r)
}

func (s *Service) mux() *http.ServeMux {
	m := http.NewServeMux()

	reg := func(pattern string, h http.HandlerFunc) {
		m.HandleFunc(pattern, h)
	}

	reg("GET /quote", s.handleGetQuote)
	reg("GET /kline", s.handleGetKline)
	reg("GET /minute", s.handleGetMinute)
	reg("GET /trade", s.handleGetTrade)
	reg("GET /search", s.handleSearchCode)
	reg("GET /stock-info", s.handleGetStockInfo)

	reg("GET /codes", s.handleGetCodes)
	reg("POST /batch-quote", s.handleBatchQuote)
	reg("GET /kline-history", s.handleGetKlineHistory)
	reg("GET /index/all", s.handleGetIndexAll)
	reg("GET /index", s.handleGetIndex)
	reg("GET /market-stats", s.handleGetMarketStats)
	reg("GET /market-count", s.handleGetMarketCount)
	reg("GET /stock-codes", s.handleGetStockCodes)
	reg("GET /etf-codes", s.handleGetETFCodes)
	reg("GET /server-status", s.handleGetServerStatus)
	reg("GET /health", s.handleHealthCheck)
	reg("GET /etf", s.handleGetETFList)
	reg("GET /trade-history/full", s.handleGetTradeHistoryFull)
	reg("GET /trade-history", s.handleGetTradeHistory)
	reg("GET /minute-trade-all", s.handleGetMinuteTradeAll)
	reg("GET /kline-all/tdx", s.handleGetKlineAllTDX)
	reg("GET /kline-all/ths", s.handleGetKlineAllTHS)
	reg("GET /kline-all", s.handleGetKlineAllTDX)
	reg("GET /workday/range", s.handleGetWorkdayRange)
	reg("GET /workday", s.handleGetWorkday)
	reg("GET /income", s.handleGetIncome)

	reg("POST /tasks/pull-kline", s.handleCreatePullKlineTask)
	reg("POST /tasks/pull-trade", s.handleCreatePullTradeTask)
	reg("GET /tasks/", s.handleTaskOperations)
	reg("POST /tasks/", s.handleTaskOperations)
	reg("GET /tasks", s.handleListTasks)

	return m
}
