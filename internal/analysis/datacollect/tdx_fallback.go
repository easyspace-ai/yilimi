package datacollect

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/easyspace-ai/tdx"
	"github.com/easyspace-ai/tdx/protocol"
)

// DailyBarFetcher 可选：tsdb 无数据时拉通达信日线。
type DailyBarFetcher interface {
	FetchDailyBars(ctx context.Context, tsCode string, maxCount int) (text string, err error)
}

// TDXBarFetcher 使用 github.com/easyspace-ai/tdx 客户端（需网络可达行情主站）。
type TDXBarFetcher struct {
	Dial func() (*tdx.Client, error)
}

// EnabledFromEnv 当 AIGOSTOCK_TDX_FALLBACK=1 时启用。
func TDXFallbackEnabled() bool {
	return strings.TrimSpace(os.Getenv("AIGOSTOCK_TDX_FALLBACK")) == "1"
}

// FetchDailyBars 返回格式化的日 K 文本（仅用于补洞）。
func (f *TDXBarFetcher) FetchDailyBars(ctx context.Context, tsCode string, maxCount int) (string, error) {
	if f == nil || f.Dial == nil {
		return "", nil
	}
	dial := f.Dial
	select {
	case <-ctx.Done():
		return "", ctx.Err()
	default:
	}
	cli, err := dial()
	if err != nil {
		return "", err
	}

	six, err := SixDigitFromTSCode(tsCode)
	if err != nil {
		return "", err
	}
	if maxCount <= 0 {
		maxCount = 800
	}
	if maxCount > 800 {
		maxCount = 800
	}
	resp, err := cli.GetKlineDay(six, 0, uint16(maxCount))
	if err != nil || resp == nil || len(resp.List) == 0 {
		return "", err
	}
	var b strings.Builder
	b.WriteString("（通达信补数）日线（时间倒序展示最新在前）：\n")
	// 展示最近 120 条避免过大
	show := resp.List
	if len(show) > 120 {
		show = show[len(show)-120:]
	}
	for i := len(show) - 1; i >= 0; i-- {
		k := show[i]
		b.WriteString(formatOneKline(k))
		b.WriteByte('\n')
	}
	return b.String(), nil
}

func formatOneKline(k *protocol.Kline) string {
	if k == nil {
		return ""
	}
	return fmt.Sprintf("%s O=%.4f H=%.4f L=%.4f C=%.4f V=%d",
		k.Time.Format("2006-01-02"),
		k.Open.Float64(), k.High.Float64(), k.Low.Float64(), k.Close.Float64(),
		k.Volume)
}
