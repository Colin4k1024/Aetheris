# Compliance Evidence System Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Build a production-ready Compliance Evidence System with signed evidence packages, policy-driven automated collection, Forensics API, semantic verification, and evidence lifecycle management.

**Architecture:** The system builds on the existing `pkg/proof/` package which already provides ZIP-based evidence export with hash chain validation. New work adds: (1) Ed25519 digital signatures on evidence packages, (2) a compliance policy engine that auto-triggers exports based on job characteristics, (3) a Forensics API for querying evidence, (4) semantic verification to detect "claimed success but actually failed" fraud, and (5) evidence retention/lifecycle management. All evidence operations are scoped to the current tenant.

**Tech Stack:** Go 1.25, Ed25519 (golang.org/x/crypto/ed25519), archive/zip, PostgreSQL (existing jobstore), Viper for config.

---

## Task 1: Add Digital Signatures to Evidence Packages

**Files:**

- Modify: `pkg/proof/types.go:46-53` — add Signature fields to ProofSummary
- Modify: `pkg/proof/export.go:27-154` — sign evidence package during export
- Create: `pkg/proof/signature.go` — Ed25519 signing and verification logic
- Create: `pkg/proof/signature_test.go`
- Modify: `pkg/proof/export_test.go` — add signature test
- Modify: `pkg/proof/verify.go:26-152` — verify signature during verification

**Step 1: Add signature types**

Modify `pkg/proof/types.go` to extend `ProofSummary` and add signing configuration:

```go
// SigningConfig contains Ed25519 private key for signing evidence packages.
type SigningConfig struct {
    PrivateKey ed25519.PrivateKey // loaded from config/env
}

// SignedProof extends ProofSummary with cryptographic signature.
type SignedProof struct {
    ProofSummary
    Signature     string `json:"signature,omitempty"`       // Base64-encoded Ed25519 signature
    SignedAt      string `json:"signed_at,omitempty"`       // ISO 8601 timestamp
    SignerKeyID  string `json:"signer_key_id,omitempty"` // Identifier for the signing key
}
```

**Step 2: Create signature.go**

```go
package proof

import (
    "crypto/ed25519"
    "encoding/base64"
    "fmt"
    "time"

    "golang.org/x/crypto/ed25519"
)

// SignEvidence signs the evidence package contents and returns a signature.
func SignEvidence(privateKey ed25519.PrivateKey, manifest Manifest, events []Event, ledger []ToolInvocation) (string, error) {
    // Build canonical signing payload: manifest JSON + events NDJSON + ledger NDJSON
    manifestBytes, _ := json.Marshal(manifest)
    eventsBytes, _ := eventsToNDJSON(events)
    ledgerBytes, _ := ledgerToNDJSON(ledger)

    payload := append(manifestBytes, eventsBytes...)
    payload = append(payload, ledgerBytes...)

    sig := ed25519.Sign(privateKey, payload)
    return base64.StdEncoding.EncodeToString(sig), nil
}

// VerifySignature verifies an Ed25519 signature against the evidence package contents.
func VerifySignature(publicKey ed25519.PublicKey, manifest Manifest, events []Event, ledger []ToolInvocation, signature string) error {
    sigBytes, err := base64.StdEncoding.DecodeString(signature)
    if err != nil {
        return fmt.Errorf("invalid signature encoding: %w", err)
    }

    manifestBytes, _ := json.Marshal(manifest)
    eventsBytes, _ := eventsToNDJSON(events)
    ledgerBytes, _ := ledgerToNDJSON(ledger)

    payload := append(manifestBytes, eventsBytes...)
    payload = append(payload, ledgerBytes...)

    if !ed25519.Verify(publicKey, payload, sigBytes) {
        return fmt.Errorf("signature verification failed")
    }
    return nil
}
```

**Step 3: Modify ExportEvidenceZip to sign**

In `pkg/proof/export.go`, after building proofJSON, add:

```go
var proofBytes []byte
if opts.SigningConfig != nil && len(opts.SigningConfig.PrivateKey) > 0 {
    sig, err := SignEvidence(opts.SigningConfig.PrivateKey, manifest, events, toolInvocations)
    if err != nil {
        return nil, fmt.Errorf("failed to sign evidence: %w", err)
    }
    signedProof := SignedProof{
        ProofSummary: proofSummary,
        Signature:    sig,
        SignedAt:    time.Now().UTC().Format(time.RFC3339),
        SignerKeyID: opts.SigningConfig.KeyID,
    }
    proofBytes, _ = json.MarshalIndent(signedProof, "", "  ")
} else {
    proofBytes = proofJSON
}
fileHashes["proof.json"] = ComputeFileHash(proofBytes)
```

