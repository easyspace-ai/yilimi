package tools

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
	"github.com/easyspace-ai/stock_api/pkg/tsdb"
)

// StockTools 封装 stockdb 为 eino Tools
type StockTools struct {
	client *tsdb.UnifiedClient
}

// UnifiedClient 返回底层 tsdb 客户端（供 datacollect 等复用同一数据源）。
func (s *StockTools) UnifiedClient() *tsdb.UnifiedClient {
	if s == nil {
		return nil
	}
	return s.client
}

// NewStockTools 创建股票数据工具集
func NewStockTools(dataDir string) (*StockTools, error) {
	client, err := tsdb.NewUnifiedClient(tsdb.UnifiedConfig{
		PrimaryDataSource: tsdb.DataSourceStockSDK,
		DataDir:           dataDir,
		CacheMode:         tsdb.CacheModeAuto,
	})
	if err != nil {
		return nil, fmt.Errorf("create stockdb client: %w", err)
	}
	return &StockTools{client: client}, nil
}

// Close 关闭客户端
func (s *StockTools) Close() error {
	return s.client.Close()
}

// formatDataFrame 将 DataFrame 格式化为字符串供 LLM 使用
func formatDataFrame(df *tsdb.DataFrame) string {
	if df == nil || len(df.Rows) == 0 {
		return "无数据"
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("共 %d 条数据，列：%v\n\n", len(df.Rows), df.Columns))

	// 最多显示 20 行
	limit := len(df.Rows)
	if limit > 20 {
		limit = 20
	}

	for i, row := range df.Rows {
		if i >= limit {
			sb.WriteString(fmt.Sprintf("...（省略 %d 条）\n", len(df.Rows)-limit))
			break
		}
		sb.WriteString(fmt.Sprintf("%d. %v\n", i+1, row))
	}

	return sb.String()
}

// GetStockData 获取股票日线数据（K线）
func (s *StockTools) GetStockData(ctx context.Context, tsCode string, startDate string, endDate string) (string, error) {
	df, err := s.client.GetStockDaily(ctx, tsCode, startDate, endDate, tsdb.AdjustQFQ)
	if err != nil {
		return "", fmt.Errorf("get stock daily: %w", err)
	}
	return fmt.Sprintf("股票 %s 日线数据（%s 至 %s，前复权）：\n%s",
		tsCode, startDate, endDate, formatDataFrame(df)), nil
}

// GetStockBasic 获取股票基础信息
func (s *StockTools) GetStockBasic(ctx context.Context, tsCode string) (string, error) {
	filter := tsdb.StockBasicFilter{TSCode: tsCode}
	df, err := s.client.GetStockBasic(ctx, filter)
	if err != nil {
		return "", fmt.Errorf("get stock basic: %w", err)
	}
	return fmt.Sprintf("股票 %s 基础信息：\n%s", tsCode, formatDataFrame(df)), nil
}

// GetAllStockBasic 获取所有上市股票列表
func (s *StockTools) GetAllStockBasic(ctx context.Context) (string, error) {
	filter := tsdb.StockBasicFilter{ListStatus: "L"}
	df, err := s.client.GetStockBasic(ctx, filter)
	if err != nil {
		return "", fmt.Errorf("get all stock basic: %w", err)
	}
	return fmt.Sprintf("所有上市股票（%d 只）：\n%s", len(df.Rows), formatDataFrame(df)), nil
}

// GetTradeCalendar 获取交易日历
func (s *StockTools) GetTradeCalendar(ctx context.Context, startDate string, endDate string) (string, error) {
	filter := tsdb.TradeCalendarFilter{
		StartDate: startDate,
		EndDate:   endDate,
	}
	df, err := s.client.GetTradeCalendar(ctx, filter)
	if err != nil {
		return "", fmt.Errorf("get trade calendar: %w", err)
	}
	return fmt.Sprintf("交易日历（%s 至 %s）：\n%s", startDate, endDate, formatDataFrame(df)), nil
}

// GetDailyBasic 获取每日基本面数据
func (s *StockTools) GetDailyBasic(ctx context.Context, tsCode string, startDate string, endDate string) (string, error) {
	df, err := s.client.GetDailyBasic(ctx, tsCode, startDate, endDate)
	if err != nil {
		return "", fmt.Errorf("get daily basic: %w", err)
	}
	return fmt.Sprintf("股票 %s 每日基本面数据（%s 至 %s）：\n%s",
		tsCode, startDate, endDate, formatDataFrame(df)), nil
}

