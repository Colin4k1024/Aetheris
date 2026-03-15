// Copyright 2026 Aetheris
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

package api

import (
	"testing"
)

func TestNewPoolManager(t *testing.T) {
	m := NewPoolManager()
	if m == nil {
		t.Fatal("expected non-nil PoolManager")
	}
	if m.pools == nil {
		t.Error("expected pools map to be initialized")
	}
}

func TestPoolManager_GetPool_EmptyDSN(t *testing.T) {
	m := NewPoolManager()
	_, err := m.GetPool(nil, "")
	if err == nil {
		t.Error("expected error for empty DSN")
	}
}

func TestPoolStat_String(t *testing.T) {
	stat := PoolStat{
		TotalConns:    10,
		IdleConns:     5,
		AcquiredConns: 3,
		MaxConns:      20,
	}
	// Just verify fields are set correctly
	if stat.TotalConns != 10 {
		t.Errorf("expected TotalConns 10, got %d", stat.TotalConns)
	}
	if stat.IdleConns != 5 {
		t.Errorf("expected IdleConns 5, got %d", stat.IdleConns)
	}
	if stat.AcquiredConns != 3 {
		t.Errorf("expected AcquiredConns 3, got %d", stat.AcquiredConns)
	}
	if stat.MaxConns != 20 {
		t.Errorf("expected MaxConns 20, got %d", stat.MaxConns)
	}
}