Add to `ExportOptions`:

```go
type ExportOptions struct {
    // ... existing fields ...
    SigningConfig *SigningConfig
}
```

**Step 4: VerifyEvidenceZip with signature**

In `pkg/proof/verify.go`, after parsing proof.json, add signature verification if present. If `SignedProof` is detected (has Signature field), verify against the same canonical payload.

**Step 5: Add tests**

```go
func TestSignAndVerifyEvidence(t *testing.T) {
    publicKey, privateKey, err := ed25519.GenerateKey(nil)
    if err != nil {
        t.Fatalf("generate key: %v", err)
    }

    events := []Event{{ID: "1", JobID: "job_test", Type: "job_created", Hash: "abc"}}
    ledger := []ToolInvocation{{ID: "inv1", JobID: "job_test", IdempotencyKey: "key1"}}
    manifest := Manifest{JobID: "job_test", Version: "2.0"}

    sig, err := SignEvidence(privateKey, manifest, events, ledger)
    if err != nil {
        t.Fatalf("sign: %v", err)
    }

    err = VerifySignature(publicKey, manifest, events, ledger, sig)
    if err != nil {
        t.Fatalf("verify: %v", err)
    }
}
```

**Step 6: Run tests**

```bash
go test ./pkg/proof/... -v -run TestSignAndVerifyEvidence
```

**Step 7: Commit**

```bash
git add pkg/proof/signature.go pkg/proof/types.go pkg/proof/export.go pkg/proof/verify.go pkg/proof/signature_test.go
git commit -m "feat(proof): add Ed25519 digital signatures on evidence packages"
```

---

## Task 2: Integrate Evidence Export into REST API (already exists, verify + add signing config)

**Files:**

- Modify: `internal/api/http/forensics.go:32-77` — wire signing config into evidence export
- Modify: `configs/api.yaml` — add signing key configuration section

**Step 1: Add signing key to handler**

In `internal/api/http/handler.go`, check if a signing key is configured and pass it to `buildForensicsPackage`. Read the key from environment or config file.

**Step 2: Add config for signing keys**

```bash
# Add to configs/api.yaml
evidence:
  signing:
    enabled: true
    key_file: "/path/to/ed25519_private_key.pem"  # or env: ED25519_PRIVATE_KEY
    key_id: "default-signer"
```

**Step 3: Run build to verify compilation**

```bash
go build ./cmd/api
```

**Step 4: Commit**

```bash
git add internal/api/http/forensics.go configs/api.yaml
git commit -m "feat(api): wire Ed25519 signing config into evidence export"
```

---

## Task 3: Add SHA256 Hash for All Files in ZIP

**Files:**

- Modify: `pkg/proof/export.go:79-84` — compute hashes for all files, not just 3
- Modify: `pkg/proof/export_test.go` — verify all file hashes in manifest

**Step 1: Modify export.go**

Currently only events.ndjson, ledger.ndjson, and metadata.json are hashed. Add proof.json and manifest.json:

```go
fileHashes := map[string]string{
    "manifest.json":  ComputeFileHash(manifestJSON),
    "events.ndjson":   ComputeFileHash(eventsNDJSON),
    "ledger.ndjson":  ComputeFileHash(ledgerNDJSON),
    "proof.json":     ComputeFileHash(proofJSON),
    "metadata.json":  ComputeFileHash(metadataJSON),
}
```

**Step 2: Verify the change**

```bash
go test ./pkg/proof/... -v -run TestExport
```

**Step 3: Commit**

```bash
git add pkg/proof/export.go
git commit -m "feat(proof): include SHA256 hash for all files in evidence ZIP manifest"
```

---

## Task 4: Evidence Package Merge for Multi-Job Compliance Reports

**Files:**

- Create: `pkg/proof/merge.go` — merge multiple evidence packages into one
- Create: `pkg/proof/merge_test.go`

**Step 1: Design merge structure**

A merged evidence package is a ZIP containing multiple `job_XXX.zip` entries plus a `manifest.json` that lists all included jobs and their root hashes.

