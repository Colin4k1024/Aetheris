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

package vector

import (
	"testing"

	"github.com/Colin4k1024/Aetheris/v2/pkg/config"
)

func TestNewStore_Memory(t *testing.T) {
	cfg := config.VectorConfig{
		Type: "memory",
	}
	store, err := NewStore(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if store == nil {
		t.Fatal("expected non-nil store")
	}
}

func TestNewStore_EmptyType(t *testing.T) {
	cfg := config.VectorConfig{
		Type: "",
	}
	store, err := NewStore(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if store == nil {
		t.Fatal("expected non-nil store")
	}
}

func TestNewStore_Unsupported(t *testing.T) {
	cfg := config.VectorConfig{
		Type: "pgvector",
	}
	store, err := NewStore(cfg)
	if err == nil {
		t.Fatal("expected error for unsupported type")
	}
	if store != nil {
		t.Error("expected nil store")
	}
}
