package httpapi

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/easyspace-ai/yilimi/internal/workbench/domain/ai"
	"github.com/easyspace-ai/yilimi/internal/workbench/ports"
)

// AIHandler AI API 处理器
type AIHandler struct {
	aiService ports.AIService
}

// NewAIHandler 创建 AI 处理器
func NewAIHandler(aiService ports.AIService) *AIHandler {
	return &AIHandler{
		aiService: aiService,
	}
}

// RegisterRoutes 注册 AI 相关路由
func (h *AIHandler) RegisterRoutes(r *gin.RouterGroup) {
	aiGroup := r.Group("/ai")
	{
		// AI 配置
		aiGroup.GET("/configs", h.GetConfigs)

		// 提示词模板
		aiGroup.GET("/prompts", h.GetPrompts)

		// 会话管理
		aiGroup.GET("/session", h.GetSession)
		aiGroup.POST("/session", h.SaveSession)

		// 聊天
		aiGroup.POST("/chat/summary-stream", h.ChatSummaryStream)
		aiGroup.POST("/chat/stream", h.ChatStream)
		aiGroup.POST("/share", h.Share)

		// Agent 工具
		agentGroup := r.Group("/agent")
		{
			agentGroup.GET("/tools", h.GetAgentTools)
			agentGroup.POST("/execute", h.ExecuteAgentTool)
		}
	}
}

// GetConfigs 获取 AI 配置
func (h *AIHandler) GetConfigs(c *gin.Context) {
	configs, err := h.aiService.GetConfigs()
	if err != nil {
		c.JSON(http.StatusOK, Error(err.Error()))
		return
	}
	c.JSON(http.StatusOK, Success(configs))
}

// GetPrompts 获取提示词模板
func (h *AIHandler) GetPrompts(c *gin.Context) {
	var query ai.PromptTemplateQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		c.JSON(http.StatusOK, Error(err.Error()))
		return
	}

	if query.Page <= 0 {
		query.Page = 1
	}
	if query.PageSize <= 0 {
		query.PageSize = 50
	}

	templates, total, err := h.aiService.GetPromptTemplates(&query)
	if err != nil {
		c.JSON(http.StatusOK, Error(err.Error()))
		return
	}

	c.JSON(http.StatusOK, PageData(templates, total, query.Page, query.PageSize))
}

// GetSession 获取会话
func (h *AIHandler) GetSession(c *gin.Context) {
	sessionId := c.Query("sessionId")
	userId := c.GetHeader("X-User-Id")
	if userId == "" {
		userId = "default"
	}

	session, err := h.aiService.GetSession(sessionId, userId)
	if err != nil {
		c.JSON(http.StatusOK, Error(err.Error()))
		return
	}

	c.JSON(http.StatusOK, Success(session))
}

// SaveSession 保存会话
func (h *AIHandler) SaveSession(c *gin.Context) {
	var req struct {
		SessionId string           `json:"sessionId"`
		Messages  []ai.ChatMessage `json:"messages"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, Error(err.Error()))
		return
	}

	userId := c.GetHeader("X-User-Id")
	if userId == "" {
		userId = "default"
	}

	if err := h.aiService.SaveSession(req.SessionId, userId, req.Messages); err != nil {
		c.JSON(http.StatusOK, Error(err.Error()))
		return
	}

	c.JSON(http.StatusOK, Success(nil))
}

// ChatSummaryStream 聊天总结流
func (h *AIHandler) ChatSummaryStream(c *gin.Context) {
	c.JSON(http.StatusOK, Error("not implemented"))
}

// ChatStream 聊天流
func (h *AIHandler) ChatStream(c *gin.Context) {
	c.JSON(http.StatusOK, Error("not implemented"))
}

// Share 分享
func (h *AIHandler) Share(c *gin.Context) {
	c.JSON(http.StatusOK, Error("not implemented"))
}

// GetAgentTools 获取 Agent 工具列表
func (h *AIHandler) GetAgentTools(c *gin.Context) {
	tools, err := h.aiService.GetAgentTools()
	if err != nil {
		c.JSON(http.StatusOK, Error(err.Error()))
		return
	}
	c.JSON(http.StatusOK, Success(tools))
}

// ExecuteAgentTool 执行 Agent 工具
func (h *AIHandler) ExecuteAgentTool(c *gin.Context) {
	var req ai.AgentExecutionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, Error(err.Error()))
		return
	}

	result, err := h.aiService.ExecuteAgentTool(&req)
	if err != nil {
		c.JSON(http.StatusOK, Error(err.Error()))
		return
	}

	c.JSON(http.StatusOK, Success(result))
}

// GetVipStatus 获取 VIP 状态（兼容旧 API）
func GetVipStatus(c *gin.Context) {
	c.JSON(http.StatusOK, Success(gin.H{
		"vip":  true,
		"vip2": true,
	}))
}

// Health 健康检查
func Health(c *gin.Context) {
	c.JSON(http.StatusOK, Success(gin.H{
		"status": "ok",
	}))
}
