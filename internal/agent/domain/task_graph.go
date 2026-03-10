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

package domain

import (
	"errors"
	"time"
)

// NodeExecutionState 节点执行状态
type NodeExecutionState string

const (
	NodeExecutionStatePending   NodeExecutionState = "pending"   // 待执行
	NodeExecutionStateReady     NodeExecutionState = "ready"     // 就绪（依赖已满足）
	NodeExecutionStateRunning   NodeExecutionState = "running"   // 执行中
	NodeExecutionStateCompleted NodeExecutionState = "completed" // 已完成
	NodeExecutionStateFailed    NodeExecutionState = "failed"    // 执行失败
	NodeExecutionStateSkipped   NodeExecutionState = "skipped"   // 已跳过（因依赖失败）
)

// GraphNode 任务图中的节点（含执行状态）
type GraphNode struct {
	ID       string                 `json:"id"`     // 节点 ID
	Type     string                 `json:"type"`   // 节点类型
	Config   map[string]interface{} `json:"config"` // 配置
	ToolName string                 `json:"tool_name,omitempty"`
	Workflow string                 `json:"workflow,omitempty"`

	// 执行状态
	State      NodeExecutionState     `json:"state"`       // 执行状态
	Output     map[string]interface{} `json:"output"`      // 执行输出
	Error      string                 `json:"error"`       // 错误信息
	StartedAt  int64                  `json:"started_at"`  // 开始时间戳
	FinishedAt int64                  `json:"finished_at"` // 结束时间戳

	// 依赖关系
	DependsOn []string `json:"depends_on"` // 依赖的节点 ID 列表
}

// GraphEdge 任务图中的边
type GraphEdge struct {
	From string `json:"from"`
	To   string `json:"to"`
}

// TaskGraphExecution TaskGraph 执行上下文聚合根 - 负责 DAG 执行状态管理
type TaskGraphExecution struct {
	// 原始任务图数据
	Nodes []GraphNode `json:"nodes"`
	Edges []GraphEdge `json:"edges"`

	// 执行状态
	completedNodes map[string]struct{} // 已完成的节点集合
	readyQueue     []string            // 就绪队列（依赖已满足的节点）

	// 节点索引
	nodeIndex map[string]*GraphNode
}

// NewTaskGraphExecution 创建 TaskGraphExecution
func NewTaskGraphExecution() *TaskGraphExecution {
	return &TaskGraphExecution{
		Nodes:          []GraphNode{},
		Edges:          []GraphEdge{},
		completedNodes: make(map[string]struct{}),
		readyQueue:     []string{},
		nodeIndex:      make(map[string]*GraphNode),
	}
}

// AddNode 添加节点到任务图
func (g *TaskGraphExecution) AddNode(node GraphNode) {
	g.Nodes = append(g.Nodes, node)
	g.nodeIndex[node.ID] = &g.Nodes[len(g.Nodes)-1]
}

// AddEdge 添加边到任务图
func (g *TaskGraphExecution) AddEdge(edge GraphEdge) {
	g.Edges = append(g.Edges, edge)
}

// BuildIndex 构建节点索引和依赖关系
func (g *TaskGraphExecution) BuildIndex() error {
	g.nodeIndex = make(map[string]*GraphNode)
	for i := range g.Nodes {
		g.nodeIndex[g.Nodes[i].ID] = &g.Nodes[i]
	}

	// 构建依赖关系（反向：每个节点依赖哪些节点）
	dependencies := make(map[string]map[string]struct{})
	for _, edge := range g.Edges {
		if dependencies[edge.To] == nil {
			dependencies[edge.To] = make(map[string]struct{})
		}
		dependencies[edge.To][edge.From] = struct{}{}
	}

	// 设置每个节点的依赖列表
	for i := range g.Nodes {
		nodeID := g.Nodes[i].ID
		if deps, ok := dependencies[nodeID]; ok {
			g.Nodes[i].DependsOn = make([]string, 0, len(deps))
			for depID := range deps {
				g.Nodes[i].DependsOn = append(g.Nodes[i].DependsOn, depID)
			}
		}
	}

	// 初始化就绪队列（找出没有依赖的节点）
	g.readyQueue = []string{}
	for i := range g.Nodes {
		if len(g.Nodes[i].DependsOn) == 0 {
			g.readyQueue = append(g.readyQueue, g.Nodes[i].ID)
			g.Nodes[i].State = NodeExecutionStateReady
		} else {
			// 有依赖的节点初始为 Pending
			g.Nodes[i].State = NodeExecutionStatePending
		}
	}

	return nil
}

