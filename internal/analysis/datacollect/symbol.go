package datacollect

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/easyspace-ai/tdx/protocol"
)

var sixDigitCode = regexp.MustCompile(`^\d{6}$`)

// ToTdxMarketCode 将 600820.SH 转为通达信 GetKlineDay* 使用的市场前缀+6位代码（小写），如 sh600820。
func ToTdxMarketCode(normalized string) (string, error) {
	n := strings.TrimSpace(normalized)
	if n == "" {
		return "", fmt.Errorf("empty symbol")
	}
	up := strings.ToUpper(n)
	switch {
	case strings.HasSuffix(up, ".SH"):
		base := strings.TrimSuffix(up, ".SH")
		if len(base) != 6 || !sixDigitCode.MatchString(base) {
			return "", fmt.Errorf("unsupported symbol: %s", normalized)
		}
		return "sh" + strings.ToLower(base), nil
	case strings.HasSuffix(up, ".SZ"):
		base := strings.TrimSuffix(up, ".SZ")
		if len(base) != 6 || !sixDigitCode.MatchString(base) {
			return "", fmt.Errorf("unsupported symbol: %s", normalized)
		}
		return "sz" + strings.ToLower(base), nil
	case strings.HasSuffix(up, ".BJ"):
		base := strings.TrimSuffix(up, ".BJ")
		if len(base) != 6 || !sixDigitCode.MatchString(base) {
			return "", fmt.Errorf("unsupported symbol: %s", normalized)
		}
		return "bj" + strings.ToLower(base), nil
	}
	lower := strings.ToLower(n)
	if len(lower) >= 8 && (strings.HasPrefix(lower, "sh") || strings.HasPrefix(lower, "sz") || strings.HasPrefix(lower, "bj")) {
		if protocol.IsSHStock(lower) || protocol.IsSZStock(lower) || protocol.IsBJStock(lower) || protocol.IsETF(lower) {
			return lower, nil
		}
	}
	if sixDigitCode.MatchString(up) {
		out := strings.ToLower(protocol.AddPrefix(up))
		if len(out) != 8 {
			return "", fmt.Errorf("unsupported symbol: %s", normalized)
		}
		return out, nil
	}
	return "", fmt.Errorf("unsupported symbol: %s", normalized)
}

// SixDigitFromTSCode 从 600820.SH / 裸六位 解析出六位代码。
func SixDigitFromTSCode(tsCode string) (string, error) {
	n := strings.TrimSpace(tsCode)
	if i := strings.IndexByte(strings.ToUpper(n), '.'); i > 0 {
		n = n[:i]
	}
	n = strings.TrimSpace(n)
	if !sixDigitCode.MatchString(n) {
		return "", fmt.Errorf("need six-digit code: %s", tsCode)
	}
	return n, nil
}
