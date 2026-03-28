package market

import (
	"time"

	"gorm.io/gorm"
)

// ensure gorm is used
var _ = gorm.Model{}

// LongTigerRank 龙虎榜数据
type LongTigerRank struct {
	ID        uint      `json:"id" gorm:"primarykey"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`

	TradeDate        string  `json:"tradeDate" gorm:"index"`
	SecurityCode     string  `json:"securityCode" gorm:"index"`
	SecuCode         string  `json:"secuCode"`
	SecurityNameAbbr string  `json:"securityNameAbbr"`
	ClosePrice       float64 `json:"closePrice"`
	ChangeRate       float64 `json:"changeRate"`
	AccumAmount      float64 `json:"accumAmount"`
	BillboardBuyAmt  float64 `json:"billboardBuyAmt"`
	BillboardSellAmt float64 `json:"billboardSellAmt"`
	BillboardNetAmt  float64 `json:"billboardNetAmt"`
	BillboardDealAmt float64 `json:"billboardDealAmt"`
	Explanation      string  `json:"explanation"`
	TurnoverRate     float64 `json:"turnoverRate"`
	FreeMarketCap    float64 `json:"freeMarketCap"`
}

func (LongTigerRank) TableName() string {
	return "long_tiger_rank"
}

// MarketNews 市场新闻
type MarketNews struct {
	ID        uint      `json:"id" gorm:"primarykey"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`

	Title       string    `json:"title" gorm:"index"`
	Content     string    `json:"content"`
	Source      string    `json:"source"`
	Url         string    `json:"url"`
	PublishTime time.Time `json:"publishTime" gorm:"index"`
	StockCodes  string    `json:"stockCodes"` // 相关股票代码，逗号分隔
	Tags        string    `json:"tags"`       // 标签，逗号分隔
}

func (MarketNews) TableName() string {
	return "market_news"
}

// Telegraph 快讯
type Telegraph struct {
	ID        uint      `json:"id" gorm:"primarykey"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`

	Title           string     `json:"title" gorm:"index"`
	Content         string     `json:"content" gorm:"index"`
	Source          string     `json:"source" gorm:"index"`
	Url             string     `json:"url"`
	DataTime        *time.Time `json:"dataTime" gorm:"index"`
	IsRed           bool       `json:"isRed" gorm:"index"`
	SubjectTags     string     `json:"subjectTags"`                  // 主题标签，JSON数组
	StockTags       string     `json:"stockTags"`                    // 股票标签，JSON数组
	SentimentResult string     `json:"sentimentResult" gorm:"index"` // 情感分析结果
}

func (Telegraph) TableName() string {
	return "telegraph_list"
}

// HotStock 热门股票
type HotStock struct {
	Code       string  `json:"code"`
	Name       string  `json:"name"`
	Value      float64 `json:"value"`      // 热度值
	Increment  int     `json:"increment"`  // 热度变化
	RankChange int     `json:"rankChange"` // 排名变化
	Percent    float64 `json:"percent"`    // 涨跌幅
	Current    float64 `json:"current"`    // 当前价格
	Chg        float64 `json:"chg"`        // 价格变化
	Exchange   string  `json:"exchange"`   // 交易所
}

// HotEvent 热门事件
type HotEvent struct {
	ID          int    `json:"id"`
	Title       string `json:"title"`
	Content     string `json:"content"`
	Tag         string `json:"tag"`
	Pic         string `json:"pic"`
	Hot         int    `json:"hot"`
	StatusCount int    `json:"statusCount"`
}

// HotTopic 热门话题
type HotTopic struct {
	ID         int    `json:"id"`
	Title      string `json:"title"`
	Content    string `json:"content"`
	Hot        int    `json:"hot"`
	StockCount int    `json:"stockCount"`
}

// ResearchReport 研报
type ResearchReport struct {
	ID        uint      `json:"id" gorm:"primarykey"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`

	Title       string `json:"title"`
	Content     string `json:"content"`
	StockCode   string `json:"stockCode" gorm:"index"`
	StockName   string `json:"stockName"`
	Author      string `json:"author"`
	OrgName     string `json:"orgName"`
	PublishDate string `json:"publishDate" gorm:"index"`
	ReportType  string `json:"reportType"` // "stock" | "industry"
	Url         string `json:"url"`
}

func (ResearchReport) TableName() string {
	return "research_report"
}

// StockNotice 个股公告
type StockNotice struct {
	ID        uint      `json:"id" gorm:"primarykey"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`

	Title       string `json:"title"`
	Content     string `json:"content"`
	StockCode   string `json:"stockCode" gorm:"index"`
	StockName   string `json:"stockName"`
	NoticeType  string `json:"noticeType"`
	PublishDate string `json:"publishDate" gorm:"index"`
	UpdateTime  string `json:"updateTime"`
	Url         string `json:"url"`
}

