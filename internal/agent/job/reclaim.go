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

package job

import (
	"context"

	"github.com/Colin4k1024/Aetheris/v2/internal/runtime/jobstore"
)

// ReclaimOrphanedFromEventStore 以 event store 租约为准回收孤儿（design/runtime-contract.md §3）：
// Session1-3 fix: 使用 DeriveStatusFromEvents 判断 event stream 真实状态，按结果分流处理：
//   - 事件流为 terminal (Completed/Failed/Cancelled)：同步 metadata 为对应终态，不重新入队。
//   - 事件流为 Blocked (Waiting/Parked)：跳过，不重新入队。
//   - 其余情况（Running/Pending）：metadata 置回 Pending 重新入队。
func ReclaimOrphanedFromEventStore(ctx context.Context, metadata JobStore, eventStore jobstore.JobStore) (int, error) {
	if metadata == nil || eventStore == nil {
		return 0, nil
	}
	ids, err := eventStore.ListJobIDsWithExpiredClaim(ctx)
	if err != nil || len(ids) == 0 {
		return 0, err
	}
	var reclaimed int
	for _, jobID := range ids {
		events, _, err := eventStore.ListEvents(ctx, jobID)
		if err != nil {
			continue
		}
		derived := DeriveStatusFromEvents(events)
		switch derived {
		case StatusCompleted, StatusFailed, StatusCancelled:
			// 事件流已有终态事件，仅同步 metadata（不重新入队）
			j, err := metadata.Get(ctx, jobID)
			if err != nil || j == nil {
				continue
			}
			if j.Status != derived {
				_ = metadata.UpdateStatus(ctx, jobID, derived)
			}
			continue
		case StatusWaiting, StatusParked:
			// Job 处于 Blocked 状态，不回收
			continue
		}
		// 事件流状态为 Running/Pending（租约已过期），回收到 Pending 重新执行
		j, err := metadata.Get(ctx, jobID)
		if err != nil || j == nil {
			continue
		}
		if j.Status != StatusRunning {
			continue
		}
		if err := metadata.UpdateStatus(ctx, jobID, StatusPending); err != nil {
			continue
		}
		reclaimed++
	}
	return reclaimed, nil
}
