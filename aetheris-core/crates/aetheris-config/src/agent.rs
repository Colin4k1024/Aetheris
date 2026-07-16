//! Agent configuration types.

use serde::{Deserialize, Serialize};

/// Configuration for an agent type.
#[derive(Clone, Debug, Serialize, Deserialize)]
pub struct AgentConfig {
    /// Unique agent identifier.
    pub agent_id: String,
    /// Human-readable name.
    #[serde(default)]
    pub name: String,
    /// Description.
    #[serde(default)]
    pub description: String,
    /// Agent capabilities (e.g., "tool_use", "planning", "memory").
    #[serde(default)]
    pub capabilities: Vec<String>,
    /// Tools bound to this agent.
    #[serde(default)]
    pub tool_bindings: Vec<ToolBinding>,
    /// Model configuration (JSON-encoded).
    #[serde(default)]
    pub model_config: serde_json::Value,
    /// Behavior configuration (JSON-encoded).
    #[serde(default)]
    pub behavior_config: serde_json::Value,
}

/// Binding of a tool to an agent.
#[derive(Clone, Debug, Serialize, Deserialize)]
pub struct ToolBinding {
    /// Tool name.
    pub tool_name: String,
    /// Tool type: "builtin", "mcp", "http".
    #[serde(default = "default_tool_type")]
    pub tool_type: String,
    /// Tool configuration (JSON-encoded).
    #[serde(default)]
    pub config: serde_json::Value,
}

fn default_tool_type() -> String {
    "builtin".to_string()
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_agent_config_deserialize() {
        let yaml = r#"
agent_id: test-agent
name: Test Agent
description: A test agent
capabilities:
  - tool_use
  - planning
tool_bindings:
  - tool_name: web_search
    tool_type: builtin
  - tool_name: code_runner
    tool_type: mcp
    config:
      endpoint: "http://localhost:3000"
"#;
        let config: AgentConfig = serde_yaml::from_str(yaml).unwrap();
        assert_eq!(config.agent_id, "test-agent");
        assert_eq!(config.capabilities.len(), 2);
        assert_eq!(config.tool_bindings.len(), 2);
        assert_eq!(config.tool_bindings[0].tool_type, "builtin");
        assert_eq!(config.tool_bindings[1].tool_type, "mcp");
    }
}
