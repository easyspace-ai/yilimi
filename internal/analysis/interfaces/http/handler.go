package httpapi

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/easyspace-ai/yilimi/internal/analysis/agents/common"
	"github.com/easyspace-ai/yilimi/internal/workbench/klinecompat"
	"github.com/easyspace-ai/yilimi/internal/workbench/klinefetch"
	"github.com/easyspace-ai/yilimi/internal/workbench/ports"
	"github.com/easyspace-ai/yilimi/internal/workbench/tdxapi"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// Handler API 处理器
type Handler struct {
	jobManager *JobManager
	ctx        context.Context

	reports     *ReportStore
	watchlist   map[string]*WatchlistItem
	scheduled   map[string]*ScheduledAnalysis
	tokens      map[string]*UserToken
	watchlistMu sync.RWMutex
	scheduledMu sync.RWMutex
	tokenMu     sync.RWMutex

	// stockRepo 为 nil 时（例如独立运行 cmd/analysis）K 线接口退回模拟数据。
	stockMU   sync.RWMutex
	stockRepo ports.StockRepository
}

// NewHandler 创建 API 处理器；reports 可为 nil（不推荐生产环境）。
func NewHandler(ctx context.Context, jobManager *JobManager, reports *ReportStore) *Handler {
	return &Handler{
		jobManager: jobManager,
		ctx:        ctx,
		reports:    reports,
		watchlist:  make(map[string]*WatchlistItem),
		scheduled:  make(map[string]*ScheduledAnalysis),
		tokens:     make(map[string]*UserToken),
	}
}

// SetStockRepository 合并服务注入工作台行情仓储后，/market/kline 返回真实日线（前复权）。
func (h *Handler) SetStockRepository(repo ports.StockRepository) {
	h.stockMU.Lock()
	defer h.stockMU.Unlock()
	h.stockRepo = repo
}

func (h *Handler) getStockRepo() ports.StockRepository {
	h.stockMU.RLock()
	defer h.stockMU.RUnlock()
	return h.stockRepo
}

// ========== 健康检查 ==========

// Healthz 健康检查
func (h *Handler) Healthz(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status": "ok",
		"time":   time.Now().Format(time.RFC3339),
	})
}

// ========== 分析 API ==========

// StartAnalysis 开始分析
func (h *Handler) StartAnalysis(c *gin.Context) {
	var req AnalysisRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"detail": "无效的请求参数",
		})
		return
	}

	if req.Symbol == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"detail": "股票代码不能为空",
		})
		return
	}

	job := h.jobManager.CreateJob(req)
	h.jobManager.StartJob(h.ctx, job.ID)

	c.JSON(http.StatusOK, AnalysisResponse{
		JobID:     job.ID,
		Status:    job.Status,
		CreatedAt: job.CreatedAt,
	})
}

// GetJobStatus 获取任务状态
func (h *Handler) GetJobStatus(c *gin.Context) {
	jobID := c.Param("job_id")

	job, ok := h.jobManager.GetJob(jobID)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{
			"detail": "任务不存在",
		})
		return
	}

	resp := JobStatusResponse{
		JobID:     job.ID,
		Status:    job.Status,
		CreatedAt: job.CreatedAt,
		UpdatedAt: job.UpdatedAt,
		Symbol:    job.Request.Symbol,
		TradeDate: job.Request.TradeDate,
	}
	c.JSON(http.StatusOK, resp)
}

// GetJobResult 获取任务结果
func (h *Handler) GetJobResult(c *gin.Context) {
	jobID := c.Param("job_id")

	job, ok := h.jobManager.GetJob(jobID)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{
			"detail": "任务不存在",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"job_id": job.ID,
		"status": job.Status,
		"result": job.Result,
	})
}

