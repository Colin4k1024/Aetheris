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

package main

import (
	"go/ast"
	"go/parser"
	"go/token"
	"strings"
	"testing"
)

func TestGetPos_NilSafe(t *testing.T) {
	pos := getPos(nil)
	if pos.Line != 1 || pos.Column != 1 {
		t.Errorf("expected 1:1 for nil node, got %d:%d", pos.Line, pos.Column)
	}
}

func TestGetPos_RealPosition(t *testing.T) {
	src := `package main

type Foo struct {
	Bar string
}
`
	fset := token.NewFileSet()
	pkgFset = fset
	file, err := parser.ParseFile(fset, "test.go", src, 0)
	if err != nil {
		t.Fatal(err)
	}

	var foundNode ast.Node
	for _, decl := range file.Decls {
		gd, ok := decl.(*ast.GenDecl)
		if !ok {
			continue
		}
		for _, spec := range gd.Specs {
			if ts, ok := spec.(*ast.TypeSpec); ok && ts.Name.Name == "Foo" {
				foundNode = ts
				break
			}
		}
		if foundNode != nil {
			break
		}
	}
	if foundNode == nil {
		t.Fatal("Foo type spec not found")
	}

	pos := getPos(foundNode)
	if pos.Line != 3 {
		t.Errorf("expected line 3, got %d", pos.Line)
	}
	if pos.Column != 6 {
		t.Errorf("expected column 6 (Foo identifier), got %d", pos.Column)
	}
}

func TestLitString(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{`"hello"`, "hello"},
		{`123`, ""},
		{``, ""},
	}
	for _, tt := range tests {
		expr, _ := parser.ParseExpr(tt.input)
		got := litString(expr)
		if got != tt.expected {
			t.Errorf("litString(%q) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}

func TestFindMapKey_Found(t *testing.T) {
	src := `package main

var x = map[string]any{
	"type": "object",
	"properties": map[string]any{},
	"description": "test",
}
`
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "test.go", src, 0)
	if err != nil {
		t.Fatal(err)
	}

	ast.Inspect(file, func(n ast.Node) bool {
		if cl, ok := n.(*ast.CompositeLit); ok {
			kv, found := findMapKey(cl, "properties")
			if found {
				if kv == nil {
					t.Error("found is true but kv is nil")
				}
				return false
			}
		}
		return true
	})
}

func TestFindMapKey_NotFound(t *testing.T) {
	src := `package main

var x = map[string]any{
	"type": "object",
}
`
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "test.go", src, 0)
	if err != nil {
		t.Fatal(err)
	}

	ast.Inspect(file, func(n ast.Node) bool {
		if cl, ok := n.(*ast.CompositeLit); ok {
			_, found := findMapKey(cl, "nonexistent")
			if found {
				t.Error("expected not found")
			}
		}
		return true
	})
}

func TestLintSchemaFunc_MissingDescription(t *testing.T) {
	// Reset state
	errors = nil
	warnings = nil

	src := `package main

// MySchemaFunc returns a schema with properties missing descriptions.
func MySchemaFunc() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"name": map[string]any{
				"type": "string",
			},
			"age": map[string]any{
				"type":        "integer",
				"description": "User age",
			},
		},
	}
}
`
	fset := token.NewFileSet()
	pkgFset = fset
	file, err := parser.ParseFile(fset, "test.go", src, 0)
	if err != nil {
		t.Fatal(err)
	}

	ast.Inspect(file, func(n ast.Node) bool {
		if fn, ok := n.(*ast.FuncDecl); ok {
			lintSchemaFunc("test.go", fn)
		}
		return true
	})

	// Should have a warning for "name" property missing description
	foundWarning := false
	for _, w := range warnings {
		if strings.Contains(w.Msg, "name") && strings.Contains(w.Msg, "description") {
			foundWarning = true
			break
		}
	}
	if !foundWarning {
		t.Error("expected warning for 'name' property missing description")
	}

	// Should NOT have a warning for "age" property (it has description)
	for _, w := range warnings {
		if strings.Contains(w.Msg, "age") && strings.Contains(w.Msg, "missing") {
			t.Error("should not warn about 'age' property which has description")
		}
	}
}

func TestLintSchemaFunc_WithDocumentation(t *testing.T) {
	errors = nil
	warnings = nil

	src := `package main

// documentedFunc is documented.
func documentedFunc() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"x": map[string]any{
				"type":        "string",
				"description": "X value",
			},
		},
	}
}
`
	fset := token.NewFileSet()
	pkgFset = fset
	file, err := parser.ParseFile(fset, "test.go", src, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}

	ast.Inspect(file, func(n ast.Node) bool {
		if fn, ok := n.(*ast.FuncDecl); ok {
			lintSchemaFunc("test.go", fn)
		}
		return true
	})

	// Should have no warnings (function is documented, all properties have descriptions)
	for _, w := range warnings {
		if strings.Contains(w.Msg, "documentation") {
			t.Error("should not warn about missing documentation")
		}
		if strings.Contains(w.Msg, "missing description") {
			t.Error("should not warn about missing property descriptions")
		}
	}
}

func TestLintSchemaFunc_NoDocWarning(t *testing.T) {
	errors = nil
	warnings = nil

	src := `package main

func undocumentedFunc() map[string]any {
	return map[string]any{}
}
`
	fset := token.NewFileSet()
	pkgFset = fset
	file, err := parser.ParseFile(fset, "test.go", src, 0)
	if err != nil {
		t.Fatal(err)
	}

	ast.Inspect(file, func(n ast.Node) bool {
		if fn, ok := n.(*ast.FuncDecl); ok {
			lintSchemaFunc("test.go", fn)
		}
		return true
	})

	foundDocWarning := false
	for _, w := range warnings {
		if strings.Contains(w.Msg, "documentation") {
			foundDocWarning = true
			break
		}
	}
	if !foundDocWarning {
		t.Error("expected documentation warning for undocumented function")
	}
}
