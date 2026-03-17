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

package metrics

import (
	"io"
	"strconv"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/common/expfmt"
)

// P99Buckets P99 延迟直方图专用桶
var P99Buckets = []float64{1, 5, 10, 25, 50, 100, 250, 500, 1000, 2500, 5000, 10000, 30000, 60000}

// LongTaskBuckets 长时间任务专用桶 (1min - 1hour)
var LongTaskBuckets = []float64{60000, 120000, 300000, 600000, 1800000, 3600000}

// 全局 Registry，供 API/Worker 注册与暴露
var DefaultRegistry = prometheus.NewRegistry()

// MetricsLabels 标准标签定义
type MetricsLabels struct {
	Tenant   string
	AgentID  string
	StepType string
	Tool     string
	Provider string
	Status   string
	Result   string
	Queue    string
}

// Observer 用于观测指标的结构
type Observer struct {
	mu           sync.RWMutex
	labelValues  map[string]string
	counters     map[string]prometheus.Counter
	gauges       map[string]prometheus.Gauge
	histograms   map[string]prometheus.Observer
	countersLock sync.Mutex
}

// NewObserver 创建新的观测器
func NewObserver(labels MetricsLabels) *Observer {
	return &Observer{
		labelValues: map[string]string{
			"tenant":   labels.Tenant,
			"agent_id": labels.AgentID,
			"step_type": labels.StepType,
			"tool":     labels.Tool,
			"provider": labels.Provider,
			"status":   labels.Status,
			"result":   labels.Result,
			"queue":    labels.Queue,
		},
		counters:   make(map[string]prometheus.Counter),
		gauges:     make(map[string]prometheus.Gauge),
		histograms: make(map[string]prometheus.Observer),
	}
}

// labels 根据标签生成 prometheus Labels
func (o *Observer) labels(baseLabels []string) prometheus.Labels {
	labels := prometheus.Labels{}
	for _, k := range baseLabels {
		if v, ok := o.labelValues[k]; ok {
			labels[k] = v
		}
	}
	return labels
}

// IncCounter 增加计数器
func (o *Observer) IncCounter(name string, baseLabels []string) {
	o.countersLock.Lock()
	defer o.countersLock.Unlock()

	if c, ok := o.counters[name]; ok {
		c.Inc()
		return
	}
	// 创建新的 counter
	counter := promauto.NewCounter(prometheus.CounterOpts{
		Name: name,
		Help: "Auto-created counter",
	})
	counter.Inc()
	o.counters[name] = counter
}

// ObserveHistogram 观测直方图
func (o *Observer) ObserveHistogram(name string, value float64, baseLabels []string) {
	if h, ok := o.histograms[name]; ok {
		h.Observe(value)
	}
}

// SetGauge 设置 gauge
func (o *Observer) SetGauge(name string, value float64, baseLabels []string) {
	if g, ok := o.gauges[name]; ok {
		g.Set(value)
	}
}

func init() {
	DefaultRegistry.MustRegister(
		JobDuration, JobTotal, JobFailTotal,
		ToolDuration, LLMTokensTotal,
		WorkerBusy,
		QueueBacklog, StuckJobCount,
		// 2.0 Rate limiting metrics
		RateLimitWaitSeconds, RateLimitRejectionsTotal,
		ToolConcurrentGauge, LLMConcurrentGauge,
		JobParkedDuration,
		// 3.0-M4 Advanced metrics
		DecisionQualityScore, AnomalyDetectedTotal, SignatureVerificationTotal,
		// P0 SLO metrics
		JobStateGauge, StepDurationSeconds, LeaseConflictTotal, ToolInvocationTotal,
		// Metrics MVP: tenant-aware + SLO
		JobsTotal, JobLatencySeconds,
		StepRetriesTotal, StepTimeoutTotal,
		LeaseAcquireTotal, SchedulerTickDurationSeconds,
		ToolInvocationsTotal, ToolErrorsTotal, ConfirmationReplayFailTotal, ConfirmationReplayWarnTotal,
		// Runtime tracing metrics
		PlanDurationSeconds, CompileDurationSeconds, NodeExecutionTotal,
		RunPauseTotal, RunResumeTotal, HumanDecisionTotal,
		// SLA metrics
		SLAQuotaLimitTotal, SLAQuotaUsed, SLAQuotaExceededTotal,
		SLOSLOStatus, SLAViolationTotal,
	)
}

