package e2e

import (
	"encoding/json"
	"os"
	"regexp"
	"strings"
)

// Verdict L3 机审结果（写入 e2e-verdict.json）。
type Verdict struct {
	AllPass bool              `json:"all_pass"`
	Gates   []GateResult      `json:"gates"`
	Summary string            `json:"summary"`
	Hints   []string          `json:"hints,omitempty"`
	Meta    map[string]string `json:"meta,omitempty"`
}

// GateResult 单项闸门。
type GateResult struct {
	Name   string `json:"name"`
	Pass   bool   `json:"pass"`
	Detail string `json:"detail,omitempty"`
}

var (
	reDigits       = regexp.MustCompile(`\d+\.?\d*`)
	reFunctionCall = regexp.MustCompile(`(?i)<\|FunctionCallBegin\|>`)
	reToolPerm     = regexp.MustCompile(`请(提供|授予).*工具|工具调用权限|请先提供工具`)
	reLegacyGap    = regexp.MustCompile(`未接工具|get_news|get_global_news|get_stock_data`)
)

// EvaluateReports 对 buildResultMap 产出的 result 做静态规则验收。
func EvaluateReports(result map[string]any) Verdict {
	v := Verdict{AllPass: true, Gates: nil, Meta: map[string]string{}}

	if result == nil {
		v.AllPass = false
		v.Summary = "result is nil"
		v.Gates = append(v.Gates, GateResult{Name: "non_nil", Pass: false, Detail: "no result map"})
		return v
	}

	combine := func(keys ...string) string {
		var b strings.Builder
		for _, k := range keys {
			if s, ok := result[k].(string); ok && s != "" {
				if b.Len() > 0 {
					b.WriteByte('\n')
				}
				b.WriteString(s)
			}
		}
		return b.String()
	}

	allText := combine(
		"market_report", "sentiment_report", "news_report", "fundamentals_report",
		"macro_report", "smart_money_report", "game_theory_report", "investment_plan",
		"trader_investment_plan", "final_trade_decision",
	)

	// G1: 禁止伪工具调用与索要权限
	g1 := GateResult{Name: "no_fake_tool_calls", Pass: true}
	if reFunctionCall.MatchString(allText) {
		g1.Pass, g1.Detail = false, "contains <|FunctionCallBegin|>"
	}
	if reToolPerm.MatchString(allText) {
		g1.Pass, g1.Detail = false, "asks for tool permission"
	}
	v.Gates = append(v.Gates, g1)

	// G2: 市场段有实质内容（数字或 OHLC 语义）
	mkt, _ := result["market_report"].(string)
	g2 := GateResult{Name: "market_substantive", Pass: true}
	if strings.TrimSpace(mkt) == "" {
		g2.Pass, g2.Detail = false, "market_report empty"
	} else {
		low := strings.ToLower(mkt)
		hasNum := reDigits.MatchString(mkt)
		hasOHLC := strings.Contains(low, "open") || strings.Contains(low, "high") ||
			strings.Contains(low, "low") || strings.Contains(low, "close") ||
			strings.Contains(low, "成交量") || strings.Contains(low, "volume") ||
			strings.Contains(low, "指标") || strings.Contains(low, "rsi") || strings.Contains(low, "macd")
		if !hasNum && !hasOHLC {
			g2.Pass, g2.Detail = false, "market_report lacks numbers/OHLC hints"
		}
	}
	v.Gates = append(v.Gates, g2)

	// G3: 新闻段无空壳（有长度且不假调用）
	news, _ := result["news_report"].(string)
	g3 := GateResult{Name: "news_present", Pass: strings.TrimSpace(news) != "" && len(strings.TrimSpace(news)) > 80}
	if !g3.Pass {
		g3.Detail = "news_report empty or too short"
	}
	v.Gates = append(v.Gates, g3)

	// G4: 舆情段
	sent, _ := result["sentiment_report"].(string)
	g4 := GateResult{Name: "sentiment_present", Pass: strings.TrimSpace(sent) != "" && len(strings.TrimSpace(sent)) > 80}
	if !g4.Pass {
		g4.Detail = "sentiment_report empty or too short"
	}
	v.Gates = append(v.Gates, g4)

	// G5: 宏观：资金流或资讯关键词 / 最小长度
	macro, _ := result["macro_report"].(string)
	g5 := GateResult{Name: "macro_substantive", Pass: true}
	ms := strings.TrimSpace(macro)
	if len(ms) < 100 {
		g5.Pass, g5.Detail = false, "macro_report too short"
	} else {
		l := strings.ToLower(ms)
		if !strings.Contains(l, "板块") && !strings.Contains(l, "资金") && !strings.Contains(l, "流向") &&
			!strings.Contains(l, "行业") && !strings.Contains(l, "政策") {
			g5.Pass, g5.Detail = false, "macro_report lacks sector/money-flow/policy cues"
		}
	}
	v.Gates = append(v.Gates, g5)

	// G6: 主力：资金/龙虎榜语义
	sm, _ := result["smart_money_report"].(string)
	g6 := GateResult{Name: "smart_money_substantive", Pass: true}
	ss := strings.TrimSpace(sm)
	if len(ss) < 80 {
		g6.Pass, g6.Detail = false, "smart_money_report too short"
	} else {
		l := strings.ToLower(ss)
		if !strings.Contains(l, "资金") && !strings.Contains(l, "龙虎") && !strings.Contains(l, "主力") &&
			!strings.Contains(l, "净流入") {
			g6.Pass, g6.Detail = false, "smart_money_report lacks money-flow / LHB cues"
		}
	}
	v.Gates = append(v.Gates, g6)

	// G7: 基本面 / 日指标
	fund, _ := result["fundamentals_report"].(string)
	g7 := GateResult{Name: "fundamentals_substantive", Pass: true}
	fs := strings.TrimSpace(fund)
	if len(fs) < 80 {
		g7.Pass, g7.Detail = false, "fundamentals_report too short"
	} else {
		l := strings.ToLower(fs)
		if !strings.Contains(l, "pe") && !strings.Contains(l, "pb") && !strings.Contains(l, "市值") &&
			!strings.Contains(l, "估值") && !strings.Contains(l, "市盈率") {
			g7.Pass, g7.Detail = false, "fundamentals_report lacks PE/PB/市值 cues"
		}
	}
	v.Gates = append(v.Gates, g7)

	// G8: data_gaps 不把「旧版未接工具」当主因刷屏（允许真实 API 失败）
	if dg, ok := result["data_gaps"].([]string); ok && len(dg) > 0 {
		g8 := GateResult{Name: "data_gaps_not_legacy", Pass: true}
		for _, line := range dg {
			if reLegacyGap.MatchString(line) {
				g8.Pass, g8.Detail = false, "legacy gap line: "+line
				break
			}
		}
		v.Gates = append(v.Gates, g8)
	}

	for _, g := range v.Gates {
		if !g.Pass {
			v.AllPass = false
		}
	}
	if v.AllPass {
		v.Summary = "all gates passed"
	} else {
		var failed []string
		for _, g := range v.Gates {
			if !g.Pass {
				failed = append(failed, g.Name+": "+g.Detail)
			}
		}
		v.Summary = "failed: " + strings.Join(failed, "; ")
		v.Hints = failureHints(v.Gates)
	}
	return v
}

func failureHints(gates []GateResult) []string {
	var h []string
	for _, g := range gates {
		if g.Pass {
			continue
		}
		switch g.Name {
		case "no_fake_tool_calls":
			h = append(h, "L3: 仍出现伪工具调用→检查 datacollect 注入与分析师 system prompt。")
		case "market_substantive":
			h = append(h, "L3: 市场段空→回看 L1 日线是否成功、Collect 是否 no_ohlcv。")
		case "macro_substantive", "smart_money_substantive", "news_present", "sentiment_present":
			h = append(h, "L3: 分块过短→检查 realtimedata 网络与 pool.Errors；或调大 LLM 超时。")
		case "data_gaps_not_legacy":
			h = append(h, "L3: data_gaps 含旧版话术→清理工具失败文案或修正上游。")
		}
	}
	return h
}

// WriteVerdict 将 Verdict 写入 path。
func WriteVerdict(path string, v Verdict) error {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0o644)
}
