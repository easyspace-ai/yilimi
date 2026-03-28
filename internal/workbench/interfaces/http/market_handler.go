package httpapi

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/easyspace-ai/yilimi/internal/workbench/ports"
)

// MarketHandler 市场 API 处理器
type MarketHandler struct {
	marketRepo ports.MarketRepository
}

// NewMarketHandler 创建市场处理器
func NewMarketHandler(marketRepo ports.MarketRepository) *MarketHandler {
	return &MarketHandler{
		marketRepo: marketRepo,
	}
}

// RegisterRoutes 注册市场相关路由
func (h *MarketHandler) RegisterRoutes(r *gin.RouterGroup) {
	market := r.Group("/market")
	{
		// 龙虎榜
		market.GET("/long-tiger", h.GetLongTiger)

		// 热门数据
		market.GET("/hot-stock", h.GetHotStock)
		market.GET("/hot-event", h.GetHotEvent)
		market.GET("/hot-topic", h.GetHotTopic)

		// 日历
		market.GET("/invest-calendar", h.GetInvestCalendar)
		market.GET("/cls-calendar", h.GetCLSCalendar)

		// 全球指数
		market.GET("/global-indexes", h.GetGlobalIndexes)
		market.GET("/global-indexes-readable", h.GetGlobalIndexesReadable)

		// 资金排名
		market.GET("/industry-rank", h.GetIndustryRank)
		market.GET("/industry-money-rank", h.GetIndustryMoneyRank)
		market.GET("/money-rank", h.GetMoneyRank)
		market.GET("/stock-money-trend", h.GetStockMoneyTrend)

		// 新闻
		market.GET("/news24h", h.GetNews24h)
		market.GET("/sina-news", h.GetSinaNews)
		market.GET("/stock-news", h.GetStockNews)

		// 研报和公告
		market.GET("/stock-research-report", h.GetStockResearchReport)
		market.GET("/stock-notice", h.GetStockNotice)
		market.GET("/industry-research-report", h.GetIndustryResearchReport)
	}
}

// GetLongTiger 龙虎榜
func (h *MarketHandler) GetLongTiger(c *gin.Context) {
	date := c.Query("date")
	list, err := h.marketRepo.GetLongTigerList(date)
	if err != nil {
		c.JSON(http.StatusOK, Error(err.Error()))
		return
	}
	c.JSON(http.StatusOK, Success(list))
}

// GetHotStock 热门股票
func (h *MarketHandler) GetHotStock(c *gin.Context) {
	source := c.DefaultQuery("source", "xueqiu")
	list, err := h.marketRepo.GetHotStocks(source)
	if err != nil {
		c.JSON(http.StatusOK, Error(err.Error()))
		return
	}
	c.JSON(http.StatusOK, Success(list))
}

// GetHotEvent 热门事件
func (h *MarketHandler) GetHotEvent(c *gin.Context) {
	list, err := h.marketRepo.GetHotEvents()
	if err != nil {
		c.JSON(http.StatusOK, Error(err.Error()))
		return
	}
	c.JSON(http.StatusOK, Success(list))
}

// GetHotTopic 热门话题
func (h *MarketHandler) GetHotTopic(c *gin.Context) {
	list, err := h.marketRepo.GetHotTopics()
	if err != nil {
		c.JSON(http.StatusOK, Error(err.Error()))
		return
	}
	c.JSON(http.StatusOK, Success(list))
}

// GetInvestCalendar 投资日历
func (h *MarketHandler) GetInvestCalendar(c *gin.Context) {
	startDate := c.Query("startDate")
	endDate := c.Query("endDate")
	list, err := h.marketRepo.GetInvestCalendar(startDate, endDate)
	if err != nil {
		c.JSON(http.StatusOK, Error(err.Error()))
		return
	}
	c.JSON(http.StatusOK, Success(list))
}

// GetCLSCalendar 财联社日历
func (h *MarketHandler) GetCLSCalendar(c *gin.Context) {
	startDate := c.Query("startDate")
	endDate := c.Query("endDate")
	list, err := h.marketRepo.GetCLSCalendar(startDate, endDate)
	if err != nil {
		c.JSON(http.StatusOK, Error(err.Error()))
		return
	}
	c.JSON(http.StatusOK, Success(list))
}

// GetGlobalIndexes 全球指数
func (h *MarketHandler) GetGlobalIndexes(c *gin.Context) {
	list, err := h.marketRepo.GetGlobalIndexes()
	if err != nil {
		c.JSON(http.StatusOK, Error(err.Error()))
		return
	}
	c.JSON(http.StatusOK, Success(list))
}

