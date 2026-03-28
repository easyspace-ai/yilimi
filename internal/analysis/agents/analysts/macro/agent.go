package macro

import (
	"context"

	"github.com/easyspace-ai/yilimi/internal/analysis/agents/common"
	"github.com/easyspace-ai/yilimi/internal/analysis/datacollect"

	"github.com/cloudwego/eino/adk"
)

// NewAgent 创建宏观分析师 Agent。
func NewAgent(ctx context.Context, pool *datacollect.Pool) (adk.Agent, error) {
	if pool == nil {
		pool = &datacollect.Pool{}
	}
	return common.NewAgentBuilder("宏观分析师", "专业的宏观经济分析专家，擅长从政策和经济大环境角度分析。").
		WithInstruction(pool.MacroInstruction(macroAnalystInstruction)).
		WithModel(common.NewQuickThinkModel()).
		Build(ctx)
}
