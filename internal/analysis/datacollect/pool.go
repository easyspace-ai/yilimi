package datacollect

// PoolMeta 采集窗口与交易日解析元信息。
type PoolMeta struct {
	RequestedTradeDateISO string // 用户请求 YYYY-MM-DD
	ResolvedTradeDateISO  string // 解析后的最近交易日 YYYY-MM-DD
	ResolvedYYYYMMDD      string // 同上 YYYYMMDD
	FetchLookbackDays     int    // 拉取日线向前日历天数（对标 Python ~365）
	NewsLookbackDays      int    // 新闻窗口（对标 90）
}

// Pool 对标 Python DataCollector 单轮缓存，供各分析师注入。
type Pool struct {
	Meta PoolMeta

	StockDataText string // OHLCV 可读文本
	Indicators    string // 技术指标块
	StockBasic    string
	DailyBasic    string
	News          string // 个股资讯
	GlobalNews    string // 市场资讯（global_news 近似）
	FundFlowBoard string // 行业板块资金流向排名
	FundFlowStock string // 个股资金流向
	LHB           string // 龙虎榜（个股）
	HotTopics     string // 热门话题（可选）

	FundamentalsNote string // 财报三表缺失说明

	Errors map[string]string // 键如 stock_daily, news；供调试与 data_gaps
}

// DataGaps 返回人类可读的数据缺口列表（仅真实错误）。
func (p *Pool) DataGaps() []string {
	if p == nil || len(p.Errors) == 0 {
		return nil
	}
	out := make([]string, 0, len(p.Errors))
	for k, v := range p.Errors {
		out = append(out, k+": "+v)
	}
	return out
}
