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

package determinism

import (
	"context"
	"testing"
)

func TestWithReplay(t *testing.T) {
	ctx := context.Background()

	// Non-replay mode
	if IsReplay(ctx) {
		t.Error("expected false for default context")
	}

	// Enable replay mode
	ctx = WithReplay(ctx, true)
	if !IsReplay(ctx) {
		t.Error("expected true after WithReplay(true)")
	}

	// Disable replay mode
	ctx = WithReplay(ctx, false)
	if IsReplay(ctx) {
		t.Error("expected false after WithReplay(false)")
	}
}

func TestIsReplay_NilContext(t *testing.T) {
	if IsReplay(nil) {
		t.Error("expected false for nil context")
	}
}

func TestIsReplay_NonBoolValue(t *testing.T) {
	ctx := context.WithValue(context.Background(), replayModeKey, "not-a-bool")
	if IsReplay(ctx) {
		t.Error("expected false for non-bool value")
	}
}

func TestReplayGuard_CheckEffectAllowed_NonReplay(t *testing.T) {
	ctx := context.Background()
	guard := &ReplayGuard{StrictReplay: true}

	// Should not panic in non-replay mode
	guard.CheckEffectAllowed(ctx, "job-1", "step-1", OpRandom)
}

func TestReplayGuard_CheckEffectAllowed_ReplayNonStrict(t *testing.T) {
	ctx := WithReplay(context.Background(), true)
	guard := &ReplayGuard{StrictReplay: false}

	// Should not panic in replay mode with non-strict
	guard.CheckEffectAllowed(ctx, "job-1", "step-1", OpRandom)
}

func TestForbiddenOps(t *testing.T) {
	ops := ForbiddenOps()
	if len(ops) != 6 {
		t.Errorf("expected 6 forbidden ops, got %d", len(ops))
	}
}

func TestForbiddenOp_Description(t *testing.T) {
	tests := []struct {
		op   ForbiddenOp
		desc string
	}{
		{OpWallClock, "读取系统时间（time.Now）在 Replay 时forbidden；请使用 runtime.Now(ctx)"},
		{OpRandom, "使用随机数/uuid 在 Replay 时forbidden；请使用 runtime.UUID(ctx) 或 runtime.Random(ctx)"},
		{OpUnrecordedIO, "未记录的外部 IO（http、db）在 Replay 时forbidden；请通过 Tool 或 runtime.HTTP(ctx)"},
		{OpGoroutine, "在 step 内启动 goroutine forbidden；会破坏确定性"},
		{OpChannel, "在 step 内使用 channel forbidden；请使用纯计算或 Tool"},
		{OpSleep, "time.Sleep 在 step 内forbidden；会引入非确定性"},
	}

	for _, tt := range tests {
		desc := tt.op.Description()
		if desc != tt.desc {
			t.Errorf("expected %s, got %s", tt.desc, desc)
		}
	}
}

func TestForbiddenOp_Description_Unknown(t *testing.T) {
	desc := ForbiddenOp("unknown").Description()
	if desc != "未记录的非确定性操作" {
		t.Errorf("expected default description, got %s", desc)
	}
}
