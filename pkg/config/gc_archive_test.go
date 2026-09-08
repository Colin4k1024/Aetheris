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

package config

import (
	"os"
	"path/filepath"
	"testing"
)

// TestRuntimeGCConfig_ArchiveDefaults 验证归档开关默认关闭、归档副本默认永久保留。
func TestRuntimeGCConfig_ArchiveDefaults(t *testing.T) {
	cfg := DefaultDevConfig()

	if cfg.Runtime.GC.ArchiveEnabled {
		t.Fatal("expected archive_enabled=false by default")
	}
	if cfg.Runtime.GC.ArchiveTTLDays != 0 {
		t.Fatalf("expected archive_ttl_days=0 (永久保留), got %d", cfg.Runtime.GC.ArchiveTTLDays)
	}
}

// TestRuntimeGCConfig_ArchiveYAMLOverride 验证 archive_enabled / archive_ttl_days 可从 YAML 读取，
// 否则归档能力对外不可配置。
func TestRuntimeGCConfig_ArchiveYAMLOverride(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "gc-archive.yaml")
	configContent := `
runtime:
  gc:
    enabled: true
    interval: 24h
    ttl_days: 90
    batch_size: 1000
    archive_enabled: true
    archive_ttl_days: 730
`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	if !cfg.Runtime.GC.ArchiveEnabled {
		t.Error("archive_enabled = false, want true")
	}
	if cfg.Runtime.GC.ArchiveTTLDays != 730 {
		t.Errorf("archive_ttl_days = %d, want 730", cfg.Runtime.GC.ArchiveTTLDays)
	}
	if !cfg.Runtime.GC.Enabled || cfg.Runtime.GC.TTLDays != 90 || cfg.Runtime.GC.BatchSize != 1000 {
		t.Errorf("既有 GC 字段被破坏: %+v", cfg.Runtime.GC)
	}
}
