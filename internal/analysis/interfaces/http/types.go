package httpapi

import (
	"encoding/json"
	"time"
)

// ========== 请求类型 ==========

// AnalysisRequest 分析请求
type AnalysisRequest struct {
	Symbol            string   `json:"symbol"`
	TradeDate         string   `json:"trade_date,omitempty"`
	StartDate         string   `json:"start_date,omitempty"`
	EndDate           string   `json:"end_date,omitempty"`
	SelectedAnalysts  []string `json:"selected_analysts,omitempty"`
	Objective         string   `json:"objective,omitempty"`
	RiskProfile       string   `json:"risk_profile,omitempty"`
	InvestmentHorizon string   `json:"investment_horizon,omitempty"`
}

// ChatCompletionRequest 聊天补全请求
type ChatCompletionRequest struct {
	Messages         []Message `json:"messages"`
	Stream           bool      `json:"stream"`
	SelectedAnalysts []string  `json:"selected_analysts,omitempty"`
}

// Message 聊天消息
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ========== 响应类型 ==========

// AnalysisResponse 分析响应
type AnalysisResponse struct {
	JobID     string    `json:"job_id"`
	Status    string    `json:"status,omitempty"`
	CreatedAt time.Time `json:"created_at,omitempty"`
}

// JobStatusResponse 任务状态响应
type JobStatusResponse struct {
	JobID     string    `json:"job_id"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Symbol    string    `json:"symbol,omitempty"`
	TradeDate string    `json:"trade_date,omitempty"`
	Error     string    `json:"error,omitempty"`
}

// KlineResponse K线响应
type KlineResponse struct {
	Symbol    string        `json:"symbol"`
	StartDate string        `json:"start_date"`
	EndDate   string        `json:"end_date"`
	Candles   []KlineCandle `json:"candles"`
}

// KlineCandle K线数据项（前端格式）
type KlineCandle struct {
	Date          string  `json:"date"`
	Open          float64 `json:"open"`
	High          float64 `json:"high"`
	Low           float64 `json:"low"`
	Close         float64 `json:"close"`
	Volume        float64 `json:"volume,omitempty"`
	Amount        float64 `json:"amount,omitempty"`
	Change        float64 `json:"change,omitempty"`
	ChangePercent float64 `json:"change_percent,omitempty"`
	TurnoverRate  float64 `json:"turnover_rate,omitempty"`
}

// MinuteChartResponse 分时（约 1 分钟一步），数据源自通达信。
type MinuteChartResponse struct {
	Symbol string             `json:"symbol"`
	Date   string             `json:"date"`
	Points []MinuteChartPoint `json:"points"`
}

// MinuteChartPoint 分时点：时间与现价、累计量（与 TDX PriceNumber 对齐）。
type MinuteChartPoint struct {
	Time   string  `json:"time"`
	Price  float64 `json:"price"`
	Volume int     `json:"volume"`
}

// KlineItem 旧版K线数据项（向后兼容）
type KlineItem struct {
	Date      string  `json:"date"`
	Open      float64 `json:"open"`
	High      float64 `json:"high"`
	Low       float64 `json:"low"`
	Close     float64 `json:"close"`
	Volume    float64 `json:"volume"`
	Turnover  float64 `json:"turnover,omitempty"`
	Amplitude float64 `json:"amplitude,omitempty"`
}

// ConfigResponse 配置响应
type ConfigResponse struct {
	Features Features `json:"features"`
	Limits   Limits   `json:"limits"`
}

type Features struct {
	Backtest    bool `json:"backtest"`
	MultiAgent  bool `json:"multi_agent"`
	Chat        bool `json:"chat"`
	Scheduled   bool `json:"scheduled"`
	Watchlist   bool `json:"watchlist"`
	Reports     bool `json:"reports"`
	TokenManage bool `json:"token_manage"`
}

type Limits struct {
	MaxAnalysesPerDay int `json:"max_analyses_per_day"`
	MaxChatPerDay     int `json:"max_chat_per_day"`
}

// RuntimeConfig 运行时配置
type RuntimeConfig struct {
	LLMProvider           string `json:"llm_provider"`
	DeepThinkLLM          string `json:"deep_think_llm"`
	QuickThinkLLM         string `json:"quick_think_llm"`
	BackendURL            string `json:"backend_url"`
	MaxDebateRounds       int    `json:"max_debate_rounds"`
	MaxRiskDiscussRounds  int    `json:"max_risk_discuss_rounds"`
	HasAPIKey             bool   `json:"has_api_key,omitempty"`
	ServerFallbackEnabled bool   `json:"server_fallback_enabled,omitempty"`
}

// RuntimeConfigUpdate 配置更新请求
type RuntimeConfigUpdate struct {
	LLMProvider          *string `json:"llm_provider,omitempty"`
	DeepThinkLLM         *string `json:"deep_think_llm,omitempty"`
	QuickThinkLLM        *string `json:"quick_think_llm,omitempty"`
	BackendURL           *string `json:"backend_url,omitempty"`
	MaxDebateRounds      *int    `json:"max_debate_rounds,omitempty"`
	MaxRiskDiscussRounds *int    `json:"max_risk_discuss_rounds,omitempty"`
	APIKey               *string `json:"api_key,omitempty"`
	ClearAPIKey          *bool   `json:"clear_api_key,omitempty"`
}

// RuntimeConfigUpdateResponse 配置更新响应
type RuntimeConfigUpdateResponse struct {
	Message   string              `json:"message"`
	Applied   RuntimeConfigUpdate `json:"applied"`
	HasAPIKey bool                `json:"has_api_key"`
	Current   RuntimeConfig       `json:"current"`
}

// ReportListResponse 报告列表响应
type ReportListResponse struct {
	Total   int      `json:"total"`
	Reports []Report `json:"reports"`
}

// Report 报告
type Report struct {
	ID            string  `json:"id"`
	UserID        string  `json:"user_id,omitempty"`
	Symbol        string  `json:"symbol"`
	TradeDate     string  `json:"trade_date"`
	Status        string  `json:"status"`
	Error         string  `json:"error,omitempty"`
	Decision      string  `json:"decision,omitempty"`
	Direction     string  `json:"direction,omitempty"`
	Confidence    float64 `json:"confidence,omitempty"`
	TargetPrice   float64 `json:"target_price,omitempty"`
	StopLossPrice float64 `json:"stop_loss_price,omitempty"`
	CreatedAt     string  `json:"created_at,omitempty"`
	UpdatedAt     string  `json:"updated_at,omitempty"`
}

// ReportDetail 报告详情
type ReportDetail struct {
	Report
	ResultData           json.RawMessage `json:"result_data,omitempty"`
	RiskItems            json.RawMessage `json:"risk_items,omitempty"`
	KeyMetrics           json.RawMessage `json:"key_metrics,omitempty"`
	AnalystTraces        json.RawMessage `json:"analyst_traces,omitempty"`
	MarketReport         string          `json:"market_report,omitempty"`
	SentimentReport      string          `json:"sentiment_report,omitempty"`
	NewsReport           string          `json:"news_report,omitempty"`
	FundamentalsReport   string          `json:"fundamentals_report,omitempty"`
	MacroReport          string          `json:"macro_report,omitempty"`
	SmartMoneyReport     string          `json:"smart_money_report,omitempty"`
	GameTheoryReport     string          `json:"game_theory_report,omitempty"`
	InvestmentPlan       string          `json:"investment_plan,omitempty"`
	TraderInvestmentPlan string          `json:"trader_investment_plan,omitempty"`
	FinalTradeDecision   string          `json:"final_trade_decision,omitempty"`
}

// WatchlistItem 自选股项
type WatchlistItem struct {
	ID           string `json:"id"`
	Symbol       string `json:"symbol"`
	Name         string `json:"name"`
	SortOrder    int    `json:"sort_order"`
	CreatedAt    string `json:"created_at"`
	HasScheduled bool   `json:"has_scheduled"`
}

// ScheduledAnalysis 定时分析
type ScheduledAnalysis struct {
	ID                  string `json:"id"`
	Symbol              string `json:"symbol"`
	Name                string `json:"name"`
	Horizon             string `json:"horizon"`
	TriggerTime         string `json:"trigger_time"`
	IsActive            bool   `json:"is_active"`
	LastRunDate         string `json:"last_run_date,omitempty"`
	LastRunStatus       string `json:"last_run_status,omitempty"`
	LastReportID        string `json:"last_report_id,omitempty"`
	ConsecutiveFailures int    `json:"consecutive_failures"`
	CreatedAt           string `json:"created_at"`
}

// StockSearchResult 股票搜索结果
type StockSearchResult struct {
	Symbol string `json:"symbol"`
	Name   string `json:"name"`
}

// LatestAnnouncementResponse 最新公告响应
type LatestAnnouncementResponse struct {
	Announcement *Announcement `json:"announcement"`
}

// Announcement 公告
type Announcement struct {
	ID          string             `json:"id"`
	Tag         string             `json:"tag,omitempty"`
	Title       string             `json:"title"`
	Summary     string             `json:"summary,omitempty"`
	PublishedAt string             `json:"published_at"`
	Items       []AnnouncementItem `json:"items"`
	CTALabel    string             `json:"cta_label,omitempty"`
	CTAPath     string             `json:"cta_path,omitempty"`
}

// AnnouncementItem 公告项
type AnnouncementItem struct {
	Title  string `json:"title"`
	Detail string `json:"detail"`
}

// AuthUser 认证用户
type AuthUser struct {
	ID          string `json:"id"`
	Email       string `json:"email"`
	CreatedAt   string `json:"created_at,omitempty"`
	LastLoginAt string `json:"last_login_at,omitempty"`
}

// AuthVerifyResponse 认证验证响应
type AuthVerifyResponse struct {
	AccessToken string   `json:"access_token"`
	TokenType   string   `json:"token_type"`
	User        AuthUser `json:"user"`
}

// UserToken 用户Token
type UserToken struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Token      string `json:"token,omitempty"`
	TokenHint  string `json:"token_hint,omitempty"`
	LastUsedAt string `json:"last_used_at,omitempty"`
	CreatedAt  string `json:"created_at"`
}

// UserTokenCreateRequest 创建Token请求
type UserTokenCreateRequest struct {
	Name string `json:"name"`
}

// ========== 任务状态 ==========

const (
	JobStatusPending   = "pending"
	JobStatusRunning   = "running"
	JobStatusCompleted = "completed"
	JobStatusFailed    = "failed"
)

// Job 任务结构
type Job struct {
	ID        string
	Status    string
	Request   AnalysisRequest
	Result    map[string]any
	CreatedAt time.Time
	UpdatedAt time.Time
	Events    []JobEvent
}

// JobEvent 任务事件
type JobEvent struct {
	Timestamp time.Time `json:"timestamp"`
	Type      string    `json:"type"`
	Data      any       `json:"data,omitempty"`
	Message   string    `json:"message,omitempty"`
}
