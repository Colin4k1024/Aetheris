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

package log

import (
	"bytes"
	"context"
	"log/slog"
	"testing"
)

func TestNewLogger_Default(t *testing.T) {
	logger, err := NewLogger(nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if logger == nil {
		t.Fatal("expected non-nil logger")
	}
}

func TestNewLogger_WithLevel(t *testing.T) {
	cfg := &Config{Level: "debug"}
	logger, err := NewLogger(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if logger == nil {
		t.Fatal("expected non-nil logger")
	}
}

func TestNewLogger_WithFormat(t *testing.T) {
	cfg := &Config{Format: "text"}
	logger, err := NewLogger(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if logger == nil {
		t.Fatal("expected non-nil logger")
	}
}

func TestNewLogger_WithInvalidLevel(t *testing.T) {
	cfg := &Config{Level: "invalid"}
	logger, err := NewLogger(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if logger == nil {
		t.Fatal("expected non-nil logger")
	}
}

func TestMultiHandler_New(t *testing.T) {
	handler := NewMultiHandler()
	if handler == nil {
		t.Fatal("expected non-nil handler")
	}
	if len(handler.handlers) != 0 {
		t.Errorf("expected 0 handlers, got %d", len(handler.handlers))
	}
}

func TestMultiHandler_Enabled(t *testing.T) {
	buf := &bytes.Buffer{}
	h1 := slog.NewJSONHandler(buf, nil)
	h2 := slog.NewTextHandler(buf, nil)
	handler := NewMultiHandler(h1, h2)

	ctx := context.Background()
	if !handler.Enabled(ctx, slog.LevelInfo) {
		t.Error("expected Enabled to return true")
	}
}

func TestMultiHandler_Handle(t *testing.T) {
	buf := &bytes.Buffer{}
	h := slog.NewJSONHandler(buf, nil)
	handler := NewMultiHandler(h)

	ctx := context.Background()
	record := slog.Record{
		Message: "test",
		Level:   slog.LevelInfo,
	}

	err := handler.Handle(ctx, record)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestMultiHandler_WithAttrs(t *testing.T) {
	buf := &bytes.Buffer{}
	h := slog.NewJSONHandler(buf, nil)
	handler := NewMultiHandler(h)

	newHandler := handler.WithAttrs([]slog.Attr{slog.String("key", "value")})
	if newHandler == nil {
		t.Error("expected non-nil handler")
	}
}

func TestMultiHandler_WithGroup(t *testing.T) {
	buf := &bytes.Buffer{}
	h := slog.NewJSONHandler(buf, nil)
	handler := NewMultiHandler(h)

	newHandler := handler.WithGroup("group")
	if newHandler == nil {
		t.Error("expected non-nil handler")
	}
}

func TestLogger_Log(t *testing.T) {
	logger, _ := NewLogger(nil)
	logger.Info("test message", "key", "value")
}

func TestConfig_Defaults(t *testing.T) {
	cfg := &Config{}
	if cfg.Level != "" {
		t.Errorf("expected empty level, got %s", cfg.Level)
	}
	if cfg.Format != "" {
		t.Errorf("expected empty format, got %s", cfg.Format)
	}
	if cfg.File != "" {
		t.Errorf("expected empty file, got %s", cfg.File)
	}
}
