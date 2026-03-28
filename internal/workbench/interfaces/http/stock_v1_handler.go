package httpapi

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	stocklib "github.com/easyspace-ai/stock_api/pkg/stockapi"
)

type StockV1Handler struct {
	client *stocklib.Client
}

func NewStockV1Handler(client *stocklib.Client) *StockV1Handler {
	return &StockV1Handler{client: client}
}

func (h *StockV1Handler) RegisterRoutes(r *gin.RouterGroup) {
	r.GET("/quotes/all", h.GetAllAShareQuotes)
	r.GET("/kline/history", h.GetHistoryKline)
	r.GET("/timeline/:symbol", h.GetTodayTimeline)
	r.POST("/timeline/batch", h.GetTodayTimelineBatch)
}

func (h *StockV1Handler) GetAllAShareQuotes(c *gin.Context) {
	quotes, err := h.client.GetAllAShareQuotes(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Code:    500,
			Message: "Failed to get quotes: " + err.Error(),
		})
		return
	}
	c.JSON(http.StatusOK, Success(quotes))
}

type v1KlineData struct {
	Date   string  `json:"date"`
	Open   float64 `json:"open"`
	High   float64 `json:"high"`
	Low    float64 `json:"low"`
	Close  float64 `json:"close"`
	Volume float64 `json:"volume"`
	Amount float64 `json:"amount"`
}

func (h *StockV1Handler) GetHistoryKline(c *gin.Context) {
	symbol := strings.TrimSpace(c.Query("symbol"))
	if symbol == "" {
		c.JSON(http.StatusBadRequest, Response{Code: 400, Message: "symbol is required"})
		return
	}
	items, err := h.client.GetHistoryKline(
		c.Request.Context(),
		symbol,
		strings.TrimSpace(c.DefaultQuery("period", "daily")),
		strings.TrimSpace(c.DefaultQuery("adjust", "")),
		strings.TrimSpace(c.Query("start_date")),
		strings.TrimSpace(c.Query("end_date")),
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Code:    500,
			Message: "Failed to get kline: " + err.Error(),
		})
		return
	}

	result := make([]v1KlineData, 0, len(items))
	for _, item := range items {
		result = append(result, v1KlineData{
			Date:   item.Date,
			Open:   item.Open,
			High:   item.High,
			Low:    item.Low,
			Close:  item.Close,
			Volume: item.Volume,
			Amount: item.Amount,
		})
	}

	c.JSON(http.StatusOK, Success(result))
}

type v1TimelineData struct {
	Time     string  `json:"time"`
	Price    float64 `json:"price"`
	AvgPrice float64 `json:"avgPrice"`
	Volume   float64 `json:"volume"`
}

type v1TimelineResponse struct {
	Symbol    string           `json:"symbol"`
	PrevClose float64          `json:"prevClose"`
	Data      []v1TimelineData `json:"data"`
}

func (h *StockV1Handler) GetTodayTimeline(c *gin.Context) {
	symbol := strings.TrimSpace(c.Param("symbol"))
	if symbol == "" {
		c.JSON(http.StatusBadRequest, Response{Code: 400, Message: "symbol is required"})
		return
	}

	timeline, err := h.client.GetTodayTimeline(c.Request.Context(), symbol)
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Code:    500,
			Message: "Failed to get timeline: " + err.Error(),
		})
		return
	}

	data := make([]v1TimelineData, 0, len(timeline.Data))
	for _, item := range timeline.Data {
		data = append(data, v1TimelineData{
			Time:     item.Time,
			Price:    item.Price,
			AvgPrice: item.AvgPrice,
			Volume:   item.Volume,
		})
	}

	c.JSON(http.StatusOK, Success(v1TimelineResponse{
		Symbol:    symbol,
		PrevClose: timeline.PrevClose,
		Data:      data,
	}))
}

type v1TimelineBatchRequest struct {
	Symbols []string `json:"symbols"`
	Codes   []string `json:"codes"`
}

func (h *StockV1Handler) GetTodayTimelineBatch(c *gin.Context) {
	var req v1TimelineBatchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, Response{Code: 400, Message: "invalid request body"})
		return
	}

	symbols := req.Symbols
	if len(symbols) == 0 {
		symbols = req.Codes
	}
	if len(symbols) == 0 {
		c.JSON(http.StatusBadRequest, Response{Code: 400, Message: "symbols is required"})
		return
	}

	raw := h.client.GetTodayTimelineBatch(c.Request.Context(), symbols)
	success := make(map[string]v1TimelineResponse, len(raw.Success))
	for symbol, timeline := range raw.Success {
		items := make([]v1TimelineData, 0, len(timeline.Data))
		for _, item := range timeline.Data {
			items = append(items, v1TimelineData{
				Time:     item.Time,
				Price:    item.Price,
				AvgPrice: item.AvgPrice,
				Volume:   item.Volume,
			})
		}
		success[symbol] = v1TimelineResponse{
			Symbol:    symbol,
			PrevClose: timeline.PrevClose,
			Data:      items,
		}
	}

	c.JSON(http.StatusOK, Success(gin.H{
		"success": success,
		"failed":  raw.Failed,
	}))
}
