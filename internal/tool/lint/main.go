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
	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, path, func(info os.FileInfo) bool {
		return !strings.HasSuffix(info.Name(), "_test.go")
	}, parser.ParseComments)

	if err != nil {
		fmt.Printf("Error parsing %s: %v\n", path, err)
		return
	}

	for _, pkg := range pkgs {
		for _, file := range pkg.Files {
			lintFile(fset, file)
		}
	}
}

func lintFile(fset *token.FileSet, file *ast.File) {
	filename := fset.Position(file.Pos()).Filename

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
	// 检查 Schema 的 Properties 是否都有 Description
	// 这是一个简化检查，实际应该更复杂
	if fn.Doc == nil {
		addWarning(filename, fn, "info", "Tool function should have documentation")
	}
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

func getPos(node ast.Node) token.Position {
	// 简化实现
	return token.Position{
		Line:   1,
		Column: 1,
	}
}