// ===== Runtime Layer Metrics =====

// RuntimeLLMCallDurationSeconds LLM 调用耗时（秒）- Runtime 层
var RuntimeLLMCallDurationSeconds = prometheus.NewHistogramVec(
	prometheus.HistogramOpts{
		Name:    "aetheris_runtime_llm_call_duration_seconds",
		Help:    "Runtime 层 LLM 调用耗时（秒）",
		Buckets: P99Buckets,
	},
	[]string{"tenant", "model", "status"}, // status: success | error
)

// RuntimeLLMTokensTotal Runtime 层 LLM token 计数
var RuntimeLLMTokensTotal = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Name: "aetheris_runtime_llm_tokens_total",
		Help: "Runtime 层 LLM token 总数",
	},
	[]string{"tenant", "model", "direction"}, // direction: input | output
)

// RuntimeLLMRetriesTotal LLM 重试次数
var RuntimeLLMRetriesTotal = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Name: "aetheris_runtime_llm_retries_total",
		Help: "LLM 调用重试次数",
	},
	[]string{"tenant", "model", "reason"}, // reason: rate_limit | timeout | error
)

// RuntimeNodeExecutionDurationSeconds DAG 节点执行耗时（秒）
var RuntimeNodeExecutionDurationSeconds = prometheus.NewHistogramVec(
	prometheus.HistogramOpts{
		Name:    "aetheris_runtime_node_execution_duration_seconds",
		Help:    "DAG 节点执行耗时（秒）",
		Buckets: P99Buckets,
	},
	[]string{"tenant", "node_type", "status"},
)

// RuntimeNodeRetriesTotal 节点重试次数
var RuntimeNodeRetriesTotal = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Name: "aetheris_runtime_node_retries_total",
		Help: "DAG 节点重试次数",
	},
	[]string{"tenant", "node_type", "reason"},
)

// RuntimeIngestPipelineDurationSeconds Ingest Pipeline 执行耗时
var RuntimeIngestPipelineDurationSeconds = prometheus.NewHistogramVec(
	prometheus.HistogramOpts{
		Name:    "aetheris_runtime_ingest_pipeline_duration_seconds",
		Help:    "Ingest Pipeline 各阶段执行耗时",
		Buckets: P99Buckets,
	},
	[]string{"tenant", "step"}, // step: loader | parser | splitter | embedding | indexer
)

// RuntimeQueryPipelineDurationSeconds Query Pipeline 执行耗时
var RuntimeQueryPipelineDurationSeconds = prometheus.NewHistogramVec(
	prometheus.HistogramOpts{
		Name:    "aetheris_runtime_query_pipeline_duration_seconds",
		Help:    "Query Pipeline 各阶段执行耗时",
		Buckets: P99Buckets,
	},
	[]string{"tenant", "step"}, // step: query_embed | retrieve | generate
)

// ===== Adapter Layer Metrics =====

// AdapterLLMRequestDurationSeconds Adapter 层 LLM 请求耗时
var AdapterLLMRequestDurationSeconds = prometheus.NewHistogramVec(
	prometheus.HistogramOpts{
		Name:    "aetheris_adapter_llm_request_duration_seconds",
		Help:    "Adapter 层 LLM 请求耗时（秒）",
		Buckets: P99Buckets,
	},
	[]string{"tenant", "adapter_type", "model", "status"},
)

// AdapterLLMRequestTotal Adapter 层 LLM 请求总数
var AdapterLLMRequestTotal = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Name: "aetheris_adapter_llm_request_total",
		Help: "Adapter 层 LLM 请求总数",
	},
	[]string{"tenant", "adapter_type", "model", "status"},
)

