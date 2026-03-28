package httpapi

import (
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/easyspace-ai/yilimi/internal/workbench/domain/stock"
	"github.com/easyspace-ai/yilimi/internal/workbench/klinecompat"
	"github.com/easyspace-ai/yilimi/internal/workbench/klinefetch"
	"github.com/easyspace-ai/yilimi/internal/workbench/ports"
)

// StockHandler 股票 API 处理器
type StockHandler struct {
	stockRepo ports.StockRepository
}

// NewStockHandler 创建股票处理器
func NewStockHandler(stockRepo ports.StockRepository) *StockHandler {
	return &StockHandler{
		stockRepo: stockRepo,
	}
}

// RegisterRoutes 注册股票相关路由
func (h *StockHandler) RegisterRoutes(r *gin.RouterGroup) {
	r.GET("/kline", h.GetKlineCompat)
	r.POST("/kline/batch", h.GetKlineCompatBatch)

	// ========== 选股API ==========
	picker := r.Group("/picker")
	{
		picker.GET("/quotes", h.GetAllQuotes)
		picker.GET("/timeline/:code", h.GetTimeline)
		picker.POST("/end-of-day", h.EndOfDayPicker)
		picker.POST("/momentum", h.MomentumPicker)
		picker.POST("/kunpeng", h.KunpengPicker)
	}

	stock := r.Group("/stock")
	{
		// 基础查询
		stock.GET("/realtime", h.GetRealtime)
		stock.GET("/search", h.Search)
		stock.GET("/list", h.List)
		stock.GET("/hot-strategy", h.GetHotStrategy)
		stock.POST("/all", h.SelectStocks)
		stock.GET("/all-info", h.GetAllStockInfo)

		// 市场、行业、概念
		stock.GET("/markets", h.GetMarkets)
		stock.GET("/industries", h.GetIndustries)
		stock.GET("/concepts", h.GetConcepts)

		// 关注股票
		stock.GET("/followed", h.GetFollowed)
		stock.POST("/:code/follow", h.Follow)
		stock.DELETE("/:code/follow", h.Unfollow)
		stock.POST("/:code/cost", h.SetCost)
		stock.POST("/:code/alarm", h.SetAlarm)

		// 单只股票详情
		stock.GET("/:code", h.GetInfo)
		stock.GET("/:code/kline", h.GetKLine)
		stock.GET("/:code/common-kline", h.GetCommonKLine)
		stock.POST("/common-kline/batch", h.GetBatchCommonKLine)
		stock.GET("/:code/minute-price", h.GetMinutePrice)
		stock.GET("/:code/money-history", h.GetMoneyHistory)
		stock.GET("/:code/money-trend", h.GetMoneyTrend)
		stock.GET("/:code/concept-info", h.GetConceptInfo)
		stock.GET("/:code/financial-info", h.GetFinancialInfo)
		stock.GET("/:code/holder-num", h.GetHolderNum)
		stock.GET("/:code/rzrq", h.GetRZRQ)
	}
}

type klineCompatCandle struct {
	Date          string   `json:"date"`
	Open          float64  `json:"open"`
	High          float64  `json:"high"`
	Low           float64  `json:"low"`
	Close         float64  `json:"close"`
	Volume        int64    `json:"volume"`
	Amount        float64  `json:"amount"`
	Change        float64  `json:"change"`
	ChangePercent *float64 `json:"change_percent"`
	TurnoverRate  *float64 `json:"turnover_rate"`
}

type klineCompatResponse struct {
	Symbol    string              `json:"symbol"`
	StartDate string              `json:"start_date"`
	EndDate   string              `json:"end_date"`
	Candles   []klineCompatCandle `json:"candles"`
}

// GetKlineCompat 兼容 Python /v1/market/kline 响应格式
func (h *StockHandler) GetKlineCompat(c *gin.Context) {
	rawSymbol := c.Query("symbol")
	if strings.TrimSpace(rawSymbol) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"detail": "symbol is required"})
		return
	}

	endDate := strings.TrimSpace(c.Query("end_date"))
	if endDate == "" {
		endDate = time.Now().Format("2006-01-02")
	}
	endTime, err := klinecompat.ParseKlineDate(endDate)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"detail": "end_date must be YYYY-MM-DD"})
		return
	}

	startDate := strings.TrimSpace(c.Query("start_date"))
	if startDate == "" {
		startDate = endTime.AddDate(0, 0, -120).Format("2006-01-02")
	}
	startTime, err := klinecompat.ParseKlineDate(startDate)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"detail": "start_date must be YYYY-MM-DD"})
		return
	}
	if startTime.After(endTime) {
		c.JSON(http.StatusBadRequest, gin.H{"detail": "start_date must be <= end_date"})
		return
	}

	resp, err := h.buildKlineCompatResponse(rawSymbol, startTime, endTime, startDate, endDate)
	if err != nil {
		if err.Error() == "no kline data" {
			c.JSON(http.StatusNotFound, gin.H{"detail": err.Error()})
			return
		}
		c.JSON(http.StatusBadGateway, gin.H{"detail": err.Error()})
		return
	}
	c.JSON(http.StatusOK, resp)
}

