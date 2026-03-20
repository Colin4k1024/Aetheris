// Copyright 2026 fanjia1024
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package worker

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prometheus/common/expfmt"

	"rag-platform/internal/agent/instance"
	"rag-platform/internal/agent/job"
	"rag-platform/internal/agent/messaging"
	"rag-platform/internal/agent/planner"
	"rag-platform/internal/agent/replay"
	replaysandbox "rag-platform/internal/agent/replay/sandbox"
	"rag-platform/internal/agent/runtime"
	agentexec "rag-platform/internal/agent/runtime/executor"
	"rag-platform/internal/agent/runtime/executor/verifier"
	"rag-platform/internal/agent/tools"
	"rag-platform/internal/agent/tools/mcp"
	"rag-platform/internal/app"
	"rag-platform/internal/app/api"
	"rag-platform/internal/app/approvalexpiry"
	"rag-platform/internal/ingestqueue"
	llmmod "rag-platform/internal/model/llm"
	"rag-platform/internal/runtime/eino"
	"rag-platform/internal/runtime/jobstore"
	"rag-platform/internal/storage/metadata"
	"rag-platform/internal/storage/vector"
	"rag-platform/pkg/config"
	"rag-platform/pkg/log"
	"rag-platform/pkg/metrics"
)

// App Worker 应用（Pipeline 由 eino 调度；JobStore=postgres 时拉取 Agent Job 执行）
type App struct {
	config         *config.Config
	logger         *log.Logger
	engine         *eino.Engine
	metadataStore  metadata.Store
	vectorStore    vector.Store
	shutdown       chan struct{}
	agentJobRunner *AgentJobRunner
	agentJobCancel context.CancelFunc
	jobEventStore  jobstore.JobStore // 用于 Snapshot 自动化与 GC goroutine（仅 postgres 模式下非 nil）
	replayBuilder  replay.ReplayContextBuilder
	mcpManager     *mcp.Manager
	jobStore       job.JobStore
	wakeupQueue    job.WakeupQueue
}