```go
// MergedManifest extends the per-job Manifest to cover multiple jobs.
type MergedManifest struct {
    Version         string            `json:"version"`
    MergedAt        time.Time         `json:"merged_at"`
    JobCount        int               `json:"job_count"`
    Jobs            []MergedJobEntry  `json:"jobs"`
    TotalEventCount int               `json:"total_event_count"`
    OverallRootHash string            `json:"overall_root_hash"` // Hash of all job root hashes
}

type MergedJobEntry struct {
    JobID          string `json:"job_id"`
    RootHash       string `json:"root_hash"`
    EventCount     int    `json:"event_count"`
    LedgerCount    int    `json:"ledger_count"`
    Filename       string `json:"filename"` // inside ZIP
}
```

**Step 2: Implement merge.go**

```go
package proof

import (
    "archive/zip"
    "bytes"
    "crypto/sha256"
    "encoding/hex"
    "encoding/json"
    "fmt"
    "sort"
    "time"
)

// MergeEvidencePackages merges multiple evidence ZIP bytes into a single ZIP.
func MergeEvidencePackages(packages [][]byte, opts ExportOptions) ([]byte, error) {
    if len(packages) == 0 {
        return nil, fmt.Errorf("no packages to merge")
    }

    var mergedJobs []MergedJobEntry
    totalEvents := 0
    var rootHashes []string

    buf := new(bytes.Buffer)
    zw := zip.NewWriter(buf)

    for i, pkgBytes := range packages {
        entries, err := readZipEntries(bytes.NewReader(pkgBytes))
        if err != nil {
            return nil, fmt.Errorf("package %d: %w", i, err)
        }

        jobFilename := fmt.Sprintf("job_%03d.zip", i+1)
        w, err := zw.Create(jobFilename)
        if err != nil {
            return nil, fmt.Errorf("create entry %s: %w", jobFilename, err)
        }
        if _, err := w.Write(pkgBytes); err != nil {
            return nil, fmt.Errorf("write entry %s: %w", jobFilename, err)
        }

        // Extract metadata from manifest
        if manifestData, ok := entries["manifest.json"]; ok {
            var m Manifest
            if err := json.Unmarshal(manifestData, &m); err == nil {
                mergedJobs = append(mergedJobs, MergedJobEntry{
                    JobID:       m.JobID,
                    RootHash:    m.LastEventHash,
                    EventCount:  m.EventCount,
                    LedgerCount: m.LedgerCount,
                    Filename:    jobFilename,
                })
                rootHashes = append(rootHashes, m.LastEventHash)
                totalEvents += m.EventCount
            }
        }
    }

    // Sort by JobID for deterministic output
    sort.Slice(mergedJobs, func(i, j int) bool {
        return mergedJobs[i].JobID < mergedJobs[j].JobID
    })

    // Compute overall root hash
    overallRootHash := computeOverallRootHash(rootHashes)

    mergedManifest := MergedManifest{
        Version:         "2.0",
        MergedAt:        time.Now().UTC(),
        JobCount:        len(mergedJobs),
        Jobs:            mergedJobs,
        TotalEventCount: totalEvents,
        OverallRootHash: overallRootHash,
    }

    mergedManifestJSON, _ := json.MarshalIndent(mergedManifest, "", "  ")
    fw, _ := zw.Create("manifest.json")
    fw.Write(mergedManifestJSON)
    zw.Close()

    return buf.Bytes(), nil
}

func computeOverallRootHash(hashes []string) string {
    sort.Strings(hashes)
    h := sha256.New()
    for _, rh := range hashes {
        h.Write([]byte(rh))
    }
    return hex.EncodeToString(h.Sum(nil))
}
```

**Step 3: Add test**

```go
func TestMergeEvidencePackages(t *testing.T) {
    // Create two minimal evidence packages
    events1 := []Event{{ID: "1", JobID: "job_1", Type: "job_created", Hash: "hash1"}}
    events2 := []Event{{ID: "2", JobID: "job_2", Type: "job_created", Hash: "hash2"}}
    m1 := Manifest{JobID: "job_1", LastEventHash: "hash1", EventCount: 1}
    m2 := Manifest{JobID: "job_2", LastEventHash: "hash2", EventCount: 1}

    pkg1, _ := buildTestPackage("job_1", events1, m1)
    pkg2, _ := buildTestPackage("job_2", events2, m2)

    merged, err := MergeEvidencePackages([][]byte{pkg1, pkg2}, ExportOptions{})
    if err != nil {
        t.Fatalf("merge failed: %v", err)
    }

    // Verify merged ZIP can be opened
    zr, err := zip.NewReader(bytes.NewReader(merged), int64(len(merged)))
    if err != nil {
        t.Fatalf("merged zip unreadable: %v", err)
    }
    names := make(map[string]bool)
    for _, f := range zr.File {
        names[f.Name] = true
    }
    if !names["manifest.json"] {
        t.Error("merged zip missing manifest.json")
    }
    if !names["job_0001.zip"] || !names["job_0002.zip"] {
        t.Error("merged zip missing job entries")
    }
}
```