type klineBatchRequest struct {
	Symbols   []string `json:"symbols"`
	StartDate string   `json:"start_date"`
	EndDate   string   `json:"end_date"`
}

func (h *StockHandler) GetKlineCompatBatch(c *gin.Context) {
	var req klineBatchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"detail": "invalid request body"})
		return
	}
	if len(req.Symbols) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"detail": "symbols is required"})
		return
	}

	endDate := strings.TrimSpace(req.EndDate)
	if endDate == "" {
		endDate = time.Now().Format("2006-01-02")
	}
	endTime, err := klinecompat.ParseKlineDate(endDate)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"detail": "end_date must be YYYY-MM-DD"})
		return
	}
	startDate := strings.TrimSpace(req.StartDate)
	if startDate == "" {
		startDate = endTime.AddDate(0, 0, -120).Format("2006-01-02")
	}
	startTime, err := klinecompat.ParseKlineDate(startDate)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"detail": "start_date must be YYYY-MM-DD"})
		return
	}
	if startTime.After(endTime) {
		c.JSON(http.StatusBadRequest, gin.H{"detail": "start_date must be <= end_date"})
		return
	}

	items := make(map[string]klineCompatResponse, len(req.Symbols))
	errors := make([]map[string]string, 0)

	var (
		mu       sync.Mutex
		wg       sync.WaitGroup
		sem      = make(chan struct{}, 8)
		seenCode = make(map[string]struct{}, len(req.Symbols))
	)

	for _, raw := range req.Symbols {
		symbol := strings.TrimSpace(raw)
		if symbol == "" {
			continue
		}
		if _, ok := seenCode[symbol]; ok {
			continue
		}
		seenCode[symbol] = struct{}{}

		wg.Add(1)
		go func(sym string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			resp, buildErr := h.buildKlineCompatResponse(sym, startTime, endTime, startDate, endDate)

			mu.Lock()
			defer mu.Unlock()
			if buildErr != nil {
				errors = append(errors, map[string]string{
					"symbol":  sym,
					"message": buildErr.Error(),
				})
				return
			}
			items[sym] = resp
		}(symbol)
	}
	wg.Wait()

	c.JSON(http.StatusOK, gin.H{
		"items":  items,
		"errors": errors,
	})
}

func (h *StockHandler) buildKlineCompatResponse(rawSymbol string, startTime time.Time, endTime time.Time, startDate string, endDate string) (klineCompatResponse, error) {
	items, err := klinefetch.DailyBars(h.stockRepo, rawSymbol, startTime, endTime)
	if err != nil {
		return klineCompatResponse{}, err
	}

	candles := make([]klineCompatCandle, 0, len(items))
	var prevClose float64
	for _, item := range items {
		changePct := 0.0
		if prevClose > 0 {
			changePct = (item.Close - prevClose) / prevClose * 100
		}
		prevClose = item.Close
		candles = append(candles, klineCompatCandle{
			Date:          item.Date,
			Open:          item.Open,
			High:          item.High,
			Low:           item.Low,
			Close:         item.Close,
			Volume:        item.Volume,
			Amount:        item.Amount,
			Change:        item.Change,
			ChangePercent: &changePct,
			TurnoverRate:  nil,
		})
	}
	return klineCompatResponse{
		Symbol:    rawSymbol,
		StartDate: startDate,
		EndDate:   endDate,
		Candles:   candles,
	}, nil
}

// GetRealtime 获取实时行情
func (h *StockHandler) GetRealtime(c *gin.Context) {
	codes := c.Query("codes")
	if codes == "" {
		c.JSON(http.StatusOK, Error("codes is required"))
		return
	}

	codeList := strings.Split(codes, ",")
	quotes, err := h.stockRepo.GetQuotes(codeList)
	if err != nil {
		c.JSON(http.StatusOK, Error(err.Error()))
		return
	}

	c.JSON(http.StatusOK, Success(quotes))
}

