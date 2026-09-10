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

package executor

import (
	"context"
	"strings"
	"testing"
)

// --- mock providers ---

type mockSearchProvider struct {
	results map[string]string
	err     error
}

func (m *mockSearchProvider) Search(_ context.Context, query string) (string, error) {
	if m.err != nil {
		return "", m.err
	}
	if r, ok := m.results[query]; ok {
		return r, nil
	}
	return "no results found for: " + query, nil
}

type mockWeatherProvider struct {
	results map[string]string
	err     error
}

func (m *mockWeatherProvider) GetWeather(_ context.Context, city string) (string, error) {
	if m.err != nil {
		return "", m.err
	}
	if r, ok := m.results[city]; ok {
		return r, nil
	}
	return city + ": weather data unavailable", nil
}

// --- calculator tests (unchanged behavior) ---

func TestBuiltinTools_Calculator_Add(t *testing.T) {
	b := NewBuiltinTools()
	result, err := b.Execute(context.Background(), "calculator", map[string]any{
		"operation": "add",
		"value1":    float64(3),
		"value2":    float64(4),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "7" {
		t.Errorf("expected 7, got %s", result)
	}
}

func TestBuiltinTools_Calculator_DivideByZero(t *testing.T) {
	b := NewBuiltinTools()
	_, err := b.Execute(context.Background(), "calculator", map[string]any{
		"operation": "divide",
		"value1":    float64(1),
		"value2":    float64(0),
	})
	if err == nil {
		t.Fatal("expected error for division by zero")
	}
}

// --- search tests ---

func TestBuiltinTools_Search_NoProvider_ReturnsError(t *testing.T) {
	b := NewBuiltinTools()
	_, err := b.Execute(context.Background(), "search", map[string]any{
		"query": "test",
	})
	if err == nil {
		t.Fatal("expected error when no search provider configured")
	}
}

func TestBuiltinTools_Search_NoProvider_DoesNotReturnFixedData(t *testing.T) {
	b := NewBuiltinTools()
	result, _ := b.Execute(context.Background(), "search", map[string]any{
		"query": "test",
	})
	// Should NOT contain the old fixed "Result A/B/C" data
	if strings.Contains(result, "Result A") || strings.Contains(result, "Result B") || strings.Contains(result, "Result C") {
		t.Error("search without provider should not return fixed mock data")
	}
}

func TestBuiltinTools_Search_WithProvider_ReturnsRealResults(t *testing.T) {
	b := NewBuiltinToolsWithProviders(
		&mockSearchProvider{results: map[string]string{"golang": "Go is a programming language"}},
		nil,
	)
	result, err := b.Execute(context.Background(), "search", map[string]any{
		"query": "golang",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result, "Go is a programming language") {
		t.Errorf("expected real search result, got %s", result)
	}
}

func TestBuiltinTools_Search_SetProvider(t *testing.T) {
	b := NewBuiltinTools()
	b.SetSearchProvider(&mockSearchProvider{results: map[string]string{"test": "found it"}})
	result, err := b.Execute(context.Background(), "search", map[string]any{
		"query": "test",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "found it" {
		t.Errorf("expected 'found it', got %s", result)
	}
}

// --- weather tests ---

func TestBuiltinTools_Weather_NoProvider_ReturnsError(t *testing.T) {
	b := NewBuiltinTools()
	_, err := b.Execute(context.Background(), "weather", map[string]any{
		"city": "beijing",
	})
	if err == nil {
		t.Fatal("expected error when no weather provider configured")
	}
}

func TestBuiltinTools_Weather_NoProvider_DoesNotReturnFixedData(t *testing.T) {
	b := NewBuiltinTools()
	result, _ := b.Execute(context.Background(), "weather", map[string]any{
		"city": "beijing",
	})
	// Should NOT contain the old fixed weather data
	if strings.Contains(result, "晴, 25") || strings.Contains(result, "多云, 28") {
		t.Error("weather without provider should not return fixed mock data")
	}
}

func TestBuiltinTools_Weather_WithProvider_ReturnsRealResults(t *testing.T) {
	b := NewBuiltinToolsWithProviders(
		nil,
		&mockWeatherProvider{results: map[string]string{"beijing": "Beijing: 晴, 25°C"}},
	)
	result, err := b.Execute(context.Background(), "weather", map[string]any{
		"city": "beijing",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result, "Beijing: 晴, 25°C") {
		t.Errorf("expected real weather result, got %s", result)
	}
}

func TestBuiltinTools_Weather_SetProvider(t *testing.T) {
	b := NewBuiltinTools()
	b.SetWeatherProvider(&mockWeatherProvider{results: map[string]string{"shanghai": "Shanghai: 多云, 28°C"}})
	result, err := b.Execute(context.Background(), "weather", map[string]any{
		"city": "shanghai",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result, "Shanghai: 多云, 28°C") {
		t.Errorf("expected real weather result, got %s", result)
	}
}

// --- unknown tool ---

func TestBuiltinTools_UnknownTool(t *testing.T) {
	b := NewBuiltinTools()
	_, err := b.Execute(context.Background(), "nonexistent", map[string]any{})
	if err == nil {
		t.Fatal("expected error for unknown tool")
	}
}
