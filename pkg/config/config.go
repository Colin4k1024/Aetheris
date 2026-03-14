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

package config

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"
)

// Config 应用配置结构体
type Config struct {
	API             APIConfig             `mapstructure:"api"`
	Agent           AgentConfig           `mapstructure:"agent"`
	Agents          AgentsConfig          `mapstructure:"agents"`
	Runtime         RuntimeConfig         `mapstructure:"runtime"`
	JobStore        JobStoreConfig        `mapstructure:"jobstore"`
	EffectStore     EffectStoreConfig     `mapstructure:"effect_store"`
	CheckpointStore CheckpointStoreConfig `mapstructure:"checkpoint_store"`
	Worker          WorkerConfig          `mapstructure:"worker"`
	Model           ModelConfig           `mapstructure:"model"`
	Storage         StorageConfig         `mapstructure:"storage"`
	Log             LogConfig             `mapstructure:"log"`
	Monitoring      MonitoringConfig      `mapstructure:"monitoring"`
	RateLimits      RateLimitsConfig      `mapstructure:"rate_limits"`
	Security        SecurityConfig        `mapstructure:"security"`
	MCP             MCPConfig             `mapstructure:"mcp"`
}

// RuntimeConfig 运行时环境配置
type RuntimeConfig struct {
	Profile string `mapstructure:"profile"` // dev | prod
	Strict  bool   `mapstructure:"strict"`  // true 时启用生产强校验门禁
}

// RateLimitsConfig 限流配置（Tool + LLM）
type RateLimitsConfig struct {
	Tools map[string]ToolRateLimitConfig `mapstructure:"tools"`
	LLM   map[string]LLMRateLimitConfig  `mapstructure:"llm"`
}

// ToolRateLimitConfig 单个 Tool 的限流配置
type ToolRateLimitConfig struct {
	QPS           float64 `mapstructure:"qps"`
	MaxConcurrent int     `mapstructure:"max_concurrent"`
	Burst         int     `mapstructure:"burst"`
}

// LLMRateLimitConfig 单个 LLM Provider 的限流配置
type LLMRateLimitConfig struct {
	TokensPerMinute   int     `mapstructure:"tokens_per_minute"`
	RequestsPerMinute float64 `mapstructure:"requests_per_minute"`
	MaxConcurrent     int     `mapstructure:"max_concurrent"`
}

// SecurityConfig 安全配置
type SecurityConfig struct {
	MTLS        MTLSConfig        `mapstructure:"mtls"`
	APISigning  APISigningConfig  `mapstructure:"api_signing"`
	IPAllowList IPAllowListConfig `mapstructure:"ip_allowlist"`
	Secrets     SecretsConfig     `mapstructure:"secrets"`
}

// MTLSConfig mTLS 配置
type MTLSConfig struct {
	Enabled            bool   `mapstructure:"enabled"`
	CertFile           string `mapstructure:"cert_file"`
	KeyFile            string `mapstructure:"key_file"`
	CAFile             string `mapstructure:"ca_file"`
	ClientCertFile     string `mapstructure:"client_cert_file"`
	ClientKeyFile      string `mapstructure:"client_key_file"`
	InsecureSkipVerify bool   `mapstructure:"insecure_skip_verify"`
}

// APISigningConfig API 签名配置
type APISigningConfig struct {
	Enabled       bool     `mapstructure:"enabled"`
	Algorithm     string   `mapstructure:"algorithm"`
	ClockSkew     string   `mapstructure:"clock_skew"`
	RequiredPaths []string `mapstructure:"required_paths"`
}

// IPAllowListConfig IP 白名单配置
type IPAllowListConfig struct {
	Enabled        bool     `mapstructure:"enabled"`
	AllowIPs       []string `mapstructure:"allow_ips"`
	BlockIPs       []string `mapstructure:"block_ips"`
	TrustedProxies []string `mapstructure:"trusted_proxies"`
}

