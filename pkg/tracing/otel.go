// Copyright 2026 fanjia1024
// OpenTelemetry integration for distributed tracing - Extended for DAG observability

package tracing

import (
	"context"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/jaeger"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/metric"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
	"go.opentelemetry.io/otel/trace"
)

// OTelConfig OpenTelemetry 配置
type OTelConfig struct {
	ServiceName    string
	ExportEndpoint string // OTLP HTTP endpoint
	GRPCEndpoint   string // OTLP gRPC endpoint
	Insecure       bool
	JaegerAgent    string // Jaeger agent address (e.g., "localhost:14250")
	PrometheusPort int    // Prometheus exporter port (e.g., 9090)
	Stdout         bool   // Print traces to stdout for debugging
}

// TracerProviderHolder holds the tracer provider and meter provider
type TracerProviderHolder struct {
	TracerProvider *sdktrace.TracerProvider
	MeterProvider  *sdkmetric.MeterProvider
}

// InitTracer 初始化 OpenTelemetry tracer (HTTP OTLP)
func InitTracer(config OTelConfig) (*TracerProviderHolder, error) {
	ctx := context.Background()

	var exporter sdktrace.SpanExporter
	var err error

	// 选择导出器
	if config.Stdout {
		// stdout exporter for debugging
		stdoutExporter, err := stdouttrace.New(stdouttrace.WithPrettyPrint())
		if err != nil {
			return nil, err
		}
		exporter = stdoutExporter
	} else if config.JaegerAgent != "" {
		// Jaeger exporter
		je, err := jaeger.New(jaeger.WithAgentEndpoint(jaeger.WithAgentHost(config.JaegerAgent)))
		if err != nil {
			return nil, err
		}
		exporter = je
	} else if config.GRPCEndpoint != "" {
		// OTLP gRPC exporter
		exporter, err = otlptrace.New(ctx, otlptracegrpc.NewClient(
			otlptracegrpc.WithEndpoint(config.GRPCEndpoint),
			otlptracegrpc.WithInsecure(),
		))
		if err != nil {
			return nil, err
		}
	} else {
		// OTLP HTTP exporter (default)
		opts := []otlptracehttp.Option{
			otlptracehttp.WithEndpoint(config.ExportEndpoint),
		}
		if config.Insecure {
			opts = append(opts, otlptracehttp.WithInsecure())
		}
		exporter, err = otlptrace.New(ctx, otlptracehttp.NewClient(opts...))
		if err != nil {
			return nil, err
		}
	}

	// 创建 resource
	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceName(config.ServiceName),
		),
	)
	if err != nil {
		return nil, err
	}

	// 创建 tracer provider with sampling
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
	)

	otel.SetTracerProvider(tp)

	// 创建 meter provider for metrics
	mp := sdkmetric.NewMeterProvider()
	otel.SetMeterProvider(mp)

	return &TracerProviderHolder{
		TracerProvider: tp,
		MeterProvider:  mp,
	}, nil
}

// InitTracerWithPrometheus 初始化支持 Prometheus 的 OpenTelemetry
func InitTracerWithPrometheus(config OTelConfig) (*TracerProviderHolder, error) {
	ctx := context.Background()

	// 创建 OTLP exporter
	var exporter sdktrace.SpanExporter
	var err error

	if config.GRPCEndpoint != "" {
		exporter, err = otlptrace.New(ctx, otlptracegrpc.NewClient(
			otlptracegrpc.WithEndpoint(config.GRPCEndpoint),
			otlptracegrpc.WithInsecure(),
		))
	} else if config.ExportEndpoint != "" {
		exporter, err = otlptrace.New(ctx, otlptracehttp.NewClient(
			otlptracehttp.WithEndpoint(config.ExportEndpoint),
			otlptracehttp.WithInsecure(),
		))
	} else {
		// Default to stdout if no endpoint provided
		stdoutExporter, err := stdouttrace.New(stdouttrace.WithPrettyPrint())
		if err != nil {
			return nil, err
		}
		exporter = stdoutExporter
	}

	if err != nil {
		return nil, err
	}

	// 创建 resource
	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceName(config.ServiceName),
		),
	)
	if err != nil {
		return nil, err
	}

	// 创建 tracer provider
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
	)

	otel.SetTracerProvider(tp)

	// 创建 Prometheus meter provider
	prometheusExporter, err := prometheus.New()
	if err != nil {
		return nil, err
	}

	mp := sdkmetric.NewMeterProvider(sdkmetric.WithReader(prometheusExporter))
	otel.SetMeterProvider(mp)

	return &TracerProviderHolder{
		TracerProvider: tp,
		MeterProvider:  mp,
	}, nil
}

