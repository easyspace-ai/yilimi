package httpapi

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// ReportStore persists reports to SQLite, schema-aligned with Python api.database.ReportDB.
type ReportStore struct {
	db *sql.DB
}

// OpenReportStore opens (and migrates) the reports database. path 通常为 {AI_DATA_DIR}/aigostock.db。
func OpenReportStore(path string) (*ReportStore, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil && filepath.Dir(path) != "." {
		return nil, fmt.Errorf("mkdir report db dir: %w", err)
	}
	// 与 workbench 子系统一致使用 mattn/go-sqlite3，避免与 glebarez/sqlite 注册的 modernc 驱动名冲突。
	dsn := path + "?_foreign_keys=on&_busy_timeout=5000&_journal_mode=WAL"
	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, fmt.Errorf("sqlite open: %w", err)
	}
	db.SetMaxOpenConns(1)
	if err := migrateReports(db); err != nil {
		_ = db.Close()
		return nil, err
	}
	return &ReportStore{db: db}, nil
}

func migrateReports(db *sql.DB) error {
	ddl := `
CREATE TABLE IF NOT EXISTS reports (
  id TEXT PRIMARY KEY,
  user_id TEXT,
  symbol TEXT NOT NULL,
  trade_date TEXT NOT NULL,
  status TEXT NOT NULL DEFAULT 'completed',
  error TEXT,
  decision TEXT,
  direction TEXT,
  confidence INTEGER,
  target_price REAL,
  stop_loss_price REAL,
  result_data TEXT,
  risk_items TEXT,
  key_metrics TEXT,
  analyst_traces TEXT,
  market_report TEXT,
  sentiment_report TEXT,
  news_report TEXT,
  fundamentals_report TEXT,
  macro_report TEXT,
  smart_money_report TEXT,
  game_theory_report TEXT,
  investment_plan TEXT,
  trader_investment_plan TEXT,
  final_trade_decision TEXT,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_reports_symbol ON reports(symbol);
CREATE INDEX IF NOT EXISTS idx_reports_status ON reports(status);
CREATE INDEX IF NOT EXISTS idx_reports_created_at ON reports(created_at);
`
	if _, err := db.Exec(ddl); err != nil {
		return fmt.Errorf("reports migrate: %w", err)
	}
	return nil
}

// Close releases the database handle.
func (s *ReportStore) Close() error {
	if s == nil || s.db == nil {
		return nil
	}
	return s.db.Close()
}

func utcNowRFC3339() string {
	return time.Now().UTC().Format(time.RFC3339Nano)
}

func sqlString(s string) sql.NullString {
	s = strings.TrimSpace(s)
	if s == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: s, Valid: true}
}

func marshalJSONString(v any) sql.NullString {
	if v == nil {
		return sql.NullString{}
	}
	b, err := json.Marshal(v)
	if err != nil || len(b) == 0 || string(b) == "null" {
		return sql.NullString{}
	}
	return sql.NullString{String: string(b), Valid: true}
}

func nullFloatPayload(payload map[string]any, key string) sql.NullFloat64 {
	if payload == nil {
		return sql.NullFloat64{}
	}
	v, ok := payload[key]
	if !ok || v == nil {
		return sql.NullFloat64{}
	}
	switch x := v.(type) {
	case float64:
		return sql.NullFloat64{Float64: x, Valid: true}
	case int:
		return sql.NullFloat64{Float64: float64(x), Valid: true}
	case int64:
		return sql.NullFloat64{Float64: float64(x), Valid: true}
	case json.Number:
		f, err := x.Float64()
		if err != nil {
			return sql.NullFloat64{}
		}
		return sql.NullFloat64{Float64: f, Valid: true}
	default:
		return sql.NullFloat64{}
	}
}

func nullIntConfidence(payload map[string]any) sql.NullInt64 {
	if payload == nil {
		return sql.NullInt64{}
	}
	v, ok := payload["confidence"]
	if !ok || v == nil {
		return sql.NullInt64{}
	}
	switch x := v.(type) {
	case float64:
		i := int64(x)
		if x == float64(i) && i >= 0 && i <= 100 {
			return sql.NullInt64{Int64: i, Valid: true}
		}
	case int:
		if x >= 0 && x <= 100 {
			return sql.NullInt64{Int64: int64(x), Valid: true}
		}
	case int64:
		if x >= 0 && x <= 100 {
			return sql.NullInt64{Int64: x, Valid: true}
		}
	}
	return sql.NullInt64{}
}