// SecretsConfig Secrets 配置
type SecretsConfig struct {
	Provider string            `mapstructure:"provider"` // vault, aws, k8s, env
	Config   map[string]string `mapstructure:"config"`
}

// JobStoreConfig 任务事件存储配置（事件流 + 租约）
type JobStoreConfig struct {
	Type          string `mapstructure:"type"`           // memory | postgres | embedded
	DSN           string `mapstructure:"dsn"`            // Postgres 连接串或 embedded 数据目录路径
	LeaseDuration string `mapstructure:"lease_duration"` // 租约时长，如 "30s"，空则默认 30s
}

// EffectStoreConfig 副作用存储配置
type EffectStoreConfig struct {
	Type string `mapstructure:"type"` // memory | postgres | embedded
	DSN  string `mapstructure:"dsn"`  // Postgres 连接串；embedded 时可为空（沿用 jobstore.dsn）
}

// CheckpointStoreConfig Checkpoint 存储配置
type CheckpointStoreConfig struct {
	Type string `mapstructure:"type"` // memory | postgres | embedded
	DSN  string `mapstructure:"dsn"`  // Postgres 连接串；embedded 时可为空（沿用 jobstore.dsn）
}

// AgentConfig Agent 与 Job 调度相关配置
type AgentConfig struct {
	JobScheduler JobSchedulerConfig `mapstructure:"job_scheduler"`
	ADK          AgentADKConfig     `mapstructure:"adk"` // Eino ADK 主 Runner（对话 run/resume/stream）
}

// AgentsConfig 本地 Agent 配置（从配置文件加载）
type AgentsConfig struct {
	Agents map[string]AgentDefConfig `mapstructure:"agents"`
	LLM    AgentLLMConfig            `mapstructure:"llm"`
	Tools  ToolsConfig               `mapstructure:"tools"`
}

// AgentDefConfig 单个 Agent 的配置
type AgentDefConfig struct {
	Type          string         `mapstructure:"type"`           // react, deer, manus, chain, graph, workflow
	Description   string         `mapstructure:"description"`    // 描述
	LLM           string         `mapstructure:"llm"`            // 使用的 LLM
	MaxIterations int            `mapstructure:"max_iterations"` // 最大迭代次数
	SystemPrompt  string         `mapstructure:"system_prompt"`  // 系统提示词
	Tools         []string       `mapstructure:"tools"`          // 工具名列表（空 = 全部）
	ChainType     string         `mapstructure:"chain_type"`     // chain 类型
	GraphType     string         `mapstructure:"graph_type"`     // graph 类型
	WorkflowType  string         `mapstructure:"workflow_type"`  // workflow 类型
	Config        map[string]any `mapstructure:"config"`         // 其他配置
}

// AgentLLMConfig 默认 LLM 配置（用于本地 Agent）
type AgentLLMConfig struct {
	Provider string `mapstructure:"provider"` // openai, anthropic, ollama 等
	Model    string `mapstructure:"model"`
	APIKey   string `mapstructure:"api_key"`
}

// ToolsConfig 工具配置
type ToolsConfig struct {
	Enabled     []string              `mapstructure:"enabled"`
	WebSearch   WebSearchToolConfig   `mapstructure:"web_search"`
	Calculator  CalculatorToolConfig  `mapstructure:"calculator"`
	FileReader  FileReaderToolConfig  `mapstructure:"file_reader"`
	HTTPRequest HTTPRequestToolConfig `mapstructure:"http_request"`
}

// WebSearchToolConfig 网页搜索工具配置
type WebSearchToolConfig struct {
	APIKey string `mapstructure:"api_key"`
	Engine string `mapstructure:"engine"`
}

// CalculatorToolConfig 计算器工具配置
type CalculatorToolConfig struct {
	Precision int `mapstructure:"precision"`
}

// FileReaderToolConfig 文件读取工具配置
type FileReaderToolConfig struct {
	AllowedPaths []string `mapstructure:"allowed_paths"`
}

