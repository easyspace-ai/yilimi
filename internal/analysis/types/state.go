package types

import "time"

// AgentState 完整的 Agent 状态，对应 Python 的 TypedDict
type AgentState struct {
	// 基本信息
	CompanyOfInterest string    `json:"company_of_interest"`
	TradeDate         time.Time `json:"trade_date"`
	Horizon           string    `json:"horizon"` // "short" 或 "medium"

	// 上下文
	InstrumentContext InstrumentContext `json:"instrument_context"`
	MarketContext     MarketContext     `json:"market_context"`
	UserContext       UserContext       `json:"user_context"`
	WorkflowContext   WorkflowContext   `json:"workflow_context"`

	// 分析师报告
	MarketReport       string `json:"market_report"`
	SentimentReport    string `json:"sentiment_report"`
	NewsReport         string `json:"news_report"`
	FundamentalsReport string `json:"fundamentals_report"`
	MacroReport        string `json:"macro_report"`
	SmartMoneyReport   string `json:"smart_money_report"`

	// 投资辩论状态
	InvestmentDebateState InvestmentDebateState `json:"investment_debate_state"`

	// 风控辩论状态
	RiskDebateState RiskDebateState `json:"risk_debate_state"`

	// 最终输出
	InvestmentPlan       string `json:"investment_plan"`
	TraderInvestmentPlan string `json:"trader_investment_plan"`
	FinalTradeDecision   string `json:"final_trade_decision"`

	// 双周期结果
	ShortTermResult  *AnalysisResult `json:"short_term_result"`
	MediumTermResult *AnalysisResult `json:"medium_term_result"`
}

// InstrumentContext 标的上下文
type InstrumentContext struct {
	Symbol      string `json:"symbol"`
	CompanyName string `json:"company_name"`
	Exchange    string `json:"exchange"`
}

// MarketContext 市场上下文
type MarketContext struct {
	IndexData       map[string]any `json:"index_data"`
	SectorData      map[string]any `json:"sector_data"`
	MarketSentiment string         `json:"market_sentiment"`
}

// UserContext 用户上下文
type UserContext struct {
	RiskTolerance string   `json:"risk_tolerance"`
	TimeHorizon   string   `json:"time_horizon"`
	FocusAreas    []string `json:"focus_areas"`
}

// WorkflowContext 工作流上下文
type WorkflowContext struct {
	CurrentStep         string `json:"current_step"`
	DebateRound         int    `json:"debate_round"`
	RiskDebateRound     int    `json:"risk_debate_round"`
	MaxDebateRounds     int    `json:"max_debate_rounds"`
	MaxRiskDebateRounds int    `json:"max_risk_debate_rounds"`
}

// InvestmentDebateState 投资辩论状态
type InvestmentDebateState struct {
	BullArguments    []string `json:"bull_arguments"`
	BearArguments    []string `json:"bear_arguments"`
	CommonGround     []string `json:"common_ground"`
	KeyDisagreements []string `json:"key_disagreements"`
	CurrentRound     int      `json:"current_round"`
}

// RiskDebateState 风控辩论状态
type RiskDebateState struct {
	AggressiveView   string `json:"aggressive_view"`
	ConservativeView string `json:"conservative_view"`
	NeutralView      string `json:"neutral_view"`
	CurrentRound     int    `json:"current_round"`
	NeedsRevision    bool   `json:"needs_revision"`
	RevisionFeedback string `json:"revision_feedback"`
}

// AnalysisResult 分析结果
type AnalysisResult struct {
	Horizon     string    `json:"horizon"`
	Decision    string    `json:"decision"`
	Confidence  float64   `json:"confidence"`
	Reasoning   string    `json:"reasoning"`
	RiskScore   float64   `json:"risk_score"`
	GeneratedAt time.Time `json:"generated_at"`
}

// NewAgentState 创建新的状态
func NewAgentState(companyName string, tradeDate time.Time, horizon string) *AgentState {
	return &AgentState{
		CompanyOfInterest: companyName,
		TradeDate:         tradeDate,
		Horizon:           horizon,
		InvestmentDebateState: InvestmentDebateState{
			BullArguments: []string{},
			BearArguments: []string{},
			CommonGround:  []string{},
			CurrentRound:  0,
		},
		RiskDebateState: RiskDebateState{
			CurrentRound:  0,
			NeedsRevision: false,
		},
		WorkflowContext: WorkflowContext{
			CurrentStep:         "start",
			DebateRound:         0,
			RiskDebateRound:     0,
			MaxDebateRounds:     3,
			MaxRiskDebateRounds: 2,
		},
	}
}
