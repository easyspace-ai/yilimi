package httpapi

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

var (
	reConfidenceZH = regexp.MustCompile(`(?i)置信度[:：]\s*(\d+)%`)
	reConfidenceEN = regexp.MustCompile(`(?i)confidence[:：]\s*(\d+)%`)
	reTargetPrice  = regexp.MustCompile(`(?i)(?:目标价|目标价格|target)[:：]\s*[¥$]?\s*(\d+\.?\d*)`)
	reStopLoss     = regexp.MustCompile(`(?i)(?:止损价|止损价格|stop[-\s_]?loss)[:：]\s*[¥$]?\s*(\d+\.?\d*)`)
	reVerdict      = regexp.MustCompile(`(?is)<!--\s*VERDICT:\s*(\{.*?\})\s*-->`)
)

func extractConfidenceRegex(text string) *int {
	if text == "" {
		return nil
	}
	for _, re := range []*regexp.Regexp{reConfidenceZH, reConfidenceEN} {
		m := re.FindStringSubmatch(text)
		if len(m) > 1 {
			v, err := strconv.Atoi(m[1])
			if err == nil && v >= 0 && v <= 100 {
				return &v
			}
		}
	}
	return nil
}

func extractTargetPriceRegex(text string) *float64 {
	if text == "" {
		return nil
	}
	m := reTargetPrice.FindStringSubmatch(text)
	if len(m) > 1 {
		f, err := strconv.ParseFloat(m[1], 64)
		if err == nil {
			return &f
		}
	}
	return nil
}

func extractStopLossRegex(text string) *float64 {
	if text == "" {
		return nil
	}
	m := reStopLoss.FindStringSubmatch(text)
	if len(m) > 1 {
		f, err := strconv.ParseFloat(m[1], 64)
		if err == nil {
			return &f
		}
	}
	return nil
}

// extractVerdictDirection mirrors Python _extract_verdict → direction string for reports.direction.
func extractVerdictDirection(text string) string {
	if text == "" {
		return ""
	}
	m := reVerdict.FindStringSubmatch(text)
	if len(m) < 2 {
		return ""
	}
	raw := strings.TrimSpace(strings.ReplaceAll(strings.ReplaceAll(m[1], "\n", " "), "\r", " "))
	var payload struct {
		Direction string `json:"direction"`
	}
	if json.Unmarshal([]byte(raw), &payload) != nil || strings.TrimSpace(payload.Direction) == "" {
		return ""
	}
	return strings.TrimSpace(payload.Direction)
}

func stringFromAny(v any) string {
	if v == nil {
		return ""
	}
	switch t := v.(type) {
	case string:
		return t
	default:
		return fmt.Sprint(t)
	}
}

// mergePayloadExtras fills nil confidence / prices / verdict direction like Python resolve_report_fields.
func mergePayloadExtras(result map[string]any, payload map[string]any) {
	if result == nil || payload == nil {
		return
	}
	final := stringFromAny(result["final_trade_decision"])
	trader := stringFromAny(result["trader_investment_plan"])

	if vdir := extractVerdictDirection(final); vdir != "" {
		payload["direction"] = vdir
	}

	if payload["confidence"] == nil {
		if c := extractConfidenceRegex(final); c != nil {
			payload["confidence"] = float64(*c)
		}
	}

	if payload["target_price"] == nil {
		if p := extractTargetPriceRegex(final); p != nil {
			payload["target_price"] = *p
		} else if p := extractTargetPriceRegex(trader); p != nil {
			payload["target_price"] = *p
		}
	}

	if payload["stop_loss_price"] == nil {
		if p := extractStopLossRegex(final); p != nil {
			payload["stop_loss_price"] = *p
		} else if p := extractStopLossRegex(trader); p != nil {
			payload["stop_loss_price"] = *p
		}
	}
}
