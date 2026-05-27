// Package main implements a minimal OpenAI-compatible mock LLM server for CI testing.
//
// It handles POST /v1/chat/completions and GET /v1/models requests, returning valid
// OpenAI-format responses immediately without calling any real model. This lets
// performance gates measure pure Aetheris runtime throughput without requiring
// an Ollama or OpenAI instance in the CI environment.
//
// Usage: llm-mock listens on :11434 by default.
// Override: LLM_MOCK_ADDR=:8899 llm-mock
package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"
)

func main() {
	addr := os.Getenv("LLM_MOCK_ADDR")
	if addr == "" {
		addr = ":11434"
	}

	mux := http.NewServeMux()
	mux.HandleFunc("POST /v1/chat/completions", handleCompletions)
	mux.HandleFunc("GET /v1/models", handleModels)
	mux.HandleFunc("GET /health", handleHealth)

	log.Printf("[llm-mock] CI mock LLM server listening on %s", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("[llm-mock] server error: %v", err)
	}
}

// handleCompletions handles POST /v1/chat/completions.
// It discards the request body and returns a minimal valid OpenAI completion.
func handleCompletions(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	io.Copy(io.Discard, r.Body) //nolint:errcheck

	resp := map[string]any{
		"id":      "chatcmpl-ci",
		"object":  "chat.completion",
		"created": time.Now().Unix(),
		"model":   "ci-mock",
		"choices": []map[string]any{{
			"index": 0,
			"message": map[string]any{
				"role":    "assistant",
				"content": "CI mock: task acknowledged.",
			},
			"finish_reason": "stop",
		}},
		"usage": map[string]any{
			"prompt_tokens":     10,
			"completion_tokens": 6,
			"total_tokens":      16,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp) //nolint:errcheck
}

// handleModels handles GET /v1/models (used for health checks and model listing).
func handleModels(w http.ResponseWriter, _ *http.Request) {
	resp := map[string]any{
		"object": "list",
		"data": []map[string]any{
			{"id": "ci-mock", "object": "model", "owned_by": "ci"},
		},
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp) //nolint:errcheck
}

// handleHealth handles GET /health (Docker healthcheck probe).
func handleHealth(w http.ResponseWriter, _ *http.Request) {
	fmt.Fprint(w, "ok")
}