// HTTPRequestToolConfig HTTP 请求工具配置
type HTTPRequestToolConfig struct {
	Timeout    int `mapstructure:"timeout"`
	MaxRetries int `mapstructure:"max_retries"`
}

// MCPConfig MCP (Model Context Protocol) overall configuration
type MCPConfig struct {
	// Servers maps server names to their connection configurations.
	Servers map[string]MCPServerConfig `mapstructure:"servers"`
	// InitTimeout is how long to wait for each server's initialize handshake, e.g. "30s".
	InitTimeout string `mapstructure:"init_timeout"`
}

// MCPServerConfig configures a single MCP server connection.
type MCPServerConfig struct {
	// Type is the transport type: "stdio" or "sse".
	Type string `mapstructure:"type"`
	// Command is the executable to launch for stdio transport.
	Command string `mapstructure:"command"`
	// Args are command arguments for stdio transport.
	Args []string `mapstructure:"args"`
	// Env are additional environment variables (KEY: VALUE) for stdio transport.
	Env map[string]string `mapstructure:"env"`
	// Dir is the working directory for stdio transport.
	Dir string `mapstructure:"dir"`
	// URL is the endpoint URL for SSE transport.
	URL string `mapstructure:"url"`
	// Headers are optional HTTP headers for SSE transport.
	Headers map[string]string `mapstructure:"headers"`
	// Timeout is per-call timeout, e.g. "60s".
	Timeout string `mapstructure:"timeout"`
}

// AgentADKConfig ADK Runner 配置（主对话入口）
type AgentADKConfig struct {
	Enabled         *bool  `mapstructure:"enabled"`          // 为 false 时禁用 ADK，使用原 Plan→Execute Agent；未配置时默认 true
	CheckpointStore string `mapstructure:"checkpoint_store"` // memory | 留空；后续可扩展 postgres/redis
}

// JobSchedulerConfig Scheduler 并发、重试、backoff 与队列优先级
type JobSchedulerConfig struct {
	// Enabled 为 false 时 API 不启动进程内 Scheduler，由独立 Worker 进程拉取执行（分布式模式）；未配置时默认 true
	Enabled        *bool    `mapstructure:"enabled"`
	MaxConcurrency int      `mapstructure:"max_concurrency"` // 最大并发执行数，<=0 使用默认 2
	RetryMax       int      `mapstructure:"retry_max"`       // 失败后最大重试次数（不含首次），<0 使用默认 2
	Backoff        string   `mapstructure:"backoff"`         // 重试前等待时间，如 "1s"，空则默认 1s
	Queues         []string `mapstructure:"queues"`          // 按优先级轮询的队列列表，如 ["realtime","default","background"]；空则不区分队列
}

// APIConfig API 服务配置
type APIConfig struct {
	Port       int              `mapstructure:"port"`
	Host       string           `mapstructure:"host"`
	Timeout    string           `mapstructure:"timeout"`
	CORS       CORSConfig       `mapstructure:"cors"`
	Middleware MiddlewareConfig `mapstructure:"middleware"`
	Forensics  ForensicsConfig  `mapstructure:"forensics"`
	Grpc       GrpcConfig       `mapstructure:"grpc"`
}

// ForensicsConfig 取证查询类接口配置
type ForensicsConfig struct {
	Experimental bool `mapstructure:"experimental"`
}

// GrpcConfig gRPC 服务配置
type GrpcConfig struct {
	Enable bool `mapstructure:"enable"`
	Port   int  `mapstructure:"port"`
}

// CORSConfig CORS 配置
type CORSConfig struct {
	Enable       bool     `mapstructure:"enable"`
	AllowOrigins []string `mapstructure:"allow_origins"`
}

// MiddlewareConfig 中间件配置
type MiddlewareConfig struct {
	Auth          bool   `mapstructure:"auth"`
	RateLimit     bool   `mapstructure:"rate_limit"`
	RateLimitRPS  int    `mapstructure:"rate_limit_rps"`
	JWTKey        string `mapstructure:"jwt_key"`
	JWTTimeout    string `mapstructure:"jwt_timeout"`     // 如 "1h"
	JWTMaxRefresh string `mapstructure:"jwt_max_refresh"` // 如 "1h"
}

