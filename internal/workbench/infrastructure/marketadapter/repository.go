package marketadapter

import (
	md "github.com/easyspace-ai/stock_api/pkg/marketdata"

	"github.com/easyspace-ai/yilimi/internal/workbench/domain/market"
	"github.com/easyspace-ai/yilimi/internal/workbench/ports"
)

// Repository implements ports.MarketRepository by delegating to tusharedb-go marketdata.Client.
type Repository struct {
	client *md.Client
}

// NewRepository wraps a marketdata client.
func NewRepository(client *md.Client) ports.MarketRepository {
	return &Repository{client: client}
}

func (r *Repository) GetLongTigerList(date string) ([]market.LongTigerRank, error) {
	list, err := r.client.GetLongTigerList(date)
	if err != nil {
		return nil, err
	}
	out := make([]market.LongTigerRank, len(list))
	for i, x := range list {
		out[i] = market.LongTigerRank{
			TradeDate:        x.TradeDate,
			SecurityCode:     x.SecurityCode,
			SecuCode:         x.SecuCode,
			SecurityNameAbbr: x.SecurityNameAbbr,
			ClosePrice:       x.ClosePrice,
			AccumAmount:      x.AccumAmount,
			ChangeRate:       x.ChangeRate,
			BillboardBuyAmt:  x.BillboardBuyAmt,
			BillboardSellAmt: x.BillboardSellAmt,
			BillboardNetAmt:  x.BillboardNetAmt,
			BillboardDealAmt: x.BillboardDealAmt,
			Explanation:      x.Explanation,
			TurnoverRate:     x.TurnoverRate,
			FreeMarketCap:    x.FreeMarketCap,
		}
	}
	return out, nil
}

func (r *Repository) GetHotStocks(source string) ([]market.HotStock, error) {
	list, err := r.client.GetHotStocks(source)
	if err != nil {
		return nil, err
	}
	out := make([]market.HotStock, len(list))
	for i, x := range list {
		out[i] = market.HotStock{
			Code: x.Code, Name: x.Name, Value: x.Value, Increment: x.Increment,
			RankChange: x.RankChange, Percent: x.Percent, Current: x.Current, Chg: x.Chg, Exchange: x.Exchange,
		}
	}
	return out, nil
}

func (r *Repository) GetHotEvents() ([]market.HotEvent, error) {
	list, err := r.client.GetHotEvents()
	if err != nil {
		return nil, err
	}
	out := make([]market.HotEvent, len(list))
	for i, x := range list {
		out[i] = market.HotEvent{
			ID: x.ID, Title: x.Title, Content: x.Content, Tag: x.Tag, Pic: x.Pic, Hot: x.Hot, StatusCount: x.StatusCount,
		}
	}
	return out, nil
}

func (r *Repository) GetHotTopics() ([]market.HotTopic, error) {
	list, err := r.client.GetHotTopics()
	if err != nil {
		return nil, err
	}
	out := make([]market.HotTopic, len(list))
	for i, x := range list {
		out[i] = market.HotTopic{
			ID: x.ID, Title: x.Title, Content: x.Content, Hot: x.Hot, StockCount: x.StockCount,
		}
	}
	return out, nil
}

func (r *Repository) GetNews24h(page, pageSize int) ([]market.MarketNews, int64, error) {
	list, total, err := r.client.GetNews24h(page, pageSize)
	if err != nil {
		return nil, 0, err
	}
	return toDomainNews(list), total, nil
}

func (r *Repository) GetSinaNews(page, pageSize int) ([]market.MarketNews, int64, error) {
	list, total, err := r.client.GetSinaNews(page, pageSize)
	if err != nil {
		return nil, 0, err
	}
	return toDomainNews(list), total, nil
}

func (r *Repository) GetStockNews(code string, page, pageSize int) ([]market.MarketNews, int64, error) {
	list, total, err := r.client.GetStockNews(code, page, pageSize)
	if err != nil {
		return nil, 0, err
	}
	return toDomainNews(list), total, nil
}

func toDomainNews(list []md.MarketNews) []market.MarketNews {
	out := make([]market.MarketNews, len(list))
	for i, x := range list {
		out[i] = market.MarketNews{
			ID: x.ID, Title: x.Title, Content: x.Content, Source: x.Source, Url: x.Url,
			PublishTime: x.PublishTime, StockCodes: x.StockCodes, Tags: x.Tags,
		}
	}
	return out
}

func (r *Repository) GetStockResearchReport(code string, page, pageSize int) ([]market.ResearchReport, int64, error) {
	list, total, err := r.client.GetStockResearchReport(code, page, pageSize)
	if err != nil {
		return nil, 0, err
	}
	return toDomainReports(list), total, nil
}

func (r *Repository) GetIndustryResearchReport(industry string, page, pageSize int) ([]market.ResearchReport, int64, error) {
	list, total, err := r.client.GetIndustryResearchReport(industry, page, pageSize)
	if err != nil {
		return nil, 0, err
	}
	return toDomainReports(list), total, nil
}

func toDomainReports(list []md.ResearchReport) []market.ResearchReport {
	out := make([]market.ResearchReport, len(list))
	for i, x := range list {
		out[i] = market.ResearchReport{
			ID: x.ID, Title: x.Title, Content: x.Content, StockCode: x.StockCode, StockName: x.StockName,
			Author: x.Author, OrgName: x.OrgName, PublishDate: x.PublishDate, ReportType: x.ReportType, Url: x.Url,
		}
	}
	return out
}

