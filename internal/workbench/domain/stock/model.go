package stock

import (
	"time"

	"gorm.io/gorm"
)

// StockInfo A股基本信息
type StockInfo struct {
	ID        uint           `json:"id" gorm:"primarykey"`
	CreatedAt time.Time      `json:"createdAt"`
	UpdatedAt time.Time      `json:"updatedAt"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`

	Code     string `json:"code" gorm:"index;uniqueIndex:idx_code_market"`
	Name     string `json:"name"`
	Market   string `json:"market" gorm:"index;uniqueIndex:idx_code_market"` // 沪/深/京
	Industry string `json:"industry"`
	Concept  string `json:"concept"`
	ListDate string `json:"listDate"`
}

func (StockInfo) TableName() string {
	return "stock_info"
}

// StockQuote 实时行情数据
type StockQuote struct {
	Symbol     string  `json:"symbol"`
	Name       string  `json:"name"`
	Price      float64 `json:"price"`
	Change     float64 `json:"change"`
	ChangePct  float64 `json:"changePct"`
	Open       float64 `json:"open"`
	High       float64 `json:"high"`
	Low        float64 `json:"low"`
	PrevClose  float64 `json:"prevClose"`
	Volume     int64   `json:"volume"`
	Amount     float64 `json:"amount"`
	UpdateTime string  `json:"updateTime"`
}

// KLineItem K线数据项
type KLineItem struct {
	Date   string  `json:"date"`
	Open   float64 `json:"open"`
	Close  float64 `json:"close"`
	High   float64 `json:"high"`
	Low    float64 `json:"low"`
	Volume int64   `json:"volume"`
	Amount float64 `json:"amount"`
	Change float64 `json:"change"`
}

// KLineData K线完整数据
type KLineData struct {
	Code string      `json:"code"`
	Name string      `json:"name"`
	List []KLineItem `json:"list"`
}

// DailyKLineCache 日线K线持久化缓存
type DailyKLineCache struct {
	ID         uint      `json:"id" gorm:"primarykey"`
	CreatedAt  time.Time `json:"createdAt"`
	UpdatedAt  time.Time `json:"updatedAt"`
	Symbol     string    `json:"symbol" gorm:"index;uniqueIndex:idx_daily_kline_unique"`
	Date       string    `json:"date" gorm:"index;uniqueIndex:idx_daily_kline_unique"` // YYYY-MM-DD
	KLineType  string    `json:"klineType" gorm:"index;uniqueIndex:idx_daily_kline_unique"`
	AdjustFlag string    `json:"adjustFlag" gorm:"index;uniqueIndex:idx_daily_kline_unique"`
	Name       string    `json:"name"`
	Open       float64   `json:"open"`
	Close      float64   `json:"close"`
	High       float64   `json:"high"`
	Low        float64   `json:"low"`
	Volume     int64     `json:"volume"`
	Amount     float64   `json:"amount"`
	Change     float64   `json:"change"`
}

func (DailyKLineCache) TableName() string {
	return "daily_kline_cache"
}

// TechnicalIndicator 技术指标枚举
type TechnicalIndicator string

const (
	// 形态类
	MacdGoldenCross    TechnicalIndicator = "MACD_GOLDEN_CROSS"
	KdjGoldenCross     TechnicalIndicator = "KDJ_GOLDEN_CROSS"
	BreakThrough       TechnicalIndicator = "BREAK_THROUGH"
	MaLongArrangement  TechnicalIndicator = "LONG_AVG_ARRAY"
	MaShortArrangement TechnicalIndicator = "SHORT_AVG_ARRAY"
	MorningStar        TechnicalIndicator = "MORNING_STAR"
	EveningStar        TechnicalIndicator = "EVENING_STAR"

	// 成交量类
	HighVolumeUpside  TechnicalIndicator = "UPSIDE_VOLUME"
	LowVolumeDownside TechnicalIndicator = "DOWN_NARROW_VOLUME"

	// 均线突破类
	BreakMa5      TechnicalIndicator = "BREAKUP_MA_5DAYS"
	ConsecutiveUp TechnicalIndicator = "UPP_DAYS"
)