// WorkerConfig Worker 服务配置
type WorkerConfig struct {
	Concurrency  int      `mapstructure:"concurrency"`
	QueueSize    int      `mapstructure:"queue_size"`
	RetryCount   int      `mapstructure:"retry_count"`
	RetryDelay   string   `mapstructure:"retry_delay"`
	Timeout      string   `mapstructure:"timeout"`
	PollInterval string   `mapstructure:"poll_interval"` // Agent Job Claim 轮询间隔，如 "2s"
	MaxAttempts  int      `mapstructure:"max_attempts"`  // Agent Job 最大执行次数（含首次），达此后标记 Failed 不再调度；<=0 时默认 3
	Capabilities []string `mapstructure:"capabilities"`  // Worker 能力列表（如 llm, tool, rag）；Scheduler 仅派发 RequiredCapabilities 满足的 Job；空表示接受任意 Job
}

// ModelConfig 模型配置
type ModelConfig struct {
	LLM       LLMConfig       `mapstructure:"llm"`
	Embedding EmbeddingConfig `mapstructure:"embedding"`
	Vision    VisionConfig    `mapstructure:"vision"`
	Defaults  DefaultsConfig  `mapstructure:"defaults"`
}

// LLMConfig LLM 模型配置
type LLMConfig struct {
	Providers map[string]ProviderConfig `mapstructure:"providers"`
}

// EmbeddingConfig Embedding 模型配置
type EmbeddingConfig struct {
	Providers map[string]ProviderConfig `mapstructure:"providers"`
}

// VisionConfig Vision 模型配置
type VisionConfig struct {
	Providers map[string]ProviderConfig `mapstructure:"providers"`
}

// ProviderConfig 模型提供商配置
type ProviderConfig struct {
	APIKey  string               `mapstructure:"api_key"`
	BaseURL string               `mapstructure:"base_url"`
	Models  map[string]ModelInfo `mapstructure:"models"`
}

// ModelInfo 模型信息
type ModelInfo struct {
	Name          string  `mapstructure:"name"`
	ContextWindow int     `mapstructure:"context_window"`
	Temperature   float64 `mapstructure:"temperature"`
	Dimension     int     `mapstructure:"dimension"`
	InputLimit    int     `mapstructure:"input_limit"`
	MaxTokens     int     `mapstructure:"max_tokens"`
}

// DefaultsConfig 默认模型配置
type DefaultsConfig struct {
	LLM       string `mapstructure:"llm"`
	Embedding string `mapstructure:"embedding"`
	Vision    string `mapstructure:"vision"`
}

// StorageConfig 存储配置
type StorageConfig struct {
	Metadata MetadataConfig `mapstructure:"metadata"`
	Vector   VectorConfig   `mapstructure:"vector"`
	Object   ObjectConfig   `mapstructure:"object"`
	Cache    CacheConfig    `mapstructure:"cache"`
	Ingest   IngestConfig   `mapstructure:"ingest"`
}

// IngestConfig 入库管线配置（索引批大小、并发等）
type IngestConfig struct {
	BatchSize   int `mapstructure:"batch_size"`
	Concurrency int `mapstructure:"concurrency"`
}

// MetadataConfig 元数据存储配置
type MetadataConfig struct {
	Type     string `mapstructure:"type"`
	DSN      string `mapstructure:"dsn"`
	PoolSize int    `mapstructure:"pool_size"`
}

// VectorConfig 向量存储配置（memory 为内置内存；redis/milvus2/es8 等使用 eino-ext 对应组件）
type VectorConfig struct {
	Type       string `mapstructure:"type"`
	Addr       string `mapstructure:"addr"`
	DB         string `mapstructure:"db"`         // memory 忽略；Redis 为 DB 编号，如 "0"
	Collection string `mapstructure:"collection"` // 默认索引/集合名，ingest 与 query 共用
	Password   string `mapstructure:"password"`   // Redis 等后端密码，可选
}

