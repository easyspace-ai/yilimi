package tushare

import (
	"encoding/json"
	"fmt"

	"github.com/go-resty/resty/v2"
)

const (
	apiURL = "http://api.tushare.pro"
)

// Request Tushare 请求
type Request struct {
	API    string `json:"api_name"`
	Token  string `json:"token"`
	Params any    `json:"params"`
	Fields string `json:"fields"`
}

// Response Tushare 响应
type Response struct {
	RequestID string `json:"request_id"`
	Code      int    `json:"code"`
	Data      struct {
		Fields  []string `json:"fields"`
		Items   [][]any  `json:"items"`
		HasMore bool     `json:"has_more"`
	} `json:"data"`
	Msg string `json:"msg"`
}

// StockBasicInfo 股票基本信息
type StockBasicInfo struct {
	TsCode     string `json:"ts_code"`
	Symbol     string `json:"symbol"`
	Name       string `json:"name"`
	Area       string `json:"area"`
	Industry   string `json:"industry"`
	FullName   string `json:"fullname"`
	EnName     string `json:"enname"`
	Market     string `json:"market"`
	Exchange   string `json:"exchange"`
	ListDate   string `json:"list_date"`
	DelistDate string `json:"delist_date"`
	IsHS       string `json:"is_hs"`
}

// Client Tushare 客户端
type Client struct {
	httpClient *resty.Client
	token      string
}

// NewClient 创建 Tushare 客户端
func NewClient(token string) *Client {
	return &Client{
		httpClient: resty.New(),
		token:      token,
	}
}

// doRequest 执行请求
func (c *Client) doRequest(apiName string, params any, fields string) (*Response, error) {
	req := Request{
		API:    apiName,
		Token:  c.token,
		Params: params,
		Fields: fields,
	}

	resp, err := c.httpClient.R().
		SetHeader("Content-Type", "application/json").
		SetBody(req).
		Post(apiURL)

	if err != nil {
		return nil, err
	}

	var result Response
	if err := json.Unmarshal(resp.Body(), &result); err != nil {
		return nil, err
	}

	if result.Code != 0 {
		return nil, fmt.Errorf("Tushare error: %d - %s", result.Code, result.Msg)
	}

	return &result, nil
}

// GetStockBasic 获取股票列表
func (c *Client) GetStockBasic(market string) ([]StockBasicInfo, error) {
	params := map[string]any{}
	if market != "" {
		params["market"] = market
	}
	params["list_status"] = "L"

	fields := "ts_code,symbol,name,area,industry,fullname,enname,market,exchange,list_date,delist_date,is_hs"

	resp, err := c.doRequest("stock_basic", params, fields)
	if err != nil {
		return nil, err
	}

	var result []StockBasicInfo
	for _, item := range resp.Data.Items {
		if len(item) < 12 {
			continue
		}
		info := StockBasicInfo{
			TsCode:   asString(item[0]),
			Symbol:   asString(item[1]),
			Name:     asString(item[2]),
			Area:     asString(item[3]),
			Industry: asString(item[4]),
			FullName: asString(item[5]),
			EnName:   asString(item[6]),
			Market:   asString(item[7]),
			Exchange: asString(item[8]),
			ListDate: asString(item[9]),
		}
		result = append(result, info)
	}

	return result, nil
}

// GetDaily 获取日线行情
func (c *Client) GetDaily(tsCode, startDate, endDate string) ([][]any, error) {
	params := map[string]any{
		"ts_code": tsCode,
	}
	if startDate != "" {
		params["start_date"] = startDate
	}
	if endDate != "" {
		params["end_date"] = endDate
	}

	fields := "ts_code,trade_date,open,high,low,close,pre_close,change,pct_chg,vol,amount"

	resp, err := c.doRequest("daily", params, fields)
	if err != nil {
		return nil, err
	}

	return resp.Data.Items, nil
}

func asString(v any) string {
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}
