// Package klinecompat 提供日线前复权 K 线在日期区间内的裁剪，供多路由复用（/api/v1/kline 与 /api/v1/analysis/market/kline）。
package klinecompat

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/easyspace-ai/yilimi/internal/workbench/domain/stock"
	"github.com/easyspace-ai/yilimi/internal/workbench/ports"
)

var plainCodePattern = regexp.MustCompile(`^\d{6}$`)
var reFollowPrefixed = regexp.MustCompile(`^(SH|SZ|BJ)(\d{6})$`)
var reFollowSuffixed = regexp.MustCompile(`^(\d{6})\.(SH|SZ|BJ)$`)

// NormalizeFollowCode 统一自选股代码：sh600519 / SH600519 → 600519.SH；已带后缀则规范大写。
func NormalizeFollowCode(raw string) string {
	s := strings.TrimSpace(raw)
	if s == "" {
		return ""
	}
	u := strings.ToUpper(s)
	if m := reFollowSuffixed.FindStringSubmatch(u); len(m) == 3 {
		return m[1] + "." + m[2]
	}
	if m := reFollowPrefixed.FindStringSubmatch(u); len(m) == 3 {
		return m[2] + "." + m[1]
	}
	return NormalizeSymbol(u)
}

// NormalizeSymbol 与 StockHandler 一致：裸 6 位补交易所后缀等。
func NormalizeSymbol(symbol string) string {
	trimmed := strings.ToUpper(strings.TrimSpace(symbol))
	if trimmed == "" {
		return ""
	}
	if plainCodePattern.MatchString(trimmed) {
		if strings.HasPrefix(trimmed, "6") || strings.HasPrefix(trimmed, "5") || strings.HasPrefix(trimmed, "9") {
			return trimmed + ".SH"
		}
		return trimmed + ".SZ"
	}
	if strings.HasPrefix(trimmed, "SH") || strings.HasPrefix(trimmed, "SZ") || strings.HasPrefix(trimmed, "BJ") {
		return trimmed
	}
	if strings.HasSuffix(trimmed, ".SH") || strings.HasSuffix(trimmed, ".SZ") || strings.HasSuffix(trimmed, ".BJ") {
		return trimmed
	}
	return strings.TrimSpace(symbol)
}

// ParseKlineDate 解析 YYYY-MM-DD。
func ParseKlineDate(value string) (time.Time, error) {
	return time.Parse("2006-01-02", value)
}

// DailyBarsInRange 拉取日线前复权数据并保留 [startTime, endTime] 内的 bar（按日期字符串比较）。
func DailyBarsInRange(repo ports.StockRepository, rawSymbol string, startTime, endTime time.Time) ([]stock.KLineItem, error) {
	symbol := NormalizeSymbol(rawSymbol)
	data, err := repo.GetKLine(symbol, "day", 500, "qfq")
	if err != nil {
		data, err = repo.GetKLine(symbol, "day", 500, "")
	}
	if err != nil {
		return nil, err
	}
	out := make([]stock.KLineItem, 0, len(data.List))
	for _, item := range data.List {
		itemTime, parseErr := ParseKlineDate(item.Date)
		if parseErr != nil {
			continue
		}
		if itemTime.Before(startTime) || itemTime.After(endTime) {
			continue
		}
		out = append(out, item)
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("no kline data")
	}
	return out, nil
}