// NewApp 创建新的 Worker 应用
func NewApp(cfg *config.Config) (*App, error) {
	if err := validateProductionRuntimeConfig(cfg); err != nil {
		return nil, err
	}
	// 初始化日志
	logCfg := &log.Config{}
	if cfg != nil {
		logCfg.Level = cfg.Log.Level
		logCfg.Format = cfg.Log.Format
		logCfg.File = cfg.Log.File
	}
	logger, err := log.NewLogger(logCfg)
	if err != nil {
		return nil, fmt.Errorf("初始化日志failed: %w", err)
	}

	// 初始化存储
	metadataStore, err := metadata.NewStore(cfg.Storage.Metadata)
	if err != nil {
		return nil, fmt.Errorf("初始化元数据存储failed: %w", err)
	}

	vectorStore, err := vector.NewStore(cfg.Storage.Vector)
	if err != nil {
		return nil, fmt.Errorf("初始化向量存储failed: %w", err)
	}

	// 初始化 eino 引擎（ingest 任务通过 ExecuteWorkflow(ctx, "ingest_pipeline", payload) 执行）
	engine, err := eino.NewEngine(cfg, logger)
	if err != nil {
		return nil, fmt.Errorf("初始化 eino 引擎failed: %w", err)
	}

	appObj := &App{
		config:        cfg,
		logger:        logger,
		engine:        engine,
		metadataStore: metadataStore,
		vectorStore:   vectorStore,
		shutdown:      make(chan struct{}),
	}

	// Agent Job 模式：jobstore.type=postgres 时，从事件存储 Claim、从元数据存储取 Job、执行 DAG Runner
	embeddedBaseDir := embeddedDataDir(cfg)
	if cfg != nil && cfg.JobStore.Type == "postgres" && cfg.JobStore.DSN != "" {
		dsn := cfg.JobStore.DSN
		leaseDur := 30 * time.Second
		if cfg.JobStore.LeaseDuration != "" {
			if d, err := time.ParseDuration(cfg.JobStore.LeaseDuration); err == nil && d > 0 {
				leaseDur = d
			}
		}
		pgEventStore, err := jobstore.NewPostgresStore(context.Background(), dsn, leaseDur)
		if err != nil {
			return nil, fmt.Errorf("初始化 JobStore 事件(postgres) failed: %w", err)
		}
		pgJobStore, err := job.NewJobStorePg(context.Background(), dsn)
		if err != nil {
			return nil, fmt.Errorf("初始化 Job 元数据(postgres) failed: %w", err)
		}
		llmClientRaw, err := app.NewLLMClientFromConfig(cfg)
		if err != nil {
			return nil, fmt.Errorf("初始化 LLM 客户端failed: %w", err)
		}
		// LLM 限流包装
		var llmClient llmmod.Client = llmClientRaw
		if len(cfg.RateLimits.LLM) > 0 {
			llmLimiterConfigs := make(map[string]llmmod.LLMLimitConfig, len(cfg.RateLimits.LLM))
			for provider, c := range cfg.RateLimits.LLM {
				if provider == "_default" {
					continue
				}
				llmLimiterConfigs[provider] = llmmod.LLMLimitConfig{
					TokensPerMinute:   c.TokensPerMinute,
					RequestsPerMinute: c.RequestsPerMinute,
					MaxConcurrent:     c.MaxConcurrent,
				}
			}
			var llmDefaults *llmmod.LLMLimitConfig
			if d, ok := cfg.RateLimits.LLM["_default"]; ok {
				llmDefaults = &llmmod.LLMLimitConfig{
					TokensPerMinute:   d.TokensPerMinute,
					RequestsPerMinute: d.RequestsPerMinute,
					MaxConcurrent:     d.MaxConcurrent,
				}
			}
			llmRateLimiter := llmmod.NewLLMRateLimiter(llmLimiterConfigs, llmDefaults)
			llmClient = llmmod.NewRateLimitedClient(llmClientRaw, llmRateLimiter)
			logger.Info("Worker LLM 限流已启用", "providers", len(llmLimiterConfigs))
		}
		toolsReg := tools.NewRegistry()
		tools.RegisterBuiltin(toolsReg, engine, nil)
		appObj.mcpManager = initMCPManager(cfg, toolsReg, logger)
		var v1Planner planner.Planner
		if os.Getenv("PLANNER_TYPE") == "rule" {
			v1Planner = planner.NewRulePlanner()
			logger.Info("Worker 使用规则规划器")
		} else {
			v1Planner = planner.NewLLMPlanner(llmClient)
		}
		nodeEventSink := api.NewNodeEventSink(pgEventStore)
		var invocationStore agentexec.ToolInvocationStore
		if invPoolConfig, errPool := pgxpool.ParseConfig(dsn); errPool == nil {
			if invPool, errPool := pgxpool.NewWithConfig(context.Background(), invPoolConfig); errPool == nil {
				invocationStore = agentexec.NewToolInvocationStorePg(invPool)
			}
		}
		if invocationStore == nil {
			invocationStore = agentexec.NewToolInvocationStoreMem()
		}
		var effectStore agentexec.EffectStore
		if cfg.EffectStore.Type == "postgres" && cfg.EffectStore.DSN != "" {
			effPoolConfig, errPool := pgxpool.ParseConfig(cfg.EffectStore.DSN)
			if errPool != nil {
				return nil, fmt.Errorf("解析 EffectStore DSN failed: %w", errPool)
			}
			effPool, errPool := pgxpool.NewWithConfig(context.Background(), effPoolConfig)
			if errPool != nil {
				return nil, fmt.Errorf("创建 EffectStore 连接池failed: %w", errPool)
			}
			effectStore = agentexec.NewEffectStorePg(effPool)
		} else {
			effectStore = agentexec.NewEffectStoreMem()
		}
		var resourceVerifier agentexec.ResourceVerifier
		if token := os.Getenv("GITHUB_TOKEN"); token != "" {
			resourceVerifier = verifier.NewGitHubVerifier(token)
		}
		// Tool 限流器（可选）
		var toolRateLimiter *agentexec.ToolRateLimiter
		if len(cfg.RateLimits.Tools) > 0 {
			toolLimiterConfigs := make(map[string]agentexec.ToolLimitConfig, len(cfg.RateLimits.Tools))
			for toolName, c := range cfg.RateLimits.Tools {
				if toolName == "_default" {
					continue
				}
				toolLimiterConfigs[toolName] = agentexec.ToolLimitConfig{QPS: c.QPS, MaxConcurrent: c.MaxConcurrent, Burst: c.Burst}
			}
			var toolDefaults *agentexec.ToolLimitConfig
			if d, ok := cfg.RateLimits.Tools["_default"]; ok {
				toolDefaults = &agentexec.ToolLimitConfig{QPS: d.QPS, MaxConcurrent: d.MaxConcurrent, Burst: d.Burst}
			}
			toolRateLimiter = agentexec.NewToolRateLimiter(toolLimiterConfigs, toolDefaults)
		}
		dagCompiler := api.NewDAGCompilerWithOptions(llmClient, toolsReg, engine, nodeEventSink, nodeEventSink, invocationStore, effectStore, resourceVerifier, api.NewAttemptValidator(pgEventStore), toolRateLimiter, &cfg.Agents)
		dagRunner := api.NewDAGRunner(dagCompiler)
		checkpointStore := runtime.NewCheckpointStoreMem()
		if cfg.CheckpointStore.Type == "postgres" && cfg.CheckpointStore.DSN != "" {
			cpPoolConfig, errPool := pgxpool.ParseConfig(cfg.CheckpointStore.DSN)
			if errPool != nil {
				return nil, fmt.Errorf("解析 CheckpointStore DSN failed: %w", errPool)
			}
			cpPool, errPool := pgxpool.NewWithConfig(context.Background(), cpPoolConfig)
			if errPool != nil {
				return nil, fmt.Errorf("创建 CheckpointStore 连接池failed: %w", errPool)
			}
			checkpointStore = runtime.NewCheckpointStorePg(cpPool)
		} else if cfg.CheckpointStore.Type == "embedded" || cfg.JobStore.Type == "embedded" {
			embeddedCheckpointPath := filepath.Join(embeddedBaseDir, "checkpoints.json")
			embeddedCheckpointStore, err := runtime.NewCheckpointStoreEmbedded(embeddedCheckpointPath)
			if err != nil {
				return nil, fmt.Errorf("初始化 CheckpointStore(embedded) failed: %w", err)
			}
			checkpointStore = embeddedCheckpointStore
		}
		agentStateStore, errState := runtime.NewAgentStateStorePg(context.Background(), dsn)
		if errState != nil {
			return nil, fmt.Errorf("初始化 AgentStateStore(postgres) failed: %w", errState)
		}
		dagRunner.SetCheckpointStores(checkpointStore, &jobStoreForRunnerAdapter{JobStore: pgJobStore})
		dagRunner.SetPlanGeneratedSink(api.NewPlanGeneratedSink(pgEventStore))
		dagRunner.SetNodeEventSink(nodeEventSink)
		dagRunner.SetRecordedEffectsRecorder(api.NewRecordedEffectsRecorder(pgEventStore))
		dagRunner.SetReplayContextBuilder(api.NewReplayContextBuilder(pgEventStore))
		dagRunner.SetReplayPolicy(replaysandbox.DefaultPolicy{})
		if cfg.Worker.Timeout != "" {
			if d, err := time.ParseDuration(cfg.Worker.Timeout); err == nil && d > 0 {
				dagRunner.SetStepTimeout(d)
			}
		}
		maxAttempts := cfg.Worker.MaxAttempts
		if maxAttempts <= 0 {
			maxAttempts = 3
		}
		// 捕获 logger 供内部函数使用
		workerLogger := logger
		waitPlanReady := func(ctx context.Context, jobID string, maxWait time.Duration) error {
			if maxWait <= 0 {
				maxWait = 15 * time.Second
			}
			deadline := time.Now().Add(maxWait)
			for {
				events, _, err := pgEventStore.ListEvents(ctx, jobID)
				if err == nil {
					for _, e := range events {
						if e.Type == jobstore.PlanGenerated {
							return nil
						}
					}
				}
				if time.Now().After(deadline) {
					return fmt.Errorf("plan_generated not ready within %s", maxWait)
				}
				select {
				case <-ctx.Done():
					return ctx.Err()
				case <-time.After(200 * time.Millisecond):
				}
			}
		}
		runJob := func(ctx context.Context, j *job.Job) error {
			sessionID := j.SessionID
			if sessionID == "" {
				sessionID = j.AgentID
			}
			state, _ := agentStateStore.LoadAgentState(ctx, j.AgentID, sessionID)
			sess := runtime.NewSession(sessionID, j.AgentID)
			if state != nil {
				runtime.ApplyAgentState(sess, state)
			}
			plannerProv := newPlannerProviderAdapter(v1Planner)
			toolsProv := newToolsProviderAdapter(toolsReg)
			agent := runtime.NewAgent(j.AgentID, j.AgentID, sess, nil, plannerProv, toolsProv)
			tenantID := j.TenantID
			if tenantID == "" {
				tenantID = "default"
			}
			if err := waitPlanReady(ctx, j.ID, 20*time.Second); err != nil {
				return err
			}
			err := dagRunner.RunForJob(ctx, agent, &agentexec.JobForRunner{
				ID: j.ID, AgentID: j.AgentID, Goal: j.Goal, Cursor: j.Cursor, TenantID: tenantID,
			})
			if agentStateStore != nil && agent.Session != nil {
				if err := agentStateStore.SaveAgentState(ctx, j.AgentID, agent.Session.ID, runtime.SessionToAgentState(agent.Session)); err != nil {
					workerLogger.Warn("failed to save agent state", "error", err, "agent_id", j.AgentID)
				}
			}
			if err != nil && errors.Is(err, agentexec.ErrJobWaiting) {
				// Job 在 Wait 节点挂起，已写 job_waiting 并置为 Waiting；等待 signal 后重新入队，不写终端事件
				return err
			}
			if err != nil {
				// 毒任务保护：达到 max_attempts 后标记 Failed 并写 job_failed，不再调度；否则 Requeue（不写终端事件）供再次 Claim
				if j.RetryCount+1 >= maxAttempts {
					errStr := err.Error()
					payload, _ := json.Marshal(map[string]interface{}{"goal": j.Goal, "error": errStr})
					if er := appendTerminalEventAndUpdateStatus(ctx, pgEventStore, pgJobStore, j.ID, payload, jobstore.JobFailed, job.StatusFailed); er != nil {
						workerLogger.Warn("failed to persist job_failed terminal state", "error", er, "job_id", j.ID)
					}
				} else {
					if er := pgJobStore.Requeue(ctx, j); er != nil {
						workerLogger.Warn("failed to requeue job", "error", er, "job_id", j.ID)
					}
				}
			} else {
				payload, _ := json.Marshal(map[string]interface{}{"goal": j.Goal})
				if er := appendTerminalEventAndUpdateStatus(ctx, pgEventStore, pgJobStore, j.ID, payload, jobstore.JobCompleted, job.StatusCompleted); er != nil {
					workerLogger.Warn("failed to persist job_completed terminal state", "error", er, "job_id", j.ID)
				}
			}
			return err
		}
		pollInterval := 2 * time.Second
		if cfg.Worker.PollInterval != "" {
			if d, err := time.ParseDuration(cfg.Worker.PollInterval); err == nil && d > 0 {
				pollInterval = d
			}
		}
		maxConcurrency := cfg.Worker.Concurrency
		if maxConcurrency <= 0 {
			maxConcurrency = 2
		}
		runner := NewAgentJobRunner(
			DefaultWorkerID(),
			pgEventStore,
			pgJobStore,
			runJob,
			pollInterval,
			leaseDur,
			maxConcurrency,
			cfg.Worker.Capabilities,
			logger,
		)
		// 唤醒队列：无 job 时用 Receive(pollInterval) 替代固定 sleep，API 侧 JobSignal/JobMessage 若设置同一 WakeupQueue 可立即唤醒（单进程部署时注入同一实例）
		wakeupQueue := job.NewWakeupQueueMem(256)
		runner.SetWakeupQueue(wakeupQueue)
		// Inbox 驱动创建 Job：轮询 agent_messages 未消费消息，创建 Job 后 NotifyReady（design/plan.md Phase A）
		if inboxStore, errInbox := messaging.NewStorePg(context.Background(), dsn); errInbox == nil {
			runner.SetInboxReader(inboxStore)
			logger.Info("Worker Inbox 轮询已启用，支持 message arrival → job run")
		}
		// Instance current_job_id：Job 认领/结束时更新（design/plan.md Phase B）
		if instanceStore, errInst := instance.NewStorePg(context.Background(), dsn); errInst == nil {
			runner.SetInstanceStore(instanceStore)
		}
		appObj.agentJobRunner = runner
		appObj.jobEventStore = pgEventStore
		appObj.replayBuilder = replay.NewReplayContextBuilder(pgEventStore)
		appObj.jobStore = pgJobStore
		appObj.wakeupQueue = wakeupQueue
		logger.Info("Worker Agent Job 模式已启用", "worker_id", DefaultWorkerID(), "dsn", dsn)
	} else if cfg != nil && cfg.JobStore.Type == "embedded" {
		embeddedEventsPath := filepath.Join(embeddedBaseDir, "job_events.json")
		embeddedJobsPath := filepath.Join(embeddedBaseDir, "jobs.json")
		embeddedEventStore, err := jobstore.NewEmbeddedStore(embeddedEventsPath)
		if err != nil {
			return nil, fmt.Errorf("初始化 JobStore 事件(embedded) failed: %w", err)
		}
		embeddedJobStore, err := job.NewJobStoreEmbedded(embeddedJobsPath)
		if err != nil {
			return nil, fmt.Errorf("初始化 Job 元数据(embedded) failed: %w", err)
		}
		llmClientRaw, err := app.NewLLMClientFromConfig(cfg)
		if err != nil {
			return nil, fmt.Errorf("初始化 LLM 客户端failed: %w", err)
		}
		toolsReg := tools.NewRegistry()
		tools.RegisterBuiltin(toolsReg, engine, nil)
		appObj.mcpManager = initMCPManager(cfg, toolsReg, logger)
		v1Planner := planner.NewLLMPlanner(llmClientRaw)
		nodeEventSink := api.NewNodeEventSink(embeddedEventStore)
		embeddedInvocationStore, err := agentexec.NewToolInvocationStoreEmbedded(filepath.Join(embeddedBaseDir, "tool_invocations.json"))
		if err != nil {
			return nil, fmt.Errorf("初始化 ToolInvocationStore(embedded) failed: %w", err)
		}
		embeddedEffectStore, err := agentexec.NewEffectStoreEmbedded(filepath.Join(embeddedBaseDir, "effects.json"))
		if err != nil {
			return nil, fmt.Errorf("初始化 EffectStore(embedded) failed: %w", err)
		}
		dagCompiler := api.NewDAGCompilerWithOptions(llmClientRaw, toolsReg, engine, nodeEventSink, nodeEventSink, embeddedInvocationStore, embeddedEffectStore, nil, api.NewAttemptValidator(embeddedEventStore), nil, &cfg.Agents)
		dagRunner := api.NewDAGRunner(dagCompiler)
		embeddedCheckpointStore, err := runtime.NewCheckpointStoreEmbedded(filepath.Join(embeddedBaseDir, "checkpoints.json"))
		if err != nil {
			return nil, fmt.Errorf("初始化 CheckpointStore(embedded) failed: %w", err)
		}
		embeddedStateStore, err := runtime.NewAgentStateStoreEmbedded(filepath.Join(embeddedBaseDir, "agent_state.json"))
		if err != nil {
			return nil, fmt.Errorf("初始化 AgentStateStore(embedded) failed: %w", err)
		}
		dagRunner.SetCheckpointStores(embeddedCheckpointStore, &jobStoreForRunnerAdapter{JobStore: embeddedJobStore})
		dagRunner.SetPlanGeneratedSink(api.NewPlanGeneratedSink(embeddedEventStore))
		dagRunner.SetNodeEventSink(nodeEventSink)
		dagRunner.SetRecordedEffectsRecorder(api.NewRecordedEffectsRecorder(embeddedEventStore))
		dagRunner.SetReplayContextBuilder(api.NewReplayContextBuilder(embeddedEventStore))
		dagRunner.SetReplayPolicy(replaysandbox.DefaultPolicy{})
		runJob := func(ctx context.Context, j *job.Job) error {
			sessionID := j.SessionID
			if sessionID == "" {
				sessionID = j.AgentID
			}
			state, _ := embeddedStateStore.LoadAgentState(ctx, j.AgentID, sessionID)
			sess := runtime.NewSession(sessionID, j.AgentID)
			if state != nil {
				runtime.ApplyAgentState(sess, state)
			}
			agent := runtime.NewAgent(j.AgentID, j.AgentID, sess, nil, newPlannerProviderAdapter(v1Planner), newToolsProviderAdapter(toolsReg))
			err := dagRunner.RunForJob(ctx, agent, &agentexec.JobForRunner{ID: j.ID, AgentID: j.AgentID, Goal: j.Goal, Cursor: j.Cursor, TenantID: "default"})
			_ = embeddedStateStore.SaveAgentState(ctx, j.AgentID, sess.ID, runtime.SessionToAgentState(sess))
			return err
		}
		runner := NewAgentJobRunner(DefaultWorkerID(), embeddedEventStore, embeddedJobStore, runJob, 2*time.Second, 30*time.Second, 1, nil, logger)
		wakeupQueue := job.NewWakeupQueueMem(256)
		runner.SetWakeupQueue(wakeupQueue)
		appObj.agentJobRunner = runner
		appObj.jobEventStore = embeddedEventStore
		appObj.replayBuilder = replay.NewReplayContextBuilder(embeddedEventStore)
		appObj.jobStore = embeddedJobStore
		appObj.wakeupQueue = wakeupQueue
		logger.Info("Worker Agent Job Embedded 模式已启用", "path", embeddedBaseDir)
	}

	return appObj, nil
}