// GetJobEvents SSE 事件流
func (h *Handler) GetJobEvents(c *gin.Context) {
	jobID := c.Param("job_id")

	// 检查任务是否存在
	_, ok := h.jobManager.GetJob(jobID)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{
			"detail": "任务不存在",
		})
		return
	}

	// 设置 SSE 响应头
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("Access-Control-Allow-Origin", "*")

	// 订阅事件
	eventCh, unsubscribe := h.jobManager.SubscribeEvents(jobID)
	defer unsubscribe()

	// 直接写入响应
	w := c.Writer
	flusher, ok := w.(http.Flusher)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{
			"detail": "不支持流式输出",
		})
		return
	}

	for {
		select {
		case event, ok := <-eventCh:
			if !ok {
				return
			}
			payload := map[string]any{
				"timestamp": event.Timestamp.Format(time.RFC3339),
			}
			if event.Message != "" {
				payload["message"] = event.Message
			}
			if event.Data != nil {
				if m, ok := event.Data.(map[string]any); ok {
					for k, v := range m {
						payload[k] = v
					}
				} else {
					payload["data"] = event.Data
				}
			}
			_ = writeSSE(w, flusher, event.Type, payload)
			// 已结束的任务重放完毕后立刻结束 SSE，避免前端 fetch 一直挂起
			if event.Type == "job.completed" || event.Type == "job.failed" {
				writeSSEDone(w, flusher)
				return
			}
		case <-c.Request.Context().Done():
			return
		}
	}
}

// ========== 市场数据 API ==========

// GetKline 获取 K线数据
func (h *Handler) GetKline(c *gin.Context) {
	symbol := strings.TrimSpace(c.Query("symbol"))
	startDate := strings.TrimSpace(c.Query("start_date"))
	endDate := strings.TrimSpace(c.Query("end_date"))

	if symbol == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"detail": "股票代码不能为空",
		})
		return
	}

	if endDate == "" {
		endDate = time.Now().Format("2006-01-02")
	}
	endTime, err := klinecompat.ParseKlineDate(endDate)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"detail": "end_date 须为 YYYY-MM-DD"})
		return
	}

	if startDate == "" {
		startDate = endTime.AddDate(0, 0, -180).Format("2006-01-02")
	}
	startTime, err := klinecompat.ParseKlineDate(startDate)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"detail": "start_date 须为 YYYY-MM-DD"})
		return
	}
	if startTime.After(endTime) {
		c.JSON(http.StatusBadRequest, gin.H{"detail": "start_date 必须早于或等于 end_date"})
		return
	}

	items, fetchErr := klinefetch.DailyBars(h.getStockRepo(), symbol, startTime, endTime)
	if len(items) > 0 {
		candles := make([]KlineCandle, 0, len(items))
		var prevClose float64
		for _, item := range items {
			changePct := 0.0
			if prevClose > 0 {
				changePct = (item.Close - prevClose) / prevClose * 100
			}
			prevClose = item.Close
			candles = append(candles, KlineCandle{
				Date:          item.Date,
				Open:          item.Open,
				High:          item.High,
				Low:           item.Low,
				Close:         item.Close,
				Volume:        float64(item.Volume),
				Amount:        item.Amount,
				Change:        item.Change,
				ChangePercent: changePct,
			})
		}
		c.JSON(http.StatusOK, KlineResponse{
			Symbol:    symbol,
			StartDate: startDate,
			EndDate:   endDate,
			Candles:   candles,
		})
		return
	}

	// 无任何行情源（独立分析进程且未启用 TDX）时使用占位数据
	if h.getStockRepo() == nil && tdxapi.ActiveService() == nil {
		candles := generateMockCandles(symbol, startDate, endDate)
		c.JSON(http.StatusOK, KlineResponse{
			Symbol:    symbol,
			StartDate: startDate,
			EndDate:   endDate,
			Candles:   candles,
		})
		return
	}

	if fetchErr != nil {
		if fetchErr.Error() == "no kline data" {
			c.JSON(http.StatusNotFound, gin.H{"detail": fetchErr.Error()})
			return
		}
		c.JSON(http.StatusBadGateway, gin.H{"detail": fetchErr.Error()})
		return
	}
	c.JSON(http.StatusNotFound, gin.H{"detail": "no kline data"})
}

