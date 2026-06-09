package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/Colin4k1024/Aetheris/v2/internal/agent/planner"
	"github.com/Colin4k1024/Aetheris/v2/pkg/config"
)

const frameworkManifestSchemaV1 = "aetheris.framework.v1"

type FrameworkManifest struct {
	SchemaVersion string                  `json:"schema_version"`
	Name          string                  `json:"name"`
	Framework     string                  `json:"framework"`
	InputNode     string                  `json:"input_node,omitempty"`
	OutputNode    string                  `json:"output_node,omitempty"`
	Nodes         []FrameworkManifestNode `json:"nodes"`
	Edges         []FrameworkManifestEdge `json:"edges"`
}

type FrameworkManifestNode struct {
	ID       string         `json:"id"`
	Kind     string         `json:"kind"`
	ToolName string         `json:"tool_name,omitempty"`
	Workflow string         `json:"workflow,omitempty"`
	Callable string         `json:"callable,omitempty"`
	Config   map[string]any `json:"config,omitempty"`
}

type FrameworkManifestEdge struct {
	From string `json:"from"`
	To   string `json:"to"`
}

func LoadFrameworkManifestForAgent(ctx context.Context, agentID string, agent config.AgentDefConfig) (*FrameworkManifest, error) {
	var data []byte
	var err error
	switch {
	case strings.TrimSpace(agent.External.ManifestPath) != "":
		data, err = os.ReadFile(agent.External.ManifestPath)
	case strings.TrimSpace(agent.External.ManifestURL) != "":
		data, err = fetchFrameworkManifest(ctx, agent.External.ManifestURL, agent.External.TokenEnv)
	default:
		if strings.TrimSpace(agent.External.URL) == "" {
			return nil, fmt.Errorf("agents.%s.external.url, manifest_path, or manifest_url is required when mode=embedded", agentID)
		}
		manifestURL, buildErr := joinFrameworkManifestURL(agent.External.URL)
		if buildErr != nil {
			return nil, fmt.Errorf("agents.%s.external.url is invalid: %w", agentID, buildErr)
		}
		data, err = fetchFrameworkManifest(ctx, manifestURL, agent.External.TokenEnv)
	}
	if err != nil {
		return nil, fmt.Errorf("load framework manifest for %s: %w", agentID, err)
	}
	var manifest FrameworkManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return nil, fmt.Errorf("decode framework manifest for %s: %w", agentID, err)
	}
	if err := ValidateFrameworkManifest(agentID, agent, &manifest); err != nil {
		return nil, err
	}
	return &manifest, nil
}

func fetchFrameworkManifest(ctx context.Context, rawURL, tokenEnv string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, err
	}
	if tokenEnv != "" {
		token := os.Getenv(tokenEnv)
		if token == "" {
			return nil, fmt.Errorf("token env %q is not set", tokenEnv)
		}
		req.Header.Set("Authorization", "Bearer "+token)
	}
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("manifest endpoint returned HTTP %d", resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxExternalResponseBytes+1))
	if err != nil {
		return nil, err
	}
	if int64(len(body)) > maxExternalResponseBytes {
		return nil, fmt.Errorf("manifest response exceeds %d MiB limit", maxExternalResponseBytes/(1024*1024))
	}
	return body, nil
}

func joinFrameworkManifestURL(base string) (string, error) {
	u, err := url.Parse(base)
	if err != nil {
		return "", err
	}
	u.Path = strings.TrimRight(u.Path, "/") + "/aetheris/manifest"
	u.RawQuery = ""
	u.Fragment = ""
	return u.String(), nil
}

