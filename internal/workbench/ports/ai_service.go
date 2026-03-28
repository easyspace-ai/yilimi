package ports

import (
	"github.com/easyspace-ai/yilimi/internal/workbench/domain/ai"
)

// AIService AI 服务接口
type AIService interface {
	// AI 配置
	GetConfigs() (*ai.AIConfig, error)

	// 提示词模板
	GetPromptTemplates(query *ai.PromptTemplateQuery) ([]ai.PromptTemplate, int64, error)
	GetPromptTemplate(id uint) (*ai.PromptTemplate, error)
	CreatePromptTemplate(template *ai.PromptTemplate) error
	UpdatePromptTemplate(template *ai.PromptTemplate) error
	DeletePromptTemplate(id uint) error

	// 会话管理
	GetSession(sessionId, userId string) (*ai.ChatSession, error)
	SaveSession(sessionId, userId string, messages []ai.ChatMessage) error

	// 聊天
	ChatStream(request *ai.ChatRequest, ch chan<- string) error
	SummaryStream(request *ai.ChatRequest, ch chan<- string) error

	// 股票分析
	AnalyzeStock(request *ai.StockAnalysisRequest) (*ai.StockAnalysisResponse, error)

	// AI 响应结果
	GetAIResults(query *ai.AIResponseResultQuery) ([]ai.AIResponseResult, int64, error)
	SaveAIResult(result *ai.AIResponseResult) error

	// 推荐股票
	GetRecommendStocks(chatId string) ([]ai.AiRecommendStocks, error)
	SaveRecommendStock(recommend *ai.AiRecommendStocks) error

	// Agent 工具
	GetAgentTools() ([]ai.AgentTool, error)
	ExecuteAgentTool(request *ai.AgentExecutionRequest) (*ai.AgentExecutionResult, error)

	// 设置
	GetSettings() (*ai.Settings, error)
	SaveSettings(settings *ai.Settings) error

	// 定时任务
	GetCronTasks() ([]ai.CronTask, error)
	CreateCronTask(task *ai.CronTask) error
	UpdateCronTask(task *ai.CronTask) error
	DeleteCronTask(id uint) error
	ToggleCronTask(id uint, enabled bool) error
}