// UpsertCompleted inserts or replaces a completed report (id = job_id, same as Python create_report(..., report_id=job_id)).
func (s *ReportStore) UpsertCompleted(jobID, userID, symbol, tradeDate string, result map[string]any, payload map[string]any) error {
	if s == nil || s.db == nil {
		return nil
	}
	mergePayloadExtras(result, payload)

	resJSON := marshalJSONString(result)
	dec := stringFromAny(payload["decision"])
	dir := stringFromAny(payload["direction"])

	mkt := sqlString(stringFromAny(result["market_report"]))
	sent := sqlString(stringFromAny(result["sentiment_report"]))
	news := sqlString(stringFromAny(result["news_report"]))
	fund := sqlString(stringFromAny(result["fundamentals_report"]))
	macro := sqlString(stringFromAny(result["macro_report"]))
	smart := sqlString(stringFromAny(result["smart_money_report"]))
	game := sqlString(stringFromAny(result["game_theory_report"]))
	inv := sqlString(stringFromAny(result["investment_plan"]))
	trader := sqlString(stringFromAny(result["trader_investment_plan"]))
	final := sqlString(stringFromAny(result["final_trade_decision"]))

	risk := marshalJSONString(payload["risk_items"])
	keym := marshalJSONString(payload["key_metrics"])
	traces := marshalJSONString(payload["analyst_traces"])

	conf := nullIntConfidence(payload)
	tgt := nullFloatPayload(payload, "target_price")
	stop := nullFloatPayload(payload, "stop_loss_price")

	now := utcNowRFC3339()

	const q = `
INSERT INTO reports (
  id, user_id, symbol, trade_date, status, error,
  decision, direction, confidence, target_price, stop_loss_price,
  result_data, risk_items, key_metrics, analyst_traces,
  market_report, sentiment_report, news_report, fundamentals_report,
  macro_report, smart_money_report, game_theory_report,
  investment_plan, trader_investment_plan, final_trade_decision,
  created_at, updated_at
) VALUES (
  ?, ?, ?, ?, 'completed', NULL,
  ?, ?, ?, ?, ?,
  ?, ?, ?, ?,
  ?, ?, ?, ?,
  ?, ?, ?,
  ?, ?, ?,
  COALESCE((SELECT created_at FROM reports WHERE id = ?), ?),
  ?
)
ON CONFLICT(id) DO UPDATE SET
  user_id = excluded.user_id,
  symbol = excluded.symbol,
  trade_date = excluded.trade_date,
  status = 'completed',
  error = NULL,
  decision = excluded.decision,
  direction = excluded.direction,
  confidence = excluded.confidence,
  target_price = excluded.target_price,
  stop_loss_price = excluded.stop_loss_price,
  result_data = excluded.result_data,
  risk_items = excluded.risk_items,
  key_metrics = excluded.key_metrics,
  analyst_traces = excluded.analyst_traces,
  market_report = excluded.market_report,
  sentiment_report = excluded.sentiment_report,
  news_report = excluded.news_report,
  fundamentals_report = excluded.fundamentals_report,
  macro_report = excluded.macro_report,
  smart_money_report = excluded.smart_money_report,
  game_theory_report = excluded.game_theory_report,
  investment_plan = excluded.investment_plan,
  trader_investment_plan = excluded.trader_investment_plan,
  final_trade_decision = excluded.final_trade_decision,
  updated_at = excluded.updated_at
`
	uq := sql.NullString{String: userID, Valid: userID != ""}

	var confArg any
	if conf.Valid {
		confArg = conf.Int64
	} else {
		confArg = nil
	}
	var tgtArg, stopArg any
	if tgt.Valid {
		tgtArg = tgt.Float64
	}
	if stop.Valid {
		stopArg = stop.Float64
	}

	_, err := s.db.Exec(q,
		jobID, uq, symbol, tradeDate,
		sqlString(dec), sqlString(dir), confArg, tgtArg, stopArg,
		nullStringToInterface(resJSON), nullStringToInterface(risk), nullStringToInterface(keym), nullStringToInterface(traces),
		nullStringToInterface(mkt), nullStringToInterface(sent), nullStringToInterface(news), nullStringToInterface(fund),
		nullStringToInterface(macro), nullStringToInterface(smart), nullStringToInterface(game),
		nullStringToInterface(inv), nullStringToInterface(trader), nullStringToInterface(final),
		jobID, now,
		now,
	)
	if err != nil {
		return fmt.Errorf("upsert completed report: %w", err)
	}
	return nil
}