// GetMeter 获取 meter 实例
func GetMeter(name string) metric.Meter {
	return otel.Meter(name)
}

// ===== Span 创建函数 =====

// StartJobSpan 开始 job execution span
func StartJobSpan(ctx context.Context, jobID string, agentID string) (context.Context, trace.Span) {
	tracer := otel.Tracer("aetheris")
	ctx, span := tracer.Start(ctx, "job.execute",
		trace.WithAttributes(
			attribute.String("job.id", jobID),
			attribute.String("agent.id", agentID),
		),
	)
	return ctx, span
}

// StartNodeSpan 开始 node execution span
func StartNodeSpan(ctx context.Context, nodeID string, nodeType string) (context.Context, trace.Span) {
	tracer := otel.Tracer("aetheris")
	ctx, span := tracer.Start(ctx, "node.execute",
		trace.WithAttributes(
			attribute.String("node.id", nodeID),
			attribute.String("node.type", nodeType),
		),
	)
	return ctx, span
}

// StartToolSpan 开始 tool invocation span
func StartToolSpan(ctx context.Context, toolName string, idempotencyKey string) (context.Context, trace.Span) {
	tracer := otel.Tracer("aetheris")
	ctx, span := tracer.Start(ctx, "tool.invoke",
		trace.WithAttributes(
			attribute.String("tool.name", toolName),
			attribute.String("tool.idempotency_key", idempotencyKey),
		),
	)
	return ctx, span
}

// StartPlanSpan 开始 plan 生成 span
func StartPlanSpan(ctx context.Context, goal string) (context.Context, trace.Span) {
	tracer := otel.Tracer("aetheris")
	ctx, span := tracer.Start(ctx, "plan.generate",
		trace.WithAttributes(
			attribute.String("goal", goal),
		),
	)
	return ctx, span
}

// StartCompileSpan 开始 TaskGraph 编译 span
func StartCompileSpan(ctx context.Context) (context.Context, trace.Span) {
	tracer := otel.Tracer("aetheris")
	ctx, span := tracer.Start(ctx, "graph.compile")
	return ctx, span
}

// StartInvokeSpan 开始 DAG invoke span
func StartInvokeSpan(ctx context.Context) (context.Context, trace.Span) {
	tracer := otel.Tracer("aetheris")
	ctx, span := tracer.Start(ctx, "graph.invoke")
	return ctx, span
}

// StartLLMSpan 开始 LLM 调用 span
func StartLLMSpan(ctx context.Context, model string) (context.Context, trace.Span) {
	tracer := otel.Tracer("aetheris")
	ctx, span := tracer.Start(ctx, "llm.generate",
		trace.WithAttributes(
			attribute.String("llm.model", model),
		),
	)
	return ctx, span
}

// ===== DAG 节点 Span 函数 =====

// IngestPipelineSpan 用于 ingest 工作流的细粒度 span
type IngestPipelineSpan struct {
	ctx    context.Context
	span   trace.Span
	start  time.Time
	step   string
}

// StartIngestStepSpan 开始 ingest 步骤 span (loader/parser/splitter/embedding/indexer)
func StartIngestStepSpan(ctx context.Context, ingestID, step string) (*IngestPipelineSpan, context.Context) {
	tracer := otel.Tracer("aetheris")
	ctx, span := tracer.Start(ctx, "ingest."+step,
		trace.WithAttributes(
			attribute.String("ingest.id", ingestID),
			attribute.String("ingest.step", step),
		),
	)
	return &IngestPipelineSpan{
		ctx:   ctx,
		span:  span,
		start: time.Now(),
		step:  step,
	}, ctx
}

// End 结束 span 并记录耗时
func (s *IngestPipelineSpan) End(err error) {
	if err != nil {
		s.span.SetAttributes(attribute.String("error", err.Error()))
		s.span.SetStatus(1, err.Error())
	}
	s.span.SetAttributes(attribute.Float64("duration_ms", float64(time.Since(s.start).Milliseconds())))
	s.span.End()
}

// QueryPipelineSpan 用于 query 工作流的细粒度 span
type QueryPipelineSpan struct {
	ctx    context.Context
	span   trace.Span
	start  time.Time
	step   string
}

// StartQueryStepSpan 开始 query 步骤 span (query_embed/retrieve/generate)
func StartQueryStepSpan(ctx context.Context, queryID, step string) (*QueryPipelineSpan, context.Context) {
	tracer := otel.Tracer("aetheris")
	ctx, span := tracer.Start(ctx, "query."+step,
		trace.WithAttributes(
			attribute.String("query.id", queryID),
			attribute.String("query.step", step),
		),
	)
	return &QueryPipelineSpan{
		ctx:   ctx,
		span:  span,
		start: time.Now(),
		step:  step,
	}, ctx
}