func (StockNotice) TableName() string {
	return "stock_notice"
}

// IndustryMoneyRank 行业资金排名
type IndustryMoneyRank struct {
	IndustryName  string  `json:"industryName"`
	ChangePct     float64 `json:"changePct"`
	Inflow        float64 `json:"inflow"`
	Outflow       float64 `json:"outflow"`
	NetInflow     float64 `json:"netInflow"`
	NetRatio      float64 `json:"netRatio"`
	LeadStock     string  `json:"leadStock"`
	LeadStockCode string  `json:"leadStockCode"`
	LeadChange    float64 `json:"leadChange"`
	LeadPrice     float64 `json:"leadPrice"`
	LeadNetRatio  float64 `json:"leadNetRatio"`
}

// IndustryRank 行业涨幅排名
type IndustryRank struct {
	IndustryName  string  `json:"industryName"`
	IndustryCode  string  `json:"industryCode"`
	ChangePct     float64 `json:"changePct"`
	ChangePct5d   float64 `json:"changePct5d"`
	ChangePct20d  float64 `json:"changePct20d"`
	LeadStock     string  `json:"leadStock"`
	LeadStockCode string  `json:"leadStockCode"`
	LeadChange    float64 `json:"leadChange"`
	LeadPrice     float64 `json:"leadPrice"`
}

// StockMoneyRank 股票资金排名
type StockMoneyRank struct {
	Code         string  `json:"code"`
	Name         string  `json:"name"`
	Price        float64 `json:"price"`
	ChangePct    float64 `json:"changePct"`
	TurnoverRate float64 `json:"turnoverRate"`
	Amount       float64 `json:"amount"`
	OutAmount    float64 `json:"outAmount"`
	InAmount     float64 `json:"inAmount"`
	NetAmount    float64 `json:"netAmount"`
	NetRatio     float64 `json:"netRatio"`
	R0Out        float64 `json:"r0Out"`
	R0In         float64 `json:"r0In"`
	R0Net        float64 `json:"r0Net"`
	R0Ratio      float64 `json:"r0Ratio"`
	R3Out        float64 `json:"r3Out"`
	R3In         float64 `json:"r3In"`
	R3Net        float64 `json:"r3Net"`
	R3Ratio      float64 `json:"r3Ratio"`
}

// GlobalIndex 全球指数
type GlobalIndex struct {
	Name       string  `json:"name"`
	Code       string  `json:"code"`
	Price      float64 `json:"price"`
	Change     float64 `json:"change"`
	ChangePct  float64 `json:"changePct"`
	UpdateTime string  `json:"updateTime"`
	// Region from data source board: common | america | asia | europe | other
	Region string `json:"region"`
}

// InvestCalendarItem 投资日历项
type InvestCalendarItem struct {
	Date    string `json:"date"`
	Title   string `json:"title"`
	Content string `json:"content"`
	Type    string `json:"type"` // "ipo" | "listing" | "dividend" | "meeting" | "other"
}

// BKDict 板块词典
type BKDict struct {
	ID        uint      `json:"id" gorm:"primarykey"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`

	BkCode      string `json:"bkCode"`
	BkName      string `json:"bkName"`
	FirstLetter string `json:"firstLetter"`
	FubkCode    string `json:"fubkCode"`
	PublishCode string `json:"publishCode"`
}

func (BKDict) TableName() string {
	return "bk_dict"
}

// MarketIndex 市场指数
type MarketIndex struct {
	Code      string  `json:"code"`
	Name      string  `json:"name"`
	Price     float64 `json:"price"`
	Change    float64 `json:"change"`
	ChangePct float64 `json:"changePct"`
	Volume    int64   `json:"volume"`
	Amount    float64 `json:"amount"`
}

// MoneyFlowInfo 资金流向信息（市场模块复用）
type MoneyFlowInfo struct {
	Date                string  `json:"date"`
	MainNetInflow       float64 `json:"mainNetInflow"`
	MainNetRatio        float64 `json:"mainNetRatio"`
	SuperLargeNetInflow float64 `json:"superLargeNetInflow"`
	LargeNetInflow      float64 `json:"largeNetInflow"`
	MediumNetInflow     float64 `json:"mediumNetInflow"`
	SmallNetInflow      float64 `json:"smallNetInflow"`
}

// InteractiveAnswer 互动问答
type InteractiveAnswer struct {
	ID        uint      `json:"id" gorm:"primarykey"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`

	StockCode      string `json:"stockCode" gorm:"index"`
	StockName      string `json:"stockName"`
	Question       string `json:"question"`
	Answer         string `json:"answer"`
	QuestionAuthor string `json:"questionAuthor"`
	AnswerAuthor   string `json:"answerAuthor"`
	QuestionDate   string `json:"questionDate"`
	AnswerDate     string `json:"answerDate"`
}

func (InteractiveAnswer) TableName() string {
	return "interactive_answer"
}
