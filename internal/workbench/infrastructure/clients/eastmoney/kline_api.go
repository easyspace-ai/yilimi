package eastmoney

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/go-resty/resty/v2"

	"github.com/easyspace-ai/yilimi/internal/workbench/infrastructure/clients"
)

const (
	baseURL = "https://push2his.eastmoney.com/api/qt/stock/kline/get"
)

// KLineType K线类型
type KLineType string

const (
	KLineType1Min     KLineType = "1"
	KLineType5Min     KLineType = "5"
	KLineType15Min    KLineType = "15"
	KLineType30Min    KLineType = "30"
	KLineType60Min    KLineType = "60"
	KLineType120Min   KLineType = "120"
	KLineTypeDay      KLineType = "101"
	KLineTypeWeek     KLineType = "102"
	KLineTypeMonth    KLineType = "103"
	KLineTypeQuarter  KLineType = "104"
	KLineTypeHalfYear KLineType = "105"
	KLineTypeYear     KLineType = "106"
)

// KLineData K线数据项
type KLineData struct {
	Date   string  `json:"date"`
	Open   float64 `json:"open"`
	Close  float64 `json:"close"`
	High   float64 `json:"high"`
	Low    float64 `json:"low"`
	Volume int64   `json:"volume"`
	Amount float64 `json:"amount"`
	Change float64 `json:"change"`
}

// EastMoneyKLineResponse 东方财富 K线响应
type EastMoneyKLineResponse struct {
	Version string `json:"version"`
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    struct {
		ID     int      `json:"id"`
		Klines []string `json:"klines"`
		Name   string   `json:"name"`
		Code   string   `json:"code"`
		Market any      `json:"market"`
		Period string   `json:"period"`
	} `json:"data"`
}

// Client 东方财富 K线客户端
type Client struct {
	httpClient *resty.Client
	config     *clients.SettingConfig
}

// NewClient 创建东方财富客户端
func NewClient(config *clients.SettingConfig) *Client {
	return &Client{
		httpClient: resty.New(),
		config:     config,
	}
}

// GetKLine 获取 K线数据
func (c *Client) GetKLine(stockCode string, klineType KLineType, adjustFlag string, days int) ([]KLineData, string, error) {
	secid := c.convertStockCode(stockCode)
	if secid == "" {
		return nil, "", fmt.Errorf("invalid stock code: %s", stockCode)
	}

	fqt := c.getAdjustType(adjustFlag)
	result, name, err := c.fetchKLineRaw(secid, klineType, adjustFlag, fqt, days)
	if err == nil {
		return result, name, nil
	}
	// 指数等品种前复权(fqt=1)可能返回 API 错误，回落不复权再拉一次
	if fqt == "1" {
		result2, name2, err2 := c.fetchKLineRaw(secid, klineType, "", "0", days)
		if err2 == nil {
			return result2, name2, nil
		}
	}
	return nil, "", err
}

func (c *Client) fetchKLineRaw(secid string, klineType KLineType, adjustFlag string, fqt string, days int) ([]KLineData, string, error) {
	var result []KLineData
	fields := "f51,f52,f53,f54,f55,f56,f57,f58,f59,f60,f61,f116"
	if adjustFlag != "" {
		fields = "f51,f52,f53,f54,f55,f56,f57,f58,f59,f60,f61,f116,f113,f114,f115"
	}

	url := fmt.Sprintf("%s?secid=%s&klt=%s&fqt=%s&end=20500101&lmt=%d&fields1=f1,f2,f3,f4,f5,f6&fields2=%s&wbp2u=|0|0|0|web&_=%d",
		baseURL, secid, string(klineType), fqt, days, fields, time.Now().UnixMilli())

	timeout := time.Duration(c.config.CrawlTimeOut) * time.Second
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	resp, err := c.httpClient.SetTimeout(timeout).R().
		SetHeader("Host", "push2his.eastmoney.com").
		SetHeader("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36").
		SetHeader("Referer", "https://quote.eastmoney.com/").
		Get(url)

	if err != nil {
		return nil, "", err
	}

	var emResp EastMoneyKLineResponse
	if err := json.Unmarshal(resp.Body(), &emResp); err != nil {
		return nil, "", err
	}

	if emResp.Code != 0 {
		return nil, "", fmt.Errorf("API error: %d - %s", emResp.Code, emResp.Message)
	}

	for _, klineStr := range emResp.Data.Klines {
		kline := c.parseKLine(klineStr)
		if kline != nil {
			result = append(result, *kline)
		}
	}

	return result, emResp.Data.Name, nil
}

// convertStockCode 转换股票代码格式
func (c *Client) convertStockCode(code string) string {
	code = strings.ToUpper(strings.TrimSpace(code))

	if strings.Contains(code, ".") {
		parts := strings.Split(code, ".")
		if len(parts) == 2 {
			cd := parts[0]
			market := parts[1]
			switch market {
			case "SH", "SS":
				return "1." + cd
			case "SZ":
				return "0." + cd
			case "BJ":
				// 北交所标的 / 北证指数与东财 push2his 一致用 0. 前缀（如 899050）
				return "0." + cd
			case "HK":
				return "128." + cd
			}
		}
	}

	if len(code) == 6 {
		if strings.HasPrefix(code, "6") {
			return "1." + code
		}
		return "0." + code
	}

	return ""
}

// getAdjustType 获取复权类型
func (c *Client) getAdjustType(flag string) string {
	switch flag {
	case "qfq":
		return "1"
	case "hfq":
		return "2"
	default:
		return "0"
	}
}

// parseKLine 解析 K线字符串
func (c *Client) parseKLine(line string) *KLineData {
	parts := strings.Split(line, ",")
	if len(parts) < 7 {
		return nil
	}

	data := &KLineData{
		Date: parts[0],
	}

	if v, err := strconv.ParseFloat(parts[1], 64); err == nil {
		data.Open = v
	}
	if v, err := strconv.ParseFloat(parts[2], 64); err == nil {
		data.Close = v
	}
	if v, err := strconv.ParseFloat(parts[3], 64); err == nil {
		data.High = v
	}
	if v, err := strconv.ParseFloat(parts[4], 64); err == nil {
		data.Low = v
	}
	if v, err := strconv.ParseInt(parts[5], 10, 64); err == nil {
		data.Volume = v
	}
	if v, err := strconv.ParseFloat(parts[6], 64); err == nil {
		data.Amount = v
	}
	if len(parts) > 10 {
		if v, err := strconv.ParseFloat(parts[10], 64); err == nil {
			data.Change = v
		}
	}

	return data
}
