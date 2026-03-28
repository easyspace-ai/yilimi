package common

import (
	"context"
	"fmt"
	"os"

	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/components/model"
)

// TryChatModel 从环境变量创建 ChatModel（与 backend/.env 中 OPENAI_* 命名一致；由 godotenv 加载）。
func TryChatModel(ctx context.Context) (model.ToolCallingChatModel, error) {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("OPENAI_API_KEY is required")
	}

	baseURL := os.Getenv("OPENAI_BASE_URL")
	if baseURL == "" {
		return nil, fmt.Errorf("OPENAI_BASE_URL is required")
	}

	modelName := os.Getenv("OPENAI_MODEL")
	if modelName == "" {
		return nil, fmt.Errorf("OPENAI_MODEL is required")
	}

	cm, err := openai.NewChatModel(ctx, &openai.ChatModelConfig{
		APIKey:  apiKey,
		BaseURL: baseURL,
		Model:   modelName,
	})
	if err != nil {
		return nil, fmt.Errorf("openai chat model: %w", err)
	}
	return cm, nil
}

// NewChatModel 创建默认的 ChatModel（供各 Agent 构建使用；未配置密钥时 panic）
func NewChatModel() model.ToolCallingChatModel {
	cm, err := TryChatModel(context.Background())
	if err != nil {
		panic(err.Error())
	}
	return cm
}

// NewDeepThinkModel 创建深度思考模型（用于复杂决策）
func NewDeepThinkModel() model.ToolCallingChatModel {
	// 可以用不同的模型配置
	return NewChatModel()
}

// NewQuickThinkModel 创建快速思考模型（用于分析师）
func NewQuickThinkModel() model.ToolCallingChatModel {
	// 可以用更快速、更便宜的模型
	return NewChatModel()
}
