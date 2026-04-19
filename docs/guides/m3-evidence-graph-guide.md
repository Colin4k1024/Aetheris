# M3 Evidence Graph Guide - 决策依据可视化

## 概述

Aetheris 2.0-M3 提供 Evidence Graph，将"发生了什么"转化为"为什么这么做"，通过可视化的因果关系图展示决策依据。

---

## 核心概念

### Evidence Graph（证据图）

决策依据的 DAG（有向无环图）：
- **节点**: Steps（计划、执行、工具调用）
- **边**: 因果关系（uses_output、invokes_tool）
- **证据**: 每个节点附带的证据（RAG文档、Tool调用、LLM决策）

### Evidence Node（证据节点）

7 种证据类型：

| 类型 | 说明 | 示例 |
|------|------|------|
| `rag_doc` | RAG 检索的文档 | doc_123 (similarity: 0.95) |
| `tool_invocation` | 工具调用 | stripe.charge (inv_456) |
| `memory_entry` | 记忆条目 | mem_789 |
| `llm_decision` | LLM 决策 | gpt-4o (temp: 0.7) |
| `human_approval` | 人类审批 | user_admin approved |
| `policy_rule` | 策略规则 | rule_refund_under_30_days |
| `signal` | 外部信号 | signal_123 |

### Causal Dependency（因果依赖）

通过 `input_keys` 和 `output_keys` 构建依赖关系：

```
Step A: output_keys = ["order_status"]
Step B: input_keys = ["order_status"], output_keys = ["refund_amount"]
Step C: input_keys = ["refund_amount"]

依赖关系: A → B → C
```

---

## API 使用

### 获取 Evidence Graph

```bash
GET /api/jobs/:job_id/evidence-graph

# 响应
{
  "job_id": "job_123",
  "graph": {
    "nodes": [
      {
        "step_id": "step_a",
        "node_id": "node_a",
        "type": "plan",
        "label": "Generate Plan",
        "evidence": {
          "nodes": [
            {"type": "rag_doc", "id": "doc_123"},
            {"type": "llm_decision", "id": "gpt-4o"}
          ],
          "input_keys": [],
          "output_keys": ["order_id"]
        }
      },
      {
        "step_id": "step_b",
        "node_id": "node_b",
        "type": "tool",
        "label": "Process Payment",
        "evidence": {
          "nodes": [
            {"type": "tool_invocation", "id": "inv_456"}
          ],
          "input_keys": ["order_id"],
          "output_keys": ["payment_result"]
        }
      }
    ],
    "edges": [
      {
        "from": "step_a",
        "to": "step_b",
        "relation": "uses_output",
        "data_key": "order_id"
      }
    ]
  }
}
```

---

## UI 可视化

### 访问 Evidence Graph

1. 打开 Job trace 页面: `/api/jobs/job_123/trace/page`
2. 点击 **"Evidence Graph"** tab
3. 查看可交互的因果关系图

### 交互功能

- **Zoom/Pan**: 鼠标滚轮缩放，拖拽平移
- **点击节点**: 显示证据详情面板
- **高亮路径**: 点击节点高亮因果路径
- **过滤**: 按证据类型过滤（只显示 LLM 决策、只显示 Tool 调用等）

### 证据详情面板

点击节点后显示：

```
Step: Process Payment (step_b)
Type: Tool Invocation

Evidence Nodes:
  🔧 Tool: stripe.charge (inv_456)
    - Status: success
    - Amount: $99.99
    - External ID: ch_abc123xyz

Causal Dependencies:
  ← Reads: order_id (from Generate Plan)
  → Writes: payment_result (to Send Confirmation Email)

Timestamp: 2026-02-12 10:30:45 UTC
Duration: 1.2s
```

---

## 典型场景

### 场景 1: 审计邮件发送

**问题**: "这封错误邮件是谁让 AI 发的？"

