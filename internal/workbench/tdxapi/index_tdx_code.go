package tdxapi

import (
	"strings"
)

// AShareIndexToTdxCode 识别常见 A 股指数并返回通达信指数通道代码（如 sh000001、sz399001）。
// 个股返回 ("", false)。用于日线走 GetIndexDayAll，而非个股 K 线接口。
func AShareIndexToTdxCode(normalized string) (string, bool) {
	n := strings.ToUpper(strings.TrimSpace(normalized))
	if !strings.Contains(n, ".") {
		return "", false
	}
	dot := strings.LastIndex(n, ".")
	base, suf := n[:dot], n[dot+1:]
	if len(base) != 6 || (suf != "SH" && suf != "SZ" && suf != "BJ") {
		return "", false
	}
	switch suf {
	case "SZ":
		// 深证系列指数多为 399xxx（成指、创业板指等）
		if strings.HasPrefix(base, "399") {
			return "sz" + strings.ToLower(base), true
		}
	case "SH":
		// 上证系列指数多为 000xxx（上证、沪深300、中证500 等）
		if strings.HasPrefix(base, "000") {
			return "sh" + strings.ToLower(base), true
		}
	case "BJ":
		// 北证50 等：899xxx
		if strings.HasPrefix(base, "899") {
			return "bj" + strings.ToLower(base), true
		}
	}
	return "", false
}
