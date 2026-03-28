package macro

import (
	"context"
	"github.com/easyspace-ai/yilimi/internal/analysis/agents/common"

	"github.com/cloudwego/eino/adk"
)

// NewAgent 创建宏观分析师 Agent
func NewAgent(ctx context.Context) (adk.Agent, error) {
	return common.NewAgentBuilder("宏观分析师", "专业的宏观经济分析专家，擅长从政策和经济大环境角度分析。").
		WithInstruction(macroAnalystInstruction).
		WithModel(common.NewQuickThinkModel()).
		Build(ctx)
}
