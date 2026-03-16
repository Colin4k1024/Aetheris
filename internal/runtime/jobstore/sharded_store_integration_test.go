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

package jobstore

import (
	"context"
	"testing"
)

// TestShardedStore_BasicOperations 测试 ShardedStore 基本操作：跨分片 Append/ListEvents/Claim
func TestShardedStore_BasicOperations(t *testing.T) {
	ctx := context.Background()

	// 创建 3 个分片
	shards := []JobStore{
		NewMemoryStore(),
		NewMemoryStore(),
		NewMemoryStore(),
	}
	store := NewShardedStore(shards)

	jobID := "job-sharded-1"

	// Append 到 job-1（会根据 hash 路由到某个分片）
	ev1 := JobEvent{JobID: jobID, Type: JobCreated}
	ver, err := store.Append(ctx, jobID, 0, ev1)
	if err != nil {
		t.Fatalf("Append: %v", err)
	}
	if ver != 1 {
		t.Errorf("version: got %d want 1", ver)
	}

	// ListEvents 应该返回正确的事件
	events, version, err := store.ListEvents(ctx, jobID)
	if err != nil {
		t.Fatalf("ListEvents: %v", err)
	}
	if version != 1 || len(events) != 1 {
		t.Errorf("events: version=%d len=%d", version, len(events))
	}
}

// TestShardedStore_ConsistentHashing 测试相同 jobID 始终路由到同一分片
func TestShardedStore_ConsistentHashing(t *testing.T) {
	ctx := context.Background()

	shards := []JobStore{
		NewMemoryStore(),
		NewMemoryStore(),
		NewMemoryStore(),
		NewMemoryStore(),
		NewMemoryStore(),
	}
	store := NewShardedStore(shards)

	// 相同 jobID 多次操作应该在同一分片
	jobID := "consistent-hash-job"

	for i := 0; i < 10; i++ {
		ev := JobEvent{JobID: jobID, Type: JobCreated, Payload: []byte(`{"i":` + string(rune('0'+i)) + `}`)}
		_, err := store.Append(ctx, jobID, i, ev)
		if err != nil {
			t.Fatalf("Append iteration %d: %v", i, err)
		}
	}

	events, version, err := store.ListEvents(ctx, jobID)
	if err != nil {
		t.Fatalf("ListEvents: %v", err)
	}
	if version != 10 {
		t.Errorf("final version: got %d want 10", version)
	}
	if len(events) != 10 {
		t.Errorf("events count: got %d want 10", len(events))
	}
}

// TestShardedStore_MultipleJobsAcrossShards 测试多个 Job 分布在不同分片
func TestShardedStore_MultipleJobsAcrossShards(t *testing.T) {
	ctx := context.Background()

	shards := []JobStore{
		NewMemoryStore(),
		NewMemoryStore(),
		NewMemoryStore(),
	}
	store := NewShardedStore(shards)

	// 创建多个 Job，期望它们分布在不同分片
	jobs := []string{
		"job-aaa",
		"job-bbb",
		"job-ccc",
		"job-ddd",
		"job-eee",
	}

	versions := make(map[string]int)
	for _, jobID := range jobs {
		ev := JobEvent{JobID: jobID, Type: JobCreated}
		ver, err := store.Append(ctx, jobID, 0, ev)
		if err != nil {
			t.Fatalf("Append %s: %v", jobID, err)
		}
		versions[jobID] = ver
	}

	// 验证每个 Job 都能正确读取
	for _, jobID := range jobs {
		events, version, err := store.ListEvents(ctx, jobID)
		if err != nil {
			t.Fatalf("ListEvents %s: %v", jobID, err)
		}
		if version != versions[jobID] {
			t.Errorf("%s version mismatch: got %d want %d", jobID, version, versions[jobID])
		}
		if len(events) != 1 {
			t.Errorf("%s events: got %d want 1", jobID, len(events))
		}
	}
}

// TestShardedStore_ClaimAcrossShards 测试跨分片 Claim
func TestShardedStore_ClaimAcrossShards(t *testing.T) {
	ctx := context.Background()

	shards := []JobStore{
		NewMemoryStore(),
		NewMemoryStore(),
		NewMemoryStore(),
	}
	store := NewShardedStore(shards)

	// 在不同分片创建可 Claim 的 Job
	jobID1 := "job-claim-1"
	jobID2 := "job-claim-2"
	jobID3 := "job-claim-3"

	// 每个分片各创建一个可 Claim 的 Job
	_, _ = store.Append(ctx, jobID1, 0, JobEvent{JobID: jobID1, Type: JobCreated})
	_, _ = store.Append(ctx, jobID2, 0, JobEvent{JobID: jobID2, Type: JobCreated})
	_, _ = store.Append(ctx, jobID3, 0, JobEvent{JobID: jobID3, Type: JobCreated})

	// Claim 第一个可用的 Job
	claimedID, version, attemptID, err := store.Claim(ctx, "worker-1")
	if err != nil {
		t.Fatalf("Claim: %v", err)
	}
	if claimedID == "" {
		t.Error("claimed job ID should not be empty")
	}
	if version != 1 {
		t.Errorf("claimed version: got %d want 1", version)
	}
	if attemptID == "" {
		t.Error("attempt ID should not be empty")
	}

	// 再次 Claim 应该得到不同的 Job
	claimedID2, _, _, err := store.Claim(ctx, "worker-1")
	if err != nil {
		t.Fatalf("Second Claim: %v", err)
	}
	if claimedID2 == claimedID {
		t.Error("second claim should return different job")
	}
}