// Search 搜索股票
func (h *StockHandler) Search(c *gin.Context) {
	keyword := c.Query("keyword")
	if keyword == "" {
		c.JSON(http.StatusOK, Error("keyword is required"))
		return
	}

	stocks, err := h.stockRepo.Search(keyword)
	if err != nil {
		c.JSON(http.StatusOK, Error(err.Error()))
		return
	}

	c.JSON(http.StatusOK, Success(stocks))
}

// List 获取股票列表
func (h *StockHandler) List(c *gin.Context) {
	market := c.Query("market")
	industry := c.Query("industry")
	concept := c.Query("concept")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "50"))

	stocks, total, err := h.stockRepo.List(market, industry, concept, page, pageSize)
	if err != nil {
		c.JSON(http.StatusOK, Error(err.Error()))
		return
	}

	c.JSON(http.StatusOK, PageData(stocks, total, page, pageSize))
}

// GetHotStrategy 获取热门策略
func (h *StockHandler) GetHotStrategy(c *gin.Context) {
	strategies, err := h.stockRepo.GetHotStrategies()
	if err != nil {
		c.JSON(http.StatusOK, Error(err.Error()))
		return
	}

	c.JSON(http.StatusOK, Success(strategies))
}

// SelectStocks 技术指标选股
func (h *StockHandler) SelectStocks(c *gin.Context) {
	var criteria stock.StockSelectionCriteria
	if err := c.ShouldBindJSON(&criteria); err != nil {
		c.JSON(http.StatusOK, Error(err.Error()))
		return
	}

	result, err := h.stockRepo.SelectStocks(&criteria)
	if err != nil {
		c.JSON(http.StatusOK, Error(err.Error()))
		return
	}

	c.JSON(http.StatusOK, Success(result))
}

// GetAllStockInfo 获取扩展股票信息
func (h *StockHandler) GetAllStockInfo(c *gin.Context) {
	var criteria stock.StockSelectionCriteria
	if err := c.ShouldBindQuery(&criteria); err != nil {
		c.JSON(http.StatusOK, Error(err.Error()))
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "50"))

	stocks, total, err := h.stockRepo.GetAllStockInfo(&criteria, page, pageSize)
	if err != nil {
		c.JSON(http.StatusOK, Error(err.Error()))
		return
	}

	c.JSON(http.StatusOK, PageData(stocks, total, page, pageSize))
}

// GetMarkets 获取市场列表
func (h *StockHandler) GetMarkets(c *gin.Context) {
	markets, err := h.stockRepo.GetMarkets()
	if err != nil {
		c.JSON(http.StatusOK, Error(err.Error()))
		return
	}

	c.JSON(http.StatusOK, Success(markets))
}

// GetIndustries 获取行业列表
func (h *StockHandler) GetIndustries(c *gin.Context) {
	industries, err := h.stockRepo.GetIndustries()
	if err != nil {
		c.JSON(http.StatusOK, Error(err.Error()))
		return
	}

	c.JSON(http.StatusOK, Success(industries))
}

// GetConcepts 获取概念列表
func (h *StockHandler) GetConcepts(c *gin.Context) {
	concepts, err := h.stockRepo.GetConcepts()
	if err != nil {
		c.JSON(http.StatusOK, Error(err.Error()))
		return
	}

	c.JSON(http.StatusOK, Success(concepts))
}

// GetFollowed 获取关注列表
func (h *StockHandler) GetFollowed(c *gin.Context) {
	userId := c.GetHeader("X-User-Id")
	if userId == "" {
		userId = "default"
	}

	followed, err := h.stockRepo.GetFollowedStocks(userId)
	if err != nil {
		c.JSON(http.StatusOK, Error(err.Error()))
		return
	}

	c.JSON(http.StatusOK, Success(followed))
}

// Follow 关注股票
func (h *StockHandler) Follow(c *gin.Context) {
	code := klinecompat.NormalizeFollowCode(c.Param("code"))
	if code == "" {
		c.JSON(http.StatusOK, Error("code is required"))
		return
	}

	userId := c.GetHeader("X-User-Id")
	if userId == "" {
		userId = "default"
	}

	var req struct {
		StockName string `json:"stockName"`
		Note      string `json:"note"`
	}
	if c.Request.ContentLength > 0 {
		_ = c.ShouldBindJSON(&req)
	}
	name := strings.TrimSpace(req.StockName)
	if name == "" {
		name = code
	}
	note := strings.TrimSpace(req.Note)

	if err := h.stockRepo.FollowStock(userId, code, name, note); err != nil {
		c.JSON(http.StatusOK, Error(err.Error()))
		return
	}

	c.JSON(http.StatusOK, Success(nil))
}

