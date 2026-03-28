package trader

import (
	"context"
	"github.com/easyspace-ai/yilimi/internal/analysis/agents/common"
	"github.com/easyspace-ai/yilimi/internal/analysis/tools"

	"github.com/cloudwego/eino/adk"
)

// NewAgent 创建交易员 Agent
func NewAgent(ctx context.Context) (adk.Agent, error) {
	return common.NewAgentBuilder("交易员", "专业的交易计划制定者，擅长将研究结论转化为可执行的交易计划。").
		WithInstruction(traderInstruction).
		WithTools(tools.GetGlobalTools()...).
		WithModel(common.NewDeepThinkModel()).
		Build(ctx)
}
