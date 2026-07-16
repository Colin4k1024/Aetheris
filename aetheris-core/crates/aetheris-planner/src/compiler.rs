//! DAG compiler: converts a TaskGraph into an ExecutionPlan.

use std::collections::{HashMap, HashSet, VecDeque};

use crate::error::PlannerError;
use crate::plan::{ExecutionPlan, PlanNode, PlanNodeStatus};
use crate::types::{NodeType, TaskGraph};

/// Compiles a TaskGraph protobuf into an ExecutionPlan.
pub struct DagCompiler<'a> {
    graph: &'a TaskGraph,
}

impl<'a> DagCompiler<'a> {
    pub fn new(graph: &'a TaskGraph) -> Self {
        Self { graph }
    }

    /// Compile the graph into an execution plan.
    pub fn compile(&self) -> Result<ExecutionPlan, PlannerError> {
        if self.graph.nodes.is_empty() {
            return Err(PlannerError::EmptyGraph);
        }

        // Build adjacency lists
        let mut in_degree: HashMap<&str, usize> = HashMap::new();
        let mut out_edges: HashMap<&str, Vec<&str>> = HashMap::new();
        let mut node_map: HashMap<&str, &crate::types::TaskNode> = HashMap::new();

        for node in &self.graph.nodes {
            node_map.insert(node.id.as_str(), node);
            in_degree.entry(node.id.as_str()).or_insert(0);
            out_edges.entry(node.id.as_str()).or_default();
        }

        for edge in &self.graph.edges {
            let from = edge.from_node.as_str();
            let to = edge.to_node.as_str();

            if !node_map.contains_key(from) {
                return Err(PlannerError::InvalidEdge {
                    from: from.to_string(),
                    to: to.to_string(),
                });
            }
            if !node_map.contains_key(to) {
                return Err(PlannerError::InvalidEdge {
                    from: from.to_string(),
                    to: to.to_string(),
                });
            }

            out_edges.entry(from).or_default().push(to);
            *in_degree.entry(to).or_insert(0) += 1;
        }

        // Topological sort with BFS (Kahn's algorithm)
        let mut queue: VecDeque<&str> = VecDeque::new();
        for node in &self.graph.nodes {
            if in_degree.get(node.id.as_str()).copied().unwrap_or(0) == 0 {
                queue.push_back(node.id.as_str());
            }
        }

        let mut topo_order: Vec<&str> = Vec::new();
        let mut visited: HashSet<&str> = HashSet::new();

        while let Some(current) = queue.pop_front() {
            if visited.contains(current) {
                continue;
            }
            visited.insert(current);
            topo_order.push(current);

            if let Some(children) = out_edges.get(current) {
                for child in children {
                    let deg = in_degree.get_mut(*child).unwrap();
                    *deg -= 1;
                    if *deg == 0 {
                        queue.push_back(child);
                    }
                }
            }
        }

        // Check for cycles
        if topo_order.len() != self.graph.nodes.len() {
            let missing = self
                .graph
                .nodes
                .iter()
                .find(|n| !visited.contains(n.id.as_str()))
                .map(|n| n.id.clone())
                .unwrap_or_default();
            return Err(PlannerError::CycleDetected(missing));
        }

        // Compute parallel groups (BFS level)
        let mut parallel_groups: HashMap<&str, usize> = HashMap::new();
        let mut group_queue: VecDeque<(&str, usize)> = VecDeque::new();

        // Start from nodes with no dependencies
        for node_id in &topo_order {
            if in_degree.get(node_id).copied().unwrap_or(0) == 0 {
                group_queue.push_back((node_id, 0));
            }
        }

        // BFS to assign parallel groups
        let reverse_edges: HashMap<&str, Vec<&str>> = {
            let mut rev: HashMap<&str, Vec<&str>> = HashMap::new();
            for edge in &self.graph.edges {
                rev.entry(edge.to_node.as_str())
                    .or_default()
                    .push(edge.from_node.as_str());
            }
            rev
        };

        for (i, node_id) in topo_order.iter().enumerate() {
            // Group = max group of all predecessors + 1
            let group = if let Some(predecessors) = reverse_edges.get(node_id) {
                predecessors
                    .iter()
                    .filter_map(|p| parallel_groups.get(p))
                    .max()
                    .map(|g| g + 1)
                    .unwrap_or(0)
            } else {
                0
            };
            parallel_groups.insert(node_id, group);
        }

        // Build dependency map (from edges)
        let mut depends_on: HashMap<&str, Vec<String>> = HashMap::new();
        for edge in &self.graph.edges {
            depends_on
                .entry(edge.to_node.as_str())
                .or_default()
                .push(edge.from_node.clone());
        }

        // Build gates map (outgoing edges)
        let mut gates_map: HashMap<&str, Vec<String>> = HashMap::new();
        for edge in &self.graph.edges {
            gates_map
                .entry(edge.from_node.as_str())
                .or_default()
                .push(edge.to_node.clone());
        }

        // Build plan nodes
        let mut nodes: Vec<PlanNode> = topo_order
            .iter()
            .enumerate()
            .map(|(idx, node_id)| {
                let task_node = node_map[node_id];
                let node_type = match task_node.r#type() {
                    NodeType::ToolCall => "tool_call",
                    NodeType::LlmCall => "llm_call",
                    NodeType::Condition => "condition",
                    NodeType::Parallel => "parallel",
                    NodeType::Subgraph => "subgraph",
                    NodeType::HumanGate => "human_gate",
                    _ => "unknown",
                };

                PlanNode {
                    id: task_node.id.clone(),
                    name: task_node.name.clone(),
                    node_type: node_type.to_string(),
                    config: task_node.config.clone(),
                    depends_on: depends_on
                        .get(node_id)
                        .cloned()
                        .unwrap_or_default(),
                    gates: gates_map.get(node_id).cloned().unwrap_or_default(),
                    max_retries: task_node.max_retries.max(0) as u32,
                    timeout_ms: task_node.timeout_ms.max(0) as u64,
                    status: PlanNodeStatus::Pending,
                    attempt: 0,
                    output: None,
                    error: None,
                    topo_order: idx,
                    parallel_group: parallel_groups[node_id],
                }
            })
            .collect();

