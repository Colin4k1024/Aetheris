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
	"os"
	"testing"
	"time"
)

func TestNewLogger_Default(t *testing.T) {
	logger, err := NewLogger(nil)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if logger == nil {
		t.Fatal("expected logger, got nil")
	}

	logger.Info("test info message")
}

func TestNewLogger_WithLevel(t *testing.T) {
	tests := []struct {
		name           string
		level          string
		expectedLevel  slog.Level
	}{
		{"debug", "debug", slog.LevelDebug},
		{"info", "info", slog.LevelInfo},
		{"warn", "warn", slog.LevelWarn},
		{"error", "error", slog.LevelError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{Level: tt.level}
			logger, err := NewLogger(cfg)
			if err != nil {
				t.Fatalf("expected no error, got %v", err)
			}

			// Verify logger can write at the specified level
			switch tt.level {
			case "debug":
				logger.Debug("debug message")
			case "info":
				logger.Info("info message")
			case "warn":
				logger.Warn("warn message")
			case "error":
				logger.Error("error message")
			}
		})
	}
}

func TestNewLogger_WithFormat(t *testing.T) {
	cfg := &Config{Format: "text"}
	logger, err := NewLogger(cfg)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	logger.Info("text format test")
}

func TestMultiHandler(t *testing.T) {
	var buf1, buf2 bytes.Buffer

	h1 := slog.NewJSONHandler(&buf1, nil)
	h2 := slog.NewJSONHandler(&buf2, nil)

	mh := NewMultiHandler(h1, h2)

	ctx := context.Background()
	record := slog.Record{
		Time:    time.Now(),
		Message: "test message",
		Level:   slog.LevelInfo,
	}

	err := mh.Handle(ctx, record)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if buf1.Len() == 0 {
		t.Error("expected buffer1 to have content")
	}

	if buf2.Len() == 0 {
		t.Error("expected buffer2 to have content")
	}
}

func TestMultiHandler_Enabled(t *testing.T) {
	var buf1, buf2 bytes.Buffer

	h1 := slog.NewJSONHandler(&buf1, &slog.HandlerOptions{Level: slog.LevelInfo})
	h2 := slog.NewJSONHandler(&buf2, &slog.HandlerOptions{Level: slog.LevelWarn})

	mh := NewMultiHandler(h1, h2)

	ctx := context.Background()

	// Should be enabled for LevelInfo since h1 handles it
	if !mh.Enabled(ctx, slog.LevelInfo) {
		t.Error("expected enabled for LevelInfo")
	}

	// Should be enabled for LevelWarn since both handle it
	if !mh.Enabled(ctx, slog.LevelWarn) {
		t.Error("expected enabled for LevelWarn")
	}

	// Should not be enabled for LevelDebug (lower than both)
	if mh.Enabled(ctx, slog.LevelDebug) {
		t.Error("expected disabled for LevelDebug")
	}
}

func TestMultiHandler_WithAttrs(t *testing.T) {
	var buf bytes.Buffer

	h := slog.NewJSONHandler(&buf, nil)
	mh := NewMultiHandler(h)

	mhWithAttrs := mh.WithAttrs([]slog.Attr{
		{Key: "service", Value: slog.StringValue("test")},
	})

	ctx := context.Background()
	record := slog.Record{
		Time:    time.Now(),
		Message: "test with attrs",
		Level:   slog.LevelInfo,
	}

	err := mhWithAttrs.Handle(ctx, record)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestMultiHandler_WithGroup(t *testing.T) {
	var buf bytes.Buffer

	h := slog.NewJSONHandler(&buf, nil)
	mh := NewMultiHandler(h)

	mhWithGroup := mh.WithGroup("testGroup")

	ctx := context.Background()
	record := slog.Record{
		Time:    time.Now(),
		Message: "test with group",
		Level:   slog.LevelInfo,
	}

	err := mhWithGroup.Handle(ctx, record)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestNewLogger_WithFile(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "test-log-*.log")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	tmpPath := tmpFile.Name()
	tmpFile.Close()
	defer os.Remove(tmpPath)

	cfg := &Config{
		Level: "info",
		File:  tmpPath,
	}

	logger, err := NewLogger(cfg)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	logger.Info("file test message")

	// Verify file has content
	content, err := os.ReadFile(tmpPath)
	if err != nil {
		t.Fatalf("failed to read log file: %v", err)
	}

	if len(content) == 0 {
		t.Error("expected log file to have content")
	}
}

func TestLogger_WithContext(t *testing.T) {
	logger, _ := NewLogger(nil)

	ctx := context.WithValue(context.Background(), "request_id", "test-123")
	logger = logger.WithContext(ctx)

	logger.Info("test with context")
}

func TestLogger_With(t *testing.T) {
	logger, _ := NewLogger(nil)

	loggerWith := logger.With("key", "value")
	loggerWith.Info("test with attr")
}