// Unfollow 取消关注
func (h *StockHandler) Unfollow(c *gin.Context) {
	code := klinecompat.NormalizeFollowCode(c.Param("code"))
	if code == "" {
		c.JSON(http.StatusOK, Error("code is required"))
		return
	}

	userId := c.GetHeader("X-User-Id")
	if userId == "" {
		userId = "default"
	}

	if err := h.stockRepo.UnfollowStock(userId, code); err != nil {
		c.JSON(http.StatusOK, Error(err.Error()))
		return
	}

	c.JSON(http.StatusOK, Success(nil))
}

// SetCost 设置成本价
func (h *StockHandler) SetCost(c *gin.Context) {
	code := c.Param("code")
	if code == "" {
		c.JSON(http.StatusOK, Error("code is required"))
		return
	}

	var req struct {
		CostPrice float64 `json:"costPrice"`
		Quantity  float64 `json:"quantity"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, Error(err.Error()))
		return
	}

	userId := c.GetHeader("X-User-Id")
	if userId == "" {
		userId = "default"
	}

	if err := h.stockRepo.UpdateCost(userId, code, req.CostPrice, req.Quantity); err != nil {
		c.JSON(http.StatusOK, Error(err.Error()))
		return
	}

	c.JSON(http.StatusOK, Success(nil))
}

// SetAlarm 设置预警
func (h *StockHandler) SetAlarm(c *gin.Context) {
	code := c.Param("code")
	if code == "" {
		c.JSON(http.StatusOK, Error("code is required"))
		return
	}

	var req stock.StockAlarm
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, Error(err.Error()))
		return
	}

	userId := c.GetHeader("X-User-Id")
	if userId == "" {
		userId = "default"
	}

	req.UserId = userId
	req.StockCode = code

	if err := h.stockRepo.SetAlarm(userId, &req); err != nil {
		c.JSON(http.StatusOK, Error(err.Error()))
		return
	}

	c.JSON(http.StatusOK, Success(nil))
}

// GetInfo 获取股票基础信息
func (h *StockHandler) GetInfo(c *gin.Context) {
	code := c.Param("code")
	if code == "" {
		c.JSON(http.StatusOK, Error("code is required"))
		return
	}

	info, err := h.stockRepo.GetByCode(code)
	if err != nil {
		c.JSON(http.StatusOK, Error(err.Error()))
		return
	}

	c.JSON(http.StatusOK, Success(info))
}

// GetKLine 获取K线
func (h *StockHandler) GetKLine(c *gin.Context) {
	code := c.Param("code")
	if code == "" {
		c.JSON(http.StatusOK, Error("code is required"))
		return
	}

	klineType := c.DefaultQuery("kLineType", "day")
	days, _ := strconv.Atoi(c.DefaultQuery("days", "120"))
	adjustFlag := c.DefaultQuery("adjustFlag", "qfq")

	data, err := h.stockRepo.GetKLine(code, klineType, days, adjustFlag)
	if err != nil {
		c.JSON(http.StatusOK, Error(err.Error()))
		return
	}

	c.JSON(http.StatusOK, Success(data))
}

// GetCommonKLine 获取通用K线
func (h *StockHandler) GetCommonKLine(c *gin.Context) {
	code := c.Param("code")
	if code == "" {
		c.JSON(http.StatusOK, Error("code is required"))
		return
	}

	klineType := c.DefaultQuery("kLineType", "day")
	days, _ := strconv.Atoi(c.DefaultQuery("days", "120"))

	data, err := h.stockRepo.GetCommonKLine(code, klineType, days)
	if err != nil {
		c.JSON(http.StatusOK, Error(err.Error()))
		return
	}

	c.JSON(http.StatusOK, Success(data))
}

type batchCommonKLineRequest struct {
	Codes     []string `json:"codes"`
	KLineType string   `json:"kLineType"`
	Days      int      `json:"days"`
}

// GetBatchCommonKLine 批量获取通用K线
func (h *StockHandler) GetBatchCommonKLine(c *gin.Context) {
	var req batchCommonKLineRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, Error("invalid request body"))
		return
	}

	if len(req.Codes) == 0 {
		c.JSON(http.StatusOK, Error("codes is required"))
		return
	}

	klineType := strings.TrimSpace(req.KLineType)
	if klineType == "" {
		klineType = "day"
	}

	days := req.Days
	if days <= 0 {
		days = 120
	}
	if days > 500 {
		days = 500
	}

	type responseItem struct {
		Code string           `json:"code"`
		Data *stock.KLineData `json:"data"`
	}

	type responseError struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	}

	results := make(map[string]*stock.KLineData, len(req.Codes))
	errors := make([]responseError, 0)

	var (
		mu       sync.Mutex
		wg       sync.WaitGroup
		sem      = make(chan struct{}, 8)
		seenCode = make(map[string]struct{}, len(req.Codes))
	)

	for _, rawCode := range req.Codes {
		code := strings.TrimSpace(rawCode)
		if code == "" {
			continue
		}
		if _, exists := seenCode[code]; exists {
			continue
		}
		seenCode[code] = struct{}{}

		wg.Add(1)
		go func(symbol string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			data, err := h.stockRepo.GetCommonKLine(symbol, klineType, days)

			mu.Lock()
			defer mu.Unlock()
			if err != nil {
				errors = append(errors, responseError{
					Code:    symbol,
					Message: err.Error(),
				})
				return
			}
			results[symbol] = data
		}(code)
	}

	wg.Wait()

	c.JSON(http.StatusOK, Success(gin.H{
		"items":  results,
		"errors": errors,
	}))
}

// GetMinutePrice 获取分时数据
func (h *StockHandler) GetMinutePrice(c *gin.Context) {
	code := c.Param("code")
	if code == "" {
		c.JSON(http.StatusOK, Error("code is required"))
		return
	}

	data, err := h.stockRepo.GetMinutePrice(code)
	if err != nil {
		c.JSON(http.StatusOK, Error(err.Error()))
		return
	}

	c.JSON(http.StatusOK, Success(data))
}

// GetMoneyHistory 获取资金历史
func (h *StockHandler) GetMoneyHistory(c *gin.Context) {
	code := c.Param("code")
	if code == "" {
		c.JSON(http.StatusOK, Error("code is required"))
		return
	}

	data, err := h.stockRepo.GetMoneyHistory(code)
	if err != nil {
		c.JSON(http.StatusOK, Error(err.Error()))
		return
	}

	c.JSON(http.StatusOK, Success(data))
}

// GetMoneyTrend 获取资金趋势
func (h *StockHandler) GetMoneyTrend(c *gin.Context) {
	code := c.Param("code")
	if code == "" {
		c.JSON(http.StatusOK, Error("code is required"))
		return
	}

	data, err := h.stockRepo.GetMoneyTrend(code)
	if err != nil {
		c.JSON(http.StatusOK, Error(err.Error()))
		return
	}

	c.JSON(http.StatusOK, Success(data))
}

// GetConceptInfo 获取概念信息
func (h *StockHandler) GetConceptInfo(c *gin.Context) {
	code := c.Param("code")
	if code == "" {
		c.JSON(http.StatusOK, Error("code is required"))
		return
	}

	data, err := h.stockRepo.GetConceptInfo(code)
	if err != nil {
		c.JSON(http.StatusOK, Error(err.Error()))
		return
	}

	c.JSON(http.StatusOK, Success(data))
}

// GetFinancialInfo 获取财务信息
func (h *StockHandler) GetFinancialInfo(c *gin.Context) {
	code := c.Param("code")
	if code == "" {
		c.JSON(http.StatusOK, Error("code is required"))
		return
	}

	data, err := h.stockRepo.GetFinancialInfo(code)
	if err != nil {
		c.JSON(http.StatusOK, Error(err.Error()))
		return
	}

	c.JSON(http.StatusOK, Success(data))
}

// GetHolderNum 获取股东人数
func (h *StockHandler) GetHolderNum(c *gin.Context) {
	code := c.Param("code")
	if code == "" {
		c.JSON(http.StatusOK, Error("code is required"))
		return
	}

	num, err := h.stockRepo.GetHolderNum(code)
	if err != nil {
		c.JSON(http.StatusOK, Error(err.Error()))
		return
	}

	c.JSON(http.StatusOK, Success(num))
}

// GetRZRQ 获取融资融券
func (h *StockHandler) GetRZRQ(c *gin.Context) {
	code := c.Param("code")
	if code == "" {
		c.JSON(http.StatusOK, Error("code is required"))
		return
	}

	data, err := h.stockRepo.GetRZRQ(code)
	if err != nil {
		c.JSON(http.StatusOK, Error(err.Error()))
		return
	}

	c.JSON(http.StatusOK, Success(data))
}

// ========== 选股API ==========

// GetAllQuotes 获取全市场行情
func (h *StockHandler) GetAllQuotes(c *gin.Context) {
	quotes, err := h.stockRepo.GetAllMarketQuotes()
	if err != nil {
		c.JSON(http.StatusOK, Error(err.Error()))
		return
	}
	c.JSON(http.StatusOK, Success(quotes))
}

// GetTimeline 获取今日分时数据
func (h *StockHandler) GetTimeline(c *gin.Context) {
	code := c.Param("code")
	if code == "" {
		c.JSON(http.StatusOK, Error("code is required"))
		return
	}

	timeline, err := h.stockRepo.GetTodayTimeline(code)
	if err != nil {
		c.JSON(http.StatusOK, Error(err.Error()))
		return
	}

	c.JSON(http.StatusOK, Success(timeline))
}

// EndOfDayPicker 尾盘选股
func (h *StockHandler) EndOfDayPicker(c *gin.Context) {
	var req stock.EndOfDayPickerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, Error(err.Error()))
		return
	}

	// 设置默认值
	if req.MarketCapMin == 0 {
		req.MarketCapMin = 50
	}
	if req.MarketCapMax == 0 {
		req.MarketCapMax = 200
	}
	if req.VolumeRatioMin == 0 {
		req.VolumeRatioMin = 1.2
	}
	if req.ChangePercentMin == 0 {
		req.ChangePercentMin = 3
	}
	if req.ChangePercentMax == 0 {
		req.ChangePercentMax = 5
	}
	if req.TurnoverRateMin == 0 {
		req.TurnoverRateMin = 5
	}
	if req.TurnoverRateMax == 0 {
		req.TurnoverRateMax = 10
	}
	if req.TimelineAboveAvgRatio == 0 {
		req.TimelineAboveAvgRatio = 80
	}
	if !req.ExcludeST {
		req.ExcludeST = true
	}

	result, err := h.stockRepo.EndOfDayPicker(&req)
	if err != nil {
		c.JSON(http.StatusOK, Error(err.Error()))
		return
	}

	c.JSON(http.StatusOK, Success(result))
}

// MomentumPicker 妖股候选人扫描
func (h *StockHandler) MomentumPicker(c *gin.Context) {
	var req stock.MomentumPickerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, Error(err.Error()))
		return
	}

	// 设置默认值
	if req.MomentumThreshold == 0 {
		req.MomentumThreshold = 50
	}
	if req.AvgTurnoverMin == 0 {
		req.AvgTurnoverMin = 5
	}
	if req.MarketCapMin == 0 {
		req.MarketCapMin = 20
	}
	if req.MarketCapMax == 0 {
		req.MarketCapMax = 500
	}
	if req.PriceMin == 0 {
		req.PriceMin = 5
	}
	if req.PriceMax == 0 {
		req.PriceMax = 100
	}
	if !req.TrendAboveMA60 {
		req.TrendAboveMA60 = true
	}
	if !req.ExcludeST {
		req.ExcludeST = true
	}

	result, err := h.stockRepo.MomentumPicker(&req)
	if err != nil {
		c.JSON(http.StatusOK, Error(err.Error()))
		return
	}

	c.JSON(http.StatusOK, Success(result))
}

// KunpengPicker 鲲鹏战法筛选
func (h *StockHandler) KunpengPicker(c *gin.Context) {
	var req stock.KunpengPickerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, Error(err.Error()))
		return
	}

	// 设置默认值
	if req.MarketCapMin == 0 {
		req.MarketCapMin = 100
	}
	if req.MarketCapMax == 0 {
		req.MarketCapMax = 300
	}
	if req.NetProfitMin == 0 {
		req.NetProfitMin = 2
	}
	if req.PeMin == 0 {
		req.PeMin = 0.1
	}
	if req.PeMax == 0 {
		req.PeMax = 40
	}
	if req.PriceMin == 0 {
		req.PriceMin = 3
	}
	if req.PriceMax == 0 {
		req.PriceMax = 100
	}
	if !req.ExcludeST {
		req.ExcludeST = true
	}
	if !req.ExcludeNewStock {
		req.ExcludeNewStock = true
	}

	result, err := h.stockRepo.KunpengPicker(&req)
	if err != nil {
		c.JSON(http.StatusOK, Error(err.Error()))
		return
	}

	c.JSON(http.StatusOK, Success(result))
}