func ValidateFrameworkManifest(agentID string, agent config.AgentDefConfig, manifest *FrameworkManifest) error {
	if manifest == nil {
		return fmt.Errorf("agents.%s.framework_manifest is nil", agentID)
	}
	if manifest.SchemaVersion != frameworkManifestSchemaV1 {
		return fmt.Errorf("agents.%s.framework_manifest.schema_version must be %q", agentID, frameworkManifestSchemaV1)
	}
	if strings.TrimSpace(manifest.Name) == "" {
		return fmt.Errorf("agents.%s.framework_manifest.name is required", agentID)
	}
	if strings.TrimSpace(manifest.Framework) == "" {
		return fmt.Errorf("agents.%s.framework_manifest.framework is required", agentID)
	}
	if len(manifest.Nodes) == 0 {
		return fmt.Errorf("agents.%s.framework_manifest.nodes is required", agentID)
	}
	ids := make(map[string]FrameworkManifestNode, len(manifest.Nodes))
	for _, node := range manifest.Nodes {
		if strings.TrimSpace(node.ID) == "" {
			return fmt.Errorf("agents.%s.framework_manifest node id is required", agentID)
		}
		if strings.ContainsAny(node.ID, "/?#") {
			return fmt.Errorf("agents.%s.framework_manifest node id %q must not contain '/', '?', or '#'", agentID, node.ID)
		}
		if _, exists := ids[node.ID]; exists {
			return fmt.Errorf("agents.%s.framework_manifest duplicate node id %q", agentID, node.ID)
		}
		switch node.Kind {
		case "runtime_tool":
			if strings.TrimSpace(node.ToolName) == "" {
				return fmt.Errorf("agents.%s.framework_manifest node %s requires tool_name", agentID, node.ID)
			}
		case "runtime_llm", "wait", "approval":
			// valid
		case "runtime_workflow":
			if strings.TrimSpace(node.Workflow) == "" {
				return fmt.Errorf("agents.%s.framework_manifest node %s requires workflow", agentID, node.ID)
			}
		case "remote_callable":
			if strings.TrimSpace(node.Callable) == "" {
				return fmt.Errorf("agents.%s.framework_manifest node %s requires callable", agentID, node.ID)
			}
			if strings.TrimSpace(agent.External.URL) == "" {
				return fmt.Errorf("agents.%s.external.url is required for remote_callable node %s", agentID, node.ID)
			}
		default:
			return fmt.Errorf("agents.%s.framework_manifest node %s has unsupported kind %q", agentID, node.ID, node.Kind)
		}
		ids[node.ID] = node
	}
	if manifest.InputNode != "" {
		if _, ok := ids[manifest.InputNode]; !ok {
			return fmt.Errorf("agents.%s.framework_manifest input_node %q is not declared in nodes", agentID, manifest.InputNode)
		}
	}
	if manifest.OutputNode != "" {
		if _, ok := ids[manifest.OutputNode]; !ok {
			return fmt.Errorf("agents.%s.framework_manifest output_node %q is not declared in nodes", agentID, manifest.OutputNode)
		}
	}
	for _, edge := range manifest.Edges {
		if _, ok := ids[edge.From]; !ok {
			return fmt.Errorf("agents.%s.framework_manifest edge has unknown from node %q", agentID, edge.From)
		}
		if _, ok := ids[edge.To]; !ok {
			return fmt.Errorf("agents.%s.framework_manifest edge has unknown to node %q", agentID, edge.To)
		}
	}
	return nil
}

func FrameworkManifestToTaskGraph(agentID string, agent config.AgentDefConfig, manifest *FrameworkManifest) (*planner.TaskGraph, error) {
	if err := ValidateFrameworkManifest(agentID, agent, manifest); err != nil {
		return nil, err
	}
	nodes := make([]planner.TaskNode, 0, len(manifest.Nodes))
	for _, node := range manifest.Nodes {
		cfg := cloneMap(node.Config)
		cfg["framework"] = manifest.Framework
		cfg["framework_agent_id"] = agentID
		cfg["framework_node_id"] = node.ID
		switch node.Kind {
		case "runtime_tool":
			nodes = append(nodes, planner.TaskNode{ID: node.ID, Type: planner.NodeTool, ToolName: node.ToolName, Config: cfg})
		case "runtime_llm":
			nodes = append(nodes, planner.TaskNode{ID: node.ID, Type: planner.NodeLLM, Config: cfg})
		case "runtime_workflow":
			nodes = append(nodes, planner.TaskNode{ID: node.ID, Type: planner.NodeWorkflow, Workflow: node.Workflow, Config: cfg})
		case "wait":
			nodes = append(nodes, planner.TaskNode{ID: node.ID, Type: planner.NodeWait, Config: cfg})
		case "approval":
			nodes = append(nodes, planner.TaskNode{ID: node.ID, Type: planner.NodeApproval, Config: cfg})
		case "remote_callable":
			cfg["callable"] = node.Callable
			cfg["url"] = agent.External.URL
			cfg["token_env"] = agent.External.TokenEnv
			nodes = append(nodes, planner.TaskNode{ID: node.ID, Type: planner.NodeFrameworkCallable, Config: cfg})
		default:
			return nil, fmt.Errorf("unsupported framework node kind %q", node.Kind)
		}
	}
	edges := make([]planner.TaskEdge, 0, len(manifest.Edges))
	for _, edge := range manifest.Edges {
		edges = append(edges, planner.TaskEdge{From: edge.From, To: edge.To})
	}
	return &planner.TaskGraph{Nodes: nodes, Edges: edges}, nil
}

func cloneMap(in map[string]any) map[string]any {
	out := make(map[string]any, len(in)+4)
	for k, v := range in {
		out[k] = v
	}
	return out
}
