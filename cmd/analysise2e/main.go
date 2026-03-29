// Command analysise2e：内存 JobManager + RunTradingWorkflow，落盘事件与结果 JSON，并可选 L3 机审闸门。
//
// 在 backend 目录：go run ./cmd/analysise2e -symbol 600820.SH -date 2024-06-01 -out ./artifacts/e2e-run
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"

	"github.com/easyspace-ai/yilimi/internal/analysis/e2e"
	httpapi "github.com/easyspace-ai/yilimi/internal/analysis/interfaces/http"
	"github.com/easyspace-ai/yilimi/internal/analysis/tools"
	"github.com/easyspace-ai/yilimi/internal/appenv"
)

func main() {
	log.SetPrefix("[analysise2e] ")
	log.SetFlags(0)

	symbol := flag.String("symbol", "600820.SH", "ts_code")
	tradeDate := flag.String("date", "", "交易日 YYYY-MM-DD，空则上海当日")
	outDir := flag.String("out", "", "产出目录（e2e-events.jsonl / result.json / e2e-verdict.json）；空则 ./artifacts/analysis-e2e/<timestamp>")
	timeout := flag.Duration("timeout", 0, "工作流 ctx 超时，如 15m；0 表示用 AIGOSTOCK_ANALYSIS_TIMEOUT 或默认")
	skipGates := flag.Bool("skip-gates", false, "跳过 L3 静态闸门（仅调试）")

	flag.Parse()

	appenv.Init()
	dataDir := appenv.DataRootDir()
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		log.Fatalf("data dir: %v", err)
	}
	if err := tools.InitGlobalTools(dataDir); err != nil {
		log.Fatalf("InitGlobalTools: %v", err)
	}

	dir := *outDir
	if dir == "" {
		dir = filepath.Join("artifacts", "analysis-e2e", time.Now().Format("20060102-150405"))
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		log.Fatalf("out dir: %v", err)
	}

	eventsPath := filepath.Join(dir, "e2e-events.jsonl")
	eventsF, err := os.Create(eventsPath)
	if err != nil {
		log.Fatalf("events file: %v", err)
	}
	defer func() { _ = eventsF.Close() }()

	td := *tradeDate
	if td == "" {
		td = todayShanghai()
	}

	req := httpapi.AnalysisRequest{
		Symbol:    *symbol,
		TradeDate: td,
		Objective: fmt.Sprintf(
			"E2E 自动化验收：对 %s 于 %s 做完整投研分析，输出市场/舆情/新闻/基本面/宏观/主力等段落及交易结论。",
			*symbol, td,
		),
	}

	jobID := uuid.NewString()
	jm := httpapi.NewJobManager()
	jm.RegisterJob(jobID, req)

	emit := func(event string, data map[string]any) {
		line := map[string]any{
			"ts":    time.Now().UTC().Format(time.RFC3339Nano),
			"event": event,
			"data":  data,
		}
		b, _ := json.Marshal(line)
		_, _ = eventsF.Write(append(b, '\n'))
	}

	wfTimeout := httpapi.AnalysisWorkflowTimeout()
	if *timeout > 0 {
		wfTimeout = *timeout
	}
	ctx, cancel := context.WithTimeout(context.Background(), wfTimeout)
	defer cancel()

	log.Printf("job=%s symbol=%s date=%s timeout=%s out=%s", jobID, *symbol, td, wfTimeout, dir)

	if err := httpapi.RunTradingWorkflow(ctx, jm, jobID, req, emit); err != nil {
		_ = writeJSON(filepath.Join(dir, "workflow-error.json"), map[string]any{"error": err.Error()})
		log.Fatalf("RunTradingWorkflow: %v", err)
	}

	job, ok := jm.GetJob(jobID)
	if !ok || job == nil {
		log.Fatalf("job %s not found after workflow", jobID)
	}
	if job.Status != httpapi.JobStatusCompleted {
		_ = writeJSON(filepath.Join(dir, "workflow-error.json"), map[string]any{
			"status": job.Status,
			"note":   "expected completed",
		})
		log.Fatalf("job status=%s (want %s)", job.Status, httpapi.JobStatusCompleted)
	}
	if job.Result == nil {
		log.Fatalf("job result is nil")
	}

	if err := writeJSON(filepath.Join(dir, "result.json"), job.Result); err != nil {
		log.Fatalf("write result: %v", err)
	}

	if *skipGates {
		log.Printf("skip-gates: wrote %s and %s", eventsPath, filepath.Join(dir, "result.json"))
		return
	}

	verdict := e2e.EvaluateReports(job.Result)
	vpath := filepath.Join(dir, "e2e-verdict.json")
	if err := e2e.WriteVerdict(vpath, verdict); err != nil {
		log.Fatalf("write verdict: %v", err)
	}
	if !verdict.AllPass {
		log.Printf("L3 gates failed: %s", verdict.Summary)
		for _, h := range verdict.Hints {
			log.Print(h)
		}
		os.Exit(2)
	}
	log.Printf("PASS gates → %s", vpath)
}

func todayShanghai() string {
	loc, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		return time.Now().Format("2006-01-02")
	}
	return time.Now().In(loc).Format("2006-01-02")
}

func writeJSON(path string, v any) error {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0o644)
}
