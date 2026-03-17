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

package routing

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// RouterSelectionsTotal 模型选择总次数
	RouterSelectionsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "router_selections_total",
			Help: "Total number of model selections",
		},
		[]string{"tier", "provider", "complexity", "strategy"},
	)

	// RouterLatencyMs 模型选择延迟
	RouterLatencyMs = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "router_latency_ms",
			Help:    "Model selection latency in milliseconds",
			Buckets: []float64{1, 5, 10, 25, 50, 100, 250, 500, 1000, 2500, 5000, 10000},
		},
		[]string{"tier", "provider"},
	)

	// RouterFallbacksTotal 模型降级总次数
	RouterFallbacksTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "router_fallbacks_total",
			Help: "Total number of model fallbacks",
		},
		[]string{"from_tier", "to_tier", "reason"},
	)

	// RouterCostEstimatedTotal 预估成本
	RouterCostEstimatedTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "router_cost_estimated_total",
			Help: "Estimated cost of model selections in USD",
		},
		[]string{"tier", "provider"},
	)

	// RouterErrorsTotal 路由器错误总次数
	RouterErrorsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "router_errors_total",
			Help: "Total number of router errors",
		},
		[]string{"provider", "error_type"},
	)

	// RouterRetriesTotal 重试总次数
	RouterRetriesTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "router_retries_total",
			Help: "Total number of retries",
		},
		[]string{"model", "reason"},
	)

	// RouterContextTokensTotal 上下文 tokens
	RouterContextTokensTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "router_context_tokens_total",
			Help: "Total tokens used in routing context",
		},
		[]string{"direction"}, // input or output
	)

	// RouterActiveRequests 当前活跃请求数
	RouterActiveRequests = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "router_active_requests",
			Help: "Number of currently active routing requests",
		},
	)

	// RouterHotSwitchesTotal 热切换次数
	RouterHotSwitchesTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "router_hot_switches_total",
			Help: "Total number of hot switches",
		},
		[]string{"from_model", "to_model", "reason"},
	)
)