func nullStringToInterface(ns sql.NullString) any {
	if !ns.Valid {
		return nil
	}
	return ns.String
}

// UpsertFailed marks a report as failed (aligned with Python mark_report_failed).
func (s *ReportStore) UpsertFailed(jobID, userID, symbol, tradeDate, errMsg string) error {
	if s == nil || s.db == nil {
		return nil
	}
	now := utcNowRFC3339()
	uq := sql.NullString{String: userID, Valid: userID != ""}

	const q = `
INSERT INTO reports (id, user_id, symbol, trade_date, status, error, created_at, updated_at)
VALUES (?, ?, ?, ?, 'failed', ?, COALESCE((SELECT created_at FROM reports WHERE id = ?), ?), ?)
ON CONFLICT(id) DO UPDATE SET
  status = 'failed',
  error = excluded.error,
  symbol = excluded.symbol,
  trade_date = excluded.trade_date,
  updated_at = excluded.updated_at
`
	_, err := s.db.Exec(q, jobID, uq, symbol, tradeDate, errMsg, jobID, now, now)
	if err != nil {
		return fmt.Errorf("upsert failed report: %w", err)
	}
	return nil
}

// Delete removes a report by id.
func (s *ReportStore) Delete(id string) error {
	if s == nil || s.db == nil {
		return nil
	}
	_, err := s.db.Exec(`DELETE FROM reports WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete report: %w", err)
	}
	return nil
}

// Count returns total reports matching optional symbol filter.
func (s *ReportStore) Count(symbol string) (int, error) {
	if s == nil || s.db == nil {
		return 0, nil
	}
	var q string
	var args []any
	if strings.TrimSpace(symbol) == "" {
		q = `SELECT COUNT(1) FROM reports`
	} else {
		q = `SELECT COUNT(1) FROM reports WHERE symbol = ?`
		args = append(args, symbol)
	}
	var n int
	if err := s.db.QueryRow(q, args...).Scan(&n); err != nil {
		return 0, err
	}
	return n, nil
}

// List returns reports ordered by created_at desc.
func (s *ReportStore) List(symbol string, skip, limit int) ([]Report, error) {
	if s == nil || s.db == nil {
		return nil, nil
	}
	if limit <= 0 {
		limit = 100
	}
	if skip < 0 {
		skip = 0
	}

	var q string
	var args []any
	if strings.TrimSpace(symbol) == "" {
		q = `SELECT id, user_id, symbol, trade_date, status, error, decision, direction, confidence, target_price, stop_loss_price, created_at, updated_at
FROM reports ORDER BY created_at DESC LIMIT ? OFFSET ?`
		args = []any{limit, skip}
	} else {
		q = `SELECT id, user_id, symbol, trade_date, status, error, decision, direction, confidence, target_price, stop_loss_price, created_at, updated_at
FROM reports WHERE symbol = ? ORDER BY created_at DESC LIMIT ? OFFSET ?`
		args = []any{symbol, limit, skip}
	}

	rows, err := s.db.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Report
	for rows.Next() {
		var r Report
		var userID, errMsg, dec, dir sql.NullString
		var conf sql.NullInt64
		var tgt, stop sql.NullFloat64
		var createdAt, updatedAt string
		if err := rows.Scan(&r.ID, &userID, &r.Symbol, &r.TradeDate, &r.Status, &errMsg, &dec, &dir, &conf, &tgt, &stop, &createdAt, &updatedAt); err != nil {
			return nil, err
		}
		if userID.Valid {
			r.UserID = userID.String
		}
		if errMsg.Valid {
			r.Error = errMsg.String
		}
		if dec.Valid {
			r.Decision = dec.String
		}
		if dir.Valid {
			r.Direction = dir.String
		}
		if conf.Valid {
			r.Confidence = float64(conf.Int64)
		}
		if tgt.Valid {
			r.TargetPrice = tgt.Float64
		}
		if stop.Valid {
			r.StopLossPrice = stop.Float64
		}
		r.CreatedAt = createdAt
		r.UpdatedAt = updatedAt
		out = append(out, r)
	}
	return out, rows.Err()
}

// Get returns full report detail.
func (s *ReportStore) Get(id string) (*ReportDetail, error) {
	if s == nil || s.db == nil {
		return nil, errors.New("report store not initialized")
	}
	const q = `SELECT id, user_id, symbol, trade_date, status, error, decision, direction, confidence, target_price, stop_loss_price,
result_data, risk_items, key_metrics, analyst_traces,
market_report, sentiment_report, news_report, fundamentals_report, macro_report, smart_money_report, game_theory_report,
investment_plan, trader_investment_plan, final_trade_decision, created_at, updated_at
FROM reports WHERE id = ?`

	var d ReportDetail
	var userID, errMsg, dec, dir sql.NullString
	var conf sql.NullInt64
	var tgt, stop sql.NullFloat64
	var resData, risk, keym, traces sql.NullString
	var mkt, sent, news, fund, macro, smart, game, inv, trader, final sql.NullString
	var createdAt, updatedAt string

	err := s.db.QueryRow(q, id).Scan(
		&d.ID, &userID, &d.Symbol, &d.TradeDate, &d.Status, &errMsg, &dec, &dir, &conf, &tgt, &stop,
		&resData, &risk, &keym, &traces,
		&mkt, &sent, &news, &fund, &macro, &smart, &game, &inv, &trader, &final,
		&createdAt, &updatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if userID.Valid {
		d.UserID = userID.String
	}
	if errMsg.Valid {
		d.Error = errMsg.String
	}
	if dec.Valid {
		d.Decision = dec.String
	}
	if dir.Valid {
		d.Direction = dir.String
	}
	if conf.Valid {
		d.Confidence = float64(conf.Int64)
	}
	if tgt.Valid {
		d.TargetPrice = tgt.Float64
	}
	if stop.Valid {
		d.StopLossPrice = stop.Float64
	}
	d.CreatedAt = createdAt
	d.UpdatedAt = updatedAt

	if mkt.Valid {
		d.MarketReport = mkt.String
	}
	if sent.Valid {
		d.SentimentReport = sent.String
	}
	if news.Valid {
		d.NewsReport = news.String
	}
	if fund.Valid {
		d.FundamentalsReport = fund.String
	}
	if macro.Valid {
		d.MacroReport = macro.String
	}
	if smart.Valid {
		d.SmartMoneyReport = smart.String
	}
	if game.Valid {
		d.GameTheoryReport = game.String
	}
	if inv.Valid {
		d.InvestmentPlan = inv.String
	}
	if trader.Valid {
		d.TraderInvestmentPlan = trader.String
	}
	if final.Valid {
		d.FinalTradeDecision = final.String
	}

	if resData.Valid && resData.String != "" {
		d.ResultData = json.RawMessage(resData.String)
	}
	if risk.Valid && risk.String != "" {
		d.RiskItems = json.RawMessage(risk.String)
	}
	if keym.Valid && keym.String != "" {
		d.KeyMetrics = json.RawMessage(keym.String)
	}
	if traces.Valid && traces.String != "" {
		d.AnalystTraces = json.RawMessage(traces.String)
	}

	return &d, nil
}

// InsertManual creates a report from API (Python POST /v1/reports).
func (s *ReportStore) InsertManual(id, userID, symbol, tradeDate, decision string, resultData any) error {
	m := map[string]any{}
	if resultData != nil {
		switch t := resultData.(type) {
		case map[string]any:
			for k, v := range t {
				m[k] = v
			}
		default:
			b, err := json.Marshal(resultData)
			if err != nil {
				return fmt.Errorf("marshal result_data: %w", err)
			}
			if err := json.Unmarshal(b, &m); err != nil {
				return fmt.Errorf("result_data as map: %w", err)
			}
		}
	}
	m["symbol"] = symbol
	m["trade_date"] = tradeDate
	payload := map[string]any{}
	if strings.TrimSpace(decision) != "" {
		payload["decision"] = decision
	}
	final := stringFromAny(m["final_trade_decision"])
	dirInf, decInf := inferDirectionDecision(final)
	if strings.TrimSpace(decision) == "" {
		payload["decision"] = decInf
	} else {
		payload["decision"] = strings.TrimSpace(decision)
	}
	payload["direction"] = dirInf
	if v := extractVerdictDirection(final); v != "" {
		payload["direction"] = v
	}
	return s.UpsertCompleted(id, userID, symbol, tradeDate, m, payload)
}