**Step 4: Run tests**

```bash
go test ./pkg/proof/... -v -run TestMerge
```

**Step 5: Commit**

```bash
git add pkg/proof/merge.go pkg/proof/merge_test.go
git commit -m "feat(proof): add evidence package merge for multi-job compliance reports"
```

---

## Task 5: Compliance Policy Engine

**Files:**

- Create: `pkg/compliance/policy.go` — policy definition and matcher
- Create: `pkg/compliance/policy_test.go`
- Modify: `internal/agent/runtime/runner.go` — trigger evidence collection on job completion

**Step 1: Define policy types**

```go
package compliance

// Policy defines when evidence must be collected.
type Policy struct {
    Name        string            `yaml:"name"`
    Description string            `yaml:"description"`
    Enabled     bool              `yaml:"enabled"`
    Conditions  []PolicyCondition `yaml:"conditions"`
    Actions     []PolicyAction    `yaml:"actions"`
}

type PolicyCondition struct {
    Field    string   `yaml:"field"`     // agent_id, tool_names, job_tag, status
    Operator string   `yaml:"operator"`  // contains, equals, regex_match
    Values   []string `yaml:"values"`
}

type PolicyAction struct {
    Type string `yaml:"type"` // "collect_evidence", "notify", "require_approval"
}

// Match returns true if the job satisfies all conditions.
func (p *Policy) Match(job *JobContext) bool {
    if !p.Enabled {
        return false
    }
    for _, cond := range p.Conditions {
        if !cond.Matches(job) {
            return false
        }
    }
    return true
}
```

**Step 2: Implement condition matching**

```go
func (c *PolicyCondition) Matches(job *JobContext) bool {
    var fieldValue string
    switch c.Field {
    case "agent_id":
        fieldValue = job.AgentID
    case "status":
        fieldValue = job.Status
    case "tool_names":
        return c.matchesToolNames(job.ToolNames)
    case "job_tag":
        return c.matchesJobTags(job.Tags)
    }
    return c.matchesString(fieldValue)
}

func (c *PolicyCondition) matchesString(value string) bool {
    switch c.Operator {
    case "equals":
        for _, v := range c.Values {
            if value == v {
                return true
            }
        }
    case "contains":
        for _, v := range c.Values {
            if strings.Contains(value, v) {
                return true
            }
        }
    case "regex_match":
        for _, v := range c.Values {
            matched, _ := regexp.MatchString(v, value)
            if matched {
                return true
            }
        }
    }
    return false
}

func (c *PolicyCondition) matchesToolNames(names []string) bool {
    for _, name := range names {
        if c.matchesString(name) {
            return true
        }
    }
    return false
}
```

**Step 3: Wire into job completion**

In `internal/agent/runtime/runner.go`, after a job completes, call the policy engine:

```go
// After job completion, check compliance policies
ctx := compliance.NewContext(job)
for _, policy := range compliancePolicies {
    if policy.Match(ctx) {
        for _, action := range policy.Actions {
            if action.Type == "collect_evidence" {
                go func(jobID string) {
                    if err := h.collectEvidenceForJob(ctx, jobID); err != nil {
                        hlog.Errorf("compliance evidence collection failed for job %s: %v", jobID, err)
                    }
                }(job.ID)
            }
        }
    }
}
```

**Step 4: Add config example**

Add to `configs/compliance.yaml`:

```yaml
policies:
  - name: "payment-evidence"
    enabled: true
    description: "Collect evidence for all payment-related jobs"
    conditions:
      - field: "tool_names"
        operator: "contains"
        values: ["stripe", "payment", "paypal"]
    actions:
      - type: "collect_evidence"

  - name: "high-value-transactions"
    enabled: true
    description: "Extra evidence for high-value transactions"
    conditions:
      - field: "job_tag"
        operator: "equals"
        values: ["high_value"]
    actions:
      - type: "collect_evidence"
      - type: "notify"
```

**Step 5: Write tests**