// GetMinute 分时数据（1 分钟），数据来自通达信；需合并服务开启 TDX（TDX_ENABLED≠0）并就绪。
func (h *Handler) GetMinute(c *gin.Context) {
	symbol := strings.TrimSpace(c.Query("symbol"))
	date := strings.TrimSpace(c.Query("date"))
	if symbol == "" {
		c.JSON(http.StatusBadRequest, gin.H{"detail": "股票代码不能为空"})
		return
	}
	svc := tdxapi.ActiveService()
	if svc == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"detail": "通达信服务未就绪，请稍后再试或检查 /api/v1/tdx/health",
		})
		return
	}
	norm := klinecompat.NormalizeSymbol(symbol)
	var tdxCode string
	var err error
	if idx, ok := tdxapi.AShareIndexToTdxCode(norm); ok {
		tdxCode = idx
	} else {
		tdxCode, err = tdxapi.ToTdxLowerCodeFromNorm(norm)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"detail": err.Error()})
			return
		}
	}
	resp, usedDate, err := svc.MinuteSeries(tdxCode, date)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"detail": err.Error()})
		return
	}
	if resp == nil {
		c.JSON(http.StatusOK, MinuteChartResponse{
			Symbol: symbol,
			Date:   usedDate,
			Points: []MinuteChartPoint{},
		})
		return
	}
	points := make([]MinuteChartPoint, 0, len(resp.List))
	for _, row := range resp.List {
		points = append(points, MinuteChartPoint{
			Time:   row.Time,
			Price:  row.Price.Float64(),
			Volume: row.Number,
		})
	}
	c.JSON(http.StatusOK, MinuteChartResponse{
		Symbol: symbol,
		Date:   usedDate,
		Points: points,
	})
}

// generateMockCandles 生成模拟 K线数据（前端格式）
func generateMockCandles(symbol, startDate, endDate string) []KlineCandle {
	items := []KlineCandle{}
	baseDate := time.Now().AddDate(0, 0, -30)
	basePrice := 10.0

	for i := 0; i < 30; i++ {
		date := baseDate.AddDate(0, 0, i)
		change := (randFloat() - 0.5) * 2
		open := basePrice + change
		close := open + (randFloat()-0.5)*1.5
		high := max(open, close) + randFloat()*1
		low := min(open, close) - randFloat()*1
		volume := randFloat() * 10000000
		changeVal := close - open
		changePercent := 0.0
		if open != 0 {
			changePercent = (changeVal / open) * 100
		}
		turnoverRate := randFloat() * 5

		items = append(items, KlineCandle{
			Date:          date.Format("2006-01-02"),
			Open:          round(open, 2),
			High:          round(high, 2),
			Low:           round(low, 2),
			Close:         round(close, 2),
			Volume:        round(volume, 0),
			Amount:        round(volume*close, 2),
			Change:        round(changeVal, 2),
			ChangePercent: round(changePercent, 2),
			TurnoverRate:  round(turnoverRate, 2),
		})

		basePrice = close
	}
	return items
}

func randFloat() float64 {
	return float64(time.Now().UnixNano()%10000) / 10000
}

func round(v float64, decimals int) float64 {
	pow := 1.0
	for i := 0; i < decimals; i++ {
		pow *= 10
	}
	return float64(int64(v*pow+0.5)) / pow
}

func max(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}

