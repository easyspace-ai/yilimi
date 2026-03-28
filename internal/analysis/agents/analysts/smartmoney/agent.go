package smartmoney

import (
	"context"

	"github.com/easyspace-ai/yilimi/internal/analysis/agents/common"
	"github.com/easyspace-ai/yilimi/internal/analysis/datacollect"

	"github.com/cloudwego/eino/adk"
)

// NewAgent 创建主力资金分析师 Agent。
func NewAgent(ctx context.Context, pool *datacollect.Pool) (adk.Agent, error) {
	if pool == nil {
		pool = &datacollect.Pool{}
	}
	return common.NewAgentBuilder("主力资金分析师", "专业的资金流向分析专家，擅长追踪机构动向和聪明钱行为。").
		WithInstruction(pool.SmartMoneyInstruction(smartMoneyAnalystInstruction)).
		WithModel(common.NewQuickThinkModel()).
		Build(ctx)
}