// GetAdjFactor 获取复权因子
func (s *StockTools) GetAdjFactor(ctx context.Context, tsCode string, startDate string, endDate string) (string, error) {
	df, err := s.client.GetAdjFactor(ctx, tsCode, startDate, endDate)
	if err != nil {
		return "", fmt.Errorf("get adj factor: %w", err)
	}
	return fmt.Sprintf("股票 %s 复权因子（%s 至 %s）：\n%s",
		tsCode, startDate, endDate, formatDataFrame(df)), nil
}

// ========== eino Tool 封装 ==========

// NewGetStockDataTool 创建获取股票数据的 Tool
func (s *StockTools) NewGetStockDataTool() tool.InvokableTool {
	type Input struct {
		TSCode    string `json:"ts_code" jsonschema_description:"股票代码，如 000001.SZ 或 600519.SH"`
		StartDate string `json:"start_date" jsonschema_description:"开始日期，格式 YYYYMMDD，如 20240101"`
		EndDate   string `json:"end_date" jsonschema_description:"结束日期，格式 YYYYMMDD，如 20241231"`
	}

	t, err := utils.InferTool(
		"get_stock_data",
		"获取股票日线 K 线数据（前复权），包含 OHLCV 和技术指标基础数据",
		func(ctx context.Context, input *Input) (string, error) {
			return s.GetStockData(ctx, input.TSCode, input.StartDate, input.EndDate)
		},
	)
	if err != nil {
		log.Fatalf("创建 get_stock_data 工具失败: %v", err)
	}
	return t
}

// NewGetStockBasicTool 创建获取股票基础信息的 Tool
func (s *StockTools) NewGetStockBasicTool() tool.InvokableTool {
	type Input struct {
		TSCode string `json:"ts_code" jsonschema_description:"股票代码，如 000001.SZ 或 600519.SH；留空则获取所有上市股票"`
	}

	t, err := utils.InferTool(
		"get_stock_basic",
		"获取股票基础信息，包括名称、行业、上市日期等；如果 ts_code 留空则获取所有上市股票列表",
		func(ctx context.Context, input *Input) (string, error) {
			if input.TSCode == "" {
				return s.GetAllStockBasic(ctx)
			}
			return s.GetStockBasic(ctx, input.TSCode)
		},
	)
	if err != nil {
		log.Fatalf("创建 get_stock_basic 工具失败: %v", err)
	}
	return t
}

// NewGetTradeCalendarTool 创建获取交易日历的 Tool
func (s *StockTools) NewGetTradeCalendarTool() tool.InvokableTool {
	type Input struct {
		StartDate string `json:"start_date" jsonschema_description:"开始日期，格式 YYYYMMDD"`
		EndDate   string `json:"end_date" jsonschema_description:"结束日期，格式 YYYYMMDD"`
	}

	t, err := utils.InferTool(
		"get_trade_calendar",
		"获取交易日历，查询指定日期范围内哪些是交易日",
		func(ctx context.Context, input *Input) (string, error) {
			return s.GetTradeCalendar(ctx, input.StartDate, input.EndDate)
		},
	)
	if err != nil {
		log.Fatalf("创建 get_trade_calendar 工具失败: %v", err)
	}
	return t
}

// NewGetDailyBasicTool 创建获取每日基本面的 Tool
func (s *StockTools) NewGetDailyBasicTool() tool.InvokableTool {
	type Input struct {
		TSCode    string `json:"ts_code" jsonschema_description:"股票代码，如 000001.SZ 或 600519.SH"`
		StartDate string `json:"start_date" jsonschema_description:"开始日期，格式 YYYYMMDD"`
		EndDate   string `json:"end_date" jsonschema_description:"结束日期，格式 YYYYMMDD"`
	}

	t, err := utils.InferTool(
		"get_daily_basic",
		"获取股票每日基本面数据，包括 PE、PB、市值等指标",
		func(ctx context.Context, input *Input) (string, error) {
			return s.GetDailyBasic(ctx, input.TSCode, input.StartDate, input.EndDate)
		},
	)
	if err != nil {
		log.Fatalf("创建 get_daily_basic 工具失败: %v", err)
	}
	return t
}

// GetAllTools 获取所有股票数据 Tools
func (s *StockTools) GetAllTools() []tool.BaseTool {
	return []tool.BaseTool{
		s.NewGetStockDataTool(),
		s.NewGetStockBasicTool(),
		s.NewGetTradeCalendarTool(),
		s.NewGetDailyBasicTool(),
	}
}
