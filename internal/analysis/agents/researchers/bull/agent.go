package bull

import (
	"context"
	"github.com/easyspace-ai/yilimi/internal/analysis/agents/common"

	"github.com/cloudwego/eino/adk"
)

// NewAgent 创建多头研究员 Agent
func NewAgent(ctx context.Context) (adk.Agent, error) {
	return common.NewAgentBuilder("多头研究员", "专业的多头分析专家，善于发现投资机会，用数据支撑乐观观点。").
		WithInstruction(bullResearcherInstruction).
		WithModel(common.NewDeepThinkModel()).
		Build(ctx)
}
