package graph

import (
	"context"
	"fmt"

	"github.com/easyspace-ai/yilimi/internal/analysis/agents"
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

	"github.com/cloudwego/eino/adk"
)

// TradingWorkflow 完整的交易分析工作流
type TradingWorkflow struct {
	workflow adk.Agent
}

// NewTradingWorkflow 创建完整的交易分析工作流
func NewTradingWorkflow(ctx context.Context) (*TradingWorkflow, error) {
	// ========== 阶段 1: 6 名分析师并行分析 ==========
	analystAgents, err := agents.GetAllAnalysts(ctx)
	if err != nil {
		return nil, fmt.Errorf("get analysts: %w", err)
	}

	parallelAnalysts, err := adk.NewParallelAgent(ctx, &adk.ParallelAgentConfig{
		Name:        "分析师团队",
		Description: "6 名专业分析师（市场、舆情、新闻、基本面、宏观、主力资金）并行进行多维度分析",
		SubAgents:   analystAgents,
	})
	if err != nil {
		return nil, fmt.Errorf("create parallel analysts: %w", err)
	}

	// ========== 阶段 2: 博弈论经理 ==========
	gameTheoryManagerAgent, err := gametheory.NewAgent(ctx)
	if err != nil {
		return nil, fmt.Errorf("create game theory manager: %w", err)
	}

	// ========== 阶段 3: 多空研究员辩论 (Loop) ==========
	bullAgent, err := bull.NewAgent(ctx)
	if err != nil {
		return nil, fmt.Errorf("create bull researcher: %w", err)
	}
	bearAgent, err := bear.NewAgent(ctx)
	if err != nil {
		return nil, fmt.Errorf("create bear researcher: %w", err)
	}

	bullBearDebate, err := adk.NewLoopAgent(ctx, &adk.LoopAgentConfig{
		Name:          "多空辩论",
		Description:   "多头和空头研究员进行多轮辩论",
		SubAgents:     []adk.Agent{bullAgent, bearAgent},
		MaxIterations: 3, // 最多辩论 3 轮
	})
	if err != nil {
		return nil, fmt.Errorf("create bull bear debate: %w", err)
	}

	// ========== 阶段 4: 研究经理 ==========
	researchManagerAgent, err := research.NewAgent(ctx)
	if err != nil {
		return nil, fmt.Errorf("create research manager: %w", err)
	}

	// ========== 阶段 5: 交易员 ==========
	traderAgent, err := trader.NewAgent(ctx)
	if err != nil {
		return nil, fmt.Errorf("create trader: %w", err)
	}

	// ========== 阶段 6: 三方风控辩论 (Parallel) ==========
	aggressiveAgent, err := aggressive.NewAgent(ctx)
	if err != nil {
		return nil, fmt.Errorf("create aggressive: %w", err)
	}
	conservativeAgent, err := conservative.NewAgent(ctx)
	if err != nil {
		return nil, fmt.Errorf("create conservative: %w", err)
	}
	neutralAgent, err := neutral.NewAgent(ctx)
	if err != nil {
		return nil, fmt.Errorf("create neutral: %w", err)
	}

	riskDebate, err := adk.NewParallelAgent(ctx, &adk.ParallelAgentConfig{
		Name:        "风控三方辩论",
		Description: "激进、保守、中性三方风控辩论师并行评估风险收益",
		SubAgents:   []adk.Agent{aggressiveAgent, conservativeAgent, neutralAgent},
	})
	if err != nil {
		return nil, fmt.Errorf("create risk debate: %w", err)
	}

	// ========== 阶段 7: 风险法官 ==========
	riskJudgeAgent, err := judge.NewAgent(ctx)
	if err != nil {
		return nil, fmt.Errorf("create risk judge: %w", err)
	}

	// ========== 阶段 8: 风险经理 (最终决策) ==========
	riskManagerAgent, err := risk.NewAgent(ctx)
	if err != nil {
		return nil, fmt.Errorf("create risk manager: %w", err)
	}

	// ========== 组合成完整的顺序工作流 ==========
	fullWorkflow, err := adk.NewSequentialAgent(ctx, &adk.SequentialAgentConfig{
		Name:        "AIGoStock 完整交易分析工作流",
		Description: "从多维度分析到最终交易决策的完整投研流程",
		SubAgents: []adk.Agent{
			parallelAnalysts,       // 1. 6 名分析师并行分析
			gameTheoryManagerAgent, // 2. 博弈论经理
			bullBearDebate,         // 3. 多空辩论
			researchManagerAgent,   // 4. 研究经理
			traderAgent,            // 5. 交易员
			riskDebate,             // 6. 风控三方辩论
			riskJudgeAgent,         // 7. 风险法官
			riskManagerAgent,       // 8. 风险经理（最终决策）
		},
	})
	if err != nil {
		return nil, fmt.Errorf("create sequential workflow: %w", err)
	}

	return &TradingWorkflow{workflow: fullWorkflow}, nil
}

// GetAgent 获取工作流 Agent
func (tw *TradingWorkflow) GetAgent() adk.Agent {
	return tw.workflow
}