func min(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

// ========== 聊天 API ==========

// ChatCompletions 对齐 Python chat_completions：stream=true 时推送 job.* / agent.* SSE，最后 event: done
func (h *Handler) ChatCompletions(c *gin.Context) {
	var req ChatCompletionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"detail": "无效的请求参数",
		})
		return
	}

	if !req.Stream {
		c.JSON(http.StatusBadRequest, gin.H{
			"detail": "请使用 stream: true 调用 Copilot 分析流",
		})
		return
	}

	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")
	c.Header("Access-Control-Allow-Origin", "*")

	w := c.Writer
	flusher, ok := w.(http.Flusher)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"detail": "不支持流式输出"})
		return
	}

	flusher.Flush()
	jobID := uuid.NewString()
	reqCtx := c.Request.Context()
	wfCtx, wfCancel := context.WithTimeout(reqCtx, AnalysisWorkflowTimeout())
	defer wfCancel()

	if err := writeSSE(w, flusher, "job.ready", map[string]any{"job_id": jobID}); err != nil {
		return
	}

	if _, err := common.TryChatModel(wfCtx); err != nil {
		_ = writeSSE(w, flusher, "job.failed", map[string]any{"error": err.Error()})
		writeSSEDone(w, flusher)
		return
	}

	userText := ConcatUserMessages(req.Messages)
	if userText == "" {
		_ = writeSSE(w, flusher, "job.failed", map[string]any{"error": "消息内容为空"})
		writeSSEDone(w, flusher)
		return
	}

	sym, tradeDate, needLLM := ExtractSymbolAndDate(userText)
	var err error
	if needLLM {
		sym, tradeDate, err = ExtractSymbolWithLLM(wfCtx, userText)
	}
	if err != nil || sym == "" {
		msg := "无法识别 A 股标的，请在对话中写明代码（如 600519.SH）"
		if err != nil {
			msg = err.Error()
		}
		_ = writeSSE(w, flusher, "job.failed", map[string]any{"error": msg})
		writeSSEDone(w, flusher)
		return
	}

	analysisReq := AnalysisRequest{
		Symbol:           sym,
		TradeDate:        tradeDate,
		SelectedAnalysts: req.SelectedAnalysts,
		Objective:        userText,
	}
	if err := writeSSE(w, flusher, "job.created", map[string]any{
		"job_id":     jobID,
		"symbol":     sym,
		"trade_date": tradeDate,
	}); err != nil {
		return
	}

	h.jobManager.RegisterJob(jobID, analysisReq)
	// 记入 job.Events，便于刷新页面后 GET /jobs/:id/events 重放 job.created
	h.jobManager.addEvent(jobID, JobEvent{
		Timestamp: time.Now(),
		Type:      "job.created",
		Data: map[string]any{
			"job_id":     jobID,
			"symbol":     sym,
			"trade_date": tradeDate,
		},
	})

	emit := func(ev string, data map[string]any) {
		_ = writeSSE(w, flusher, ev, data)
		h.jobManager.addEvent(jobID, JobEvent{
			Timestamp: time.Now(),
			Type:      ev,
			Data:      data,
		})
	}

	runErr := RunTradingWorkflow(wfCtx, h.jobManager, jobID, analysisReq, emit)
	if runErr != nil {
		if jb, ok := h.jobManager.GetJob(jobID); ok && jb.Status == JobStatusRunning {
			h.jobManager.CommitJobFailure(jobID)
			h.jobManager.persistJobReportFailed(jobID, analysisReq, runErr.Error())
			_ = writeSSE(w, flusher, "job.failed", map[string]any{
				"error": runErr.Error(),
			})
		}
	}
	writeSSEDone(w, flusher)
}

// ========== 报告 API ==========

// GetReports 获取报告列表
func (h *Handler) GetReports(c *gin.Context) {
	symbol := c.Query("symbol")
	skip, _ := strconv.Atoi(c.DefaultQuery("skip", "0"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "100"))

	if h.reports == nil {
		c.JSON(http.StatusOK, ReportListResponse{Total: 0, Reports: []Report{}})
		return
	}
	total, err := h.reports.Count(symbol)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"detail": "读取报告列表失败"})
		return
	}
	reports, err := h.reports.List(symbol, skip, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"detail": "读取报告列表失败"})
		return
	}
	if reports == nil {
		reports = []Report{}
	}

	c.JSON(http.StatusOK, ReportListResponse{
		Total:   total,
		Reports: reports,
	})
}

