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

package worker

import (
	"testing"
	"time"

	"github.com/Colin4k1024/Aetheris/v2/pkg/config"
)

// TestRuntimeGCOpsConfig_ArchiveDefaults 验证归档开关默认关闭、归档保留默认永久，
// 且 nil 配置不会意外启用归档。
func TestRuntimeGCOpsConfig_ArchiveDefaults(t *testing.T) {
	gc := runtimeGCOpsConfig(nil)
	if gc.ArchiveEnabled {
		t.Errorf("ArchiveEnabled = true, want false (默认不启用归档)")
	}
	if gc.ArchiveTTLDays != 0 {
		t.Errorf("ArchiveTTLDays = %d, want 0 (默认永久保留归档副本)", gc.ArchiveTTLDays)
	}
}

// TestRuntimeGCOpsConfig_ArchiveOverrides 验证配置中的归档开关与保留天数被正确读取。
func TestRuntimeGCOpsConfig_ArchiveOverrides(t *testing.T) {
	cfg := &config.Config{
		Runtime: config.RuntimeConfig{
			GC: config.RuntimeGCConfig{
				Enabled:        true,
				Interval:       "24h",
				TTLDays:        90,
				BatchSize:      1000,
				ArchiveEnabled: true,
				ArchiveTTLDays: 730,
			},
		},
	}

	gc := runtimeGCOpsConfig(cfg)
	if !gc.ArchiveEnabled {
		t.Error("ArchiveEnabled = false, want true")
	}
	if gc.ArchiveTTLDays != 730 {
		t.Errorf("ArchiveTTLDays = %d, want 730", gc.ArchiveTTLDays)
	}
}

// TestRuntimeGCOpsConfig_NegativeArchiveTTLDays 验证非法保留天数被归一为永久保留，
// 避免误配置导致归档副本被立即清理。
func TestRuntimeGCOpsConfig_NegativeArchiveTTLDays(t *testing.T) {
	cfg := &config.Config{
		Runtime: config.RuntimeConfig{
			GC: config.RuntimeGCConfig{
				Enabled:        true,
				ArchiveEnabled: true,
				ArchiveTTLDays: -5,
			},
		},
	}

	gc := runtimeGCOpsConfig(cfg)
	if gc.ArchiveTTLDays != 0 {
		t.Errorf("ArchiveTTLDays = %d, want 0 (负值必须归一为永久保留)", gc.ArchiveTTLDays)
	}
}

// TestGCJobStoreConfig_PropagatesArchiveSettings 验证 worker 把归档配置完整传入 jobstore.GCConfig，
// 否则归档能力对外不可达（配置存在但运行时被忽略）。
func TestGCJobStoreConfig_PropagatesArchiveSettings(t *testing.T) {
	ops := runtimeGCOps{
		Enabled:        true,
		Interval:       12 * time.Hour,
		TTLDays:        30,
		BatchSize:      250,
		ArchiveEnabled: true,
		ArchiveTTLDays: 365,
	}

	got := gcJobStoreConfig(ops)
	if !got.Enable {
		t.Error("Enable = false, want true")
	}
	if got.TTLDays != 30 {
		t.Errorf("TTLDays = %d, want 30", got.TTLDays)
	}
	if got.BatchSize != 250 {
		t.Errorf("BatchSize = %d, want 250", got.BatchSize)
	}
	if got.RunInterval != 12*time.Hour {
		t.Errorf("RunInterval = %v, want 12h", got.RunInterval)
	}
	if !got.ArchiveEnabled {
		t.Error("ArchiveEnabled = false, want true (归档配置必须传入 jobstore)")
	}
	if got.ArchiveTTLDays != 365 {
		t.Errorf("ArchiveTTLDays = %d, want 365", got.ArchiveTTLDays)
	}
}

// TestGCJobStoreConfig_ArchiveDisabledByDefault 验证未开启归档时 jobstore 配置同样为关闭。
func TestGCJobStoreConfig_ArchiveDisabledByDefault(t *testing.T) {
	got := gcJobStoreConfig(runtimeGCOps{Enabled: true, TTLDays: 90, BatchSize: 1000})
	if got.ArchiveEnabled {
		t.Error("ArchiveEnabled = true, want false")
	}
	if got.ArchiveTTLDays != 0 {
		t.Errorf("ArchiveTTLDays = %d, want 0", got.ArchiveTTLDays)
	}
}