// StockSelectionCriteria 选股条件
type StockSelectionCriteria struct {
	Codes               []string             `json:"codes"`
	Market              string               `json:"market"`
	Industry            string               `json:"industry"`
	Concept             string               `json:"concept"`
	MinPrice            float64              `json:"minPrice"`
	MaxPrice            float64              `json:"maxPrice"`
	MinChange           float64              `json:"minChange"`
	MaxChange           float64              `json:"maxChange"`
	MinPe               float64              `json:"minPe"`
	MaxPe               float64              `json:"maxPe"`
	TechnicalIndicators []TechnicalIndicator `json:"technicalIndicators"`
}

// StockSelectionResult 选股结果
type StockSelectionResult struct {
	Total int64       `json:"total"`
	List  []StockInfo `json:"list"`
}

// HotStrategy 热门策略响应
type HotStrategy struct {
	ChgEffect bool               `json:"chgEffect"`
	Code      int                `json:"code"`
	Data      []*HotStrategyData `json:"data"`
	Message   string             `json:"message"`
}

// HotStrategyData 热门策略数据
type HotStrategyData struct {
	Chg       float64 `json:"chg" md:"平均涨幅(%)"`
	Code      string  `json:"code" md:"-"`
	HeatValue int     `json:"heatValue" md:"热度值"`
	Market    string  `json:"market" md:"-"`
	Question  string  `json:"question" md:"选股策略"`
	Rank      int     `json:"rank" md:"-"`
}

// FollowedStock 关注股票
type FollowedStock struct {
	ID        uint           `json:"id" gorm:"primarykey"`
	CreatedAt time.Time      `json:"createdAt"`
	UpdatedAt time.Time      `json:"updatedAt"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`

	StockCode string  `json:"stockCode" gorm:"index"`
	StockName string  `json:"stockName"`
	Note      string  `json:"note" gorm:"size:512"`
	IsStarred bool    `json:"isStarred" gorm:"default:false"`
	CostPrice float64 `json:"costPrice"`
	Quantity  float64 `json:"quantity"`
	Sort      int     `json:"sort"`
	UserId    string  `json:"userId" gorm:"index"`
}

func (FollowedStock) TableName() string {
	return "followed_stock"
}

// StockAlarm 价格预警
type StockAlarm struct {
	ID        uint           `json:"id" gorm:"primarykey"`
	CreatedAt time.Time      `json:"createdAt"`
	UpdatedAt time.Time      `json:"updatedAt"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`

	StockCode string  `json:"stockCode" gorm:"index"`
	StockName string  `json:"stockName"`
	HighPrice float64 `json:"highPrice"`
	LowPrice  float64 `json:"lowPrice"`
	Enabled   bool    `json:"enabled"`
	UserId    string  `json:"userId" gorm:"index"`
}

func (StockAlarm) TableName() string {
	return "stock_alarm"
}

// AllStockInfo 扩展股票信息（用于高级筛选）
type AllStockInfo struct {
	ID        uint           `json:"id" gorm:"primarykey"`
	CreatedAt time.Time      `json:"createdAt"`
	UpdatedAt time.Time      `json:"updatedAt"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`

	SecurityCode           string  `json:"securityCode" gorm:"index"`
	SecurityNameAbbr       string  `json:"securityNameAbbr"`
	Market                 string  `json:"market"`
	Industry               string  `json:"industry"`
	Concept                string  `json:"concept"`
	LatestPrice            float64 `json:"latestPrice"`
	ChangePercent          float64 `json:"changePercent"`
	PeRatio                float64 `json:"peRatio"`
	PbRatio                float64 `json:"pbRatio"`
	TotalMarketValue       float64 `json:"totalMarketValue"`
	CirculatingMarketValue float64 `json:"circulatingMarketValue"`
	TurnoverRate           float64 `json:"turnoverRate"`
	VolumeRatio            float64 `json:"volumeRatio"`
}