// ObjectConfig 对象存储配置
type ObjectConfig struct {
	Type     string `mapstructure:"type"`
	Endpoint string `mapstructure:"endpoint"`
	Bucket   string `mapstructure:"bucket"`
	Region   string `mapstructure:"region"`
}

// CacheConfig 缓存配置
type CacheConfig struct {
	Type     string `mapstructure:"type"`
	Addr     string `mapstructure:"addr"`
	DB       int    `mapstructure:"db"`
	Password string `mapstructure:"password"`
}

// LogConfig 日志配置
type LogConfig struct {
	Level  string `mapstructure:"level"`
	Format string `mapstructure:"format"`
	File   string `mapstructure:"file"`
}

// MonitoringConfig 监控配置
type MonitoringConfig struct {
	Prometheus PrometheusConfig `mapstructure:"prometheus"`
	Tracing    TracingConfig    `mapstructure:"tracing"`
}

// TracingConfig 链路追踪配置（OpenTelemetry）
type TracingConfig struct {
	Enable         bool   `mapstructure:"enable"`
	ServiceName    string `mapstructure:"service_name"`
	ExportEndpoint string `mapstructure:"export_endpoint"`
	Insecure       bool   `mapstructure:"insecure"`
}

// PrometheusConfig Prometheus 配置
type PrometheusConfig struct {
	Enable bool `mapstructure:"enable"`
	Port   int  `mapstructure:"port"`
}

// DefaultDevConfig 返回开发模式默认配置
func DefaultDevConfig() *Config {
	return &Config{
		Runtime: RuntimeConfig{
			Profile: "dev",
		},
		API: APIConfig{
			Port: 8080,
			Middleware: MiddlewareConfig{
				Auth: false, // 开发模式不需要认证
			},
			CORS: CORSConfig{
				Enable:       true,
				AllowOrigins: []string{"*"},
			},
		},
		JobStore: JobStoreConfig{
			Type: "memory",
		},
		EffectStore: EffectStoreConfig{
			Type: "memory",
		},
		CheckpointStore: CheckpointStoreConfig{
			Type: "memory",
		},
		Storage: StorageConfig{
			Vector: VectorConfig{
				Type: "memory",
			},
			Cache: CacheConfig{
				Type: "memory",
			},
		},
	}
}

// LoadConfig 加载配置文件
func LoadConfig(configPath string) (*Config, error) {
	viper.SetConfigFile(configPath)
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	if err := viper.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("无法读取配置文件: %w", err)
	}

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("无法解析配置文件: %w", err)
	}

	// 替换环境变量
	if err := replaceEnvVars(&config); err != nil {
		return nil, err
	}

	return &config, nil
}

// replaceEnvVars 替换配置中的环境变量
func replaceEnvVars(config *Config) error {
	// 替换模型 API Key
	for provider, providerConfig := range config.Model.LLM.Providers {
		if strings.HasPrefix(providerConfig.APIKey, "$") {
			envVar := strings.TrimPrefix(strings.TrimSuffix(providerConfig.APIKey, "}"), "${")
			if val := os.Getenv(envVar); val != "" {
				providerConfig.APIKey = val
				config.Model.LLM.Providers[provider] = providerConfig
			}
		}
	}

	for provider, providerConfig := range config.Model.Embedding.Providers {
		if strings.HasPrefix(providerConfig.APIKey, "$") {
			envVar := strings.TrimPrefix(strings.TrimSuffix(providerConfig.APIKey, "}"), "${")
			if val := os.Getenv(envVar); val != "" {
				providerConfig.APIKey = val
				config.Model.Embedding.Providers[provider] = providerConfig
			}
		}
	}

	for provider, providerConfig := range config.Model.Vision.Providers {
		if strings.HasPrefix(providerConfig.APIKey, "$") {
			envVar := strings.TrimPrefix(strings.TrimSuffix(providerConfig.APIKey, "}"), "${")
			if val := os.Getenv(envVar); val != "" {
				providerConfig.APIKey = val
				config.Model.Vision.Providers[provider] = providerConfig
			}
		}
	}

	return nil
}