```go
func TestPolicyMatching(t *testing.T) {
    policy := Policy{
        Name:    "payment-policy",
        Enabled: true,
        Conditions: []PolicyCondition{
            {Field: "tool_names", Operator: "contains", Values: []string{"stripe"}},
        },
        Actions: []PolicyAction{{Type: "collect_evidence"}},
    }

    ctx := &JobContext{
        AgentID:   "agent_1",
        Status:    "completed",
        ToolNames: []string{"http_call", "stripe.charge"},
    }

    if !policy.Match(ctx) {
        t.Error("expected policy to match job with stripe tool")
    }

    ctx2 := &JobContext{
        AgentID:   "agent_1",
        Status:    "completed",
        ToolNames: []string{"http_call", "calculator"},
    }
    if policy.Match(ctx2) {
        t.Error("expected policy NOT to match job without stripe tool")
    }
}
```

**Step 6: Run tests**

```bash
go test ./pkg/compliance/... -v
```

**Step 7: Commit**

```bash
git add pkg/compliance/policy.go pkg/compliance/policy_test.go
git commit -m "feat(compliance): add policy engine for automatic evidence collection"
```

---

## Task 6: Semantic Verification — Detect "Claimed Success but Actually Failed"

**Files:**

- Modify: `pkg/proof/verify.go:26-216` — add semantic verification pass
- Create: `pkg/proof/semantic_verify.go`
- Create: `pkg/proof/semantic_verify_test.go`

**Step 1: Add semantic verification**

The existing `ValidateLedgerConsistency` checks structural consistency. Add a semantic check that detects if the event record says "success" but the actual result contains an error:

```go
// SemanticVerificationResult holds semantic verification results.
type SemanticVerificationResult struct {
    OK              bool
    FraudIndicators []FraudIndicator
}

type FraudIndicator struct {
    Type        string // "success_claimed_but_error_in_result", "tool_not_called_but_marked_success"
    InvocationID string
    Details      string
}

// SemanticVerify performs semantic checks beyond structural integrity.
func SemanticVerify(events []Event, ledger []ToolInvocation) SemanticVerificationResult {
    result := SemanticVerificationResult{OK: true}

    // Build event-derived tool invocations map
    eventInvocations := buildInvocationMapFromEvents(events)

    for _, inv := range ledger {
        if inv.Status != "success" && inv.Status != "committed" {
            continue
        }

        eventInv, ok := eventInvocations[inv.IdempotencyKey]
        if !ok {
            result.OK = false
            result.FraudIndicators = append(result.FraudIndicators, FraudIndicator{
                Type:        "success_claimed_but_not_in_events",
                InvocationID: inv.ID,
                Details:     "Ledger claims success but no corresponding event found",
            })
            continue
        }

        // Check if result contains an error even though status is "success"
        if hasErrorInResult(inv.Result) && eventInv.Outcome == "success" {
            result.OK = false
            result.FraudIndicators = append(result.FraudIndicators, FraudIndicator{
                Type:        "success_claimed_but_error_in_result",
                InvocationID: inv.ID,
                Details:     fmt.Sprintf("Status=success but result contains error: %s", truncate(inv.Result, 200)),
            })
        }
    }

    return result
}

func hasErrorInResult(result string) bool {
    if result == "" {
        return false
    }
    // Check for common error patterns in result JSON
    var m map[string]interface{}
    if err := json.Unmarshal([]byte(result), &m); err != nil {
        return false
    }
    // Check top-level error field
    if _, ok := m["error"]; ok {
        return true
    }
    if status, ok := m["status"].(string); ok && status == "error" {
        return true
    }
    return false
}
```

**Step 2: Integrate into VerifyEvidenceZip**

In `pkg/proof/verify.go`, add to `VerifyResult`:

```go
type VerifyResult struct {
    // ... existing fields ...
    SemanticOK         bool               `json:"semantic_ok"`
    FraudIndicators    []FraudIndicator   `json:"fraud_indicators,omitempty"`
}
```

After ledger validation, call `SemanticVerify` and populate the fields.

**Step 3: Write tests**

```go
func TestSemanticVerify_DetectsFalseSuccess(t *testing.T) {
    events := []Event{{ID: "1", Type: "tool_invocation_finished", Payload: `{"idempotency_key":"k1","outcome":"success","tool_name":"payment"}`}}
    ledger := []ToolInvocation{{ID: "inv1", IdempotencyKey: "k1", Status: "success", Result: `{"error":"charge failed: insufficient funds"}`}}

    result := SemanticVerify(events, ledger)
    if result.OK {
        t.Error("expected semantic verification to fail")
    }
    if len(result.FraudIndicators) == 0 {
        t.Error("expected fraud indicator")
    }
}
```