        // Entry node is the first in topological order if not specified
        let entry_node = if self.graph.entry_node.is_empty() {
            topo_order
                .first()
                .map(|s| s.to_string())
                .unwrap_or_default()
        } else {
            self.graph.entry_node.clone()
        };

        Ok(ExecutionPlan {
            graph_id: self.graph.graph_id.clone(),
            nodes,
            entry_node,
        })
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::compile;
    use aetheris_types::Edge;

    fn make_node(id: &str, name: &str, node_type: NodeType) -> aetheris_types::TaskNode {
        aetheris_types::TaskNode {
            id: id.to_string(),
            name: name.to_string(),
            r#type: node_type.into(),
            config: vec![],
            max_retries: 3,
            timeout_ms: 30000,
        }
    }

    fn make_edge(from: &str, to: &str) -> Edge {
        Edge {
            from_node: from.to_string(),
            to_node: to.to_string(),
            condition: String::new(),
        }
    }

    fn make_graph(nodes: Vec<aetheris_types::TaskNode>, edges: Vec<Edge>) -> TaskGraph {
        TaskGraph {
            graph_id: "test-graph".to_string(),
            nodes,
            edges,
            entry_node: String::new(),
            metadata: vec![],
        }
    }

    #[test]
    fn test_single_node() {
        let graph = make_graph(
            vec![make_node("n1", "start", NodeType::ToolCall)],
            vec![],
        );
        let plan = compile(&graph).unwrap();
        assert_eq!(plan.nodes.len(), 1);
        assert_eq!(plan.entry_node, "n1");
        assert!(plan.is_complete() || plan.ready_nodes().len() == 1);
    }

    #[test]
    fn test_linear_chain() {
        let graph = make_graph(
            vec![
                make_node("n1", "step1", NodeType::ToolCall),
                make_node("n2", "step2", NodeType::LlmCall),
                make_node("n3", "step3", NodeType::ToolCall),
            ],
            vec![make_edge("n1", "n2"), make_edge("n2", "n3")],
        );
        let plan = compile(&graph).unwrap();

        assert_eq!(plan.nodes.len(), 3);
        assert_eq!(plan.nodes[0].id, "n1");
        assert_eq!(plan.nodes[1].id, "n2");
        assert_eq!(plan.nodes[2].id, "n3");

        // Only n1 should be ready (no dependencies)
        assert_eq!(plan.ready_nodes().len(), 1);
        assert_eq!(plan.ready_nodes()[0].id, "n1");
    }

    #[test]
    fn test_parallel_nodes() {
        let graph = make_graph(
            vec![
                make_node("n1", "start", NodeType::ToolCall),
                make_node("n2a", "branch_a", NodeType::ToolCall),
                make_node("n2b", "branch_b", NodeType::ToolCall),
                make_node("n3", "merge", NodeType::ToolCall),
            ],
            vec![
                make_edge("n1", "n2a"),
                make_edge("n1", "n2b"),
                make_edge("n2a", "n3"),
                make_edge("n2b", "n3"),
            ],
        );
        let plan = compile(&graph).unwrap();

        // After n1 completes, both n2a and n2b should be ready
        let mut plan_mut = plan.clone();
        plan_mut.get_node_mut("n1").unwrap().complete(vec![]);
        let ready = plan_mut.ready_nodes();
        assert_eq!(ready.len(), 2);
        let ready_ids: Vec<&str> = ready.iter().map(|n| n.id.as_str()).collect();
        assert!(ready_ids.contains(&"n2a"));
        assert!(ready_ids.contains(&"n2b"));
    }

    #[test]
    fn test_cycle_detection() {
        let graph = make_graph(
            vec![
                make_node("n1", "a", NodeType::ToolCall),
                make_node("n2", "b", NodeType::ToolCall),
            ],
            vec![make_edge("n1", "n2"), make_edge("n2", "n1")],
        );
        let result = compile(&graph);
        assert!(matches!(result, Err(PlannerError::CycleDetected(_))));
    }

    #[test]
    fn test_empty_graph() {
        let graph = make_graph(vec![], vec![]);
        let result = compile(&graph);
        assert!(matches!(result, Err(PlannerError::EmptyGraph)));
    }
}
