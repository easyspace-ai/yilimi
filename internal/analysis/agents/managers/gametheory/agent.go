package gametheory

import (
	"context"
	"github.com/easyspace-ai/yilimi/internal/analysis/agents/common"

	"github.com/cloudwego/eino/adk"
)

// NewAgent 创建博弈论经理 Agent
func NewAgent(ctx context.Context) (adk.Agent, error) {
	return common.NewAgentBuilder("博弈论经理", "擅长整合多方观点，用博弈论思维分析多空力量对比。").
		WithInstruction(gameTheoryManagerInstruction).
		WithModel(common.NewDeepThinkModel()).
		Build(ctx)
}
