package ports

import (
	"github.com/easyspace-ai/yilimi/internal/workbench/domain/market"
)

// MarketRepository 市场仓储接口（抓取逻辑已委托给 tusharedb-go/pkg/marketdata）。
type MarketRepository interface {
	// 龙虎榜
	GetLongTigerList(date string) ([]market.LongTigerRank, error)

	// 热门股票
	GetHotStocks(source string) ([]market.HotStock, error)

	// 热门事件
	GetHotEvents() ([]market.HotEvent, error)

	// 热门话题
	GetHotTopics() ([]market.HotTopic, error)

	// 新闻
	GetNews24h(page, pageSize int) ([]market.MarketNews, int64, error)
	GetSinaNews(page, pageSize int) ([]market.MarketNews, int64, error)
	GetStockNews(code string, page, pageSize int) ([]market.MarketNews, int64, error)

	// 研报
	GetStockResearchReport(code string, page, pageSize int) ([]market.ResearchReport, int64, error)
	GetIndustryResearchReport(industry string, page, pageSize int) ([]market.ResearchReport, int64, error)

	// 公告
	GetStockNotice(code string, page, pageSize int) ([]market.StockNotice, int64, error)

	// 资金排名
	GetIndustryRank(sort string, count int) ([]market.IndustryRank, error)
	GetIndustryMoneyRank(fenlei, sort string) ([]market.IndustryMoneyRank, error)
	GetStockMoneyRank(sort string) ([]market.StockMoneyRank, error)
	GetStockMoneyTrend(code string) ([]market.MoneyFlowInfo, error)

	// 全球指数
	GetGlobalIndexes() ([]market.GlobalIndex, error)

	// 投资日历
	GetInvestCalendar(startDate, endDate string) ([]market.InvestCalendarItem, error)
	GetCLSCalendar(startDate, endDate string) ([]market.InvestCalendarItem, error)
}