// AdapterToolInvocationDurationSeconds Adapter 层工具调用耗时
var AdapterToolInvocationDurationSeconds = prometheus.NewHistogramVec(
	prometheus.HistogramOpts{
		Name:    "aetheris_adapter_tool_invocation_duration_seconds",
		Help:    "Adapter 层工具调用耗时（秒）",
		Buckets: P99Buckets,
	},
	[]string{"tenant", "adapter_type", "tool", "status"},
)

// AdapterToolInvocationTotal Adapter 层工具调用总数
var AdapterToolInvocationTotal = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Name: "aetheris_adapter_tool_invocation_total",
		Help: "Adapter 层工具调用总数",
	},
	[]string{"tenant", "adapter_type", "tool", "status"},
)

// ===== Storage Layer Metrics =====

// StorageConnectionPoolSize 连接池大小
var StorageConnectionPoolSize = prometheus.NewGaugeVec(
	prometheus.GaugeOpts{
		Name: "aetheris_storage_connection_pool_size",
		Help: "存储连接池大小（当前活跃连接数）",
	},
	[]string{"tenant", "storage_type", "pool_name"}, // storage_type: redis | postgres | mysql
)

// StorageConnectionPoolMax 连接池最大容量
var StorageConnectionPoolMax = prometheus.NewGaugeVec(
	prometheus.GaugeOpts{
		Name: "aetheris_storage_connection_pool_max",
		Help: "存储连接池最大容量",
	},
	[]string{"tenant", "storage_type", "pool_name"},
)

// StorageConnectionPoolIdle 空闲连接数
var StorageConnectionPoolIdle = prometheus.NewGaugeVec(
	prometheus.GaugeOpts{
		Name: "aetheris_storage_connection_pool_idle",
		Help: "存储连接池空闲连接数",
	},
	[]string{"tenant", "storage_type", "pool_name"},
)

// StorageOperationDurationSeconds 存储操作耗时
var StorageOperationDurationSeconds = prometheus.NewHistogramVec(
	prometheus.HistogramOpts{
		Name:    "aetheris_storage_operation_duration_seconds",
		Help:    "存储操作耗时（秒）",
		Buckets: P99Buckets,
	},
	[]string{"tenant", "storage_type", "operation"}, // operation: get | set | delete | mget | mset
)

// StorageOperationTotal 存储操作总数
var StorageOperationTotal = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Name: "aetheris_storage_operation_total",
		Help: "存储操作总数",
	},
	[]string{"tenant", "storage_type", "operation", "status"},
)

// StorageOperationErrorsTotal 存储操作错误数
var StorageOperationErrorsTotal = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Name: "aetheris_storage_operation_errors_total",
		Help: "存储操作错误数",
	},
	[]string{"tenant", "storage_type", "operation", "error_type"},
)

// ===== P99 Latency Metrics =====

// P99LLMLatencySeconds P99 LLM 延迟（秒）
var P99LLMLatencySeconds = prometheus.NewHistogramVec(
	prometheus.HistogramOpts{
		Name:    "aetheris_p99_llm_latency_seconds",
		Help:    "P99 LLM 延迟（秒）",
		Buckets: P99Buckets,
	},
	[]string{"tenant", "model"},
)

// P99NodeLatencySeconds P99 节点延迟（秒）
var P99NodeLatencySeconds = prometheus.NewHistogramVec(
	prometheus.HistogramOpts{
		Name:    "aetheris_p99_node_latency_seconds",
		Help:    "P99 节点延迟（秒）",
		Buckets: P99Buckets,
	},
	[]string{"tenant", "node_type"},
)

// P99StorageLatencySeconds P99 存储延迟（秒）
var P99StorageLatencySeconds = prometheus.NewHistogramVec(
	prometheus.HistogramOpts{
		Name:    "aetheris_p99_storage_latency_seconds",
		Help:    "P99 存储延迟（秒）",
		Buckets: P99Buckets,
	},
	[]string{"tenant", "storage_type", "operation"},
)