func (r *Repository) GetStockNotice(code string, page, pageSize int) ([]market.StockNotice, int64, error) {
	list, total, err := r.client.GetStockNotice(code, page, pageSize)
	if err != nil {
		return nil, 0, err
	}
	out := make([]market.StockNotice, len(list))
	for i, x := range list {
		out[i] = market.StockNotice{
			ID: x.ID, Title: x.Title, Content: x.Content, StockCode: x.StockCode, StockName: x.StockName,
			NoticeType: x.NoticeType, PublishDate: x.PublishDate, UpdateTime: x.UpdateTime, Url: x.Url,
		}
	}
	return out, total, nil
}

func (r *Repository) GetIndustryRank(sort string, count int) ([]market.IndustryRank, error) {
	list, err := r.client.GetIndustryRank(sort, count)
	if err != nil {
		return nil, err
	}
	out := make([]market.IndustryRank, len(list))
	for i, x := range list {
		out[i] = market.IndustryRank{
			IndustryName: x.IndustryName, IndustryCode: x.IndustryCode, ChangePct: x.ChangePct,
			ChangePct5d: x.ChangePct5d, ChangePct20d: x.ChangePct20d, LeadStock: x.LeadStock,
			LeadStockCode: x.LeadStockCode, LeadChange: x.LeadChange, LeadPrice: x.LeadPrice,
		}
	}
	return out, nil
}

func (r *Repository) GetIndustryMoneyRank(fenlei, sort string) ([]market.IndustryMoneyRank, error) {
	list, err := r.client.GetIndustryMoneyRank(fenlei, sort)
	if err != nil {
		return nil, err
	}
	out := make([]market.IndustryMoneyRank, len(list))
	for i, x := range list {
		out[i] = market.IndustryMoneyRank{
			IndustryName: x.IndustryName, ChangePct: x.ChangePct, Inflow: x.Inflow, Outflow: x.Outflow,
			NetInflow: x.NetInflow, NetRatio: x.NetRatio, LeadStock: x.LeadStock, LeadStockCode: x.LeadStockCode,
			LeadChange: x.LeadChange, LeadPrice: x.LeadPrice, LeadNetRatio: x.LeadNetRatio,
		}
	}
	return out, nil
}

func (r *Repository) GetStockMoneyRank(sort string) ([]market.StockMoneyRank, error) {
	list, err := r.client.GetStockMoneyRank(sort)
	if err != nil {
		return nil, err
	}
	out := make([]market.StockMoneyRank, len(list))
	for i, x := range list {
		out[i] = market.StockMoneyRank{
			Code: x.Code, Name: x.Name, Price: x.Price, ChangePct: x.ChangePct, TurnoverRate: x.TurnoverRate,
			Amount: x.Amount, OutAmount: x.OutAmount, InAmount: x.InAmount, NetAmount: x.NetAmount, NetRatio: x.NetRatio,
			R0Out: x.R0Out, R0In: x.R0In, R0Net: x.R0Net, R0Ratio: x.R0Ratio,
			R3Out: x.R3Out, R3In: x.R3In, R3Net: x.R3Net, R3Ratio: x.R3Ratio,
		}
	}
	return out, nil
}

func (r *Repository) GetStockMoneyTrend(code string) ([]market.MoneyFlowInfo, error) {
	list, err := r.client.GetStockMoneyTrend(code)
	if err != nil {
		return nil, err
	}
	out := make([]market.MoneyFlowInfo, len(list))
	for i, x := range list {
		out[i] = market.MoneyFlowInfo{
			Date: x.Date, MainNetInflow: x.MainNetInflow, MainNetRatio: x.MainNetRatio,
			SuperLargeNetInflow: x.SuperLargeNetInflow, LargeNetInflow: x.LargeNetInflow,
			MediumNetInflow: x.MediumNetInflow, SmallNetInflow: x.SmallNetInflow,
		}
	}
	return out, nil
}

func (r *Repository) GetGlobalIndexes() ([]market.GlobalIndex, error) {
	list, err := r.client.GetGlobalIndexes()
	if err != nil {
		return nil, err
	}
	out := make([]market.GlobalIndex, len(list))
	for i, x := range list {
		out[i] = market.GlobalIndex{
			Name: x.Name, Code: x.Code, Price: x.Price, Change: x.Change, ChangePct: x.ChangePct, UpdateTime: x.UpdateTime,
			Region: x.Region,
		}
	}
	return out, nil
}

func (r *Repository) GetInvestCalendar(startDate, endDate string) ([]market.InvestCalendarItem, error) {
	list, err := r.client.GetInvestCalendar(startDate, endDate)
	if err != nil {
		return nil, err
	}
	return toDomainCalendar(list), nil
}

func (r *Repository) GetCLSCalendar(startDate, endDate string) ([]market.InvestCalendarItem, error) {
	list, err := r.client.GetCLSCalendar(startDate, endDate)
	if err != nil {
		return nil, err
	}
	return toDomainCalendar(list), nil
}

func toDomainCalendar(list []md.InvestCalendarItem) []market.InvestCalendarItem {
	out := make([]market.InvestCalendarItem, len(list))
	for i, x := range list {
		out[i] = market.InvestCalendarItem{Date: x.Date, Title: x.Title, Content: x.Content, Type: x.Type}
	}
	return out
}
