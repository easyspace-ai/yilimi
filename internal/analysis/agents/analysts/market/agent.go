package market

import (
	"context"

	"github.com/easyspace-ai/yilimi/internal/analysis/agents/common"
	"github.com/easyspace-ai/yilimi/internal/analysis/datacollect"

	"github.com/cloudwego/eino/adk"
)

// NewAgent 创建市场分析师 Agent（数据由 datacollect 注入，不启用工具调用）。
func NewAgent(ctx context.Context, pool *datacollect.Pool) (adk.Agent, error) {
	if pool == nil {
		pool = &datacollect.Pool{}
	}
	return common.NewAgentBuilder("市场分析师", "专业的技术面分析专家，擅长 K 线、技术指标和趋势判断。").
		WithInstruction(pool.MarketInstruction(marketAnalystInstruction)).
		WithModel(common.NewQuickThinkModel()).
		Build(ctx)
}
