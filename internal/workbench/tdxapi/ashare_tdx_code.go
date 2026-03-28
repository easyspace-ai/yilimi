package tdxapi

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/easyspace-ai/tdx/protocol"
)

var sixDigitCode = regexp.MustCompile(`^\d{6}$`)

// ToTdxLowerCodeFromNorm 将 600519.SH、裸六位代码或已是 sh600519 形式转为通达信底层代码（小写+前缀）。
func ToTdxLowerCodeFromNorm(normalized string) (string, error) {
	n := strings.TrimSpace(normalized)
	if n == "" {
		return "", fmt.Errorf("empty symbol")
	}
	up := strings.ToUpper(n)
	// 带 .SH/.SZ/.BJ 后缀时以交易所为准（如 516150.SH 须为 sh516150；仅用 AddPrefix(516150) 会漏判 516 段基金）
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
