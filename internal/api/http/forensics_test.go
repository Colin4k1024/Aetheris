package http

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/json"
	"testing"
	"time"

	"github.com/Colin4k1024/Aetheris/v2/internal/runtime/jobstore"
	"github.com/Colin4k1024/Aetheris/v2/pkg/proof"
)

func TestBuildForensicsPackage_ProofCompatible(t *testing.T) {
	ctx := context.Background()
	jobID := "job_forensics_export"
	store := jobstore.NewMemoryStore()

	_, ver, err := store.ListEvents(ctx, jobID)
	if err != nil {
		t.Fatalf("list events: %v", err)
	}

	appendEvent := func(ev jobstore.JobEvent) {
		newVer, err := store.Append(ctx, jobID, ver, ev)
		if err != nil {
			t.Fatalf("append %s: %v", ev.Type, err)
		}
		ver = newVer
	}

	appendEvent(jobstore.JobEvent{
		JobID: jobID,
		Type:  jobstore.JobCreated,
	})

	startedPayload, _ := json.Marshal(map[string]interface{}{
		"invocation_id":   "inv-1",
		"idempotency_key": "key-1",
		"tool_name":       "test_tool",
		"arguments_hash":  "hash-1",
		"started_at":      time.Now().UTC().Format(time.RFC3339),
	})
	appendEvent(jobstore.JobEvent{
		JobID:   jobID,
		Type:    jobstore.ToolInvocationStarted,
		Payload: startedPayload,
	})

	finishedPayload, _ := json.Marshal(map[string]interface{}{
		"invocation_id":   "inv-1",
		"idempotency_key": "key-1",
		"tool_name":       "test_tool",
		"outcome":         "success",
		"result": map[string]interface{}{
			"ok": true,
		},
		"finished_at": time.Now().UTC().Format(time.RFC3339),
	})
	appendEvent(jobstore.JobEvent{
		JobID:   jobID,
		Type:    jobstore.ToolInvocationFinished,
		Payload: finishedPayload,
	})

	h := NewHandler(nil, nil)
	h.SetJobEventStore(store)

	zipBytes, err := h.buildForensicsPackage(ctx, jobID)
	if err != nil {
		t.Fatalf("build forensics package: %v", err)
	}

	result := proof.VerifyEvidenceZip(zipBytes)
	if !result.OK {
		t.Fatalf("expected exported zip to be verifiable, errors: %v", result.Errors)
	}
	if !result.HashChainValid {
		t.Fatalf("expected hash chain valid")
	}
	if !result.ManifestValid {
		t.Fatalf("expected manifest valid")
	}
}

func TestBuildForensicsPackage_SignedProof(t *testing.T) {
	ctx := context.Background()
	jobID := "job_forensics_export_signed"
	store := jobstore.NewMemoryStore()
	_, ver, err := store.ListEvents(ctx, jobID)
	if err != nil {
		t.Fatalf("list events: %v", err)
	}
	_, err = store.Append(ctx, jobID, ver, jobstore.JobEvent{
		JobID: jobID,
		Type:  jobstore.JobCreated,
	})
	if err != nil {
		t.Fatalf("append job_created: %v", err)
	}

	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}

	h := NewHandler(nil, nil)
	h.SetJobEventStore(store)
	h.SetEvidenceSigning(EvidenceSigningConfig{
		Enabled:    true,
		KeyID:      "test-key",
		PrivateKey: privateKey,
	})

	zipBytes, err := h.buildForensicsPackage(ctx, jobID)
	if err != nil {
		t.Fatalf("build signed forensics package: %v", err)
	}

	result := proof.VerifyEvidenceZip(zipBytes, publicKey)
	if !result.OK {
		t.Fatalf("expected signed exported zip to be verifiable, errors: %v", result.Errors)
	}
	if !result.SignatureValid {
		t.Fatalf("expected signature valid")
	}
}

func TestBuildForensicsPackage_InvalidSigningKey(t *testing.T) {
	ctx := context.Background()
	jobID := "job_forensics_export_invalid_signing_key"
	store := jobstore.NewMemoryStore()
	_, err := store.Append(ctx, jobID, 0, jobstore.JobEvent{
		JobID: jobID,
		Type:  jobstore.JobCreated,
	})
	if err != nil {
		t.Fatalf("append job_created: %v", err)
	}

	h := NewHandler(nil, nil)
	h.SetJobEventStore(store)
	h.SetEvidenceSigning(EvidenceSigningConfig{
		Enabled:    true,
		KeyID:      "bad-key",
		PrivateKey: []byte("not-an-ed25519-private-key"),
	})

	if _, err := h.buildForensicsPackage(ctx, jobID); err == nil {
		t.Fatalf("expected invalid signing key error")
	}
}

func TestNormalizeProofEventHashes_RebuildsInvalidChain(t *testing.T) {
	events := []proof.Event{
		{
			ID:        "ev-1",
			JobID:     "job-hash-normalize",
			Type:      "job_created",
			Payload:   `{"goal":"hello"}`,
			CreatedAt: time.Date(2026, 2, 13, 9, 0, 0, 0, time.UTC),
			PrevHash:  "",
			Hash:      "invalid",
		},
		{
			ID:        "ev-2",
			JobID:     "job-hash-normalize",
			Type:      "job_completed",
			Payload:   `{"ok":true}`,
			CreatedAt: time.Date(2026, 2, 13, 9, 0, 1, 0, time.UTC),
			PrevHash:  "invalid",
			Hash:      "invalid",
		},
	}

	if err := proof.ValidateChain(events); err == nil {
		t.Fatalf("expected invalid chain before normalization")
	}

	normalized := normalizeProofEventHashes(events)
	if err := proof.ValidateChain(normalized); err != nil {
		t.Fatalf("expected normalized chain valid, got: %v", err)
	}
	if normalized[0].PrevHash != "" {
		t.Fatalf("first event prev_hash should be empty after normalization")
	}
	if normalized[1].PrevHash != normalized[0].Hash {
		t.Fatalf("second event prev_hash should point to first event hash")
	}
}
