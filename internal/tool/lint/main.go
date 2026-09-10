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

// Tool linter: 检查工具描述符的有效性
// 用法: go run ./internal/tool/lint/main.go ./internal/tool/...

package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"strings"
)

type LintError struct {
	File   string
	Line   int
	Column int
	Type   string // "error", "warning", "info"
	Msg    string
}

var (
	errors   []LintError
	warnings []LintError
)

func main() {
	flag.Parse()
	paths := flag.Args()

	if len(paths) == 0 {
		fmt.Println("Usage: toollint <package-paths>...")
		os.Exit(1)
	}

	for _, path := range paths {
		lintPath(path)
	}

	// 输出结果
	if len(errors) > 0 {
		fmt.Println("\n❌ Errors:")
		for _, e := range errors {
			fmt.Printf("  %s:%d:%d [%s] %s\n", e.File, e.Line, e.Column, e.Type, e.Msg)
		}
	}

	if len(warnings) > 0 {
		fmt.Println("\n⚠️  Warnings:")
		for _, w := range warnings {
			fmt.Printf("  %s:%d:%d [%s] %s\n", w.File, w.Line, w.Column, w.Type, w.Msg)
		}
	}

	if len(errors) == 0 && len(warnings) == 0 {
		fmt.Println("✅ No issues found")
	}

	if len(errors) > 0 {
		os.Exit(1)
	}
}

func lintPath(path string) {
	pkgFset = token.NewFileSet()
	pkgs, err := parser.ParseDir(pkgFset, path, func(info os.FileInfo) bool {
		return !strings.HasSuffix(info.Name(), "_test.go")
	}, parser.ParseComments)

	if err != nil {
		fmt.Printf("Error parsing %s: %v\n", path, err)
		return
	}

	for _, pkg := range pkgs {
		for _, file := range pkg.Files {
			lintFile(file)
		}
	}
}

func lintFile(file *ast.File) {
	filename := pkgFset.Position(file.Pos()).Filename

	// 检查工具定义
	ast.Inspect(file, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.TypeSpec:
			if x.Name.Name == "ToolDescriptor" {
				lintToolDescriptor(filename, x)
			}
		case *ast.FuncDecl:
			// 检查返回 Schema 的函数
			if x.Type.Results != nil {
				for _, result := range x.Type.Results.List {
					if isSchemaType(result.Type) {
						lintSchemaFunc(filename, x)
					}
				}
			}
		}
		return true
	})
}

func isSchemaType(expr ast.Expr) bool {
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name == "Schema" || t.Name == "ParameterConstraint"
	case *ast.StarExpr:
		return isSchemaType(t.X)
	}
	return false
}

func lintToolDescriptor(filename string, spec *ast.TypeSpec) {
	if spec.Doc == nil {
		addError(filename, spec, "warning", "ToolDescriptor missing documentation")
	}

	// 检查必需字段
	hasName := false
	hasDescription := false
	hasParameters := false

	if st, ok := spec.Type.(*ast.StructType); ok {
		for _, field := range st.Fields.List {
			for _, name := range field.Names {
				switch name.Name {
				case "Name":
					hasName = true
				case "Description":
					hasDescription = true
				case "Parameters":
					hasParameters = true
				}
			}
		}
	}

	if !hasName {
		addError(filename, spec, "error", "ToolDescriptor must have Name field")
	}
	if !hasDescription {
		addWarning(filename, spec, "warning", "ToolDescriptor should have Description field")
	}
	if !hasParameters {
		addWarning(filename, spec, "warning", "ToolDescriptor should have Parameters field")
	}
}

func lintSchemaFunc(filename string, fn *ast.FuncDecl) {
	// Check documentation
	if fn.Doc == nil {
		addWarning(filename, fn, "info", "Tool function should have documentation")
	}

	// Scan function body for schema construction containing "properties"
	// and verify each property has a "description" key.
	ast.Inspect(fn, func(n ast.Node) bool {
		cl, ok := n.(*ast.CompositeLit)
		if !ok {
			return true
		}

		// Look for map[string]any{...} literals containing a "properties" key
		propKV, hasProperties := findMapKey(cl, "properties")
		if !hasProperties {
			return true
		}

		// The value of "properties" should itself be a map literal;
		// check each property for a "description" sub-key.
		propMap, ok := propKV.Value.(*ast.CompositeLit)
		if !ok {
			return true
		}

		for _, el := range propMap.Elts {
			kv, ok := el.(*ast.KeyValueExpr)
			if !ok {
				continue
			}
			// Each property value should be a map with at least a "description"
			propValue, ok := kv.Value.(*ast.CompositeLit)
			if !ok {
				continue
			}
			if _, hasDesc := findMapKey(propValue, "description"); !hasDesc {
				propName := litString(kv.Key)
				addWarning(filename, fn, "warning",
					fmt.Sprintf("schema property %q missing description", propName))
			}
		}
		return true
	})
}

// findMapKey searches a CompositeLit (map literal) for a key-value pair
// whose key string matches name. Returns the KV expr and whether it was found.
func findMapKey(cl *ast.CompositeLit, name string) (*ast.KeyValueExpr, bool) {
	for _, el := range cl.Elts {
		kv, ok := el.(*ast.KeyValueExpr)
		if !ok {
			continue
		}
		if litString(kv.Key) == name {
			return kv, true
		}
	}
	return nil, false
}

// litString extracts a string value from a basic literal AST expression.
func litString(expr ast.Expr) string {
	if bl, ok := expr.(*ast.BasicLit); ok && bl.Kind == token.STRING {
		return strings.Trim(bl.Value, "\"'")
	}
	return ""
}

// 导出 JSON 格式的 lint 结果
func ExportJSON() {
	data, _ := json.MarshalIndent(struct {
		Errors   []LintError `json:"errors"`
		Warnings []LintError `json:"warnings"`
	}{
		Errors:   errors,
		Warnings: warnings,
	}, "", "  ")
	_, _ = os.Stdout.Write(data)
}

func addError(filename string, node ast.Node, typ, msg string) {
	pos := getPos(node)
	errors = append(errors, LintError{
		File:   filename,
		Line:   pos.Line,
		Column: pos.Column,
		Type:   typ,
		Msg:    msg,
	})
}

func addWarning(filename string, node ast.Node, typ, msg string) {
	pos := getPos(node)
	warnings = append(warnings, LintError{
		File:   filename,
		Line:   pos.Line,
		Column: pos.Column,
		Type:   typ,
		Msg:    msg,
	})
}

// pkgFset is the file set shared across linting passes, used to resolve
// AST node positions to real file:line:column.
var pkgFset *token.FileSet

func getPos(node ast.Node) token.Position {
	if node == nil || pkgFset == nil {
		return token.Position{Line: 1, Column: 1}
	}
	return pkgFset.Position(node.Pos())
}
