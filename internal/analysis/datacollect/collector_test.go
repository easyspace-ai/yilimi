package datacollect

import (
	"strings"
	"testing"

	"github.com/easyspace-ai/stock_api/pkg/tsdb"
)

func TestNormDateISO(t *testing.T) {
	s, err := NormDateISO("2026-03-29")
	if err != nil || s != "2026-03-29" {
		t.Fatalf("got %q %v", s, err)
	}
	s2, err := NormDateISO("20260329")
	if err != nil || s2 != "2026-03-29" {
		t.Fatalf("compact: got %q %v", s2, err)
	}
}

func TestPoolMarketInstructionContainsOHLCHint(t *testing.T) {
	p := &Pool{
		StockDataText: "open high low close volume 示例",
		Indicators:    "【rsi】\n55\n",
		StockBasic:    "name=测试",
	}
	out := p.MarketInstruction("BASE")
	if !strings.Contains(out, "BASE") || !strings.Contains(out, "数据附录") {
		t.Fatal(out)
	}
	if !strings.Contains(out, "open high low close volume") {
		t.Fatal("missing stock text")
	}
}

func TestBuildIndicatorsBlockEmpty(t *testing.T) {
	s := BuildIndicatorsBlock(&tsdb.DataFrame{})
	if s == "" {
		t.Fatal("expected message")
	}
}
