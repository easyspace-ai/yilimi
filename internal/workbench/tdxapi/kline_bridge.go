package tdxapi

import (
	"fmt"
	"time"

	"github.com/easyspace-ai/yilimi/internal/workbench/domain/stock"

	"github.com/easyspace-ai/tdx/protocol"
)

// DailyKLineItemsForNormalizedSymbol 从通达信拉取日线全量后按日历日裁剪到 [startTime, endTime]（与东财接口区间语义一致）。
func (s *Service) DailyKLineItemsForNormalizedSymbol(norm string, startTime, endTime time.Time) ([]stock.KLineItem, error) {
	if s == nil {
		return nil, fmt.Errorf("tdx service nil")
	}
	var list []*protocol.Kline
	var err error
	if idx, ok := AShareIndexToTdxCode(norm); ok {
		list, err = s.fetchIndexAll(idx, "day")
	} else {
		var code string
		code, err = ToTdxLowerCodeFromNorm(norm)
		if err != nil {
			return nil, err
		}
		list, err = s.fetchStockKlineAllTDX(code, "day")
	}
	if err != nil {
		return nil, err
	}
	return filterProtocolKlinesToItems(list, startTime, endTime)
}

func filterProtocolKlinesToItems(list []*protocol.Kline, startTime, endTime time.Time) ([]stock.KLineItem, error) {
	loc := time.Local
	out := make([]stock.KLineItem, 0, len(list))
	for _, k := range list {
		if k == nil {
			continue
		}
		t := k.Time.In(loc)
		barDay := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, loc)
		if barDay.Before(startTime) || barDay.After(endTime) {
			continue
		}
		dateStr := barDay.Format("2006-01-02")
		open := k.Open.Float64()
		high := k.High.Float64()
		low := k.Low.Float64()
		close := k.Close.Float64()
		chg := 0.0
		if k.Last > 0 {
			chg = close - k.Last.Float64()
		} else {
			chg = close - open
		}
		out = append(out, stock.KLineItem{
			Date:   dateStr,
			Open:   open,
			High:   high,
			Low:    low,
			Close:  close,
			Volume: k.Volume,
			Amount: k.Amount.Float64(),
			Change: chg,
		})
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("no kline data")
	}
	return out, nil
}

// MinuteSeries 分时 1 分钟价量序列；code 须为 sh600519 形式，date 空则由服务端择日。
func (s *Service) MinuteSeries(tdxCode, date string) (*protocol.MinuteResp, string, error) {
	if s == nil {
		return nil, "", fmt.Errorf("tdx service nil")
	}
	return s.getMinuteWithFallback(tdxCode, date)
}