func (AllStockInfo) TableName() string {
	return "all_stock_info"
}

// FinancialInfo 财务信息
type FinancialInfo struct {
	StockCode  string  `json:"stockCode"`
	StockName  string  `json:"stockName"`
	Pe         float64 `json:"pe"`
	Pb         float64 `json:"pb"`
	Ps         float64 `json:"ps"`
	Roe        float64 `json:"roe"`
	NetProfit  float64 `json:"netProfit"`
	Revenue    float64 `json:"revenue"`
	ReportDate string  `json:"reportDate"`
}

// MoneyFlowInfo 资金流向信息
type MoneyFlowInfo struct {
	Date                string  `json:"date"`
	MainNetInflow       float64 `json:"mainNetInflow"`
	MainNetRatio        float64 `json:"mainNetRatio"`
	SuperLargeNetInflow float64 `json:"superLargeNetInflow"`
	LargeNetInflow      float64 `json:"largeNetInflow"`
	MediumNetInflow     float64 `json:"mediumNetInflow"`
	SmallNetInflow      float64 `json:"smallNetInflow"`
}

// RZRQInfo 融资融券信息
type RZRQInfo struct {
	Date   string  `json:"date"`
	Rzye   float64 `json:"rzye"`   // 融资余额
	RzBuy  float64 `json:"rzBuy"`  // 融资买入
	Rqye   float64 `json:"rqye"`   // 融券余额
	RqSell float64 `json:"rqSell"` // 融券卖出
}

// StockGroup 股票分组
type StockGroup struct {
	ID        uint           `json:"id" gorm:"primarykey"`
	CreatedAt time.Time      `json:"createdAt"`
	UpdatedAt time.Time      `json:"updatedAt"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`

	Name   string `json:"name"`
	Sort   int    `json:"sort"`
	UserId string `json:"userId" gorm:"index"`
}

func (StockGroup) TableName() string {
	return "stock_group"
}

// StockGroupItem 股票分组成员
type StockGroupItem struct {
	ID        uint      `json:"id" gorm:"primarykey"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`

	GroupID   uint   `json:"groupId" gorm:"index"`
	StockCode string `json:"stockCode"`
	StockName string `json:"stockName"`
}

// MarketQuote 市场行情数据（用于选股）
type MarketQuote struct {
	Code                 string  `json:"code"`                 // 股票代码（6位）
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
	Pe                   float64 `json:"pe"`                   // 市盈率
	Pb                   float64 `json:"pb"`                   // 市净率
	Market               string  `json:"market"`               // 市场：SH/SZ/BJ
}

// PickerStrategy 选股策略类型
type PickerStrategy string

const (
	PickerStrategyEndOfDay PickerStrategy = "end_of_day" // 尾盘选股法
	PickerStrategyMomentum PickerStrategy = "momentum"   // 妖股候选人
	PickerStrategyKunpeng  PickerStrategy = "kunpeng"    // 鲲鹏战法
)

// EndOfDayPickerRequest 尾盘选股请求
type EndOfDayPickerRequest struct {
	MarketCapMin          float64 `json:"marketCapMin"`          // 最小市值（亿）
	MarketCapMax          float64 `json:"marketCapMax"`          // 最大市值（亿）
	VolumeRatioMin        float64 `json:"volumeRatioMin"`        // 最小量比
	ChangePercentMin      float64 `json:"changePercentMin"`      // 最小涨幅（%）
	ChangePercentMax      float64 `json:"changePercentMax"`      // 最大涨幅（%）
	TurnoverRateMin       float64 `json:"turnoverRateMin"`       // 最小换手率（%）
	TurnoverRateMax       float64 `json:"turnoverRateMax"`       // 最大换手率（%）
	ExcludeST             bool    `json:"excludeST"`             // 排除ST
	TimelineAboveAvgRatio float64 `json:"timelineAboveAvgRatio"` // 分时强度阈值（%）
}

