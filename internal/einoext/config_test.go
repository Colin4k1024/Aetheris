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

package einoext

import (
	"testing"

	"github.com/Colin4k1024/Aetheris/v2/pkg/config"
)

func TestRedisOptionsFromVectorConfig_Defaults(t *testing.T) {
	cfg := config.VectorConfig{
		Type: "redis",
	}
	opts, err := RedisOptionsFromVectorConfig(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if opts.Addr != "localhost:6379" {
		t.Errorf("expected localhost:6379, got %s", opts.Addr)
	}
}

func TestRedisOptionsFromVectorConfig_WithAddr(t *testing.T) {
	cfg := config.VectorConfig{
		Type: "redis",
		Addr: "redis.example.com:6379",
	}
	opts, err := RedisOptionsFromVectorConfig(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if opts.Addr != "redis.example.com:6379" {
		t.Errorf("expected redis.example.com:6379, got %s", opts.Addr)
	}
}

func TestRedisOptionsFromVectorConfig_WithDB(t *testing.T) {
	cfg := config.VectorConfig{
		Type: "redis",
		DB:   "5",
	}
	opts, err := RedisOptionsFromVectorConfig(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if opts.DB != 5 {
		t.Errorf("expected DB 5, got %d", opts.DB)
	}
}

func TestRedisOptionsFromVectorConfig_InvalidDB(t *testing.T) {
	cfg := config.VectorConfig{
		Type: "redis",
		DB:   "invalid",
	}
	opts, err := RedisOptionsFromVectorConfig(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Invalid DB should use default 0
	if opts.DB != 0 {
		t.Errorf("expected DB 0, got %d", opts.DB)
	}
}

func TestRedisOptionsFromVectorConfig_WithPassword(t *testing.T) {
	cfg := config.VectorConfig{
		Type:     "redis",
		Password: "secret",
	}
	opts, err := RedisOptionsFromVectorConfig(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if opts.Password != "secret" {
		t.Errorf("expected secret, got %s", opts.Password)
	}
}

func TestRedisOptionsFromVectorConfig_Protocol(t *testing.T) {
	cfg := config.VectorConfig{
		Type: "redis",
	}
	opts, err := RedisOptionsFromVectorConfig(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if opts.Protocol != 2 {
		t.Errorf("expected Protocol 2, got %d", opts.Protocol)
	}
	if !opts.UnstableResp3 {
		t.Error("expected UnstableResp3 to be true")
	}
}
