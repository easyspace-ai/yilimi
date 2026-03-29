package datacollect

import (
	"fmt"
	"strings"

	"github.com/easyspace-ai/stock_api/pkg/realtimedata"
	"github.com/easyspace-ai/stock_api/pkg/tsdb"
)

func formatDataFrame(df *tsdb.DataFrame, maxRows int) string {
	if df == nil || len(df.Rows) == 0 {
		return "无数据"
	}
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("共 %d 条，列：%v\n\n", len(df.Rows), df.Columns))
	limit := len(df.Rows)
	if maxRows > 0 && limit > maxRows {
		limit = maxRows
	}
	for i := 0; i < limit; i++ {
		sb.WriteString(fmt.Sprintf("%d. %v\n", i+1, df.Rows[i]))
	}
	if len(df.Rows) > limit {
		sb.WriteString(fmt.Sprintf("...（省略 %d 条）\n", len(df.Rows)-limit))
	}
	return sb.String()
}

// formatDataFrameTail 节选时间序列末尾若干条（用于 OHLCV：分析师需看最近行情而非最旧片段）。
func formatDataFrameTail(df *tsdb.DataFrame, maxRows int) string {
	if df == nil || len(df.Rows) == 0 {
		return "无数据"
	}
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("共 %d 条，列：%v\n", len(df.Rows), df.Columns))
	if maxRows > 0 && len(df.Rows) > maxRows {
		skip := len(df.Rows) - maxRows
		sb.WriteString(fmt.Sprintf("（以下节选最近 %d 条，省略较早的 %d 条）\n\n", maxRows, skip))
		for i := skip; i < len(df.Rows); i++ {
			sb.WriteString(fmt.Sprintf("%d. %v\n", i-skip+1, df.Rows[i]))
		}
		return sb.String()
	}
	sb.WriteString("\n")
	for i := 0; i < len(df.Rows); i++ {
		sb.WriteString(fmt.Sprintf("%d. %v\n", i+1, df.Rows[i]))
	}
	return sb.String()
}

func formatNewsList(items []realtimedata.News, max int) string {
	if len(items) == 0 {
		return "（近期无抓取到个股资讯条目）"
	}
	var sb strings.Builder
	n := len(items)
	if max > 0 && n > max {
		n = max
	}
	for i := 0; i < n; i++ {
		it := items[i]
		sb.WriteString(fmt.Sprintf("%d. [%s] %s\n", i+1, it.Time, it.Title))
		if strings.TrimSpace(it.Source) != "" {
			sb.WriteString(fmt.Sprintf("   来源：%s\n", it.Source))
		}
		if u := strings.TrimSpace(it.Url); u != "" {
			sb.WriteString(fmt.Sprintf("   链接：%s\n", u))
		}
	}
	return sb.String()
}

func formatSectorMoneyFlows(items []realtimedata.SectorMoneyFlow, max int) string {
	if len(items) == 0 {
		return "（未获取到板块资金流向数据）"
	}
	var sb strings.Builder
	sb.WriteString("行业/概念板块资金流向（节选）：\n")
	n := len(items)
	if max > 0 && n > max {
		n = max
	}
	for i := 0; i < n; i++ {
		x := items[i]
		sb.WriteString(fmt.Sprintf("%d. %s 净流入=%.2f 净流入占比=%.2f%% 领涨股=%s\n",
			i+1, x.Name, x.NetInflow, x.NetPct, x.LeadStock))
	}
	return sb.String()
}

func formatMoneyFlow(m *realtimedata.MoneyFlow) string {
	if m == nil {
		return "（未获取到个股资金流向）"
	}
	return fmt.Sprintf(`代码：%s 名称：%s
主力净额：%.2f 主力净占比：%.2f%%
超大单净额：%.2f 大单净额：%.2f 中单净额：%.2f 小单净额：%.2f
更新：%s`,
		m.Code, m.Name, m.MainNetInflow, m.MainNetPct,
		m.SuperLargeInflow, m.LargeInflow, m.MediumInflow, m.SmallInflow,
		m.Time)
}

func formatDragonTigerDetails(items []realtimedata.DragonTigerDetail) string {
	if len(items) == 0 {
		return "（该标的近期无龙虎榜明细或当前源未返回数据；非上榜日属正常）"
	}
	var sb strings.Builder
	for i, d := range items {
		if i >= 5 {
			break
		}
		sb.WriteString(fmt.Sprintf("日期：%s 原因：%s\n", d.Date, d.Reason))
		sb.WriteString(fmt.Sprintf("  买方席位数：%d 卖方席位数：%d\n", len(d.BuyList), len(d.SellList)))
	}
	return sb.String()
}

func formatHotTopics(items []realtimedata.HotTopic, max int) string {
	if len(items) == 0 {
		return ""
	}
	var sb strings.Builder
	sb.WriteString("市场热门话题（节选）：\n")
	n := len(items)
	if max > 0 && n > max {
		n = max
	}
	for i := 0; i < n; i++ {
		t := items[i]
		sb.WriteString(fmt.Sprintf("%d. %s 热度=%d\n", i+1, t.Title, t.Count))
	}
	return sb.String()
}