// End 结束 span 并记录耗时
func (s *QueryPipelineSpan) End(err error) {
	if err != nil {
		s.span.SetAttributes(attribute.String("error", err.Error()))
		s.span.SetStatus(1, err.Error())
	}
	s.span.SetAttributes(attribute.Float64("duration_ms", float64(time.Since(s.start).Milliseconds())))
	s.span.End()
}

// ===== LLM 调用细粒度 Span =====

// LLMSpan LLM 调用的细粒度 span
type LLMSpan struct {
	ctx    context.Context
	span   trace.Span
	start  time.Time
	model  string
}

// StartLLMCallSpan 开始 LLM 调用 span（带 tokens 跟踪）
func StartLLMCallSpan(ctx context.Context, model, prompt string) (*LLMSpan, context.Context) {
	tracer := otel.Tracer("aetheris")
	ctx, span := tracer.Start(ctx, "llm.call",
		trace.WithAttributes(
			attribute.String("llm.model", model),
			attribute.Int("llm.prompt_length", len(prompt)),
		),
	)
	return &LLMSpan{
		ctx:   ctx,
		span:  span,
		start: time.Now(),
		model: model,
	}, ctx
}

// SetTokens 设置 token 计数
func (s *LLMSpan) SetTokens(inputTokens, outputTokens int) {
	s.span.SetAttributes(
		attribute.Int("llm.input_tokens", inputTokens),
		attribute.Int("llm.output_tokens", outputTokens),
		attribute.Int("llm.total_tokens", inputTokens+outputTokens),
	)
}

// SetLatency 设置延迟
func (s *LLMSpan) SetLatency(latency time.Duration) {
	s.span.SetAttributes(attribute.Float64("llm.latency_ms", float64(latency.Milliseconds())))
}

// SetRetries 设置重试次数
func (s *LLMSpan) SetRetries(retries int) {
	s.span.SetAttributes(attribute.Int("llm.retries", retries))
}

// End 结束 span
func (s *LLMSpan) End(err error) {
	if err != nil {
		s.span.SetAttributes(attribute.String("error", err.Error()))
		s.span.SetStatus(1, err.Error())
	}
	s.span.SetAttributes(attribute.Float64("duration_ms", float64(time.Since(s.start).Milliseconds())))
	s.span.End()
}

// ===== Node 执行 Span =====

// NodeSpan DAG 节点执行的 span
type NodeSpan struct {
	ctx      context.Context
	span     trace.Span
	start    time.Time
	nodeID   string
	nodeType string
}

// StartDAGNodeSpan 开始 DAG 节点执行 span
func StartDAGNodeSpan(ctx context.Context, nodeID, nodeType string) (*NodeSpan, context.Context) {
	tracer := otel.Tracer("aetheris")
	ctx, span := tracer.Start(ctx, "dag.node."+nodeType,
		trace.WithAttributes(
			attribute.String("dag.node.id", nodeID),
			attribute.String("dag.node.type", nodeType),
		),
	)
	return &NodeSpan{
		ctx:      ctx,
		span:     span,
		start:    time.Now(),
		nodeID:   nodeID,
		nodeType: nodeType,
	}, ctx
}

// SetInputSize 设置输入大小
func (s *NodeSpan) SetInputSize(size int) {
	s.span.SetAttributes(attribute.Int("dag.input_size", size))
}

// SetOutputSize 设置输出大小
func (s *NodeSpan) SetOutputSize(size int) {
	s.span.SetAttributes(attribute.Int("dag.output_size", size))
}

// End 结束 span
func (s *NodeSpan) End(err error) {
	if err != nil {
		s.span.SetAttributes(attribute.String("error", err.Error()))
		s.span.SetStatus(1, err.Error())
	}
	s.span.SetAttributes(attribute.Float64("duration_ms", float64(time.Since(s.start).Milliseconds())))
	s.span.End()
}

// ===== Helper 函数 =====

// GetTraceID 获取当前 trace ID
func GetTraceID(ctx context.Context) string {
	span := trace.SpanFromContext(ctx)
	if span.SpanContext().HasTraceID() {
		return span.SpanContext().TraceID().String()
	}
	return ""
}

// GetSpanID 获取当前 span ID
func GetSpanID(ctx context.Context) string {
	span := trace.SpanFromContext(ctx)
	if span.SpanContext().HasSpanID() {
		return span.SpanContext().SpanID().String()
	}
	return ""
}

// AddEvent 向当前 span 添加事件
func AddEvent(ctx context.Context, name string, attrs ...attribute.KeyValue) {
	span := trace.SpanFromContext(ctx)
	if span.SpanContext().HasSpanID() {
		span.AddEvent(name, trace.WithAttributes(attrs...))
	}
}
