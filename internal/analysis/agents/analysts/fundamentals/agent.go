package fundamentals

import (
	"context"

	"github.com/easyspace-ai/yilimi/internal/analysis/agents/common"
	"github.com/easyspace-ai/yilimi/internal/analysis/datacollect"

	"github.com/cloudwego/eino/adk"
)

// NewAgent 创建基本面分析师 Agent。
func NewAgent(ctx context.Context, pool *datacollect.Pool) (adk.Agent, error) {
	if pool == nil {
		pool = &datacollect.Pool{}
	}
	return common.NewAgentBuilder("基本面分析师", "专业的财务分析专家，擅长从财报和商业模式角度评估公司价值。").
		WithInstruction(pool.FundamentalsInstruction(fundamentalsAnalystInstruction)).
		WithModel(common.NewQuickThinkModel()).
		Build(ctx)
}