// GetReport 获取报告详情
func (h *Handler) GetReport(c *gin.Context) {
	reportID := c.Param("report_id")

	if h.reports == nil {
		c.JSON(http.StatusNotFound, gin.H{"detail": "报告不存在"})
		return
	}
	report, err := h.reports.Get(reportID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"detail": "读取报告失败"})
		return
	}
	if report == nil {
		c.JSON(http.StatusNotFound, gin.H{"detail": "报告不存在"})
		return
	}

	c.JSON(http.StatusOK, report)
}

// CreateReport 创建报告
func (h *Handler) CreateReport(c *gin.Context) {
	var req struct {
		Symbol     string `json:"symbol"`
		TradeDate  string `json:"trade_date"`
		Decision   string `json:"decision,omitempty"`
		ResultData any    `json:"result_data,omitempty"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"detail": "无效的请求参数"})
		return
	}

	if h.reports == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"detail": "报告存储未启用"})
		return
	}

	reportID := uuid.NewString()
	if err := h.reports.InsertManual(reportID, "", req.Symbol, req.TradeDate, req.Decision, req.ResultData); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"detail": "保存报告失败"})
		return
	}
	detail, err := h.reports.Get(reportID)
	if err != nil || detail == nil {
		c.JSON(http.StatusOK, Report{ID: reportID, Symbol: req.Symbol, TradeDate: req.TradeDate, Status: "completed", Decision: req.Decision})
		return
	}
	c.JSON(http.StatusOK, detail.Report)
}

// DeleteReport 删除报告
func (h *Handler) DeleteReport(c *gin.Context) {
	reportID := c.Param("report_id")

	if h.reports == nil {
		c.JSON(http.StatusOK, gin.H{"message": "报告已删除"})
		return
	}
	if err := h.reports.Delete(reportID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"detail": "删除报告失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "报告已删除"})
}

// ========== 公告 API ==========

// GetLatestAnnouncement 获取最新公告
func (h *Handler) GetLatestAnnouncement(c *gin.Context) {
	c.JSON(http.StatusOK, LatestAnnouncementResponse{
		Announcement: nil,
	})
}

// ========== 自选股 API ==========

// GetWatchlist 获取自选股
func (h *Handler) GetWatchlist(c *gin.Context) {
	h.watchlistMu.RLock()
	defer h.watchlistMu.RUnlock()

	var items []WatchlistItem
	for _, item := range h.watchlist {
		items = append(items, *item)
	}

	c.JSON(http.StatusOK, gin.H{"items": items})
}

// AddToWatchlist 添加自选股
func (h *Handler) AddToWatchlist(c *gin.Context) {
	var req struct {
		Symbol string `json:"symbol"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"detail": "无效的请求参数"})
		return
	}

	itemID := uuid.NewString()
	item := &WatchlistItem{
		ID:           itemID,
		Symbol:       req.Symbol,
		Name:         req.Symbol,
		SortOrder:    len(h.watchlist),
		CreatedAt:    time.Now().Format(time.RFC3339),
		HasScheduled: false,
	}

	h.watchlistMu.Lock()
	h.watchlist[itemID] = item
	h.watchlistMu.Unlock()

	c.JSON(http.StatusOK, item)
}

// RemoveFromWatchlist 移除自选股
func (h *Handler) RemoveFromWatchlist(c *gin.Context) {
	id := c.Param("id")

	h.watchlistMu.Lock()
	delete(h.watchlist, id)
	h.watchlistMu.Unlock()

	c.Status(http.StatusNoContent)
}

// ========== 定时分析 API ==========

