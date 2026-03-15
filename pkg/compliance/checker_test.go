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

package compliance

import (
	"context"
	"testing"
)

func TestNewChecker(t *testing.T) {
	c := NewChecker()
	if c == nil {
		t.Fatal("NewChecker should not return nil")
	}
}

func TestChecker_RegisterFramework(t *testing.T) {
	c := NewChecker()

	framework := NewFramework(StandardHIPAA)
	c.RegisterFramework(framework)

	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.frameworks[StandardHIPAA] == nil {
		t.Error("expected framework to be registered")
	}
}

func TestChecker_CheckTenant(t *testing.T) {
	c := NewChecker()

	framework := NewFramework(StandardHIPAA)
	c.RegisterFramework(framework)

	ctx := context.Background()
	report, err := c.CheckTenant(ctx, "tenant-1", StandardHIPAA)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if report == nil {
		t.Fatal("expected report, got nil")
	}
}

func TestChecker_CheckTenant_UnknownStandard(t *testing.T) {
	c := NewChecker()

	ctx := context.Background()
	_, err := c.CheckTenant(ctx, "tenant-1", "unknown-standard")
	if err == nil {
		t.Error("expected error for unknown standard")
	}
}
