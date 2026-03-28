package sina

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
	"time"

	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/transform"
)

const (
	sinaURL = "http://hq.sinajs.cn/rn=%d&list=%s"
)

// Quote 实时行情数据
type Quote struct {
	Symbol    string  `json:"symbol"`
	Name      string  `json:"name"`
	Open      float64 `json:"open"`
	PrevClose float64 `json:"prevClose"`
	Price     float64 `json:"price"`
	High      float64 `json:"high"`
	Low       float64 `json:"low"`
	Volume    int64   `json:"volume"`
	Amount    float64 `json:"amount"`
	Bid1      float64 `json:"bid1"`
	Bid1Vol   int64   `json:"bid1Vol"`
	Ask1      float64 `json:"ask1"`
	Ask1Vol   int64   `json:"ask1Vol"`
	Date      string  `json:"date"`
	Time      string  `json:"time"`
}

// Client 新浪财经客户端
type Client struct {
	httpClient *http.Client
}

// NewClient 创建新浪财经客户端
func NewClient() *Client {
	return &Client{
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// GetQuotes 获取多只股票的实时行情
func (c *Client) GetQuotes(codes []string) ([]*Quote, error) {
	var result []*Quote

	url := fmt.Sprintf(sinaURL, time.Now().Unix(), strings.Join(codes, ","))

	resp, err := c.httpClient.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	body, _, err = transform.Bytes(simplifiedchinese.GBK.NewDecoder(), body)
	if err != nil {
		return nil, err
	}

	lines := strings.Split(string(body), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		quote := c.parseQuote(line)
		if quote != nil {
			result = append(result, quote)
		}
	}

	return result, nil
}

// GetQuote 获取单只股票的实时行情
func (c *Client) GetQuote(code string) (*Quote, error) {
	quotes, err := c.GetQuotes([]string{code})
	if err != nil {
		return nil, err
	}
	if len(quotes) == 0 {
		return nil, fmt.Errorf("no data for code: %s", code)
	}
	return quotes[0], nil
}

// parseQuote 解析单条行情数据
func (c *Client) parseQuote(line string) *Quote {
	eqIndex := strings.Index(line, "=")
	if eqIndex < 0 {
		return nil
	}

	varPart := line[:eqIndex]
	valuePart := line[eqIndex+1:]

	if !strings.HasPrefix(valuePart, "\"") || !strings.HasSuffix(valuePart, "\";") {
		return nil
	}

	valuePart = valuePart[1 : len(valuePart)-2]
	parts := strings.Split(valuePart, ",")

	if len(parts) < 32 {
		return nil
	}

	quote := &Quote{}

	if idx := strings.LastIndex(varPart, "_"); idx >= 0 {
		quote.Symbol = varPart[idx+1:]
	}

	quote.Name = parts[0]
	quote.Open, _ = strconv.ParseFloat(parts[1], 64)
	quote.PrevClose, _ = strconv.ParseFloat(parts[2], 64)
	quote.Price, _ = strconv.ParseFloat(parts[3], 64)
	quote.High, _ = strconv.ParseFloat(parts[4], 64)
	quote.Low, _ = strconv.ParseFloat(parts[5], 64)
	quote.Bid1, _ = strconv.ParseFloat(parts[6], 64)
	quote.Ask1, _ = strconv.ParseFloat(parts[7], 64)
	quote.Volume, _ = strconv.ParseInt(parts[8], 10, 64)
	quote.Amount, _ = strconv.ParseFloat(parts[9], 64)
	quote.Bid1Vol, _ = strconv.ParseInt(parts[10], 10, 64)
	quote.Bid1, _ = strconv.ParseFloat(parts[11], 64)
	quote.Date = parts[30]
	quote.Time = parts[31]

	return quote
}
