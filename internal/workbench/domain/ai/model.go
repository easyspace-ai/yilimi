package ai

import (
	"time"

	"gorm.io/gorm"
	"gorm.io/plugin/soft_delete"
)

// AIProvider AI 提供商配置
type AIProvider struct {
	Name    string `json:"name"`
	Type    string `json:"type"` // "openai" | "deepseek" | "ark"
	BaseURL string `json:"baseURL"`
	APIKey  string `json:"-"` // 不返回给前端
	Model   string `json:"model"`
	Enabled bool   `json:"enabled"`
}

// AIConfig AI 配置
type AIConfig struct {
	Providers []AIProvider `json:"providers"`
}

// PromptTemplate 提示词模板
type PromptTemplate struct {
	ID        uint           `json:"id" gorm:"primarykey"`
	CreatedAt time.Time      `json:"createdAt"`
	UpdatedAt time.Time      `json:"updatedAt"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`

	Name    string `json:"name"`
	Content string `json:"content"`
	Type    string `json:"type"` // "stock_analysis" | "market_summary" | "custom"
}

func (PromptTemplate) TableName() string {
	return "prompt_templates"
}

// AiAssistantSession AI 会话
type AiAssistantSession struct {
	ID        uint           `json:"id" gorm:"primarykey"`
	CreatedAt time.Time      `json:"createdAt"`
	UpdatedAt time.Time      `json:"updatedAt"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`

	SessionId string `json:"sessionId" gorm:"index"`
	Messages  string `json:"messages"` // JSON 格式的消息列表
	UserId    string `json:"userId" gorm:"index"`
}

func (AiAssistantSession) TableName() string {
	return "ai_assistant_sessions"
}

// AIResponseResult AI 响应结果
type AIResponseResult struct {
	ID        uint           `json:"id" gorm:"primarykey"`
	CreatedAt time.Time      `json:"createdAt"`
	UpdatedAt time.Time      `json:"updatedAt"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`

	ChatId    string                `json:"chatId"`
	ModelName string                `json:"modelName"`
	StockCode string                `json:"stockCode"`
	StockName string                `json:"stockName"`
	Question  string                `json:"question"`
	Content   string                `json:"content"`
	IsDel     soft_delete.DeletedAt `json:"-" gorm:"softDelete:flag"`
}

func (AIResponseResult) TableName() string {
	return "ai_response_result"
}

// AiRecommendStocks AI 推荐股票
type AiRecommendStocks struct {
	ID        uint           `json:"id" gorm:"primarykey"`
	CreatedAt time.Time      `json:"createdAt"`
	UpdatedAt time.Time      `json:"updatedAt"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`

	ChatId     string  `json:"chatId"`
	StockCode  string  `json:"stockCode"`
	StockName  string  `json:"stockName"`
	Reason     string  `json:"reason"`
	Confidence float64 `json:"confidence"`
}

func (AiRecommendStocks) TableName() string {
	return "ai_recommend_stocks"
}

// ChatMessage 聊天消息
type ChatMessage struct {
	Role      string    `json:"role"` // "user" | "assistant" | "system"
	Content   string    `json:"content"`
	Timestamp time.Time `json:"timestamp"`
}

// ChatSession 聊天会话
type ChatSession struct {
	Id        string        `json:"id"`
	Messages  []ChatMessage `json:"messages"`
	CreatedAt time.Time     `json:"createdAt"`
	UpdatedAt time.Time     `json:"updatedAt"`
}

// ChatRequest 聊天请求
type ChatRequest struct {
	SessionId string `json:"sessionId"`
	Message   string `json:"message"`
	Model     string `json:"model,omitempty"`
	StockCode string `json:"stockCode,omitempty"`
}

// StockAnalysisRequest 股票分析请求
type StockAnalysisRequest struct {
	StockCode    string `json:"stockCode" binding:"required"`
	AnalysisType string `json:"analysisType"` // "technical" | "fundamental" | "comprehensive"
	Model        string `json:"model,omitempty"`
}

// StockAnalysisResponse 股票分析响应
type StockAnalysisResponse struct {
	StockCode string `json:"stockCode"`
	StockName string `json:"stockName"`
	Analysis  string `json:"analysis"`
	Summary   string `json:"summary"`
}

// AgentTool Agent 工具定义
type AgentTool struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Parameters  any    `json:"parameters,omitempty"`
}

// AgentExecutionRequest Agent 执行请求
type AgentExecutionRequest struct {
	ToolName string `json:"toolName"`
	Input    any    `json:"input"`
}

// AgentExecutionResult Agent 执行结果
type AgentExecutionResult struct {
	Success bool   `json:"success"`
	Output  any    `json:"output"`
	Error   string `json:"error,omitempty"`
}

