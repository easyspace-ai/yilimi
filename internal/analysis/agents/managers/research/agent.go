package research

import (
	"context"
	"github.com/easyspace-ai/yilimi/internal/analysis/agents/common"

	"github.com/cloudwego/eino/adk"
)

// NewAgent 创建研究经理 Agent
func NewAgent(ctx context.Context) (adk.Agent, error) {
	return common.NewAgentBuilder("研究经理", "负责裁决多空辩论，形成最终研究结论。").
		WithInstruction(researchManagerInstruction).
		WithModel(common.NewDeepThinkModel()).
		Build(ctx)
}