// GetGlobalIndexesReadable 全球指数（易读版）
func (h *MarketHandler) GetGlobalIndexesReadable(c *gin.Context) {
	h.GetGlobalIndexes(c)
}

// GetIndustryRank 行业涨幅排名
func (h *MarketHandler) GetIndustryRank(c *gin.Context) {
	sort := c.DefaultQuery("sort", "0")
	count, _ := strconv.Atoi(c.DefaultQuery("count", "150"))
	list, err := h.marketRepo.GetIndustryRank(sort, count)
	if err != nil {
		c.JSON(http.StatusOK, Error(err.Error()))
		return
	}
	c.JSON(http.StatusOK, Success(list))
}

// GetIndustryMoneyRank 行业资金排名
func (h *MarketHandler) GetIndustryMoneyRank(c *gin.Context) {
	fenlei := c.DefaultQuery("fenlei", "0")
	sort := c.DefaultQuery("sort", "netamount")
	list, err := h.marketRepo.GetIndustryMoneyRank(fenlei, sort)
	if err != nil {
		c.JSON(http.StatusOK, Error(err.Error()))
		return
	}
	c.JSON(http.StatusOK, Success(list))
}

// GetMoneyRank 股票资金排名
func (h *MarketHandler) GetMoneyRank(c *gin.Context) {
	sort := c.DefaultQuery("sort", "netamount")
	list, err := h.marketRepo.GetStockMoneyRank(sort)
	if err != nil {
		c.JSON(http.StatusOK, Error(err.Error()))
		return
	}
	c.JSON(http.StatusOK, Success(list))
}

// GetStockMoneyTrend 个股资金趋势
func (h *MarketHandler) GetStockMoneyTrend(c *gin.Context) {
	code := c.Query("code")
	if code == "" {
		c.JSON(http.StatusOK, Error("code is required"))
		return
	}
	list, err := h.marketRepo.GetStockMoneyTrend(code)
	if err != nil {
		c.JSON(http.StatusOK, Error(err.Error()))
		return
	}
	c.JSON(http.StatusOK, Success(list))
}

// GetNews24h 24小时新闻
func (h *MarketHandler) GetNews24h(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "50"))
	list, total, err := h.marketRepo.GetNews24h(page, pageSize)
	if err != nil {
		c.JSON(http.StatusOK, Error(err.Error()))
		return
	}
	c.JSON(http.StatusOK, PageData(list, total, page, pageSize))
}

// GetStockNews 个股新闻
func (h *MarketHandler) GetStockNews(c *gin.Context) {
	code := c.Query("code")
	if code == "" {
		c.JSON(http.StatusOK, Error("code is required"))
		return
	}
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "50"))
	list, total, err := h.marketRepo.GetStockNews(code, page, pageSize)
	if err != nil {
		c.JSON(http.StatusOK, Error(err.Error()))
		return
	}
	c.JSON(http.StatusOK, PageData(list, total, page, pageSize))
}

// GetSinaNews 新浪财经快讯
func (h *MarketHandler) GetSinaNews(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))
	list, total, err := h.marketRepo.GetSinaNews(page, pageSize)
	if err != nil {
		c.JSON(http.StatusOK, Error(err.Error()))
		return
	}
	c.JSON(http.StatusOK, PageData(list, total, page, pageSize))
}

// GetStockResearchReport 个股研报
func (h *MarketHandler) GetStockResearchReport(c *gin.Context) {
	code := c.Query("code")
	if code == "" {
		c.JSON(http.StatusOK, Error("code is required"))
		return
	}
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "50"))
	list, total, err := h.marketRepo.GetStockResearchReport(code, page, pageSize)
	if err != nil {
		c.JSON(http.StatusOK, Error(err.Error()))
		return
	}
	c.JSON(http.StatusOK, PageData(list, total, page, pageSize))
}

// GetStockNotice 个股公告
func (h *MarketHandler) GetStockNotice(c *gin.Context) {
	code := c.Query("code")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "50"))
	list, total, err := h.marketRepo.GetStockNotice(code, page, pageSize)
	if err != nil {
		c.JSON(http.StatusOK, Error(err.Error()))
		return
	}
	c.JSON(http.StatusOK, PageData(list, total, page, pageSize))
}

// GetIndustryResearchReport 行业研报
func (h *MarketHandler) GetIndustryResearchReport(c *gin.Context) {
	industry := c.Query("industry")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "50"))
	list, total, err := h.marketRepo.GetIndustryResearchReport(industry, page, pageSize)
	if err != nil {
		c.JSON(http.StatusOK, Error(err.Error()))
		return
	}
	c.JSON(http.StatusOK, PageData(list, total, page, pageSize))
}
