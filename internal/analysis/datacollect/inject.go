package datacollect

import "strings"

const noToolPreamble = `

【硬性规则】
- 下文「数据附录」已包含本次分析所需的主要数据，你不得假装调用 get_news/get_stock_data 等工具，不得在输出中出现 <|FunctionCallBegin|> 或类似伪函数调用。
- 严禁向用户索要「工具权限」或额外 API；若某字段在附录中标明缺失，请说明数据缺口并降低结论置信度。
`

// MarketInstruction 市场分析师完整系统指令。
func (p *Pool) MarketInstruction(base string) string {
	if p == nil {
		return base + noToolPreamble + "\n（数据池为空）"
	}
	var b strings.Builder
	b.WriteString(base)
	b.WriteString(noToolPreamble)
	b.WriteString("\n\n---\n【数据附录】\n")
	b.WriteString(p.StockDataText)
	b.WriteString("\n\n【技术指标（末端）】\n")
	b.WriteString(p.Indicators)
	b.WriteString("\n\n【股票基础信息】\n")
	b.WriteString(p.StockBasic)
	return b.String()
}

// NewsInstruction 新闻分析师。
func (p *Pool) NewsInstruction(base string) string {
	if p == nil {
		return base + noToolPreamble
	}
	var b strings.Builder
	b.WriteString(base)
	b.WriteString(noToolPreamble)
	b.WriteString("\n\n---\n【数据附录 — 个股资讯】\n")
	b.WriteString(p.News)
	b.WriteString("\n\n【数据附录 — 市场/宏观资讯（近似 get_global_news）】\n")
	b.WriteString(p.GlobalNews)
	return b.String()
}

// SentimentInstruction 舆情分析师。
func (p *Pool) SentimentInstruction(base string) string {
	if p == nil {
		return base + noToolPreamble
	}
	var b strings.Builder
	b.WriteString(base)
	b.WriteString(noToolPreamble)
	b.WriteString("\n\n---\n【数据附录 — 可用于情绪线索的资讯】\n")
	b.WriteString(p.News)
	b.WriteString("\n")
	b.WriteString(p.GlobalNews)
	if strings.TrimSpace(p.HotTopics) != "" {
		b.WriteString("\n【热门话题】\n")
		b.WriteString(p.HotTopics)
	}
	return b.String()
}

// FundamentalsInstruction 基本面分析师。
func (p *Pool) FundamentalsInstruction(base string) string {
	if p == nil {
		return base + noToolPreamble
	}
	var b strings.Builder
	b.WriteString(base)
	b.WriteString(noToolPreamble)
	b.WriteString("\n\n---\n【数据说明】\n")
	b.WriteString(p.FundamentalsNote)
	b.WriteString("\n\n【股票基础信息】\n")
	b.WriteString(p.StockBasic)
	b.WriteString("\n\n【日频估值与基本面指标（节选）】\n")
	b.WriteString(p.DailyBasic)
	b.WriteString("\n\n【近期资讯摘要（辅助）】\n")
	b.WriteString(p.News)
	return b.String()
}

// MacroInstruction 宏观分析师。
func (p *Pool) MacroInstruction(base string) string {
	if p == nil {
		return base + noToolPreamble
	}
	var b strings.Builder
	b.WriteString(base)
	b.WriteString(noToolPreamble)
	b.WriteString("\n\n---\n【行业/板块资金流向】\n")
	b.WriteString(p.FundFlowBoard)
	b.WriteString("\n\n【相关新闻与资讯】\n")
	b.WriteString(p.News)
	b.WriteString("\n")
	b.WriteString(p.GlobalNews)
	return b.String()
}

// SmartMoneyInstruction 主力资金分析师。
func (p *Pool) SmartMoneyInstruction(base string) string {
	if p == nil {
		return base + noToolPreamble
	}
	var b strings.Builder
	b.WriteString(base)
	b.WriteString(noToolPreamble)
	b.WriteString("\n\n---\n【个股资金流向】\n")
	b.WriteString(p.FundFlowStock)
	b.WriteString("\n\n【龙虎榜】\n")
	b.WriteString(p.LHB)
	b.WriteString("\n\n【量价与技术末端（节选）】\n")
	b.WriteString(p.Indicators)
	return b.String()
}
