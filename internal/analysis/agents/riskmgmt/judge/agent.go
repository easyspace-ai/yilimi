package judge

import (
	"context"
	"github.com/easyspace-ai/yilimi/internal/analysis/agents/common"

	"github.com/cloudwego/eino/adk"
)

// NewAgent 创建风险法官 Agent
func NewAgent(ctx context.Context) (adk.Agent, error) {
	return common.NewAgentBuilder("风险法官", "负责裁决三方风控辩论，形成最终风控结论，决定是否需要修订交易计划。").
		WithInstruction(judgeInstruction).
		WithModel(common.NewDeepThinkModel()).
		Build(ctx)
}
