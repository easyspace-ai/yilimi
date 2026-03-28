package smartmoney

import (
	"context"
	"github.com/easyspace-ai/yilimi/internal/analysis/agents/common"

	"github.com/cloudwego/eino/adk"
)

// NewAgent 创建主力资金分析师 Agent
func NewAgent(ctx context.Context) (adk.Agent, error) {
	return common.NewAgentBuilder("主力资金分析师", "专业的资金流向分析专家，擅长追踪机构动向和聪明钱行为。").
		WithInstruction(smartMoneyAnalystInstruction).
		WithModel(common.NewQuickThinkModel()).
		Build(ctx)
}