// LoadAPIConfig 加载 API 配置（仅 configs/api.yaml）
func LoadAPIConfig() (*Config, error) {
	if p := os.Getenv("API_CONFIG_PATH"); p != "" {
		return LoadConfig(p)
	}
	if p := os.Getenv("CONFIG_PATH"); p != "" {
		return LoadConfig(p)
	}
	return LoadConfig("configs/api.yaml")
}

// LoadAPIConfigWithModel 加载 API 配置并合并 model 配置，便于 API 使用 LLM/Embedding；storage 仍来自 api.yaml（缺省为 memory）
func LoadAPIConfigWithModel() (*Config, error) {
	apiPath := "configs/api.yaml"
	if p := os.Getenv("API_CONFIG_PATH"); p != "" {
		apiPath = p
	} else if p := os.Getenv("CONFIG_PATH"); p != "" {
		apiPath = p
	}
	cfg, err := LoadConfig(apiPath)
	if err != nil {
		return nil, err
	}
	modelPath := "configs/model.yaml"
	if p := os.Getenv("MODEL_CONFIG_PATH"); p != "" {
		modelPath = p
	}
	modelCfg, err := LoadConfig(modelPath)
	if err == nil {
		cfg.Model = modelCfg.Model
	}
	return cfg, nil
}

// LoadWorkerConfig 加载 Worker 配置（仅 configs/worker.yaml）
func LoadWorkerConfig() (*Config, error) {
	if p := os.Getenv("WORKER_CONFIG_PATH"); p != "" {
		return LoadConfig(p)
	}
	if p := os.Getenv("CONFIG_PATH"); p != "" {
		return LoadConfig(p)
	}
	return LoadConfig("configs/worker.yaml")
}

// LoadWorkerConfigWithModel 加载 Worker 配置并合并 model 配置，便于 Worker 执行 Agent Job 时使用 LLM/Embedding。
// model 路径解析为与 worker 配置同目录（configs/），避免 cwd 导致 model.yaml 未加载。
func LoadWorkerConfigWithModel() (*Config, error) {
	workerPath := "configs/worker.yaml"
	if p := os.Getenv("WORKER_CONFIG_PATH"); p != "" {
		workerPath = p
	} else if p := os.Getenv("CONFIG_PATH"); p != "" {
		workerPath = p
	}
	cfg, err := LoadConfig(workerPath)
	if err != nil {
		return nil, err
	}
	modelPath := "configs/model.yaml"
	if p := os.Getenv("MODEL_CONFIG_PATH"); p != "" {
		modelPath = p
	} else if absWorker, errAbs := filepath.Abs(workerPath); errAbs == nil {
		modelPath = filepath.Join(filepath.Dir(absWorker), "model.yaml")
	}
	modelCfg, err := LoadConfig(modelPath)
	if err == nil {
		cfg.Model = modelCfg.Model
	} else {
		log.Printf("[config] 未加载 model 配置 %q，Worker 将无 LLM 配置: %v", modelPath, err)
	}
	return cfg, nil
}

// LoadModelConfig 加载模型配置
func LoadModelConfig() (*Config, error) {
	return LoadConfig("configs/model.yaml")
}

// ValidateProductionMode 检查生产模式配置是否安全
func (c *Config) ValidateProductionMode() error {
	if c.Runtime.Profile == "prod" {
		if !c.API.Middleware.Auth {
			return fmt.Errorf("production mode requires authentication to be enabled. Set api.middleware.auth: true in config")
		}
		if c.API.Middleware.JWTKey == "" {
			return fmt.Errorf("production mode requires JWT key to be set. Set api.middleware.jwt_key in config")
		}
	}
	return nil
}
