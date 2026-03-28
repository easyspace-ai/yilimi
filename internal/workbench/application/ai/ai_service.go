package ai

import (
	"github.com/easyspace-ai/yilimi/internal/workbench/domain/ai"
	"github.com/easyspace-ai/yilimi/internal/workbench/ports"
)

// AIServiceImpl AI 服务实现
type AIServiceImpl struct{}

// NewAIService 创建 AI 服务实例
func NewAIService() ports.AIService {
	return &AIServiceImpl{}
}

func (s *AIServiceImpl) GetConfigs() (*ai.AIConfig, error) {
	return nil, nil
}

func (s *AIServiceImpl) GetPromptTemplates(query *ai.PromptTemplateQuery) ([]ai.PromptTemplate, int64, error) {
	return nil, 0, nil
}

func (s *AIServiceImpl) GetPromptTemplate(id uint) (*ai.PromptTemplate, error) {
	return nil, nil
}

func (s *AIServiceImpl) CreatePromptTemplate(template *ai.PromptTemplate) error {
	return nil
}

func (s *AIServiceImpl) UpdatePromptTemplate(template *ai.PromptTemplate) error {
	return nil
}

func (s *AIServiceImpl) DeletePromptTemplate(id uint) error {
	return nil
}

func (s *AIServiceImpl) GetSession(sessionId, userId string) (*ai.ChatSession, error) {
	return nil, nil
}

func (s *AIServiceImpl) SaveSession(sessionId, userId string, messages []ai.ChatMessage) error {
	return nil
}

func (s *AIServiceImpl) ChatStream(request *ai.ChatRequest, ch chan<- string) error {
	return nil
}

func (s *AIServiceImpl) SummaryStream(request *ai.ChatRequest, ch chan<- string) error {
	return nil
}

func (s *AIServiceImpl) AnalyzeStock(request *ai.StockAnalysisRequest) (*ai.StockAnalysisResponse, error) {
	return nil, nil
}

func (s *AIServiceImpl) GetAIResults(query *ai.AIResponseResultQuery) ([]ai.AIResponseResult, int64, error) {
	return nil, 0, nil
}

func (s *AIServiceImpl) SaveAIResult(result *ai.AIResponseResult) error {
	return nil
}

func (s *AIServiceImpl) GetRecommendStocks(chatId string) ([]ai.AiRecommendStocks, error) {
	return nil, nil
}

func (s *AIServiceImpl) SaveRecommendStock(recommend *ai.AiRecommendStocks) error {
	return nil
}

func (s *AIServiceImpl) GetAgentTools() ([]ai.AgentTool, error) {
	return nil, nil
}

func (s *AIServiceImpl) ExecuteAgentTool(request *ai.AgentExecutionRequest) (*ai.AgentExecutionResult, error) {
	return nil, nil
}

func (s *AIServiceImpl) GetSettings() (*ai.Settings, error) {
	return nil, nil
}

func (s *AIServiceImpl) SaveSettings(settings *ai.Settings) error {
	return nil
}

func (s *AIServiceImpl) GetCronTasks() ([]ai.CronTask, error) {
	return nil, nil
}

func (s *AIServiceImpl) CreateCronTask(task *ai.CronTask) error {
	return nil
}

func (s *AIServiceImpl) UpdateCronTask(task *ai.CronTask) error {
	return nil
}

func (s *AIServiceImpl) DeleteCronTask(id uint) error {
	return nil
}

func (s *AIServiceImpl) ToggleCronTask(id uint, enabled bool) error {
	return nil
}
