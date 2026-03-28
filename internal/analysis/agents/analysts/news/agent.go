package news

import (
	"context"
	"github.com/easyspace-ai/yilimi/internal/analysis/agents/common"

	"github.com/cloudwego/eino/adk"
)

// NewAgent 创建新闻分析师 Agent
func NewAgent(ctx context.Context) (adk.Agent, error) {
	return common.NewAgentBuilder("新闻分析师", "专业的新闻分析专家，擅长从新闻中提取关键信息并分析影响。").
		WithInstruction(newsAnalystInstruction).
		WithModel(common.NewQuickThinkModel()).
		Build(ctx)
}