// GetScheduled 获取定时分析
func (h *Handler) GetScheduled(c *gin.Context) {
	h.scheduledMu.RLock()
	defer h.scheduledMu.RUnlock()

	var items []ScheduledAnalysis
	for _, item := range h.scheduled {
		items = append(items, *item)
	}

	c.JSON(http.StatusOK, gin.H{"items": items})
}

// CreateScheduled 创建定时分析
func (h *Handler) CreateScheduled(c *gin.Context) {
	var req struct {
		Symbol      string `json:"symbol"`
		Horizon     string `json:"horizon,omitempty"`
		TriggerTime string `json:"trigger_time,omitempty"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"detail": "无效的请求参数"})
		return
	}

	id := uuid.NewString()
	item := &ScheduledAnalysis{
		ID:                  id,
		Symbol:              req.Symbol,
		Name:                req.Symbol,
		Horizon:             req.Horizon,
		TriggerTime:         req.TriggerTime,
		IsActive:            true,
		ConsecutiveFailures: 0,
		CreatedAt:           time.Now().Format(time.RFC3339),
	}

	h.scheduledMu.Lock()
	h.scheduled[id] = item
	h.scheduledMu.Unlock()

	c.JSON(http.StatusOK, item)
}

// UpdateScheduled 更新定时分析
func (h *Handler) UpdateScheduled(c *gin.Context) {
	id := c.Param("id")

	var req struct {
		IsActive    *bool   `json:"is_active,omitempty"`
		Horizon     *string `json:"horizon,omitempty"`
		TriggerTime *string `json:"trigger_time,omitempty"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"detail": "无效的请求参数"})
		return
	}

	h.scheduledMu.Lock()
	item, ok := h.scheduled[id]
	if ok {
		if req.IsActive != nil {
			item.IsActive = *req.IsActive
		}
		if req.Horizon != nil {
			item.Horizon = *req.Horizon
		}
		if req.TriggerTime != nil {
			item.TriggerTime = *req.TriggerTime
		}
	}
	h.scheduledMu.Unlock()

	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"detail": "定时分析不存在"})
		return
	}

	c.JSON(http.StatusOK, item)
}

// DeleteScheduled 删除定时分析
func (h *Handler) DeleteScheduled(c *gin.Context) {
	id := c.Param("id")

	h.scheduledMu.Lock()
	delete(h.scheduled, id)
	h.scheduledMu.Unlock()

	c.Status(http.StatusNoContent)
}

// ========== 股票搜索 API ==========

// SearchStocks 搜索股票
func (h *Handler) SearchStocks(c *gin.Context) {
	q := c.Query("q")

	// 模拟搜索结果
	results := []StockSearchResult{}
	if q != "" {
		results = append(results, StockSearchResult{Symbol: "000001.SZ", Name: "平安银行"})
		results = append(results, StockSearchResult{Symbol: "600519.SH", Name: "贵州茅台"})
		results = append(results, StockSearchResult{Symbol: "300750.SZ", Name: "宁德时代"})
	}

	c.JSON(http.StatusOK, gin.H{"results": results})
}

// ========== 配置 API ==========

// GetConfig 获取配置
func (h *Handler) GetConfig(c *gin.Context) {
	c.JSON(http.StatusOK, RuntimeConfig{
		LLMProvider:           "volcengine",
		DeepThinkLLM:          "doubao-seed-2.0-code",
		QuickThinkLLM:         "doubao-seed-2.0-code",
		BackendURL:            "",
		MaxDebateRounds:       3,
		MaxRiskDiscussRounds:  2,
		HasAPIKey:             true,
		ServerFallbackEnabled: true,
	})
}