// TopologicalSort 执行拓扑排序，返回节点的执行顺序
func (g *TaskGraphExecution) TopologicalSort() ([]string, error) {
	if err := g.BuildIndex(); err != nil {
		return nil, err
	}

	// Kahn's algorithm
	inDegree := make(map[string]int)
	for _, node := range g.Nodes {
		inDegree[node.ID] = 0
	}
	for _, edge := range g.Edges {
		inDegree[edge.To]++
	}

	queue := []string{}
	for id, degree := range inDegree {
		if degree == 0 {
			queue = append(queue, id)
		}
	}

	result := []string{}
	for len(queue) > 0 {
		// 按添加顺序处理（保持确定性）
		current := queue[0]
		queue = queue[1:]
		result = append(result, current)

		for _, edge := range g.Edges {
			if edge.From == current {
				inDegree[edge.To]--
				if inDegree[edge.To] == 0 {
					queue = append(queue, edge.To)
				}
			}
		}
	}

	// 检查是否有环
	if len(result) != len(g.Nodes) {
		return nil, errors.New("cycle detected in task graph")
	}

	return result, nil
}

// GetReadyNodes 获取当前就绪的节点（依赖已满足）
func (g *TaskGraphExecution) GetReadyNodes() []string {
	var result []string
	for _, nodeID := range g.readyQueue {
		node := g.nodeIndex[nodeID]
		if node != nil && node.State == NodeExecutionStateReady {
			result = append(result, nodeID)
		}
	}
	return result
}

// MarkNodeRunning 标记节点为运行中
func (g *TaskGraphExecution) MarkNodeRunning(nodeID string) error {
	node, ok := g.nodeIndex[nodeID]
	if !ok {
		return errors.New("node not found")
	}
	if node.State != NodeExecutionStateReady {
		return errors.New("node not ready")
	}
	node.State = NodeExecutionStateRunning
	node.StartedAt = now()
	return nil
}

// MarkNodeCompleted 标记节点为已完成
func (g *TaskGraphExecution) MarkNodeCompleted(nodeID string, output map[string]interface{}) error {
	node, ok := g.nodeIndex[nodeID]
	if !ok {
		return errors.New("node not found")
	}
	node.State = NodeExecutionStateCompleted
	node.FinishedAt = now()
	node.Output = output
	g.completedNodes[nodeID] = struct{}{}

	// 更新依赖该节点的节点状态
	g.updateDependentNodes(nodeID)
	return nil
}

// MarkNodeFailed 标记节点为失败
func (g *TaskGraphExecution) MarkNodeFailed(nodeID, errMsg string) error {
	node, ok := g.nodeIndex[nodeID]
	if !ok {
		return errors.New("node not found")
	}
	node.State = NodeExecutionStateFailed
	node.FinishedAt = now()
	node.Error = errMsg

	// 标记依赖该节点的所有节点为 skipped
	g.skipDependentNodes(nodeID)
	return nil
}

// updateDependentNodes 更新依赖已完成的节点状态
func (g *TaskGraphExecution) updateDependentNodes(completedNodeID string) {
	for _, edge := range g.Edges {
		if edge.From == completedNodeID {
			targetNode := g.nodeIndex[edge.To]
			if targetNode != nil && targetNode.State == NodeExecutionStatePending {
				// 检查所有依赖是否已满足
				if g.areAllDependenciesMet(edge.To) {
					targetNode.State = NodeExecutionStateReady
					g.readyQueue = append(g.readyQueue, edge.To)
				}
			}
		}
	}
}

// skipDependentNodes 跳过依赖失败节点的节点
func (g *TaskGraphExecution) skipDependentNodes(failedNodeID string) {
	for _, edge := range g.Edges {
		if edge.From == failedNodeID {
			targetNode := g.nodeNodeIndex(edge.To)
			if targetNode != nil && targetNode.State == NodeExecutionStatePending {
				targetNode.State = NodeExecutionStateSkipped
				// 递归跳过依赖该节点的节点
				g.skipDependentNodes(edge.To)
			}
		}
	}
}

// nodeNodeIndex 获取节点（不创建索引）
func (g *TaskGraphExecution) nodeNodeIndex(nodeID string) *GraphNode {
	for i := range g.Nodes {
		if g.Nodes[i].ID == nodeID {
			return &g.Nodes[i]
		}
	}
	return nil
}

// areAllDependenciesMet 检查节点的所有依赖是否已满足
func (g *TaskGraphExecution) areAllDependenciesMet(nodeID string) bool {
	node := g.nodeIndex[nodeID]
	if node == nil {
		return false
	}
	for _, depID := range node.DependsOn {
		if _, ok := g.completedNodes[depID]; !ok {
			return false
		}
	}
	return true
}

// IsCompleted 检查任务图是否已完成
func (g *TaskGraphExecution) IsCompleted() bool {
	return len(g.completedNodes) == len(g.Nodes)
}

// GetProgress 获取执行进度
func (g *TaskGraphExecution) GetProgress() (completed, total, failed, skipped int) {
	total = len(g.Nodes)
	for _, node := range g.Nodes {
		switch node.State {
		case NodeExecutionStateCompleted:
			completed++
		case NodeExecutionStateFailed:
			failed++
		case NodeExecutionStateSkipped:
			skipped++
		}
	}
	return
}

// GetNode 获取节点
func (g *TaskGraphExecution) GetNode(nodeID string) *GraphNode {
	return g.nodeIndex[nodeID]
}

// now 返回当前时间戳
func now() int64 {
	return time.Now().UnixNano()
}