**操作**:
1. 搜索 `email_sent` 事件的 job
2. 查看 Evidence Graph
3. 回溯到上游节点：
   - LLM 决策（模型、temperature、prompt hash）
   - Tool 调用（获取用户数据）
   - RAG 文档（邮件模板）
4. 定位问题：某个 Tool 返回了错误数据

### 场景 2: 审计支付决策

**问题**: "为什么批准了这笔退款？依据是什么？"

**操作**:
1. 找到 `payment_executed` 事件
2. 查看 Evidence Graph
3. 追溯决策链：
   - Human approval（谁批准的）
   - Policy rule（符合哪条规则）
   - Tool invocation（订单状态查询）
4. 导出证据包给合规团队

### 场景 3: 复盘关键决策

**问题**: "某次 critical decision 是如何做出的？"

**操作**:
1. 搜索 `critical_decision_made` 事件
2. 查看 Evidence Graph
3. 分析证据完整性：
   - 是否有 RAG 支持？
   - 是否有 Tool 验证？
   - 是否有人类审批？
4. 验证决策合理性

---

## 图可视化技术

### Cytoscape.js 集成

UI 使用 Cytoscape.js 渲染（轻量、高性能）：

```javascript
// 初始化图
const cy = cytoscape({
  container: document.getElementById('evidence-graph'),
  
  elements: {
    nodes: graph.nodes.map(n => ({
      data: {
        id: n.step_id,
        label: n.label,
        type: n.type,
        evidence: n.evidence
      }
    })),
    
    edges: graph.edges.map(e => ({
      data: {
        source: e.from,
        target: e.to,
        label: e.data_key
      }
    }))
  },
  
  style: [
    {
      selector: 'node',
      style: {
        'label': 'data(label)',
        'background-color': '#4A90E2',
        'width': 60,
        'height': 60
      }
    },
    {
      selector: 'edge',
      style: {
        'label': 'data(label)',
        'curve-style': 'bezier',
        'target-arrow-shape': 'triangle'
      }
    }
  ],
  
  layout: {
    name: 'dagre',  // 层次布局（适合 DAG）
    rankDir: 'TB'   // 从上到下
  }
});

// 点击节点显示详情
cy.on('tap', 'node', function(evt){
  const node = evt.target;
  showEvidenceDetail(node.data());
});
```

### 布局算法

支持多种布局：
- **Dagre**: 层次布局（推荐，适合 DAG）
- **Klay**: 复杂图布局
- **Cola**: 力导向布局
- **Grid**: 网格布局

---

## 编程接口

### 构建 Evidence Graph

```go
import "github.com/Colin4k1024/Aetheris/v2/pkg/evidence"

builder := evidence.NewBuilder()
graph, err := builder.BuildFromEvents(events)

// 访问节点
for _, node := range graph.Nodes {
    fmt.Printf("Step: %s, Evidence count: %d\n", 
        node.StepID, len(node.Evidence.Nodes))
}

// 访问边
for _, edge := range graph.Edges {
    fmt.Printf("Dependency: %s → %s (via %s)\n", 
        edge.From, edge.To, edge.DataKey)
}
```

---

## 最佳实践

1. **完整记录证据**: 每个 reasoning_snapshot 应包含 evidence 字段
2. **标注关键决策**: 对重要操作发送 critical_decision_made 事件
3. **因果关系明确**: 使用 input_keys 和 output_keys 建立依赖
4. **定期审查**: 使用 Evidence Graph 回顾决策质量

---

## 性能

- **图构建**: O(N) 时间复杂度（N = 事件数）
- **渲染性能**: Cytoscape.js 可处理 1000+ 节点
- **查询延迟**: < 100ms（小型 jobs）
- **大型 jobs**: 建议使用分页或过滤

---

## 下一步

- 查看 `docs/m3-forensics-api-guide.md` 了解查询 API
- 查看 `docs/m3-ui-guide.md` 了解 UI 操作
- 查看 `docs/2.0-milestones-overview.md` 了解完整能力
