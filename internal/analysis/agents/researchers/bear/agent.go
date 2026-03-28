package bear

import (
	"context"
	"github.com/easyspace-ai/yilimi/internal/analysis/agents/common"

	"github.com/cloudwego/eino/adk"
)

// NewAgent 创建空头研究员 Agent
func NewAgent(ctx context.Context) (adk.Agent, error) {
	return common.NewAgentBuilder("空头研究员", "专业的风险分析专家，善于发现潜在风险，用数据支撑谨慎观点。").
		WithInstruction(bearResearcherInstruction).
		WithModel(common.NewDeepThinkModel()).
		Build(ctx)
}
