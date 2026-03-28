package sentiment

import (
	"context"
	"github.com/easyspace-ai/yilimi/internal/analysis/agents/common"

	"github.com/cloudwego/eino/adk"
)

// NewAgent 创建舆情分析师 Agent
func NewAgent(ctx context.Context) (adk.Agent, error) {
	return common.NewAgentBuilder("舆情分析师", "专业的市场情绪分析专家，擅长分析投资者心理和资金流向。").
		WithInstruction(sentimentAnalystInstruction).
		WithModel(common.NewQuickThinkModel()).
		Build(ctx)
}