// Start 启动应用（Pipeline 由 eino 调度；JobStore=postgres 时启动 Agent Job Claim 循环）
func (a *App) Start() error {
	a.logger.Info("启动 worker 应用")

	if a.agentJobRunner != nil {
		ctx, cancel := context.WithCancel(context.Background())
		a.agentJobCancel = cancel
		a.agentJobRunner.Start(ctx)
	}

	// 可选：Prometheus /metrics 端点；多 Worker 时可用 AETHERIS_WORKER_METRICS_PORT 指定不同端口避免冲突
	if a.config != nil && a.config.Monitoring.Prometheus.Enable && a.config.Monitoring.Prometheus.Port > 0 {
		port := a.config.Monitoring.Prometheus.Port
		if envPort := os.Getenv("AETHERIS_WORKER_METRICS_PORT"); envPort != "" {
			if p, err := strconv.Atoi(envPort); err == nil && p > 0 {
				port = p
			}
		}
		mux := http.NewServeMux()
		mux.HandleFunc("/metrics", func(w http.ResponseWriter, _ *http.Request) {
			var buf bytes.Buffer
			if err := metrics.WritePrometheus(&buf); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", string(expfmt.FmtText))
			_, _ = w.Write(buf.Bytes())
		})
		addr := fmt.Sprintf(":%d", port)
		go func() {
			if err := http.ListenAndServe(addr, mux); err != nil && err != http.ErrServerClosed {
				a.logger.Error("metrics 服务异常", "error", err)
			}
		}()
		a.logger.Info("Prometheus /metrics 已启用", "addr", addr)
	}

	// Snapshot 自动化（2.0 performance）：每小时扫描事件数 > 1000 的 Job，自动创建快照减少 Replay 开销
	if a.jobEventStore != nil {
		go a.runSnapshotLoop()
	}

	// Storage GC（2.0 operational）：每 24h 清理超 TTL 的 tool_invocations 记录
	if a.jobEventStore != nil {
		go a.runGCLoop()
	}

	if a.jobStore != nil && a.jobEventStore != nil {
		go a.runApprovalExpiryLoop()
	}

	// 启动工作队列消费者：收到入库任务时调用 engine.ExecuteWorkflow(ctx, "ingest_pipeline", payload)
	if err := a.startWorkerQueue(); err != nil {
		return fmt.Errorf("启动工作队列failed: %w", err)
	}

	a.logger.Info("worker 应用启动成功")
	return nil
}

