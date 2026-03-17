package main

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"rag-platform/pkg/proof"
)

type testJobStore struct {
	events []proof.Event
}

func (s testJobStore) ListEvents(ctx context.Context, jobID string) ([]proof.Event, error) {
	return s.events, nil
}

type testLedger struct {
	invocations []proof.ToolInvocation
}

func (l testLedger) ListToolInvocations(ctx context.Context, jobID string) ([]proof.ToolInvocation, error) {
	return l.invocations, nil
}

func makeProofEvents(jobID string, count int) []proof.Event {
	events := make([]proof.Event, 0, count)
	prevHash := ""

	for i := 0; i < count; i++ {
		e := proof.Event{
			ID:        strconv.Itoa(i + 1),
			JobID:     jobID,
			Type:      "test_event",
			Payload:   "{\"index\":" + strconv.Itoa(i) + "}",
			CreatedAt: time.Now().UTC().Add(time.Duration(i) * time.Second),
			PrevHash:  prevHash,
		}
		e.Hash = proof.ComputeEventHash(e)
		prevHash = e.Hash
		events = append(events, e)
	}

	return events
}

func TestVerifyEvidenceZip_Success(t *testing.T) {
	jobID := "job_cli_verify_success"
	zipBytes, err := proof.ExportEvidenceZip(
		context.Background(),
		jobID,
		testJobStore{events: makeProofEvents(jobID, 5)},
		testLedger{},
		proof.ExportOptions{RuntimeVersion: "test", SchemaVersion: "2.0"},
	)
	if err != nil {
		t.Fatalf("export evidence zip: %v", err)
	}

	tmpDir := t.TempDir()
	zipPath := filepath.Join(tmpDir, "ok.zip")
	if err := os.WriteFile(zipPath, zipBytes, 0644); err != nil {
		t.Fatalf("write zip: %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := verifyEvidenceZip(zipPath, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d, stderr=%s", code, stderr.String())
	}
	if !bytes.Contains(stdout.Bytes(), []byte("Verification PASSED")) {
		t.Fatalf("expected success output, got: %s", stdout.String())
	}
}