// ===== Time Window Metrics =====

// TimeWindowJobThroughput 时间窗口内 Job 吞吐量
var TimeWindowJobThroughput = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Name: "aetheris_timewindow_job_throughput_total",
		Help: "时间窗口内 Job 吞吐量",
	},
	[]string{"tenant", "window", "status"},
)

// TimeWindowLLMThroughput 时间窗口内 LLM 调用吞吐量
var TimeWindowLLMThroughput = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Name: "aetheris_timewindow_llm_throughput_total",
		Help: "时间窗口内 LLM 调用吞吐量",
	},
	[]string{"tenant", "window", "model"},
)

// TimeWindowTokenThroughput 时间窗口内 Token 吞吐量
var TimeWindowTokenThroughput = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Name: "aetheris_timewindow_token_throughput_total",
		Help: "时间窗口内 Token 吞吐量",
	},
	[]string{"tenant", "window", "direction"},
)

// ===== Queue Metrics =====

// QueueSize 当前队列大小
var QueueSize = prometheus.NewGaugeVec(
	prometheus.GaugeOpts{
		Name: "aetheris_queue_size",
		Help: "当前队列大小",
	},
	[]string{"tenant", "queue_name", "priority"},
)

// QueueAddTotal 加入队列总数
var QueueAddTotal = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Name: "aetheris_queue_add_total",
		Help: "加入队列总数",
	},
	[]string{"tenant", "queue_name", "priority"},
)

// QueueProcessTotal 出队列总数
var QueueProcessTotal = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Name: "aetheris_queue_process_total",
		Help: "出队列总数",
	},
	[]string{"tenant", "queue_name", "status"},
)

// ===== Retry Metrics =====

// RetryTotal 重试总数
var RetryTotal = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Name: "aetheris_retry_total",
		Help: "重试总数",
	},
	[]string{"tenant", "component", "reason"}, // component: llm | node | tool | storage
)

// RetryDelaySeconds 重试延迟（秒）
var RetryDelaySeconds = prometheus.NewHistogramVec(
	prometheus.HistogramOpts{
		Name:    "aetheris_retry_delay_seconds",
		Help:    "重试延迟（秒）",
		Buckets: []float64{0.1, 0.5, 1, 2, 5, 10, 30, 60},
	},
	[]string{"tenant", "component", "attempt"},
)

// ===== Cache Metrics =====

// CacheHitTotal 缓存命中总数
var CacheHitTotal = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Name: "aetheris_cache_hit_total",
		Help: "缓存命中总数",
	},
	[]string{"tenant", "cache_type"}, // cache_type: redis | memory | embedded
)

// CacheMissTotal 缓存未命中总数
var CacheMissTotal = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Name: "aetheris_cache_miss_total",
		Help: "缓存未命中总数",
	},
	[]string{"tenant", "cache_type"},
)

// CacheHitRate 缓存命中率
var CacheHitRate = prometheus.NewGaugeVec(
	prometheus.GaugeOpts{
		Name: "aetheris_cache_hit_rate",
		Help: "缓存命中率",
	},
	[]string{"tenant", "cache_type"},
)

// ===== Observer Helpers =====

// ObserveLLMCall 观测 LLM 调用
func ObserveLLMCall(tenant, model, status string, duration time.Duration, inputTokens, outputTokens int) {
	RuntimeLLMCallDurationSeconds.WithLabelValues(tenant, model, status).Observe(duration.Seconds())
	if inputTokens > 0 {
		RuntimeLLMTokensTotal.WithLabelValues(tenant, model, "input").Add(float64(inputTokens))
	}
	if outputTokens > 0 {
		RuntimeLLMTokensTotal.WithLabelValues(tenant, model, "output").Add(float64(outputTokens))
	}
}

// ObserveNodeExecution 观测节点执行
func ObserveNodeExecution(tenant, nodeType, status string, duration time.Duration) {
	RuntimeNodeExecutionDurationSeconds.WithLabelValues(tenant, nodeType, status).Observe(duration.Seconds())
}