func (a *App) runApprovalExpiryLoop() {
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()
	a.logger.Info("审批过期自动收敛 goroutine 已启动", "interval", 15*time.Second)
	for {
		select {
		case <-a.shutdown:
			return
		case <-ticker.C:
		}
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		expired, err := approvalexpiry.ExpireApprovalWaitsOnce(ctx, a.jobStore, a.jobEventStore, a.wakeupQueue)
		cancel()
		if err != nil {
			a.logger.Warn("审批过期自动收敛 failed", "error", err)
			continue
		}
		if expired > 0 {
			a.logger.Info("审批过期自动收敛完成", "jobs", expired)
		}
	}
}

func appendTerminalEventAndUpdateStatus(ctx context.Context, eventStore jobstore.JobStore, metaStore job.JobStore, jobID string, payload []byte, eventType jobstore.EventType, status job.JobStatus) error {
	if err := appendTerminalEvent(ctx, eventStore, jobID, payload, eventType); err != nil {
		return err
	}
	return metaStore.UpdateStatus(ctx, jobID, status)
}

func appendTerminalEvent(ctx context.Context, eventStore jobstore.JobStore, jobID string, payload []byte, eventType jobstore.EventType) error {
	_, ver, err := eventStore.ListEvents(ctx, jobID)
	if err != nil {
		return err
	}
	_, err = eventStore.Append(ctx, jobID, ver, jobstore.JobEvent{JobID: jobID, Type: eventType, Payload: payload})
	return err
}

