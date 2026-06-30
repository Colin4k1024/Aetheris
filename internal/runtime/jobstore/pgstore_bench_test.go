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

//go:build benchmark

package jobstore

import (
	"context"
	"fmt"
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
)

// benchmarkDSN 从环境变量读取 PostgreSQL DSN，未设置时跳过 benchmark
func benchmarkDSN() string {
	dsn := os.Getenv("BENCHMARK_DSN")
	if dsn == "" {
		dsn = "postgres://corag:corag@localhost:5432/corag?sslmode=disable"
	}
	return dsn
}

// setupBenchStore 创建测试用 pgStore；跳过不可用的数据库
func setupBenchStore(b *testing.B) *pgStore {
	b.Helper()
	dsn := benchmarkDSN()
	store, err := NewPostgresStore(context.Background(), dsn, 30*time.Second)
	if err != nil {
		b.Skipf("PostgreSQL 不可用 (%s): %v", dsn, err)
	}
	return store.(*pgStore)
}

// seedJob 创建一个带有初始事件的 Job，返回 jobID
func seedJob(b *testing.B, store *pgStore) string {
	b.Helper()
	ctx := context.Background()
	jobID := "bench-" + uuid.New().String()
	_, err := store.Append(ctx, jobID, 0, JobEvent{
		JobID: jobID,
		Type:  JobCreated,
		Payload: []byte(fmt.Sprintf(`{"goal":"benchmark-%s"}`, jobID[:8])),
	})
	if err != nil {
		b.Fatalf("seedJob Append failed: %v", err)
	}
	return jobID
}

// BenchmarkAppend 测试单 goroutine Append 吞吐量
func BenchmarkAppend(b *testing.B) {
	store := setupBenchStore(b)
	defer store.Close()

	ctx := context.Background()
	jobID := seedJob(b, store)
	version := 1

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := store.Append(ctx, jobID, version, JobEvent{
			JobID: jobID,
			Type:  "bench_step",
			Payload: []byte(fmt.Sprintf(`{"step":%d}`, i)),
		})
		if err != nil {
			b.Fatalf("Append failed at version %d: %v", version, err)
		}
		version++
	}
}

// BenchmarkAppendParallel 测试并发 Append 吞吐量（多 Job，每个 Job 串行 Append）
func BenchmarkAppendParallel(b *testing.B) {
	store := setupBenchStore(b)
	defer store.Close()

	ctx := context.Background()

	// 预创建足够多的 Job
	numJobs := 100
	jobs := make([]string, numJobs)
	versions := make([]int, numJobs)
	for i := 0; i < numJobs; i++ {
		jobs[i] = seedJob(b, store)
		versions[i] = 1
	}

	var counter atomic.Int64
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		localIdx := int(counter.Add(1)-1) % numJobs
		for pb.Next() {
			jobID := jobs[localIdx]
			_, err := store.Append(ctx, jobID, versions[localIdx], JobEvent{
				JobID: jobID,
				Type:  "bench_step",
				Payload: []byte(`{"step":1}`),
			})
			if err != nil {
				// CAS 冲突时重试（预期行为）
				if err == ErrVersionMismatch {
					continue
				}
				b.Fatalf("Append failed: %v", err)
			}
			versions[localIdx]++
			localIdx = (localIdx + 1) % numJobs
		}
	})
}

// BenchmarkClaim 测试 Claim 吞吐量（无 Job 可 Claim 时的空轮询）
func BenchmarkClaim(b *testing.B) {
	store := setupBenchStore(b)
	defer store.Close()

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, _, err := store.Claim(ctx, "bench-worker")
		if err != nil && err != ErrNoJob {
			b.Fatalf("Claim failed: %v", err)
		}
	}
}

// BenchmarkClaimWithJobs 测试有 Job 时的 Claim 吞吐量
func BenchmarkClaimWithJobs(b *testing.B) {
	store := setupBenchStore(b)
	defer store.Close()

	ctx := context.Background()

	// 预创建 Job
	numJobs := b.N
	if numJobs > 1000 {
		numJobs = 1000
	}
	for i := 0; i < numJobs; i++ {
		seedJob(b, store)
	}

	b.ResetTimer()
	claimed := 0
	for i := 0; i < b.N; i++ {
		_, _, _, err := store.Claim(ctx, "bench-worker")
		if err == ErrNoJob {
			break
		}
		if err != nil {
			b.Fatalf("Claim failed: %v", err)
		}
		claimed++
	}
	b.ReportMetric(float64(claimed), "claimed_jobs")
}

// BenchmarkClaimConcurrent 测试多 Worker 并发 Claim
func BenchmarkClaimConcurrent(b *testing.B) {
	store := setupBenchStore(b)
	defer store.Close()

	ctx := context.Background()

	// 预创建 Job
	numJobs := 500
	for i := 0; i < numJobs; i++ {
		seedJob(b, store)
	}

	var totalClaimed atomic.Int64
	b.ResetTimer()

	var wg sync.WaitGroup
	numWorkers := 10
	jobsPerWorker := b.N / numWorkers
	if jobsPerWorker < 1 {
		jobsPerWorker = 1
	}

	for w := 0; w < numWorkers; w++ {
		wg.Add(1)
		go func(workerID string) {
			defer wg.Done()
			for i := 0; i < jobsPerWorker; i++ {
				_, _, _, err := store.Claim(ctx, workerID)
				if err == ErrNoJob {
					return
				}
				if err != nil {
					continue
				}
				totalClaimed.Add(1)
			}
		}(fmt.Sprintf("bench-worker-%d", w))
	}
	wg.Wait()
	b.ReportMetric(float64(totalClaimed.Load()), "claimed_jobs")
}

// BenchmarkListEvents 测试事件查询吞吐量
func BenchmarkListEvents(b *testing.B) {
	store := setupBenchStore(b)
	defer store.Close()

	ctx := context.Background()
	jobID := seedJob(b, store)

	// 追加 50 个事件
	for i := 1; i <= 50; i++ {
		_, _ = store.Append(ctx, jobID, i, JobEvent{
			JobID:   jobID,
			Type:    "bench_step",
			Payload: []byte(fmt.Sprintf(`{"step":%d}`, i)),
		})
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, err := store.ListEvents(ctx, jobID)
		if err != nil {
			b.Fatalf("ListEvents failed: %v", err)
		}
	}
}
