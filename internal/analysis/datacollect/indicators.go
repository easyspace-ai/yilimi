package datacollect

import (
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"

	"github.com/easyspace-ai/stock_api/pkg/realtimedata"
	"github.com/easyspace-ai/stock_api/pkg/tsdb"
)

// BuildIndicatorsBlock 从日线 DataFrame 计算与 Python 分析师相近的一组末端指标。
func BuildIndicatorsBlock(df *tsdb.DataFrame) string {
	highs, lows, closes, vols, ok := rowsOHLCV(df)
	if !ok || len(closes) < 2 {
		return "（K 线不足，无法计算技术指标）"
	}
	last := len(closes) - 1
	var b strings.Builder

	write := func(name string, v float64, valid bool) {
		if valid && !math.IsNaN(v) && !math.IsInf(v, 0) {
			b.WriteString(fmt.Sprintf("【%s】\n%.4f\n\n", name, v))
		} else {
			b.WriteString(fmt.Sprintf("【%s】\nN/A\n\n", name))
		}
	}

	// close_50_sma / close_200_sma / close_10_ema
	ma50 := realtimedata.CalculateMA(closes, 50)
	ma200 := realtimedata.CalculateMA(closes, 200)
	ema10 := realtimedata.CalculateEMA(closes, 10)
	write("close_50_sma", ma50[last], last >= 49)
	write("close_200_sma", ma200[last], last >= 199)
	write("close_10_ema", ema10[last], last >= 9)

	rsi := realtimedata.CalculateRSI(closes, 14)
	if last >= 14 && !math.IsNaN(rsi[last]) {
		write("rsi", rsi[last], true)
	} else {
		write("rsi", 0, false)
	}

	macd := realtimedata.CalculateMACD(closes, 12, 26, 9)
	if macd != nil && len(macd.MACD) > last && len(macd.DIF) > last && len(macd.DEA) > last {
		write("macd_histogram", macd.MACD[last], true)
		write("macd_dif", macd.DIF[last], true)
		write("macd_dea", macd.DEA[last], true)
	} else {
		write("macd_histogram", 0, false)
	}

	boll := realtimedata.CalculateBOLL(closes, 20, 2)
	if boll != nil && len(boll.Middle) > last && len(boll.Upper) > last && len(boll.Lower) > last {
		write("boll_mid", boll.Middle[last], last >= 19)
		write("boll_ub", boll.Upper[last], last >= 19)
		write("boll_lb", boll.Lower[last], last >= 19)
	}

	atr := simpleATR(highs, lows, closes, 14)
	write("atr", atr[last], last >= 14)

	vwma := vwmaLast(closes, vols, 20)
	write("vwma", vwma, last >= 19)

	return strings.TrimSpace(b.String())
}

func rowsOHLCV(df *tsdb.DataFrame) (highs, lows, closes, vols []float64, ok bool) {
	if df == nil || len(df.Rows) == 0 {
		return nil, nil, nil, nil, false
	}
	// copy & sort by date
	r := make([]map[string]any, len(df.Rows))
	copy(r, df.Rows)
	sort.Slice(r, func(i, j int) bool {
		di := normCalCell(r[i][pickCol(r[i], "trade_date", "cal_date")])
		dj := normCalCell(r[j][pickCol(r[j], "trade_date", "cal_date")])
		return di < dj
	})
	var hi, lo, cl, vo []float64
	for _, m := range r {
		h := pickFloat(m, "high")
		l := pickFloat(m, "low")
		c := pickFloat(m, "close")
		v := pickFloat(m, "vol", "volume")
		hi = append(hi, h)
		lo = append(lo, l)
		cl = append(cl, c)
		vo = append(vo, v)
	}
	return hi, lo, cl, vo, true
}

func pickCol(m map[string]any, keys ...string) string {
	lower := make(map[string]any, len(m))
	for k, v := range m {
		lower[strings.ToLower(k)] = v
	}
	for _, k := range keys {
		if v, ok := lower[strings.ToLower(k)]; ok {
			return fmt.Sprint(v)
		}
	}
	return ""
}

func pickFloat(m map[string]any, keys ...string) float64 {
	lower := make(map[string]any, len(m))
	for k, v := range m {
		lower[strings.ToLower(k)] = v
	}
	for _, k := range keys {
		v, ok := lower[strings.ToLower(k)]
		if !ok {
			continue
		}
		switch t := v.(type) {
		case float64:
			return t
		case float32:
			return float64(t)
		case int:
			return float64(t)
		case int64:
			return float64(t)
		case string:
			f, _ := strconv.ParseFloat(strings.TrimSpace(t), 64)
			return f
		default:
			f, _ := strconv.ParseFloat(strings.TrimSpace(fmt.Sprint(v)), 64)
			return f
		}
	}
	return math.NaN()
}

func simpleATR(highs, lows, closes []float64, period int) []float64 {
	n := len(closes)
	out := make([]float64, n)
	for i := range out {
		out[i] = math.NaN()
	}
	if n < period+1 {
		return out
	}
	trs := make([]float64, n)
	for i := 1; i < n; i++ {
		hl := highs[i] - lows[i]
		hc := math.Abs(highs[i] - closes[i-1])
		lc := math.Abs(lows[i] - closes[i-1])
		trs[i] = math.Max(hl, math.Max(hc, lc))
	}
	sum := 0.0
	for i := 1; i <= period; i++ {
		sum += trs[i]
	}
	out[period] = sum / float64(period)
	for i := period + 1; i < n; i++ {
		out[i] = (out[i-1]*float64(period-1) + trs[i]) / float64(period)
	}
	return out
}

func vwmaLast(closes, vols []float64, window int) float64 {
	n := len(closes)
	if n < window {
		return math.NaN()
	}
	var num, den float64
	for i := n - window; i < n; i++ {
		v := vols[i]
		if math.IsNaN(v) || v <= 0 {
			continue
		}
		num += closes[i] * v
		den += v
	}
	if den <= 0 {
		return math.NaN()
	}
	return num / den
}