// ObserveStorageOperation 观测存储操作
func ObserveStorageOperation(tenant, storageType, operation, status string, duration time.Duration) {
	StorageOperationDurationSeconds.WithLabelValues(tenant, storageType, operation).Observe(duration.Seconds())
	StorageOperationTotal.WithLabelValues(tenant, storageType, operation, status).Inc()
}

// ObserveIngestStep 观测 Ingest Pipeline 步骤
func ObserveIngestStep(tenant, step string, duration time.Duration) {
	RuntimeIngestPipelineDurationSeconds.WithLabelValues(tenant, step).Observe(duration.Seconds())
}

// ObserveQueryStep 观测 Query Pipeline 步骤
func ObserveQueryStep(tenant, step string, duration time.Duration) {
	RuntimeQueryPipelineDurationSeconds.WithLabelValues(tenant, step).Observe(duration.Seconds())
}

// SetConnectionPoolMetrics 设置连接池指标
func SetConnectionPoolMetrics(tenant, storageType, poolName string, size, max, idle int) {
	StorageConnectionPoolSize.WithLabelValues(tenant, storageType, poolName).Set(float64(size))
	StorageConnectionPoolMax.WithLabelValues(tenant, storageType, poolName).Set(float64(max))
	StorageConnectionPoolIdle.WithLabelValues(tenant, storageType, poolName).Set(float64(idle))
}

// ObserveRetry 观测重试
func ObserveRetry(tenant, component, reason string, delay time.Duration, attempt int) {
	RetryTotal.WithLabelValues(tenant, component, reason).Inc()
	RetryDelaySeconds.WithLabelValues(tenant, component, strconv.Itoa(attempt)).Observe(delay.Seconds())
}

// ObserveCacheHit 观测缓存命中
func ObserveCacheHit(tenant, cacheType string) {
	CacheHitTotal.WithLabelValues(tenant, cacheType).Inc()
}

// ObserveCacheMiss 观测缓存未命中
func ObserveCacheMiss(tenant, cacheType string) {
	CacheMissTotal.WithLabelValues(tenant, cacheType).Inc()
}

// ===== 原有 Metrics =====

// JobDuration Job 执行耗时（秒）
var JobDuration = prometheus.NewHistogramVec(
	prometheus.HistogramOpts{
		Name:    "aetheris_job_duration_seconds",
		Help:    "Job 执行耗时（秒）",
		Buckets: prometheus.DefBuckets,
	},
	[]string{"agent_id"},
)

// JobTotal Job 总数（按状态）
var JobTotal = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Name: "aetheris_job_total",
		Help: "Job 总数（按状态）",
	},
	[]string{"status"}, // completed | failed | cancelled
)

// JobFailTotal Job 失败/取消总数（与 JobTotal 配合可算 job_fail_rate）
var JobFailTotal = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Name: "aetheris_job_fail_total",
		Help: "Job 失败/取消总数",
	},
	[]string{"status"}, // failed | cancelled
)

// ToolDuration 工具调用耗时（秒）
var ToolDuration = prometheus.NewHistogramVec(
	prometheus.HistogramOpts{
		Name:    "aetheris_tool_duration_seconds",
		Help:    "工具调用耗时（秒）",
		Buckets: prometheus.DefBuckets,
	},
	[]string{"tool"},
)

// LLMTokensTotal LLM 调用 token 数
var LLMTokensTotal = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Name: "aetheris_llm_tokens_total",
		Help: "LLM 调用 token 总数",
	},
	[]string{"direction"}, // input | output
)

// WorkerBusy 当前正在执行的 Job 数（每 Worker）
var WorkerBusy = prometheus.NewGaugeVec(
	prometheus.GaugeOpts{
		Name: "aetheris_worker_busy",
		Help: "当前正在执行的 Job 数",
	},
	[]string{"worker_id"},
)

