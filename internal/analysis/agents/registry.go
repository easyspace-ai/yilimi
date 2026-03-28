package agents

import (
	"context"
	"sync"

	"github.com/easyspace-ai/yilimi/internal/analysis/agents/analysts/fundamentals"
	"github.com/easyspace-ai/yilimi/internal/analysis/agents/analysts/macro"
	"github.com/easyspace-ai/yilimi/internal/analysis/agents/analysts/market"
	"github.com/easyspace-ai/yilimi/internal/analysis/agents/analysts/news"
	"github.com/easyspace-ai/yilimi/internal/analysis/agents/analysts/sentiment"
	"github.com/easyspace-ai/yilimi/internal/analysis/agents/analysts/smartmoney"
	"github.com/easyspace-ai/yilimi/internal/analysis/agents/managers/gametheory"
	"github.com/easyspace-ai/yilimi/internal/analysis/agents/managers/research"
	"github.com/easyspace-ai/yilimi/internal/analysis/agents/managers/risk"
	"github.com/easyspace-ai/yilimi/internal/analysis/agents/researchers/bear"
	"github.com/easyspace-ai/yilimi/internal/analysis/agents/researchers/bull"
	"github.com/easyspace-ai/yilimi/internal/analysis/agents/riskmgmt/aggressive"
	"github.com/easyspace-ai/yilimi/internal/analysis/agents/riskmgmt/conservative"
	"github.com/easyspace-ai/yilimi/internal/analysis/agents/riskmgmt/judge"
	"github.com/easyspace-ai/yilimi/internal/analysis/agents/riskmgmt/neutral"
	"github.com/easyspace-ai/yilimi/internal/analysis/agents/trader"
	"github.com/easyspace-ai/yilimi/internal/analysis/datacollect"

	"github.com/cloudwego/eino/adk"
)

// AgentInfo 智能体信息
type AgentInfo struct {
	Name        string
	Description string
	Creator     func(ctx context.Context) (adk.Agent, error)
}

var (
	// Registry 智能体注册表
	Registry     []AgentInfo
	registryOnce sync.Once
)

// initRegistry 初始化注册表
func initRegistry() {
	registryOnce.Do(func() {
		Registry = []AgentInfo{
			// 六名分析师由 GetAllAnalysts(ctx, pool) 构建（需注入数据池）。

			// 研究员团队
			{Name: "多头研究员", Description: "专业的多头分析专家", Creator: bull.NewAgent},
			{Name: "空头研究员", Description: "专业的风险分析专家", Creator: bear.NewAgent},

			// 经理层
			{Name: "博弈论经理", Description: "擅长整合多方观点", Creator: gametheory.NewAgent},
			{Name: "研究经理", Description: "负责裁决多空辩论", Creator: research.NewAgent},
			{Name: "风险经理", Description: "负责整合所有分析，形成最终决策", Creator: risk.NewAgent},

			// 交易与风控
			{Name: "交易员", Description: "专业的交易计划制定者", Creator: trader.NewAgent},
			{Name: "激进风控", Description: "从积极角度评估风险收益比", Creator: aggressive.NewAgent},
			{Name: "保守风控", Description: "从谨慎角度评估风险", Creator: conservative.NewAgent},
			{Name: "中性风控", Description: "从中立平衡角度评估风险收益", Creator: neutral.NewAgent},
			{Name: "风险法官", Description: "负责裁决三方风控辩论", Creator: judge.NewAgent},
		}
	})
}

// GetRegistry 获取智能体注册表
func GetRegistry() []AgentInfo {
	initRegistry()
	return Registry
}

// GetAllAnalysts 获取所有分析师（注入 datacollect 数据池；nil 则使用空池）。
func GetAllAnalysts(ctx context.Context, pool *datacollect.Pool) ([]adk.Agent, error) {
	if pool == nil {
		pool = &datacollect.Pool{}
	}
	builders := []func(context.Context, *datacollect.Pool) (adk.Agent, error){
		market.NewAgent,
		sentiment.NewAgent,
		news.NewAgent,
		fundamentals.NewAgent,
		macro.NewAgent,
		smartmoney.NewAgent,
	}
	var out []adk.Agent
	for _, b := range builders {
		a, err := b(ctx, pool)
		if err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, nil
}

// GetRiskDebaters 获取所有风控辩论师
func GetRiskDebaters(ctx context.Context) ([]adk.Agent, error) {
	initRegistry()
	var agents []adk.Agent
	for _, info := range Registry {
		switch info.Name {
		case "激进风控", "保守风控", "中性风控":
			agent, err := info.Creator(ctx)
			if err != nil {
				return nil, err
			}
			agents = append(agents, agent)
		}
	}
	return agents, nil
}
