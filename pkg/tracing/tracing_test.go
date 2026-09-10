// Copyright 2026 fanjia1024
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package tracing

import (
	"context"
	"testing"
)

func TestStartSpan(t *testing.T) {
	span := StartSpan("test-span")
	if span == nil {
		t.Fatal("expected non-nil span")
	}
	span.SetTag("key", "value")
	span.End()
}

func TestStartSpanWithContext(t *testing.T) {
	ctx := context.Background()
	span, newCtx := StartSpanWithContext(ctx, "ctx-span")
	if span == nil {
		t.Fatal("expected non-nil span")
	}
	if newCtx == nil {
		t.Fatal("expected non-nil context")
	}
	span.SetTag("operation", "test")
	span.AddEvent("event-1")
	span.End()
}

func TestSpan_End(t *testing.T) {
	span := &Span{}
	span.End()
}

func TestSpan_SetTag(t *testing.T) {
	span := &Span{}
	span.SetTag("key", "value")
	span.SetTag("int", 123)
	span.SetTag("bool", true)
	span.End()
}

func TestSpan_NilSafe(t *testing.T) {
	var s *Span
	s.SetTag("key", "val")
	s.AddEvent("safe")
	s.End()
}

func TestSpan_BridgesToOTel(t *testing.T) {
	ctx := context.Background()
	span, ctx := StartSpanWithContext(ctx, "bridge-test")
	if span == nil {
		t.Fatal("expected non-nil span")
	}
	span.SetTag("attr1", "val1")
	span.SetTag("attr2", 42)
	span.AddEvent("checkpoint")
	span.End()

	if ctx == nil {
		t.Fatal("expected non-nil context after StartSpanWithContext")
	}
}

func TestSpan_SetTag_Types(t *testing.T) {
	span := StartSpan("types-test")
	span.SetTag("str", "hello")
	span.SetTag("int", 42)
	span.SetTag("int64", int64(100))
	span.SetTag("float", 3.14)
	span.SetTag("bool", true)
	span.SetTag("other", []string{"a", "b"})
	span.End()
}
