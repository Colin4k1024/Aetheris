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

package gatekeeper

import (
	"testing"
)

// --- basic validation ---

func TestValidateSchema_EmptyData(t *testing.T) {
	err := ValidateSchema("", "")
	if err == nil {
		t.Fatal("expected error for empty data")
	}
}

func TestValidateSchema_InvalidJSON(t *testing.T) {
	err := ValidateSchema("not json", "")
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestValidateSchema_MultilineJSON(t *testing.T) {
	data := `{
		"name": "test",
		"age": 30
	}`
	err := ValidateSchema(data, "")
	if err != nil {
		t.Fatalf("unexpected error for multiline JSON: %v", err)
	}
}

func TestValidateSchema_NoSchema_ValidJSON(t *testing.T) {
	err := ValidateSchema(`{"key": "value"}`, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateSchema_InvalidSchema(t *testing.T) {
	err := ValidateSchema(`{"a":1}`, "not valid schema")
	if err == nil {
		t.Fatal("expected error for invalid schema")
	}
}

// --- type validation ---

func TestValidateSchema_Type_String_Pass(t *testing.T) {
	schema := `{"type": "string"}`
	err := ValidateSchema(`"hello"`, schema)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateSchema_Type_String_Fail(t *testing.T) {
	schema := `{"type": "string"}`
	err := ValidateSchema("123", schema)
	if err == nil {
		t.Fatal("expected type error: string expected, got number")
	}
}

func TestValidateSchema_Type_Number_Pass(t *testing.T) {
	schema := `{"type": "number"}`
	err := ValidateSchema("3.14", schema)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateSchema_Type_Integer_Pass(t *testing.T) {
	schema := `{"type": "integer"}`
	err := ValidateSchema("42", schema)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateSchema_Type_Integer_Fail(t *testing.T) {
	schema := `{"type": "integer"}`
	err := ValidateSchema("3.14", schema)
	if err == nil {
		t.Fatal("expected error: 3.14 is not integer")
	}
}

func TestValidateSchema_Type_Boolean_Pass(t *testing.T) {
	schema := `{"type": "boolean"}`
	err := ValidateSchema("true", schema)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateSchema_Type_Object_Fail(t *testing.T) {
	schema := `{"type": "object"}`
	err := ValidateSchema(`"string"`, schema)
	if err == nil {
		t.Fatal("expected error: string is not object")
	}
}

func TestValidateSchema_Type_Array_Pass(t *testing.T) {
	schema := `{"type": "array"}`
	err := ValidateSchema("[1,2,3]", schema)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// --- required ---

func TestValidateSchema_Required_Pass(t *testing.T) {
	schema := `{"type": "object", "required": ["name", "age"], "properties": {"name": {"type": "string"}, "age": {"type": "integer"}}}`
	data := `{"name": "Alice", "age": 30}`
	err := ValidateSchema(data, schema)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateSchema_Required_MissingField(t *testing.T) {
	schema := `{"type": "object", "required": ["name", "age"]}`
	data := `{"name": "Alice"}`
	err := ValidateSchema(data, schema)
	if err == nil {
		t.Fatal("expected error for missing required field 'age'")
	}
}

// --- enum ---

func TestValidateSchema_Enum_Pass(t *testing.T) {
	schema := `{"type": "string", "enum": ["red", "green", "blue"]}`
	err := ValidateSchema(`"red"`, schema)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateSchema_Enum_Fail(t *testing.T) {
	schema := `{"type": "string", "enum": ["red", "green", "blue"]}`
	err := ValidateSchema(`"yellow"`, schema)
	if err == nil {
		t.Fatal("expected error: yellow not in enum")
	}
}

// --- nested objects ---

func TestValidateSchema_NestedObject_Pass(t *testing.T) {
	schema := `{
		"type": "object",
		"properties": {
			"user": {
				"type": "object",
				"required": ["id"],
				"properties": {
					"id": {"type": "integer"},
					"name": {"type": "string"}
				}
			}
		}
	}`
	data := `{"user": {"id": 1, "name": "Alice"}}`
	err := ValidateSchema(data, schema)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateSchema_NestedObject_InvalidInnerType(t *testing.T) {
	schema := `{
		"type": "object",
		"properties": {
			"user": {
				"type": "object",
				"properties": {
					"id": {"type": "integer"}
				}
			}
		}
	}`
	data := `{"user": {"id": "not-an-int"}}`
	err := ValidateSchema(data, schema)
	if err == nil {
		t.Fatal("expected error: inner id should be integer")
	}
}

// --- arrays ---

func TestValidateSchema_ArrayItems_Pass(t *testing.T) {
	schema := `{"type": "array", "items": {"type": "integer"}}`
	err := ValidateSchema("[1, 2, 3]", schema)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateSchema_ArrayItems_Fail(t *testing.T) {
	schema := `{"type": "array", "items": {"type": "integer"}}`
	err := ValidateSchema(`[1, "two", 3]`, schema)
	if err == nil {
		t.Fatal("expected error: second item is not integer")
	}
}

func TestValidateSchema_MinItems_Fail(t *testing.T) {
	schema := `{"type": "array", "minItems": 3, "items": {"type": "integer"}}`
	err := ValidateSchema("[1]", schema)
	if err == nil {
		t.Fatal("expected error: less than minItems")
	}
}

// --- additionalProperties ---

func TestValidateSchema_AdditionalProperties_False_Fail(t *testing.T) {
	schema := `{"type": "object", "properties": {"a": {"type": "string"}}, "additionalProperties": false}`
	data := `{"a": "ok", "b": "extra"}`
	err := ValidateSchema(data, schema)
	if err == nil {
		t.Fatal("expected error: additional property 'b' not allowed")
	}
}

func TestValidateSchema_AdditionalProperties_False_Pass(t *testing.T) {
	schema := `{"type": "object", "properties": {"a": {"type": "string"}}, "additionalProperties": false}`
	data := `{"a": "ok"}`
	err := ValidateSchema(data, schema)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// --- string constraints ---

func TestValidateSchema_MinLength_Fail(t *testing.T) {
	schema := `{"type": "string", "minLength": 5}`
	err := ValidateSchema(`"hi"`, schema)
	if err == nil {
		t.Fatal("expected error: string too short")
	}
}

func TestValidateSchema_MaxLength_Fail(t *testing.T) {
	schema := `{"type": "string", "maxLength": 3}`
	err := ValidateSchema(`"hello"`, schema)
	if err == nil {
		t.Fatal("expected error: string too long")
	}
}

// --- number constraints ---

func TestValidateSchema_Minimum_Fail(t *testing.T) {
	schema := `{"type": "number", "minimum": 10}`
	err := ValidateSchema("5", schema)
	if err == nil {
		t.Fatal("expected error: below minimum")
	}
}

func TestValidateSchema_Maximum_Fail(t *testing.T) {
	schema := `{"type": "number", "maximum": 100}`
	err := ValidateSchema("200", schema)
	if err == nil {
		t.Fatal("expected error: above maximum")
	}
}

// --- complex schema ---

func TestValidateSchema_Complex_Pass(t *testing.T) {
	schema := `{
		"type": "object",
		"required": ["id", "tags"],
		"properties": {
			"id": {"type": "integer", "minimum": 1},
			"name": {"type": "string", "minLength": 1, "maxLength": 50},
			"tags": {"type": "array", "items": {"type": "string"}, "minItems": 1},
			"status": {"type": "string", "enum": ["active", "inactive"]},
			"metadata": {"type": "object", "additionalProperties": false}
		}
	}`
	data := `{
		"id": 1,
		"name": "test",
		"tags": ["a", "b"],
		"status": "active",
		"metadata": {}
	}`
	err := ValidateSchema(data, schema)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateSchema_Complex_Fail(t *testing.T) {
	schema := `{
		"type": "object",
		"required": ["id"],
		"properties": {
			"status": {"type": "string", "enum": ["active", "inactive"]}
		}
	}`
	data := `{"id": 1, "status": "unknown"}`
	err := ValidateSchema(data, schema)
	if err == nil {
		t.Fatal("expected error: invalid enum value")
	}
}

// --- null type ---

func TestValidateSchema_Type_Null_Pass(t *testing.T) {
	schema := `{"type": "null"}`
	err := ValidateSchema("null", schema)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateSchema_Type_Null_Fail(t *testing.T) {
	schema := `{"type": "null"}`
	err := ValidateSchema(`"not null"`, schema)
	if err == nil {
		t.Fatal("expected error: string is not null")
	}
}
