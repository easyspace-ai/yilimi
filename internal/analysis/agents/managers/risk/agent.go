package risk

import (
	"context"
	"github.com/easyspace-ai/yilimi/internal/analysis/agents/common"

	"github.com/cloudwego/eino/adk"
)

// NewAgent 创建风险经理 Agent
func NewAgent(ctx context.Context) (adk.Agent, error) {
	return common.NewAgentBuilder("风险经理", "负责整合所有分析，形成最终交易决策和投资建议。").
		WithInstruction(riskManagerInstruction).
		WithModel(common.NewDeepThinkModel()).
		Build(ctx)
}