// QueueBacklog 按队列的 Pending Job 积压数（2.0 可观测性）
var QueueBacklog = prometheus.NewGaugeVec(
	prometheus.GaugeOpts{
		Name: "aetheris_queue_backlog",
		Help: "Pending Job 积压数（按 queue 或 default）",
	},
	[]string{"queue"},
)

// StuckJobCount 卡住 Job 数：status=Running 且 updated_at 超过阈值的数量（2.0 Stuck Job Detector）
var StuckJobCount = prometheus.NewGauge(
	prometheus.GaugeOpts{
		Name: "aetheris_stuck_job_count",
		Help: "卡住的 Job 数（Running 且超过阈值未更新）",
	},
)

// RateLimitWaitSeconds 限流等待时间（秒）
var RateLimitWaitSeconds = prometheus.NewHistogramVec(
	prometheus.HistogramOpts{
		Name:    "aetheris_rate_limit_wait_seconds",
		Help:    "限流等待时间（秒）",
		Buckets: []float64{0.001, 0.01, 0.1, 0.5, 1, 2, 5, 10},
	},
	[]string{"type", "name"}, // type: tool|llm|queue, name: tool_name|provider|queue_class
)

// RateLimitRejectionsTotal 限流拒绝次数
var RateLimitRejectionsTotal = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Name: "aetheris_rate_limit_rejections_total",
		Help: "限流拒绝次数",
	},
	[]string{"type", "name"},
)

// ToolConcurrentGauge Tool 当前并发数
var ToolConcurrentGauge = prometheus.NewGaugeVec(
	prometheus.GaugeOpts{
		Name: "aetheris_tool_concurrent",
		Help: "Tool 当前并发数",
	},
	[]string{"tool"},
)

// LLMConcurrentGauge LLM Provider 当前并发数
var LLMConcurrentGauge = prometheus.NewGaugeVec(
	prometheus.GaugeOpts{
		Name: "aetheris_llm_concurrent",
		Help: "LLM Provider 当前并发数",
	},
	[]string{"provider"},
)

// JobParkedDuration Job 处于 parked 状态的时长（秒）
var JobParkedDuration = prometheus.NewHistogramVec(
	prometheus.HistogramOpts{
		Name:    "aetheris_job_parked_duration_seconds",
		Help:    "Job 处于 parked 状态的时长（秒）",
		Buckets: []float64{10, 60, 300, 600, 1800, 3600, 7200, 14400}, // 10s ~ 4h
	},
	[]string{"agent_id"},
)

// DecisionQualityScore 决策质量评分（3.0-M4）
var DecisionQualityScore = prometheus.NewHistogramVec(
	prometheus.HistogramOpts{
		Name:    "aetheris_decision_quality_score",
		Help:    "决策质量评分（0-100）",
		Buckets: []float64{0, 20, 40, 60, 80, 100},
	},
	[]string{"job_id", "step_id"},
)

// AnomalyDetectedTotal 检测到的异常决策数（3.0-M4）
var AnomalyDetectedTotal = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Name: "aetheris_anomaly_detected_total",
		Help: "检测到的异常决策数",
	},
	[]string{"anomaly_type", "severity"},
)

// SignatureVerificationTotal 签名验证次数（3.0-M4）
var SignatureVerificationTotal = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Name: "aetheris_signature_verification_total",
		Help: "签名验证次数",
	},
	[]string{"result"}, // success | failed
)

// JobStateGauge 当前各状态 Job 数量（P0 SLO）
var JobStateGauge = prometheus.NewGaugeVec(
	prometheus.GaugeOpts{
		Name: "aetheris_job_state",
		Help: "当前各状态 Job 数量",
	},
	[]string{"state"}, // pending | running | waiting | parked | completed | failed | cancelled
)

// StepDurationSeconds 单步执行耗时（秒）（P0 SLO）；tenant/step_type/ok 供 SLO
var StepDurationSeconds = prometheus.NewHistogramVec(
	prometheus.HistogramOpts{
		Name:    "aetheris_step_duration_seconds",
		Help:    "单步执行耗时（秒）",
		Buckets: prometheus.DefBuckets,
	},
	[]string{"tenant", "step_type", "ok"}, // ok: true|false
)