// TestShardedStore_ClaimJobSpecific 测试指定 Job 的 Claim
func TestShardedStore_ClaimJobSpecific(t *testing.T) {
	ctx := context.Background()

	shards := []JobStore{
		NewMemoryStore(),
		NewMemoryStore(),
		NewMemoryStore(),
	}
	store := NewShardedStore(shards)

	jobID := "job-claim-specific"
	_, _ = store.Append(ctx, jobID, 0, JobEvent{JobID: jobID, Type: JobCreated})

	// 第一次 Claim
	ver1, attemptID1, err := store.ClaimJob(ctx, "worker-1", jobID)
	if err != nil {
		t.Fatalf("ClaimJob first: %v", err)
	}
	if ver1 != 1 {
		t.Errorf("first claim version: got %d want 1", ver1)
	}

	// 同一 Worker 再次 Claim 同 Job 应该返回已 Claim
	_, _, err = store.ClaimJob(ctx, "worker-1", jobID)
	if err != ErrClaimNotFound {
		t.Errorf("second claim by same worker: got %v want %v", err, ErrClaimNotFound)
	}

	// 其他 Worker Claim 同一 Job 应该失败
	_, _, err = store.ClaimJob(ctx, "worker-2", jobID)
	if err != ErrClaimNotFound {
		t.Errorf("claim by different worker: got %v want %v", err, ErrClaimNotFound)
	}

	// 使用相同的 attempt ID 可以 Claim（模拟 lease 过期后的重试）
	// 注意：实际场景中需要先让 lease 过期，这里测试逻辑
	_ = attemptID1 // attempt ID 可用于后续验证
}

// TestShardedStore_Heartbeat 测试 Heartbeat 路由
func TestShardedStore_Heartbeat(t *testing.T) {
	ctx := context.Background()

	shards := []JobStore{
		NewMemoryStore(),
		NewMemoryStore(),
		NewMemoryStore(),
	}
	store := NewShardedStore(shards)

	jobID := "job-heartbeat"
	// Append event first
	_, _ = store.Append(ctx, jobID, 0, JobEvent{JobID: jobID, Type: JobCreated})
	_, _, _ = store.ClaimJob(ctx, "worker-1", jobID)

	// Heartbeat 应该成功
	err := store.Heartbeat(ctx, "worker-1", jobID)
	if err != nil {
		t.Errorf("Heartbeat: %v", err)
	}

	// 错误的 Worker Heartbeat 应该失败
	err = store.Heartbeat(ctx, "worker-2", jobID)
	if err != ErrClaimNotFound {
		t.Errorf("Heartbeat wrong worker: got %v want %v", err, ErrClaimNotFound)
	}
}

// TestShardedStore_Watch 测试 Watch 跨分片
func TestShardedStore_Watch(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	shards := []JobStore{
		NewMemoryStore(),
		NewMemoryStore(),
		NewMemoryStore(),
	}
	store := NewShardedStore(shards)

	jobID := "job-watch"

	// 创建 Watch
	ch, err := store.Watch(ctx, jobID)
	if err != nil {
		t.Fatalf("Watch: %v", err)
	}

	// Append 事件
	_, _ = store.Append(ctx, jobID, 0, JobEvent{JobID: jobID, Type: JobCreated})

	// 等待事件
	select {
	case event := <-ch:
		if event.JobID != jobID {
			t.Errorf("watch event jobID: got %s want %s", event.JobID, jobID)
		}
	case <-ctx.Done():
		t.Fatal("watch timeout")
	}
}

// TestShardedStore_EmptyJobList 测试空 Job 列表的 Claim
func TestShardedStore_EmptyJobList(t *testing.T) {
	ctx := context.Background()

	shards := []JobStore{
		NewMemoryStore(),
		NewMemoryStore(),
	}
	store := NewShardedStore(shards)

	// 没有可 Claim 的 Job
	_, _, _, err := store.Claim(ctx, "worker-1")
	if err != ErrNoJob {
		t.Errorf("Claim with no jobs: got %v want %v", err, ErrNoJob)
	}
}

// TestShardedStore_VersionMismatch 测试跨分片的版本冲突
func TestShardedStore_VersionMismatch(t *testing.T) {
	ctx := context.Background()

	shards := []JobStore{
		NewMemoryStore(),
		NewMemoryStore(),
		NewMemoryStore(),
	}
	store := NewShardedStore(shards)

	jobID := "job-version-conflict"

	// 第一次 Append 成功
	_, err := store.Append(ctx, jobID, 0, JobEvent{JobID: jobID, Type: JobCreated})
	if err != nil {
		t.Fatalf("First Append: %v", err)
	}

	// 错误的版本号 Append 应该失败
	_, err = store.Append(ctx, jobID, 0, JobEvent{JobID: jobID, Type: PlanGenerated})
	if err != ErrVersionMismatch {
		t.Errorf("Version mismatch: got %v want %v", err, ErrVersionMismatch)
	}

	// 正确的版本号 Append 应该成功
	_, err = store.Append(ctx, jobID, 1, JobEvent{JobID: jobID, Type: PlanGenerated})
	if err != nil {
		t.Errorf("Correct version Append: %v", err)
	}
}