func attemptAwareContext(baseCtx context.Context, attemptCtx context.Context) context.Context {
	if baseCtx == nil {
		baseCtx = context.Background()
	}
	if attemptID := jobstore.AttemptIDFromContext(attemptCtx); attemptID != "" {
		return jobstore.WithAttemptID(baseCtx, attemptID)
	}
	return baseCtx
}

// runSnapshotLoop 定时扫描高事件量的 Job 并自动创建快照（每小时运行一次）
func (a *App) runSnapshotLoop() {
	const (
		snapshotInterval = 1 * time.Hour
		eventThreshold   = 1000 // 事件数超过此值时触发快照
		batchLimit       = 50
	)
	ticker := time.NewTicker(snapshotInterval)
	defer ticker.Stop()
	a.logger.Info("Snapshot 自动化 goroutine 已启动", "interval", snapshotInterval, "event_threshold", eventThreshold)

	for {
		select {
		case <-a.shutdown:
			return
		case <-ticker.C:
		}
		a.triggerSnapshotsForHighEventJobs(eventThreshold, batchLimit)
	}
}

// triggerSnapshotsForHighEventJobs 对高事件量的 Job 触发快照创建
func (a *App) triggerSnapshotsForHighEventJobs(eventThreshold, limit int) {
	ss, ok := a.jobEventStore.(jobstore.SnapshotJobStore)
	if !ok {
		return
	}
	if a.replayBuilder == nil {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	jobIDs, err := ss.ListJobsWithHighEventCount(ctx, eventThreshold, limit)
	if err != nil {
		a.logger.Warn("Snapshot 扫描failed", "error", err)
		return
	}
	if len(jobIDs) == 0 {
		return
	}

	a.logger.Info("Snapshot 自动化触发", "jobs", len(jobIDs))
	for _, jobID := range jobIDs {
		if err := ctx.Err(); err != nil {
			break
		}
		// 使用 ReplayContextBuilder 从事件流重建上下文，再序列化为快照
		rc, err := a.replayBuilder.BuildFromEvents(ctx, jobID)
		if err != nil || rc == nil {
			continue
		}
		_, version, err := ss.ListEvents(ctx, jobID)
		if err != nil {
			continue
		}
		snapshotBytes, err := replay.SerializeReplayContext(rc)
		if err != nil {
			a.logger.Warn("快照序列化failed", "job_id", jobID, "error", err)
			continue
		}
		if err := ss.CreateSnapshot(ctx, jobID, version, snapshotBytes); err != nil {
			a.logger.Warn("快照写入failed", "job_id", jobID, "error", err)
			continue
		}
		// 清理旧快照（保留最新一个）
		if latestSnap, err := ss.GetLatestSnapshot(ctx, jobID); err == nil && latestSnap != nil {
			_ = ss.DeleteSnapshotsBefore(ctx, jobID, latestSnap.Version)
		}
		a.logger.Info("快照已创建", "job_id", jobID, "version", version)
	}
}

// runGCLoop 定时清理过期的 tool_invocations 记录（每 24h 运行一次）
func (a *App) runGCLoop() {
	const gcInterval = 24 * time.Hour
	ticker := time.NewTicker(gcInterval)
	defer ticker.Stop()
	a.logger.Info("Storage GC goroutine 已启动", "interval", gcInterval)

	for {
		select {
		case <-a.shutdown:
			return
		case <-ticker.C:
		}
		a.runGC()
	}
}

// runGC 执行一次 GC
func (a *App) runGC() {
	gcCfg := jobstore.GCConfig{
		Enable:      true,
		TTLDays:     90,
		BatchSize:   1000,
		RunInterval: 24 * time.Hour,
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()

	if err := jobstore.GC(ctx, a.jobEventStore, gcCfg); err != nil {
		a.logger.Warn("Storage GC 执行failed", "error", err)
	} else {
		a.logger.Info("Storage GC 执行完成")
	}
}

// Shutdown 优雅关闭应用。
// 顺序：①停止认领新 Job → ②等待 in-flight Job 完成（受 ctx 超时约束） → ③停止后台 goroutine → ④关闭存储。
func (a *App) Shutdown(ctx context.Context) error {
	a.logger.Info("关闭 worker 应用")

	// 1. 停止 AgentJobRunner：关闭 stopCh 以停止认领新 Job，然后 wg.Wait() 等待所有 in-flight Job 完成
	if a.agentJobRunner != nil {
		// 在后台等待 Stop()；若 ctx 超时则强制取消正在执行的 Job
		done := make(chan struct{})
		go func() {
			a.agentJobRunner.Stop()
			close(done)
		}()
		select {
		case <-done:
			a.logger.Info("in-flight Job 已全部完成")
		case <-ctx.Done():
			a.logger.Warn("优雅关闭超时，强制终止 in-flight Job")
			if a.agentJobCancel != nil {
				a.agentJobCancel()
			}
			<-done // 等待 Stop() 返回
		}
	} else if a.agentJobCancel != nil {
		a.agentJobCancel()
	}

	// 2. 停止 Snapshot 自动化与 GC goroutine
	select {
	case <-a.shutdown:
		// 已关闭，避免 double-close panic
	default:
		close(a.shutdown)
	}

	// 3. 关闭 MCP 连接
	if a.mcpManager != nil {
		_ = a.mcpManager.Close()
	}

	// 4. 关闭存储
	if err := a.metadataStore.Close(); err != nil {
		a.logger.Error("关闭元数据存储failed", "error", err)
	}

	if err := a.vectorStore.Close(); err != nil {
		a.logger.Error("关闭向量存储failed", "error", err)
	}

	// 5. 关闭 eino 引擎
	if err := a.engine.Shutdown(); err != nil {
		a.logger.Error("关闭 eino 引擎failed", "error", err)
	}

	a.logger.Info("worker 应用关闭成功")
	return nil
}

// startWorkerQueue 启动工作队列消费者；每个入库任务应调用 a.engine.ExecuteWorkflow(ctx, "ingest_pipeline", taskPayload)
func (a *App) startWorkerQueue() error {
	if a.config == nil || a.config.JobStore.Type != "postgres" || a.config.JobStore.DSN == "" {
		return nil
	}
	dsn := a.config.JobStore.DSN
	poolConfig, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return fmt.Errorf("解析入库队列 DSN failed: %w", err)
	}
	pool, err := pgxpool.NewWithConfig(context.Background(), poolConfig)
	if err != nil {
		return fmt.Errorf("创建入库队列连接池failed: %w", err)
	}
	queue := ingestqueue.NewIngestQueuePg(pool)
	workerID := fmt.Sprintf("%s-%d", getHostname(), os.Getpid())
	pollInterval := 2 * time.Second
	if a.config.Worker.PollInterval != "" {
		if d, err := time.ParseDuration(a.config.Worker.PollInterval); err == nil && d > 0 {
			pollInterval = d
		}
	}
	go a.runIngestQueueLoop(queue, workerID, pollInterval)
	a.logger.Info("入库队列消费者已启动", "worker_id", workerID, "poll_interval", pollInterval)
	return nil
}

func validateProductionRuntimeConfig(cfg *config.Config) error {
	if cfg == nil {
		return nil
	}
	prod := cfg.Runtime.Profile == "prod" || cfg.Runtime.Strict
	if !prod {
		return nil
	}
	if cfg.JobStore.Type != "postgres" || cfg.JobStore.DSN == "" {
		return fmt.Errorf("production requires jobstore.type=postgres with dsn")
	}
	if cfg.EffectStore.Type != "postgres" || cfg.EffectStore.DSN == "" {
		return fmt.Errorf("production requires effect_store.type=postgres with dsn")
	}
	if cfg.CheckpointStore.Type != "postgres" || cfg.CheckpointStore.DSN == "" {
		return fmt.Errorf("production requires checkpoint_store.type=postgres with dsn")
	}
	// Validate default passwords are not used
	if containsDefaultPassword(cfg.JobStore.DSN) || containsDefaultPassword(cfg.EffectStore.DSN) || containsDefaultPassword(cfg.CheckpointStore.DSN) {
		return fmt.Errorf("production requires changing default passwords in DSN")
	}
	// Validate SSL is enabled
	if !isSSLEnabled(cfg.JobStore.DSN) || !isSSLEnabled(cfg.EffectStore.DSN) || !isSSLEnabled(cfg.CheckpointStore.DSN) {
		return fmt.Errorf("production requires SSL to be enabled for database connections")
	}
	return nil
}

// containsDefaultPassword checks if DSN contains the default password
func containsDefaultPassword(dsn string) bool {
	return dsn != "" && (strings.Contains(dsn, "aetheris:aetheris@") || strings.Contains(dsn, "password=aetheris"))
}

// isSSLEnabled checks if SSL is enabled in the DSN
func isSSLEnabled(dsn string) bool {
	if dsn == "" {
		return true
	}
	return !strings.Contains(dsn, "sslmode=disable")
}

func getHostname() string {
	h, _ := os.Hostname()
	if h == "" {
		return "worker"
	}
	return h
} // runIngestQueueLoop 轮询认领入库任务并执行 ingest_pipeline，直到 shutdown 关闭

func embeddedDataDir(cfg *config.Config) string {
	if cfg == nil {
		return filepath.Join("data", "embedded")
	}
	if cfg.JobStore.DSN != "" {
		return cfg.JobStore.DSN
	}
	return filepath.Join("data", "embedded")
}

// initMCPManager creates and connects the MCP Manager from config, registers discovered tools.
func initMCPManager(cfg *config.Config, reg *tools.Registry, logger *log.Logger) *mcp.Manager {
	mcpMgr := mcp.NewManager(slog.Default())
	if cfg != nil && len(cfg.MCP.Servers) > 0 {
		mcpCfg := mcp.ManagerConfig{
			Servers:     make(map[string]mcp.ServerConfig, len(cfg.MCP.Servers)),
			InitTimeout: cfg.MCP.InitTimeout,
		}
		for name, sc := range cfg.MCP.Servers {
			mcpCfg.Servers[name] = mcp.ServerConfig{
				Type:    sc.Type,
				Command: sc.Command,
				Args:    sc.Args,
				Env:     sc.Env,
				Dir:     sc.Dir,
				URL:     sc.URL,
				Headers: sc.Headers,
				Timeout: sc.Timeout,
			}
		}
		if err := mcpMgr.ConnectAll(context.Background(), mcpCfg); err != nil {
			logger.Warn("mcp connect error (non-fatal)", "error", err)
		}
	}
	mcpHost := tools.NewMCPHost(mcpMgr)
	count := mcpHost.RegisterFromManager(reg, mcpMgr)
	if count > 0 {
		logger.Info("MCP 工具已注册", "count", count, "servers", len(mcpMgr.ServerNames()))
	}
	return mcpMgr
}

func (a *App) runIngestQueueLoop(queue ingestqueue.IngestQueue, workerID string, pollInterval time.Duration) {
	for {
		select {
		case <-a.shutdown:
			return
		default:
		}
		ctx := context.Background()
		taskID, payload, err := queue.ClaimOne(ctx, workerID)
		if err != nil {
			a.logger.Error("认领入库任务failed", "error", err)
			time.Sleep(pollInterval)
			continue
		}
		if taskID == "" {
			time.Sleep(pollInterval)
			continue
		}
		contentBase64, _ := payload["content_base64"].(string)
		if contentBase64 == "" {
			_ = queue.MarkFailed(ctx, taskID, "payload 缺少 content_base64")
			continue
		}
		decoded, err := base64.StdEncoding.DecodeString(contentBase64)
		if err != nil {
			_ = queue.MarkFailed(ctx, taskID, "content_base64 解码failed: "+err.Error())
			continue
		}
		params := map[string]interface{}{"content": decoded}
		if fn, ok := payload["filename"].(string); ok && fn != "" {
			params["filename"] = fn
		}
		if meta, ok := payload["metadata"]; ok {
			params["metadata"] = meta
		}
		result, err := a.engine.ExecuteWorkflow(ctx, "ingest_pipeline", params)
		if err != nil {
			_ = queue.MarkFailed(ctx, taskID, err.Error())
			a.logger.Error("入库任务执行failed", "task_id", taskID, "error", err)
			continue
		}
		if err := queue.MarkCompleted(ctx, taskID, result); err != nil {
			a.logger.Error("标记入库任务完成failed", "task_id", taskID, "error", err)
		}
	}
}
