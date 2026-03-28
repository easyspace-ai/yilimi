package eastmoney

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"
)

const (
	quoteListURL = "https://push2.eastmoney.com/api/qt/clist/get"
)

// MarketQuote 市场行情数据（包含选股所需的完整字段）
type MarketQuote struct {
	Code                 string  `json:"code"`                 // 股票代码（6位数字）
	Name                 string  `json:"name"`                 // 股票名称
	Price                float64 `json:"price"`                // 最新价
	ChangePercent        float64 `json:"changePercent"`        // 涨跌幅（%）
	Change               float64 `json:"change"`               // 涨跌额
	Open                 float64 `json:"open"`                 // 开盘价
	High                 float64 `json:"high"`                 // 最高价
	Low                  float64 `json:"low"`                  // 最低价
	PrevClose            float64 `json:"prevClose"`            // 昨收价
	Volume               int64   `json:"volume"`               // 成交量（手）
	Amount               float64 `json:"amount"`               // 成交额（元）
	TurnoverRate         float64 `json:"turnoverRate"`         // 换手率（%）
	VolumeRatio          float64 `json:"volumeRatio"`          // 量比
	CirculatingMarketCap float64 `json:"circulatingMarketCap"` // 流通市值（亿元）
	TotalMarketCap       float64 `json:"totalMarketCap"`       // 总市值（亿元）
	Pe                   float64 `json:"pe"`                   // 市盈率（动）
	Pb                   float64 `json:"pb"`                   // 市净率
	Market               string  `json:"market"`               // 市场：SH/SZ/BJ
}

// EastMoneyQuoteListResponse 东方财富行情列表响应
type EastMoneyQuoteListResponse struct {
	Version string `json:"version"`
	Result  struct {
		Data struct {
			Total int               `json:"total"`
			Diff  []json.RawMessage `json:"diff"`
		} `json:"data"`
	} `json:"result"`
	Success bool   `json:"success"`
	Message string `json:"message"`
	Code    int    `json:"code"`
}

// GetAllAShareQuotes 获取全部A股行情
func (c *Client) GetAllAShareQuotes() ([]*MarketQuote, error) {
	var result []*MarketQuote

	// 数据字段：f2最新价,f3涨跌幅,f4涨跌额,f5成交量,f6成交额,f8换手率,f10量比,f9市盈率,f12代码,f14名称,f15最高价,f16最低价,f17开盘价,f18昨收,f20总市值,f21流通市值,f23市净率
	dataFields := "f2,f3,f4,f5,f6,f8,f9,f10,f12,f14,f15,f16,f17,f18,f20,f21,f23"

	// 获取沪A
	shQuotes, err := c.getMarketQuotes("m:0+t:6,m:0+t:80", dataFields, 1, 5000)
	if err != nil {
		return nil, fmt.Errorf("get sh quotes failed: %w", err)
	}
	result = append(result, shQuotes...)

	// 获取深A
	szQuotes, err := c.getMarketQuotes("m:0+t:6,m:0+t:80", dataFields, 0, 5000)
	if err != nil {
		return nil, fmt.Errorf("get sz quotes failed: %w", err)
	}
	result = append(result, szQuotes...)

	// 获取北交所
	bjQuotes, err := c.getMarketQuotes("m:0+t:81", dataFields, 2, 5000)
	if err == nil {
		result = append(result, bjQuotes...)
	}

	return result, nil
}

// getMarketQuotes 获取指定市场的行情
func (c *Client) getMarketQuotes(fs string, fields string, marketType int, pageSize int) ([]*MarketQuote, error) {
	var result []*MarketQuote

	pn := 1
	for {
		// 按正确顺序手动构造URL参数
		params := []string{
			"pn=" + fmt.Sprintf("%d", pn),
			"pz=" + fmt.Sprintf("%d", pageSize),
			"po=1",
			"np=1",
			"fltt=2",
			"invt=2",
			"fid=f3",
			"fs=" + fs,
			"fields=" + fields,
			"_=" + fmt.Sprintf("%d", time.Now().UnixMilli()),
		}
		url := quoteListURL + "?" + strings.Join(params, "&")

		resp, err := c.httpClient.SetTimeout(time.Duration(c.config.CrawlTimeOut)*time.Second).R().
			SetHeader("Host", "push2.eastmoney.com").
			SetHeader("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36").
			SetHeader("Referer", "https://quote.eastmoney.com/").
			Get(url)

		if err != nil {
			return nil, err
		}

		var emResp EastMoneyQuoteListResponse
		if err := json.Unmarshal(resp.Body(), &emResp); err != nil {
			return nil, err
		}

		if len(emResp.Result.Data.Diff) == 0 {
			break
		}

		for _, raw := range emResp.Result.Data.Diff {
			quote, err := c.parseQuoteItem(raw, marketType)
			if err != nil {
				continue
			}
			if quote != nil {
				result = append(result, quote)
			}
		}

		if len(emResp.Result.Data.Diff) < pageSize {
			break
		}

		pn++
		if pn > 10 {
			break
		}
	}

	return result, nil
}

