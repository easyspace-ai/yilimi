package datacollect

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/easyspace-ai/stock_api/pkg/tsdb"
)

// NormDateISO 将输入规范为 YYYY-MM-DD；已是该格式则原样返回日期部分。
func NormDateISO(s string) (string, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return "", fmt.Errorf("empty date")
	}
	if len(s) == 10 && s[4] == '-' && s[7] == '-' {
		if _, err := time.Parse("2006-01-02", s); err != nil {
			return "", err
		}
		return s, nil
	}
	if len(s) == 8 && strings.IndexFunc(s, func(r rune) bool { return r < '0' || r > '9' }) < 0 {
		t, err := time.Parse("20060102", s)
		if err != nil {
			return "", err
		}
		return t.Format("2006-01-02"), nil
	}
	return "", fmt.Errorf("unsupported date format: %s", s)
}

// ToYYYYMMDD 将 YYYY-MM-DD 转为 YYYYMMDD。
func ToYYYYMMDD(iso string) string {
	return strings.ReplaceAll(iso, "-", "")
}

// PrevCalendarRangeYYYYMMDD 返回 [start,end] 各为 YYYYMMDD，覆盖 end 前 lookup 个日历日。
func PrevCalendarRangeYYYYMMDD(endISO string, lookup int) (startY, endY string, err error) {
	end, err := time.Parse("2006-01-02", endISO)
	if err != nil {
		return "", "", err
	}
	start := end.AddDate(0, 0, -lookup)
	return start.Format("20060102"), end.Format("20060102"), nil
}

// ResolveTradeDate 将请求日回退到 endISO 及之前的最近 A 股交易日（使用 trade_cal）。
func ResolveTradeDate(ctx context.Context, client *tsdb.UnifiedClient, endISO string) (resolvedISO string, resolvedYMD string, err error) {
	if client == nil {
		return endISO, ToYYYYMMDD(endISO), nil
	}
	end, err := time.Parse("2006-01-02", endISO)
	if err != nil {
		return "", "", err
	}
	start := end.AddDate(0, 0, -120)
	startY := start.Format("20060102")
	endY := end.Format("20060102")

	df, err := client.GetTradeCalendar(ctx, tsdb.TradeCalendarFilter{
		StartDate: startY,
		EndDate:   endY,
	})
	if err != nil || df == nil || len(df.Rows) == 0 {
		// 无日历则退回请求日（或简单周末回退可由调用方处理）
		return endISO, endY, nil
	}

	type row struct {
		d   string
		iso string
	}
	var opens []row
	for _, m := range df.Rows {
		raw, ok := m["cal_date"]
		if !ok {
			raw = m["trade_date"]
		}
		ds := normCalCell(raw)
		if ds == "" {
			continue
		}
		isOpen := true
		if v, ok := m["is_open"]; ok {
			switch t := v.(type) {
			case float64:
				isOpen = t != 0
			case int:
				isOpen = t != 0
			case int64:
				isOpen = t != 0
			case string:
				isOpen = t == "1" || strings.EqualFold(t, "true")
			default:
				s := strings.TrimSpace(fmt.Sprint(v))
				isOpen = s == "1" || strings.EqualFold(s, "true")
			}
		}
		if !isOpen {
			continue
		}
		iso := ymdToISO(ds)
		opens = append(opens, row{d: ds, iso: iso})
	}
	if len(opens) == 0 {
		return endISO, endY, nil
	}
	sort.Slice(opens, func(i, j int) bool { return opens[i].d < opens[j].d })
	endCmp := strings.ReplaceAll(endISO, "-", "")
	var pick row
	for i := len(opens) - 1; i >= 0; i-- {
		if opens[i].d <= endCmp {
			pick = opens[i]
			break
		}
	}
	if pick.d == "" {
		pick = opens[len(opens)-1]
	}
	return pick.iso, pick.d, nil
}

func normCalCell(v any) string {
	switch t := v.(type) {
	case string:
		s := strings.TrimSpace(t)
		s = strings.ReplaceAll(s, "-", "")
		if len(s) >= 8 {
			return s[:8]
		}
		return s
	case float64:
		return fmt.Sprintf("%.0f", t)
	default:
		s := strings.TrimSpace(fmt.Sprint(v))
		s = strings.ReplaceAll(s, "-", "")
		if len(s) >= 8 {
			return s[:8]
		}
		return s
	}
}

func ymdToISO(ymd string) string {
	if len(ymd) != 8 {
		return ymd
	}
	return ymd[:4] + "-" + ymd[4:6] + "-" + ymd[6:8]
}
