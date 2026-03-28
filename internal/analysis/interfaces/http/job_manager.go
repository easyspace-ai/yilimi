package httpapi

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/google/uuid"
)

// JobManager 任务管理器
type JobManager struct {
	jobs map[string]*Job
	mu   sync.RWMutex

	// SSE 广播
	eventListeners map[string]chan<- JobEvent
	listenerMu     sync.RWMutex

	// 可选：与 Python 一致将 job 结果落入 SQLite reports
	reportStore *ReportStore
}

// SetReportStore 设置报告持久化（由 Server 在启动时注入）。
func (jm *JobManager) SetReportStore(s *ReportStore) {
	jm.reportStore = s
}

func (jm *JobManager) persistJobReportCompleted(jobID string, req AnalysisRequest, result, payload map[string]any) {
	if jm.reportStore == nil {
		return
	}
	td := req.TradeDate
	if td == "" {
		td = todayCN()
	}
	if err := jm.reportStore.UpsertCompleted(jobID, "", req.Symbol, td, result, payload); err != nil {
		log.Printf("report store: persist completed job %s: %v", jobID, err)
	}
}

func (jm *JobManager) persistJobReportFailed(jobID string, req AnalysisRequest, errMsg string) {
	if jm.reportStore == nil || errMsg == "" {
		return
	}
	td := req.TradeDate
	if td == "" {
		td = todayCN()
	}
	if err := jm.reportStore.UpsertFailed(jobID, "", req.Symbol, td, errMsg); err != nil {
		log.Printf("report store: persist failed job %s: %v", jobID, err)
	}
}

// NewJobManager 创建任务管理器
func NewJobManager() *JobManager {
	return &JobManager{
		jobs:           make(map[string]*Job),
		eventListeners: make(map[string]chan<- JobEvent),
	}
}

// CreateJob 创建新任务
func (jm *JobManager) CreateJob(req AnalysisRequest) *Job {
	jobID := uuid.NewString()
	now := time.Now()

	job := &Job{
		ID:        jobID,
		Status:    JobStatusPending,
		Request:   req,
		CreatedAt: now,
		UpdatedAt: now,
		Events:    []JobEvent{},
	}

	jm.mu.Lock()
	jm.jobs[jobID] = job
	jm.mu.Unlock()

	jm.addEvent(jobID, JobEvent{
		Timestamp: now,
		Type:      "job.created",
		Message:   "任务已创建",
		Data: map[string]any{
			"job_id":     jobID,
			"symbol":     req.Symbol,
			"trade_date": req.TradeDate,
		},
	})

	return job
}

// GetJob 获取任务
func (jm *JobManager) GetJob(jobID string) (*Job, bool) {
	jm.mu.RLock()
	defer jm.mu.RUnlock()
	job, ok := jm.jobs[jobID]
	return job, ok
}

// UpdateJobStatus 更新任务状态
func (jm *JobManager) UpdateJobStatus(jobID, status string) {
	jm.mu.Lock()
	defer jm.mu.Unlock()

	if job, ok := jm.jobs[jobID]; ok {
		job.Status = status
		job.UpdatedAt = time.Now()

		jm.addEvent(jobID, JobEvent{
			Timestamp: time.Now(),
			Type:      "status_changed",
			Message:   fmtStatusMessage(status),
		})
	}
}

// SetJobResult 设置任务结果并广播 job.completed（供非工作流调用方使用）
func (jm *JobManager) SetJobResult(jobID string, result map[string]any) {
	jm.CommitJobResult(jobID, result)
	final, _ := result["final_trade_decision"].(string)
	dir, dec := inferDirectionDecision(final)
	jm.addEvent(jobID, JobEvent{
		Timestamp: time.Now(),
		Type:      "job.completed",
		Message:   "任务已完成",
		Data: map[string]any{
			"result":          result,
			"risk_items":      []any{},
			"key_metrics":     []any{},
			"confidence":      nil,
			"target_price":    nil,
			"stop_loss_price": nil,
			"direction":       dir,
			"decision":        dec,
		},
	})
}

// SetJobError 设置任务错误
func (jm *JobManager) SetJobError(jobID string, err error) {
	jm.CommitJobFailure(jobID)
	jm.addEvent(jobID, JobEvent{
		Timestamp: time.Now(),
		Type:      "job.failed",
		Message:   err.Error(),
		Data: map[string]any{
			"error": err.Error(),
		},
	})
}

// SubscribeEvents 订阅任务事件
func (jm *JobManager) SubscribeEvents(jobID string) (<-chan JobEvent, func()) {
	ch := make(chan JobEvent, 100)

	jm.listenerMu.Lock()
	jm.eventListeners[jobID] = ch
	jm.listenerMu.Unlock()

	// 发送历史事件
	jm.mu.RLock()
	if job, ok := jm.jobs[jobID]; ok {
		for _, event := range job.Events {
			ch <- event
		}
	}
	jm.mu.RUnlock()

	unsubscribe := func() {
		jm.listenerMu.Lock()
		delete(jm.eventListeners, jobID)
		jm.listenerMu.Unlock()
		close(ch)
	}

	return ch, unsubscribe
}

// addEvent 添加事件（内部方法，需要已持有锁）
func (jm *JobManager) addEvent(jobID string, event JobEvent) {
	if job, ok := jm.jobs[jobID]; ok {
		job.Events = append(job.Events, event)
	}

	// 广播给监听器
	jm.listenerMu.RLock()
	if ch, ok := jm.eventListeners[jobID]; ok {
		select {
		case ch <- event:
		default:
		}
	}
	jm.listenerMu.RUnlock()
}

// fmtStatusMessage 格式化状态消息
func fmtStatusMessage(status string) string {
	switch status {
	case JobStatusPending:
		return "任务等待中"
	case JobStatusRunning:
		return "任务运行中"
	case JobStatusCompleted:
		return "任务已完成"
	case JobStatusFailed:
		return "任务失败"
	default:
		return "未知状态"
	}
}

// StartJob 开始执行任务（与 Chat SSE 共用 RunTradingWorkflow）
func (jm *JobManager) StartJob(ctx context.Context, jobID string) {
	jm.mu.RLock()
	job, ok := jm.jobs[jobID]
	jm.mu.RUnlock()
	if !ok {
		return
	}

	go func() {
		defer func() {
			if r := recover(); r != nil {
				jm.SetJobError(jobID, fmt.Errorf("analysis panic: %v", r))
			}
		}()
		emit := func(ev string, data map[string]any) {
			jm.addEvent(jobID, JobEvent{
				Timestamp: time.Now(),
				Type:      ev,
				Data:      data,
			})
		}
		_ = RunTradingWorkflow(ctx, jm, jobID, job.Request, emit)
	}()
}
