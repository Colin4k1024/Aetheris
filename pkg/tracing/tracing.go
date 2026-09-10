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

// Package tracing provides tracing primitives for Aetheris.
// The legacy Span/StartSpan API now bridges to the OpenTelemetry tracer
// configured in otel.go. If no tracer provider is initialized, OTel's
// global default returns a no-op span — this is correct OTel behavior,
// not a silent fake-success placeholder.
package tracing

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// Span wraps an OpenTelemetry trace.Span for backward compatibility
// with the legacy StartSpan API.
type Span struct {
	otel trace.Span
}

// StartSpan creates a new span using the global OTel tracer.
// When no tracer provider has been initialized via InitTracer, OTel
// returns a no-op span — callers still get a valid *Span that is safe
// to End() and SetTag().
func StartSpan(name string) *Span {
	_, span := otel.Tracer("aetheris").Start(context.Background(), name)
	return &Span{otel: span}
}

// StartSpanWithContext creates a new span with context propagation,
// returning the span and the updated context for downstream calls.
func StartSpanWithContext(ctx context.Context, name string) (*Span, context.Context) {
	ctx, span := otel.Tracer("aetheris").Start(ctx, name)
	return &Span{otel: span}, ctx
}

// End finishes the span. Safe to call on a zero-value Span.
func (s *Span) End() {
	if s != nil && s.otel != nil {
		s.otel.End()
	}
}

// SetTag sets a key-value attribute on the span.
// Safe to call on a zero-value Span (no-op).
func (s *Span) SetTag(key string, value interface{}) {
	if s == nil || s.otel == nil {
		return
	}
	s.otel.SetAttributes(toKeyValue(key, value))
}

// AddEvent adds a named event to the span.
func (s *Span) AddEvent(name string) {
	if s == nil || s.otel == nil {
		return
	}
	s.otel.AddEvent(name)
}

// toKeyValue converts a key/value pair to an OTel KeyValue.
func toKeyValue(key string, value interface{}) attribute.KeyValue {
	k := attribute.Key(key)
	switch v := value.(type) {
	case string:
		return k.String(v)
	case int:
		return k.Int(v)
	case int64:
		return k.Int64(v)
	case float64:
		return k.Float64(v)
	case bool:
		return k.Bool(v)
	default:
		return k.String(fmt.Sprintf("%v", v))
	}
}