// UpdateConfig 更新配置
func (h *Handler) UpdateConfig(c *gin.Context) {
	var req RuntimeConfigUpdate
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"detail": "无效的请求参数"})
		return
	}

	c.JSON(http.StatusOK, RuntimeConfigUpdateResponse{
		Message:   "配置已更新",
		Applied:   req,
		HasAPIKey: true,
		Current: RuntimeConfig{
			LLMProvider:           "volcengine",
			DeepThinkLLM:          "doubao-seed-2.0-code",
			QuickThinkLLM:         "doubao-seed-2.0-code",
			BackendURL:            "",
			MaxDebateRounds:       3,
			MaxRiskDiscussRounds:  2,
			HasAPIKey:             true,
			ServerFallbackEnabled: true,
		},
	})
}

// ========== 认证 API (简化版) ==========

// RequestLoginCode 请求登录验证码
func (h *Handler) RequestLoginCode(c *gin.Context) {
	var req struct {
		Email string `json:"email"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"detail": "无效的请求参数",
		})
		return
	}

	// 开发模式返回固定验证码
	c.JSON(http.StatusOK, gin.H{
		"message":    "验证码已发送",
		"dev_code":   "123456",
		"is_dev":     true,
		"expires_in": 600,
	})
}

// VerifyLoginCode 验证登录验证码
func (h *Handler) VerifyLoginCode(c *gin.Context) {
	var req struct {
		Email string `json:"email"`
		Code  string `json:"code"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"detail": "无效的请求参数",
		})
		return
	}

	// 开发模式接受 123456
	if req.Code != "123456" {
		c.JSON(http.StatusUnauthorized, gin.H{
			"detail": "验证码错误",
		})
		return
	}

	// 生成简单的 token
	token := "dev-token-" + time.Now().Format("20060102150405")

	c.JSON(http.StatusOK, AuthVerifyResponse{
		AccessToken: token,
		TokenType:   "Bearer",
		User: AuthUser{
			ID:    "dev-user-1",
			Email: req.Email,
		},
	})
}

// GetMe 获取当前用户信息
func (h *Handler) GetMe(c *gin.Context) {
	// 简化版认证，返回固定用户
	c.JSON(http.StatusOK, AuthUser{
		ID:          "dev-user-1",
		Email:       "dev@example.com",
		CreatedAt:   time.Now().Format(time.RFC3339),
		LastLoginAt: time.Now().Format(time.RFC3339),
	})
}

// GetFeaturesConfig 获取功能配置
func (h *Handler) GetFeaturesConfig(c *gin.Context) {
	c.JSON(http.StatusOK, ConfigResponse{
		Features: Features{
			Backtest:    false,
			MultiAgent:  true,
			Chat:        true,
			Scheduled:   false,
			Watchlist:   true,
			Reports:     true,
			TokenManage: false,
		},
		Limits: Limits{
			MaxAnalysesPerDay: 100,
			MaxChatPerDay:     1000,
		},
	})
}

// ========== Token 管理 API ==========

// GetTokens 获取用户Token
func (h *Handler) GetTokens(c *gin.Context) {
	h.tokenMu.RLock()
	defer h.tokenMu.RUnlock()

	var tokens []UserToken
	for _, t := range h.tokens {
		tokens = append(tokens, *t)
	}

	c.JSON(http.StatusOK, tokens)
}

// CreateToken 创建Token
func (h *Handler) CreateToken(c *gin.Context) {
	var req UserTokenCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"detail": "无效的请求参数"})
		return
	}

	tokenID := uuid.NewString()
	token := &UserToken{
		ID:        tokenID,
		Name:      req.Name,
		TokenHint: "sk-...xxxx",
		CreatedAt: time.Now().Format(time.RFC3339),
	}

	h.tokenMu.Lock()
	h.tokens[tokenID] = token
	h.tokenMu.Unlock()

	c.JSON(http.StatusOK, token)
}

// DeleteToken 删除Token
func (h *Handler) DeleteToken(c *gin.Context) {
	tokenID := c.Param("token_id")

	h.tokenMu.Lock()
	delete(h.tokens, tokenID)
	h.tokenMu.Unlock()

	c.JSON(http.StatusOK, gin.H{"message": "Token已删除"})
}