**Step 4: Run tests**

```bash
go test ./pkg/proof/... -v -run TestSemantic
```

**Step 5: Commit**

```bash
git add pkg/proof/semantic_verify.go pkg/proof/semantic_verify_test.go pkg/proof/verify.go
git commit -m "feat(proof): add semantic verification to detect fraud indicators in evidence"
```

---

## Task 7: Evidence Retention Policy Per Tenant

**Files:**

- Create: `pkg/compliance/retention.go` — retention policy definition and GC
- Create: `pkg/compliance/retention_test.go`
- Modify: `configs/compliance.yaml` — add retention section

**Step 1: Define retention policy**

```go
package compliance

// RetentionPolicy defines how long evidence is kept per tenant.
type RetentionPolicy struct {
    TenantID      string        `yaml:"tenant_id"`
    EvidenceTTL   time.Duration `yaml:"evidence_ttl"`    // e.g., 7 years for financial, 1 year general
    Enabled       bool          `yaml:"enabled"`
    ArchiveBefore time.Time     `yaml:"archive_before"`  // Move to cold storage before this date
}

// ShouldRetain returns true if evidence for the given job should be retained.
func (p *RetentionPolicy) ShouldRetain(jobCreatedAt time.Time, evidenceAge time.Duration) bool {
    if !p.Enabled {
        return false
    }
    return evidenceAge < p.EvidenceTTL
}
```

**Step 2: Add GC logic**

```go
// CollectExpiredEvidence returns job IDs whose evidence has exceeded retention TTL.
func (p *RetentionPolicy) CollectExpiredEvidence(ctx context.Context, store EvidenceStore) ([]string, error) {
    if !p.Enabled {
        return nil, nil
    }

    cutoff := time.Now().UTC().Add(-p.EvidenceTTL)
    jobs, err := store.ListJobsCreatedBefore(ctx, p.TenantID, cutoff)
    if err != nil {
        return nil, err
    }

    expired := make([]string, 0)
    for _, job := range jobs {
        if !p.ShouldRetain(job.CreatedAt, time.Since(job.CreatedAt)) {
            expired = append(expired, job.ID)
        }
    }
    return expired, nil
}
```

**Step 3: Add config example**

```yaml
# configs/compliance.yaml
retention:
  policies:
    - tenant_id: "financial_customer_a"
      evidence_ttl: "87600h" # 10 years
      enabled: true
    - tenant_id: "default"
      evidence_ttl: "8760h" # 1 year
      enabled: true
```

**Step 4: Commit**

```bash
git add pkg/compliance/retention.go pkg/compliance/retention_test.go
git commit -m "feat(compliance): add evidence retention policy per tenant"
```

---

## Task 8: Documentation

**Files:**

- Modify: `docs/concepts/evidence-package.md` — update to reflect new features

**Step 1: Update docs**

Document the new features: digital signatures, semantic verification, policy engine, retention policies, merge functionality.

**Step 2: Commit**

```bash
git add docs/concepts/evidence-package.md
git commit -m "docs: update evidence package docs with Phase 1-4 features"
```

---

## Plan Summary

| Task | Description                                        | Files                                                          |
| ---- | -------------------------------------------------- | -------------------------------------------------------------- |
| 1    | Ed25519 digital signatures on evidence packages    | `pkg/proof/signature.go`, `types.go`, `export.go`, `verify.go` |
| 2    | Wire signing config into REST API export           | `internal/api/http/forensics.go`, `configs/api.yaml`           |
| 3    | SHA256 hash for all ZIP files                      | `pkg/proof/export.go`                                          |
| 4    | Evidence package merge for multi-job reports       | `pkg/proof/merge.go`                                           |
| 5    | Compliance policy engine (auto-trigger collection) | `pkg/compliance/policy.go`                                     |
| 6    | Semantic verification (fraud detection)            | `pkg/proof/semantic_verify.go`                                 |
| 7    | Evidence retention policy per tenant               | `pkg/compliance/retention.go`                                  |
| 8    | Documentation update                               | `docs/concepts/evidence-package.md`                            |

**Execution approach:** Subagent-Driven (this session) — dispatch fresh subagent per task with review between tasks.

**Which approach?**