// SentimentResult 情感分析结果
type SentimentResult struct {
	Score         float64       `json:"score"`
	Category      SentimentType `json:"category"`
	PositiveCount int           `json:"positiveCount"`
	NegativeCount int           `json:"negativeCount"`
	Description   string        `json:"description"`
}

// SentimentType 情感类型
type SentimentType int

const (
	SentimentTypeUnknown SentimentType = iota
	SentimentTypeVeryPositive
	SentimentTypePositive
	SentimentTypeNeutral
	SentimentTypeNegative
	SentimentTypeVeryNegative
)

// CronTask 定时任务
type CronTask struct {
	ID        uint           `json:"id" gorm:"primarykey"`
	CreatedAt time.Time      `json:"createdAt"`
	UpdatedAt time.Time      `json:"updatedAt"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`

	Name     string     `json:"name"`
	Type     string     `json:"type"` // "fetch_stock" | "fetch_news" | "analysis"
	CronExpr string     `json:"cronExpr"`
	Enabled  bool       `json:"enabled"`
	LastRun  *time.Time `json:"lastRun"`
	NextRun  *time.Time `json:"nextRun"`
	Config   string     `json:"config"` // JSON 配置
}

func (CronTask) TableName() string {
	return "cron_tasks"
}

// Settings 系统设置
type Settings struct {
	ID        uint      `json:"id" gorm:"primarykey"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`

	TushareToken           string `json:"tushareToken"`
	LocalPushEnable        bool   `json:"localPushEnable"`
	DingPushEnable         bool   `json:"dingPushEnable"`
	DingRobot              string `json:"dingRobot"`
	UpdateBasicInfoOnStart bool   `json:"updateBasicInfoOnStart"`
	RefreshInterval        int64  `json:"refreshInterval"`

	OpenAiEnable      bool    `json:"openAiEnable"`
	OpenAiBaseUrl     string  `json:"openAiBaseUrl"`
	OpenAiApiKey      string  `json:"openAiApiKey"`
	OpenAiModelName   string  `json:"openAiModelName"`
	OpenAiMaxTokens   int     `json:"openAiMaxTokens"`
	OpenAiTemperature float64 `json:"openAiTemperature"`
	OpenAiApiTimeOut  int     `json:"openAiApiTimeOut"`
	Prompt            string  `json:"prompt"`
	CheckUpdate       bool    `json:"checkUpdate"`
	QuestionTemplate  string  `json:"questionTemplate"`
	CrawlTimeOut      int64   `json:"crawlTimeOut"`
	KDays             int64   `json:"kDays"`
	EnableDanmu       bool    `json:"enableDanmu"`
	BrowserPath       string  `json:"browserPath"`
	EnableNews        bool    `json:"enableNews"`
	DarkTheme         bool    `json:"darkTheme"`
	BrowserPoolSize   int     `json:"browserPoolSize"`
	EnableFund        bool    `json:"enableFund"`
	EnablePushNews    bool    `json:"enablePushNews"`
	SponsorCode       string  `json:"sponsorCode"`
}

func (Settings) TableName() string {
	return "settings"
}

// VersionInfo 版本信息
type VersionInfo struct {
	ID        uint           `json:"id" gorm:"primarykey"`
	CreatedAt time.Time      `json:"createdAt"`
	UpdatedAt time.Time      `json:"updatedAt"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`

	Version           string                `json:"version"`
	Content           string                `json:"content"`
	Icon              string                `json:"icon"`
	Alipay            string                `json:"alipay"`
	Wxpay             string                `json:"wxpay"`
	Wxgzh             string                `json:"wxgzh"`
	BuildTimeStamp    int64                 `json:"buildTimeStamp"`
	OfficialStatement string                `json:"officialStatement"`
	IsDel             soft_delete.DeletedAt `json:"-" gorm:"softDelete:flag"`
}

func (VersionInfo) TableName() string {
	return "version_info"
}

// PromptTemplateQuery 分页查询参数
type PromptTemplateQuery struct {
	Page     int    `form:"page" json:"page"`
	PageSize int    `form:"pageSize" json:"pageSize"`
	Name     string `form:"name" json:"name"`
	Type     string `form:"type" json:"type"`
	Content  string `form:"content" json:"content"`
}

// AIResponseResultQuery 分页查询参数
type AIResponseResultQuery struct {
	Page      int    `form:"page" json:"page"`
	PageSize  int    `form:"pageSize" json:"pageSize"`
	ChatId    string `form:"chatId" json:"chatId"`
	ModelName string `form:"modelName" json:"modelName"`
	StockCode string `form:"stockCode" json:"stockCode"`
	StockName string `form:"stockName" json:"stockName"`
	Question  string `form:"question" json:"question"`
	StartDate string `form:"startDate" json:"startDate"`
	EndDate   string `form:"endDate" json:"endDate"`
}