func TestVerifyEvidenceZip_Tampered(t *testing.T) {
	jobID := "job_cli_verify_tampered"
	zipBytes, err := proof.ExportEvidenceZip(
		context.Background(),
		jobID,
		testJobStore{events: makeProofEvents(jobID, 5)},
		testLedger{},
		proof.ExportOptions{RuntimeVersion: "test", SchemaVersion: "2.0"},
	)
	if err != nil {
		t.Fatalf("export evidence zip: %v", err)
	}

	// 篡改 ZIP 字节，触发验证失败。
	if len(zipBytes) > 0 {
		zipBytes[len(zipBytes)-1] ^= 0xFF
	}

	tmpDir := t.TempDir()
	zipPath := filepath.Join(tmpDir, "tampered.zip")
	if err := os.WriteFile(zipPath, zipBytes, 0644); err != nil {
		t.Fatalf("write zip: %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := verifyEvidenceZip(zipPath, &stdout, &stderr)
	if code == 0 {
		t.Fatalf("expected non-zero exit code, got %d", code)
	}
	if !bytes.Contains(stdout.Bytes(), []byte("Verification FAILED")) {
		t.Fatalf("expected failure output, got: %s", stdout.String())
	}
}

func TestBackfillHashesFile(t *testing.T) {
	tmpDir := t.TempDir()
	inputPath := filepath.Join(tmpDir, "events.ndjson")
	outputPath := filepath.Join(tmpDir, "events.out.ndjson")
	input := `{"id":"1","job_id":"job_1","type":"job_created","payload":{"goal":"g1"},"created_at":"2026-02-13T10:00:00Z"}
{"id":"2","job_id":"job_1","type":"job_completed","payload":{},"created_at":"2026-02-13T10:00:01Z"}
`
	if err := os.WriteFile(inputPath, []byte(input), 0644); err != nil {
		t.Fatalf("write input: %v", err)
	}

	n, err := backfillHashesFile(inputPath, outputPath)
	if err != nil {
		t.Fatalf("backfill hashes: %v", err)
	}
	if n != 2 {
		t.Fatalf("count = %d, want 2", n)
	}

	b, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("read output: %v", err)
	}
	lines := bytes.Split(bytes.TrimSpace(b), []byte("\n"))
	if len(lines) != 2 {
		t.Fatalf("lines = %d, want 2", len(lines))
	}
	var first map[string]interface{}
	var second map[string]interface{}
	if err := json.Unmarshal(lines[0], &first); err != nil {
		t.Fatalf("unmarshal first: %v", err)
	}
	if err := json.Unmarshal(lines[1], &second); err != nil {
		t.Fatalf("unmarshal second: %v", err)
	}
	firstHash, _ := first["hash"].(string)
	secondPrev, _ := second["prev_hash"].(string)
	if firstHash == "" {
		t.Fatal("first hash should not be empty")
	}
	if secondPrev != firstHash {
		t.Fatalf("second prev_hash = %q, want %q", secondPrev, firstHash)
	}
}

func TestParsePositiveInt(t *testing.T) {
	tests := []struct {
		input    string
		wantVal  int
		wantErr  bool
	}{
		{"1", 1, false},
		{"10", 10, false},
		{"100", 100, false},
		{"0", 0, true},
		{"-1", 0, true},
		{"abc", 0, true},
		{"", 0, true},
	}

	for _, tt := range tests {
		got, err := parsePositiveInt(tt.input)
		if (err != nil) != tt.wantErr {
			t.Errorf("parsePositiveInt(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			continue
		}
		if !tt.wantErr && got != tt.wantVal {
			t.Errorf("parsePositiveInt(%q) = %d, want %d", tt.input, got, tt.wantVal)
		}
	}
}

func TestParseRFC3339(t *testing.T) {
	tests := []struct {
		input    string
		wantOK   bool
	}{
		{"2026-01-15T10:00:00Z", true},
		{"2026-01-15T10:00:00.123Z", true},
		{"2026-01-15T10:00:00+00:00", true},
		{"invalid", false},
		{"", false},
	}

	for _, tt := range tests {
		_, err := parseRFC3339(tt.input)
		if (err == nil) != tt.wantOK {
			t.Errorf("parseRFC3339(%q) error = %v, wantOK %v", tt.input, err, tt.wantOK)
		}
	}
}

func TestEventMapToProofEvent(t *testing.T) {
	tests := []struct {
		name      string
		input     map[string]interface{}
		wantJobID string
		wantErr   bool
	}{
		{
			name: "valid",
			input: map[string]interface{}{
				"id":         "e1",
				"job_id":     "job-1",
				"type":       "test",
				"payload":    `{"key":"value"}`,
				"created_at": "2026-01-15T10:00:00Z",
			},
			wantJobID: "job-1",
			wantErr:   false,
		},
		{
			name: "missing job_id",
			input: map[string]interface{}{
				"id":   "e1",
				"type": "test",
			},
			wantJobID: "",
			wantErr:   true,
		},
		{
			name: "missing type",
			input: map[string]interface{}{
				"id":     "e1",
				"job_id": "job-1",
			},
			wantJobID: "",
			wantErr:   true,
		},
		{
			name: "nil payload",
			input: map[string]interface{}{
				"id":     "e1",
				"job_id": "job-1",
				"type":   "test",
				"payload": nil,
			},
			wantJobID: "job-1",
			wantErr:   false,
		},
		{
			name: "object payload",
			input: map[string]interface{}{
				"id":     "e1",
				"job_id": "job-1",
				"type":   "test",
				"payload": map[string]interface{}{"key": "value"},
			},
			wantJobID: "job-1",
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e, err := eventMapToProofEvent(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("eventMapToProofEvent() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && e.JobID != tt.wantJobID {
				t.Errorf("JobID = %q, want %q", e.JobID, tt.wantJobID)
			}
		})
	}
}

func TestBackfillHashesFile_Empty(t *testing.T) {
	tmpDir := t.TempDir()
	inputPath := filepath.Join(tmpDir, "empty.ndjson")
	outputPath := filepath.Join(tmpDir, "empty.out.ndjson")

	// Write empty file
	if err := os.WriteFile(inputPath, []byte(""), 0644); err != nil {
		t.Fatalf("write input: %v", err)
	}

	n, err := backfillHashesFile(inputPath, outputPath)
	if err != nil {
		t.Fatalf("backfill hashes: %v", err)
	}
	if n != 0 {
		t.Fatalf("count = %d, want 0", n)
	}
}

func TestBackfillHashesFile_InvalidLine(t *testing.T) {
	tmpDir := t.TempDir()
	inputPath := filepath.Join(tmpDir, "invalid.ndjson")
	outputPath := filepath.Join(tmpDir, "invalid.out.ndjson")

	input := `{"id":"1","job_id":"job_1","type":"test","payload":invalid}`
	if err := os.WriteFile(inputPath, []byte(input), 0644); err != nil {
		t.Fatalf("write input: %v", err)
	}

	_, err := backfillHashesFile(inputPath, outputPath)
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}
