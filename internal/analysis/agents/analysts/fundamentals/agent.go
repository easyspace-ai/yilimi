package fundamentals

import (
	"context"
	"github.com/easyspace-ai/yilimi/internal/analysis/agents/common"
	"github.com/easyspace-ai/yilimi/internal/analysis/tools"

	"github.com/cloudwego/eino/adk"
)

// NewAgent 创建基本面分析师 Agent
func NewAgent(ctx context.Context) (adk.Agent, error) {
	return common.NewAgentBuilder("基本面分析师", "专业的财务分析专家，擅长从财报和商业模式角度评估公司价值。").
		WithInstruction(fundamentalsAnalystInstruction).
		WithTools(tools.GetGlobalTools()...).
		WithModel(common.NewQuickThinkModel()).
		Build(ctx)
}