// LeaseConflictTotal 租约冲突次数（ErrStaleAttempt）（P0 SLO）；tenant 未知时用 "unknown"
var LeaseConflictTotal = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Name: "aetheris_lease_conflict_total",
		Help: "Append 时 attempt_id 不匹配导致的拒绝次数",
	},
	[]string{"tenant"},
)

// ToolInvocationTotal 工具调用次数（按结果分类）（P0 SLO）
var ToolInvocationTotal = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Name: "aetheris_tool_invocation_total",
		Help: "工具调用次数（ok=真实执行成功, err=执行失败, restored=Replay/恢复注入）",
	},
	[]string{"result"}, // ok | err | restored
)

// --- Metrics MVP (SLO + tenant) ---

// JobsTotal 创建/终态 Job 总数（tenant, status）；创建时 status=pending，完成时 completed/failed/cancelled
var JobsTotal = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Name: "aetheris_jobs_total",
		Help: "Job 总数（按租户与状态）",
	},
	[]string{"tenant", "status"},
)

// JobLatencySeconds 从 created 到 done 的耗时直方图（tenant, status）
var JobLatencySeconds = prometheus.NewHistogramVec(
	prometheus.HistogramOpts{
		Name:    "aetheris_job_latency_seconds",
		Help:    "Job 从创建到完成的耗时（秒）",
		Buckets: prometheus.DefBuckets,
	},
	[]string{"tenant", "status"},
)

// StepRetriesTotal 单步重试次数（tenant, step_type, reason）
var StepRetriesTotal = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Name: "aetheris_step_retries_total",
		Help: "单步重试次数",
	},
	[]string{"tenant", "step_type", "reason"},
)

// StepTimeoutTotal 单步超时次数（tenant）
var StepTimeoutTotal = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Name: "aetheris_step_timeout_total",
		Help: "单步超时次数",
	},
	[]string{"tenant"},
)

// LeaseAcquireTotal 抢 lease 成功/失败（tenant, ok）
var LeaseAcquireTotal = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Name: "aetheris_lease_acquire_total",
		Help: "Scheduler 认领 Pending Job 次数（ok=成功）",
	},
	[]string{"tenant", "ok"},
)

// SchedulerTickDurationSeconds Scheduler 单次 tick 耗时（从 limiter 到 claim 结束）
var SchedulerTickDurationSeconds = prometheus.NewHistogram(
	prometheus.HistogramOpts{
		Name:    "aetheris_scheduler_tick_duration_seconds",
		Help:    "Scheduler 单次 tick 耗时（秒）",
		Buckets: prometheus.DefBuckets,
	},
)

// ToolInvocationsTotal 工具调用次数（tenant, tool, mode=execute|restore）
var ToolInvocationsTotal = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Name: "aetheris_tool_invocations_total",
		Help: "工具调用次数（mode=execute 真实执行, mode=restore 恢复/注入）",
	},
	[]string{"tenant", "tool", "mode"},
)

// ToolErrorsTotal 工具执行失败次数（tenant, tool）
var ToolErrorsTotal = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Name: "aetheris_tool_errors_total",
		Help: "工具执行失败次数",
	},
	[]string{"tenant", "tool"},
)

// ConfirmationReplayFailTotal confirmation replay 校验失败次数（tenant, tool）
var ConfirmationReplayFailTotal = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Name: "aetheris_confirmation_replay_fail_total",
		Help: "World-consistent replay 校验失败次数",
	},
	[]string{"tenant", "tool"},
)

// ConfirmationReplayWarnTotal ReplayVerificationWarn 模式下校验失败但继续执行的次数（tenant, tool）
var ConfirmationReplayWarnTotal = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Name: "aetheris_confirmation_replay_warn_total",
		Help: "Replay verification failed but continued (warn mode)",
	},
	[]string{"tenant", "tool"},
)