// MomentumPickerRequest 妖股候选人请求
type MomentumPickerRequest struct {
	MomentumThreshold float64 `json:"momentumThreshold"` // 动量阈值（%）
	TrendAboveMA60    bool    `json:"trendAboveMA60"`    // 高于60日均线
	AvgTurnoverMin    float64 `json:"avgTurnoverMin"`    // 最小换手率（%）
	MarketCapMin      float64 `json:"marketCapMin"`      // 最小市值（亿）
	MarketCapMax      float64 `json:"marketCapMax"`      // 最大市值（亿）
	ExcludeST         bool    `json:"excludeST"`         // 排除ST
	PriceMin          float64 `json:"priceMin"`          // 最低价格
	PriceMax          float64 `json:"priceMax"`          // 最高价格
}

// KunpengPickerRequest 鲲鹏战法请求
type KunpengPickerRequest struct {
	MarketCapMin    float64 `json:"marketCapMin"`    // 最小市值（亿）
	MarketCapMax    float64 `json:"marketCapMax"`    // 最大市值（亿）
	NetProfitMin    float64 `json:"netProfitMin"`    // 最小净利润（亿）
	PeMin           float64 `json:"peMin"`           // 最小PE
	PeMax           float64 `json:"peMax"`           // 最大PE
	ExcludeST       bool    `json:"excludeST"`       // 排除ST
	ExcludeNewStock bool    `json:"excludeNewStock"` // 排除次新股
	PriceMin        float64 `json:"priceMin"`        // 最低价格
	PriceMax        float64 `json:"priceMax"`        // 最高价格
}

// PickerStockResult 选股结果项
type PickerStockResult struct {
	// 基础信息
	Code                 string  `json:"code"`
	Name                 string  `json:"name"`
	Price                float64 `json:"price"`
	ChangePercent        float64 `json:"changePercent"`
	Change               float64 `json:"change"`
	Volume               int64   `json:"volume"`
	Amount               float64 `json:"amount"`
	TurnoverRate         float64 `json:"turnoverRate"`
	VolumeRatio          float64 `json:"volumeRatio"`
	CirculatingMarketCap float64 `json:"circulatingMarketCap"`
	TotalMarketCap       float64 `json:"totalMarketCap"`
	Pe                   float64 `json:"pe"`
	Pb                   float64 `json:"pb"`
	High                 float64 `json:"high"`
	Low                  float64 `json:"low"`
	Open                 float64 `json:"open"`
	PrevClose            float64 `json:"prevClose"`
	Market               string  `json:"market"`

	// 尾盘选股特有
	Timeline              []TimelinePoint `json:"timeline,omitempty"`
	TimelineAboveAvgRatio float64         `json:"timelineAboveAvgRatio,omitempty"`

	// 妖股候选人特有
	MomentumRatio float64 `json:"momentumRatio,omitempty"`
	Ma60Distance  float64 `json:"ma60Distance,omitempty"`
	AvgTurnover5d float64 `json:"avgTurnover5d,omitempty"`
	High20d       float64 `json:"high20d,omitempty"`
	Low20d        float64 `json:"low20d,omitempty"`
	Ma60          float64 `json:"ma60,omitempty"`

	// 鲲鹏战法特有
	NetProfit         float64 `json:"netProfit,omitempty"`
	SafetyScore       float64 `json:"safetyScore,omitempty"`
	PotentialMultiple float64 `json:"potentialMultiple,omitempty"`
}

// TimelinePoint 分时数据点
type TimelinePoint struct {
	Time     string  `json:"time"`
	Price    float64 `json:"price"`
	AvgPrice float64 `json:"avgPrice"`
}

// PickerResponse 选股响应
type PickerResponse struct {
	Strategy PickerStrategy      `json:"strategy"`
	Total    int                 `json:"total"`
	List     []PickerStockResult `json:"list"`
}

func (StockGroupItem) TableName() string {
	return "stock_group_item"
}
