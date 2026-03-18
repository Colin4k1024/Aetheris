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

package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/go-resty/resty/v2"
)

func apiBaseURL() string {
	if u := os.Getenv("AETHERIS_API_URL"); u != "" {
		return u
	}
	return "http://localhost:8080"
}

func tenantID() string {
	if cliTenantID != "" {
		return cliTenantID
	}
	if t := os.Getenv("AETHERIS_TENANT_ID"); t != "" {
		return t
	}
	return "default"
}

func newClient() *resty.Client {
	c := resty.New().
		SetBaseURL(apiBaseURL()).
		SetTimeout(30*time.Second).
		SetHeader("Content-Type", "application/json")
	if t := tenantID(); t != "" {
		c.SetHeader("X-Tenant-ID", t)
	}
	return c
}

func getJob(jobID string) (map[string]interface{}, error) {
	var out map[string]interface{}
	resp, err := newClient().R().
		SetResult(&out).
		Get("/api/jobs/" + jobID)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode() != http.StatusOK {
		return nil, fmt.Errorf("GET /api/jobs/%s: %s", jobID, resp.String())
	}
	return out, nil
}

func getJobTrace(jobID string) (map[string]interface{}, error) {
	var out map[string]interface{}
	resp, err := newClient().R().
		SetResult(&out).
		Get("/api/jobs/" + jobID + "/trace")
	if err != nil {
		return nil, err
	}
	if resp.StatusCode() != http.StatusOK {
		return nil, fmt.Errorf("GET trace: %s", resp.String())
	}
	return out, nil
}

func listAgentJobs(agentID string) ([]map[string]interface{}, error) {
	var out struct {
		Jobs []map[string]interface{} `json:"jobs"`
	}
	resp, err := newClient().R().
		SetResult(&out).
		Get("/api/agents/" + agentID + "/jobs")
	if err != nil {
		return nil, err
	}
	if resp.StatusCode() != http.StatusOK {
		return nil, fmt.Errorf("GET /api/agents/%s/jobs: %s", agentID, resp.String())
	}
	return out.Jobs, nil
}

func postMessage(agentID, message string) (jobID string, err error) {
	body := map[string]string{"message": message}
	var out struct {
		JobID string `json:"job_id"`
	}
	resp, err := newClient().R().
		SetBody(body).
		SetResult(&out).
		Post("/api/agents/" + agentID + "/message")
	if err != nil {
		return "", err
	}
	if resp.StatusCode() != http.StatusAccepted && resp.StatusCode() != http.StatusOK {
		return "", fmt.Errorf("POST message: %s", resp.String())
	}
	return out.JobID, nil
}

func tracePageURL(jobID string) string {
	return apiBaseURL() + "/api/jobs/" + jobID + "/trace/page"
}

func getJobEvents(jobID string) (map[string]interface{}, error) {
	var out map[string]interface{}
	resp, err := newClient().R().
		SetResult(&out).
		Get("/api/jobs/" + jobID + "/events")
	if err != nil {
		return nil, err
	}
	if resp.StatusCode() != http.StatusOK {
		return nil, fmt.Errorf("GET events: %s", resp.String())
	}
	return out, nil
}

func getJobVerify(jobID string) (map[string]interface{}, error) {
	var out map[string]interface{}
	resp, err := newClient().R().
		SetResult(&out).
		Get("/api/jobs/" + jobID + "/verify")
	if err != nil {
		return nil, err
	}
	if resp.StatusCode() != http.StatusOK {
		return nil, fmt.Errorf("GET verify: %s", resp.String())
	}
	return out, nil
}

func listWorkers() ([]string, error) {
	var out struct {
		Workers []string `json:"workers"`
	}
	resp, err := newClient().R().
		SetResult(&out).
		Get("/api/system/workers")
	if err != nil {
		return nil, err
	}
	if resp.StatusCode() != http.StatusOK {
		return nil, fmt.Errorf("GET /api/system/workers: %s", resp.String())
	}
	return out.Workers, nil
}

func getObservabilitySummary() (map[string]interface{}, error) {
	var out map[string]interface{}
	resp, err := newClient().R().
		SetResult(&out).
		Get("/api/observability/summary")
	if err != nil {
		return nil, err
	}
	if resp.StatusCode() != http.StatusOK {
		return nil, fmt.Errorf("GET /api/observability/summary: %s", resp.String())
	}
	return out, nil
}

func cancelJob(jobID string) (map[string]interface{}, error) {
	var out map[string]interface{}
	resp, err := newClient().R().
		SetResult(&out).
		Post("/api/jobs/" + jobID + "/stop")
	if err != nil {
		return nil, err
	}
	if resp.StatusCode() != http.StatusOK {
		return nil, fmt.Errorf("POST stop: %s", resp.String())
	}
	return out, nil
}

func pauseJob(jobID, reason string) (map[string]interface{}, error) {
	var out map[string]interface{}
	body := map[string]string{"reason": reason}
	resp, err := newClient().R().
		SetBody(body).
		SetResult(&out).
		Post("/api/jobs/" + jobID + "/pause")
	if err != nil {
		return nil, err
	}
	if resp.StatusCode() != http.StatusOK {
		return nil, fmt.Errorf("POST pause: %s", resp.String())
	}
	return out, nil
}

func resumeJob(jobID, correlationKey string) (map[string]interface{}, error) {
	var out map[string]interface{}
	body := map[string]string{"correlation_key": correlationKey}
	resp, err := newClient().R().
		SetBody(body).
		SetResult(&out).
		Post("/api/jobs/" + jobID + "/resume")
	if err != nil {
		return nil, err
	}
	if resp.StatusCode() != http.StatusOK {
		return nil, fmt.Errorf("POST resume: %s", resp.String())
	}
	return out, nil
}

func signalJob(jobID, correlationKey string) (map[string]interface{}, error) {
	var out map[string]interface{}
	body := map[string]interface{}{
		"correlation_key": correlationKey,
		"payload":         map[string]interface{}{},
	}
	resp, err := newClient().R().
		SetBody(body).
		SetResult(&out).
		Post("/api/jobs/" + jobID + "/signal")
	if err != nil {
		return nil, err
	}
	if resp.StatusCode() != http.StatusOK {
		return nil, fmt.Errorf("POST signal: %s", resp.String())
	}
	return out, nil
}

func prettyJSON(v interface{}) string {
	b, _ := json.MarshalIndent(v, "", "  ")
	return string(b)
}