// --- Runtime execution metrics ---

// PlanDurationSeconds Plan 生成耗时（秒）
var PlanDurationSeconds = prometheus.NewHistogramVec(
	prometheus.HistogramOpts{
		Name:    "aetheris_plan_duration_seconds",
		Help:    "Plan 生成耗时（秒）",
		Buckets: prometheus.DefBuckets,
	},
	[]string{"tenant"},
)

// CompileDurationSeconds TaskGraph 编译耗时（秒）
var CompileDurationSeconds = prometheus.NewHistogramVec(
	prometheus.HistogramOpts{
		Name:    "aetheris_compile_duration_seconds",
		Help:    "TaskGraph 编译耗时（秒）",
		Buckets: prometheus.DefBuckets,
	},
	[]string{"tenant"},
)

// NodeExecutionTotal Node 执行计数（按类型/状态）
var NodeExecutionTotal = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Name: "aetheris_node_execution_total",
		Help: "Node 执行计数（按类型与状态）",
	},
	[]string{"tenant", "node_type", "status"}, // node_type: llm|tool|workflow|wait, status: success|failure
)

// RunPauseTotal Run 暂停次数
var RunPauseTotal = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Name: "aetheris_run_pause_total",
		Help: "Run 暂停次数",
	},
	[]string{"tenant", "reason"},
)

// RunResumeTotal Run 恢复次数
var RunResumeTotal = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Name: "aetheris_run_resume_total",
		Help: "Run 恢复次数",
	},
	[]string{"tenant", "strategy"}, // strategy: REUSE_SUCCESSFUL_EFFECTS | REEXECUTE_FROM_POINT
)

// HumanDecisionTotal 人工决策注入次数
var HumanDecisionTotal = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Name: "aetheris_human_decision_total",
		Help: "人工决策注入次数",
	},
	[]string{"tenant", "operator"},
)

// SLA Quota metrics

// SLAQuotaLimitTotal Quota 限制
var SLAQuotaLimitTotal = prometheus.NewGaugeVec(
	prometheus.GaugeOpts{
		Name: "aetheris_sla_quota_limit_total",
		Help: "SLA Quota 限制",
	},
	[]string{"tenant", "quota_type"},
)

// SLAQuotaUsed 已使用配额
var SLAQuotaUsed = prometheus.NewGaugeVec(
	prometheus.GaugeOpts{
		Name: "aetheris_sla_quota_used",
		Help: "已使用配额",
	},
	[]string{"tenant", "quota_type"},
)

// SLAQuotaExceededTotal Quota 超限次数
var SLAQuotaExceededTotal = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Name: "aetheris_sla_quota_exceeded_total",
		Help: "Quota 超限次数",
	},
	[]string{"tenant", "quota_type"},
)

// SLOSLOStatus SLO 状态
var SLOSLOStatus = prometheus.NewGaugeVec(
	prometheus.GaugeOpts{
		Name: "aetheris_slo_status",
		Help: "SLO 状态 (1=met, 0=violated)",
	},
	[]string{"tenant", "slo_name", "slo_type"},
)

// SLAViolationTotal SLA 违规次数
var SLAViolationTotal = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Name: "aetheris_sla_violation_total",
		Help: "SLA 违规次数",
	},
	[]string{"tenant", "slo_name", "slo_type"},
)

// WritePrometheus 将 Prometheus 文本格式写入 w（供 Hertz 等复用）
func WritePrometheus(w io.Writer) error {
	metrics, err := DefaultRegistry.Gather()
	if err != nil {
		return err
	}
	enc := expfmt.NewEncoder(w, expfmt.FmtText)
	for _, mf := range metrics {
		if err := enc.Encode(mf); err != nil {
			return err
		}
	}
	return nil
}

// MustRegister 注册 metrics（初始化时调用）
func MustRegister(metrics ...prometheus.Collector) {
	DefaultRegistry.MustRegister(metrics...)
}
