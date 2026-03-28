package ports

import (
	"github.com/easyspace-ai/yilimi/internal/workbench/domain/stock"
)

// StockRepository 股票仓储接口
type StockRepository interface {
	// 基础查询
	GetByCode(code string) (*stock.StockInfo, error)
	GetByCodes(codes []string) ([]stock.StockInfo, error)
	List(market, industry, concept string, page, pageSize int) ([]stock.StockInfo, int64, error)
	Search(keyword string) ([]stock.StockInfo, error)

	// 实时行情
	GetQuote(code string) (*stock.StockQuote, error)
	GetQuotes(codes []string) ([]stock.StockQuote, error)

	// K线数据
	GetKLine(code string, klineType string, days int, adjustFlag string) (*stock.KLineData, error)
	GetCommonKLine(code string, klineType string, days int) (*stock.KLineData, error)
	GetMinutePrice(code string) ([]stock.KLineItem, error)

	// 资金流向
	GetMoneyHistory(code string) ([]stock.MoneyFlowInfo, error)
	GetMoneyTrend(code string) ([]stock.MoneyFlowInfo, error)

	// 财务信息
	GetFinancialInfo(code string) (*stock.FinancialInfo, error)

	// 概念信息
	GetConceptInfo(code string) ([]string, error)

	// 股东人数
	GetHolderNum(code string) (int, error)

	// 融资融券
	GetRZRQ(code string) ([]stock.RZRQInfo, error)

	// 关注股票
	GetFollowedStocks(userId string) ([]stock.FollowedStock, error)
	FollowStock(userId, code, name, note string) error
	UnfollowStock(userId, code string) error
	UpdateCost(userId, code string, costPrice, quantity float64) error

	// 价格预警
	GetAlarms(userId string) ([]stock.StockAlarm, error)
	SetAlarm(userId string, alarm *stock.StockAlarm) error
	DeleteAlarm(userId string, id uint) error

	// 高级筛选
	SelectStocks(criteria *stock.StockSelectionCriteria) (*stock.StockSelectionResult, error)
	GetAllStockInfo(criteria *stock.StockSelectionCriteria, page, pageSize int) ([]stock.AllStockInfo, int64, error)

	// 热门策略
	GetHotStrategies() ([]stock.HotStrategyData, error)

	// 市场列表
	GetMarkets() ([]string, error)
	GetIndustries() ([]string, error)
	GetConcepts() ([]string, error)

	// ========== 选股API ==========
	// GetAllMarketQuotes 获取全市场行情
	GetAllMarketQuotes() ([]*stock.MarketQuote, error)

	// GetTodayTimeline 获取今日分时数据
	GetTodayTimeline(code string) ([]stock.TimelinePoint, error)

	// EndOfDayPicker 尾盘选股
	EndOfDayPicker(req *stock.EndOfDayPickerRequest) (*stock.PickerResponse, error)

	// MomentumPicker 妖股候选人扫描
	MomentumPicker(req *stock.MomentumPickerRequest) (*stock.PickerResponse, error)

	// KunpengPicker 鲲鹏战法筛选
	KunpengPicker(req *stock.KunpengPickerRequest) (*stock.PickerResponse, error)
}