// parseQuoteItem 解析单个行情数据
// fields顺序: f2,f3,f4,f5,f6,f8,f9,f10,f12,f14,f15,f16,f17,f18,f20,f21,f23
func (c *Client) parseQuoteItem(raw json.RawMessage, marketType int) (*MarketQuote, error) {
	// EastMoney diff 通常是对象（f2/f3/...），历史上也可能出现数组形式，这里做双格式兼容。
	var obj map[string]any
	if err := json.Unmarshal(raw, &obj); err == nil && len(obj) > 0 {
		getFloatByKey := func(key string) float64 {
			v, ok := obj[key]
			if !ok || v == nil {
				return 0
			}
			switch vv := v.(type) {
			case float64:
				return vv
			case string:
				if f, err := strconv.ParseFloat(vv, 64); err == nil {
					return f
				}
			}
			return 0
		}
		getInt64ByKey := func(key string) int64 {
			v, ok := obj[key]
			if !ok || v == nil {
				return 0
			}
			switch vv := v.(type) {
			case float64:
				return int64(vv)
			case string:
				if i, err := strconv.ParseInt(vv, 10, 64); err == nil {
					return i
				}
			}
			return 0
		}
		getStringByKey := func(key string) string {
			v, ok := obj[key]
			if !ok || v == nil {
				return ""
			}
			switch vv := v.(type) {
			case string:
				return vv
			case float64:
				return fmt.Sprintf("%.0f", vv)
			}
			return ""
		}

		f2 := getFloatByKey("f2")
		f3 := getFloatByKey("f3")
		f4 := getFloatByKey("f4")
		f5 := getInt64ByKey("f5")
		f6 := getFloatByKey("f6")
		f8 := getFloatByKey("f8")
		f9 := getFloatByKey("f9")
		f10 := getFloatByKey("f10")
		f12 := getStringByKey("f12")
		f14 := getStringByKey("f14")
		f15 := getFloatByKey("f15")
		f16 := getFloatByKey("f16")
		f17 := getFloatByKey("f17")
		f18 := getFloatByKey("f18")
		f20 := getFloatByKey("f20")
		f21 := getFloatByKey("f21")
		f23 := getFloatByKey("f23")

		quote := &MarketQuote{
			Code:                 f12,
			Name:                 f14,
			Price:                f2,
			ChangePercent:        f3,
			Change:               f4,
			Open:                 f17,
			High:                 f15,
			Low:                  f16,
			PrevClose:            f18,
			Volume:               f5,
			Amount:               f6,
			TurnoverRate:         f8,
			VolumeRatio:          f10,
			CirculatingMarketCap: f21 / 100000000,
			TotalMarketCap:       f20 / 100000000,
			Pe:                   f9,
			Pb:                   f23,
		}

		if marketType == 1 {
			quote.Market = "SH"
		} else if marketType == 0 {
			quote.Market = "SZ"
		} else {
			quote.Market = "BJ"
		}
		if quote.Price <= 0 || quote.Code == "" || quote.Name == "" || quote.Name == "-" {
			return nil, fmt.Errorf("invalid quote")
		}
		return quote, nil
	}

	// 回退：兼容数组格式
	var data []any
	if err := json.Unmarshal(raw, &data); err != nil {
		return nil, err
	}
	if len(data) < 17 {
		return nil, fmt.Errorf("insufficient data")
	}
	getFloat := func(idx int) float64 {
		if idx >= len(data) {
			return 0
		}
		switch v := data[idx].(type) {
		case float64:
			return v
		case string:
			if f, err := strconv.ParseFloat(v, 64); err == nil {
				return f
			}
		}
		return 0
	}
	getInt64 := func(idx int) int64 {
		if idx >= len(data) {
			return 0
		}
		switch v := data[idx].(type) {
		case float64:
			return int64(v)
		case int64:
			return v
		case string:
			if i, err := strconv.ParseInt(v, 10, 64); err == nil {
				return i
			}
		}
		return 0
	}
	getString := func(idx int) string {
		if idx >= len(data) {
			return ""
		}
		switch v := data[idx].(type) {
		case string:
			return v
		case float64:
			return fmt.Sprintf("%.0f", v)
		}
		return ""
	}

	f2 := getFloat(0)
	f3 := getFloat(1)
	f4 := getFloat(2)
	f5 := getInt64(3)
	f6 := getFloat(4)
	f8 := getFloat(5)
	f9 := getFloat(6)
	f10 := getFloat(7)
	f12 := getString(8)
	f14 := getString(9)
	f15 := getFloat(10)
	f16 := getFloat(11)
	f17 := getFloat(12)
	f18 := getFloat(13)
	f20 := getFloat(14)
	f21 := getFloat(15)
	f23 := getFloat(16)

	quote := &MarketQuote{
		Code:                 f12,
		Name:                 f14,
		Price:                f2,
		ChangePercent:        f3,
		Change:               f4,
		Open:                 f17,
		High:                 f15,
		Low:                  f16,
		PrevClose:            f18,
		Volume:               f5,
		Amount:               f6,
		TurnoverRate:         f8,
		VolumeRatio:          f10,
		CirculatingMarketCap: f21 / 100000000,
		TotalMarketCap:       f20 / 100000000,
		Pe:                   f9,
		Pb:                   f23,
	}

	if marketType == 1 {
		quote.Market = "SH"
	} else if marketType == 0 {
		quote.Market = "SZ"
	} else {
		quote.Market = "BJ"
	}

	if quote.Price <= 0 || quote.Code == "" || quote.Name == "" || quote.Name == "-" {
		return nil, fmt.Errorf("invalid quote")
	}

	return quote, nil
}
