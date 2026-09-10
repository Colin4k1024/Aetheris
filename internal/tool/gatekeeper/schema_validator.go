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
	"encoding/json"
	"fmt"
	"strings"

	"github.com/Colin4k1024/Aetheris/v2/internal/tool/types"
)

// schemaMap 内部表示 JSON Schema 的解析结果
type schemaMap = map[string]interface{}

// validateAgainstSchema 递归校验 value 是否符合 schema。
// path 为当前字段的路径（用于错误消息），根路径为空字符串。
func validateAgainstSchema(value interface{}, schema schemaMap, path string) error {
	if schema == nil {
		return nil
	}

	// type 关键字
	if typeVal, ok := schema["type"]; ok {
		if typeStr, ok := typeVal.(string); ok {
			if err := validateType(value, typeStr, path); err != nil {
				return err
			}
		}
	}

	// enum 关键字
	if enumVal, ok := schema["enum"]; ok {
		if err := validateEnum(value, enumVal, path); err != nil {
			return err
		}
	}

	// object 类型校验
	if obj, ok := value.(map[string]interface{}); ok {
		// required 关键字
		if reqVal, ok := schema["required"]; ok {
			if err := validateRequired(obj, reqVal, path); err != nil {
				return err
			}
		}

		// properties 关键字
		props, hasProps := schema["properties"].(map[string]interface{})
		if hasProps {
			for key, propSchema := range props {
				propVal, exists := obj[key]
				if !exists {
					continue
				}
				fieldPath := key
				if path != "" {
					fieldPath = path + "." + key
				}
				if propMap, ok := propSchema.(map[string]interface{}); ok {
					if err := validateAgainstSchema(propVal, propMap, fieldPath); err != nil {
						return err
					}
				}
			}
		}

		// additionalProperties 关键字
		if addProps, ok := schema["additionalProperties"]; ok {
			if boolVal, ok := addProps.(bool); ok && !boolVal {
				// additionalProperties: false → 不允许未声明属性
				for key := range obj {
					if props == nil {
						if _, declared := props[key]; !declared {
							return &types.ValidationError{
								Field:   buildPath(path, key),
								Message: "additional property not allowed",
							}
						}
					} else {
						if _, declared := props[key]; !declared {
							return &types.ValidationError{
								Field:   buildPath(path, key),
								Message: "additional property not allowed",
							}
						}
					}
				}
			}
		}
	}

	// array 类型校验
	if arr, ok := value.([]interface{}); ok {
		// items 关键字
		if itemsVal, ok := schema["items"]; ok {
			if itemsSchema, ok := itemsVal.(map[string]interface{}); ok {
				for i, elem := range arr {
					elemPath := fmt.Sprintf("%s[%d]", path, i)
					if path == "" {
						elemPath = fmt.Sprintf("[%d]", i)
					}
					if err := validateAgainstSchema(elem, itemsSchema, elemPath); err != nil {
						return err
					}
				}
			}
		}
		// minItems 关键字
		if minItemsVal, ok := schema["minItems"]; ok {
			if minNum, ok := toFloat(minItemsVal); ok {
				if float64(len(arr)) < minNum {
					return &types.ValidationError{
						Field:   path,
						Message: fmt.Sprintf("array length %d less than minItems %v", len(arr), minItemsVal),
					}
				}
			}
		}
	}

	// string 类型校验
	if strVal, ok := value.(string); ok {
		if minLen, ok := schema["minLength"]; ok {
			if minNum, ok := toFloat(minLen); ok {
				if float64(len(strVal)) < minNum {
					return &types.ValidationError{
						Field:   path,
						Message: fmt.Sprintf("string length %d less than minLength %v", len(strVal), minLen),
					}
				}
			}
		}
		if maxLen, ok := schema["maxLength"]; ok {
			if maxNum, ok := toFloat(maxLen); ok {
				if float64(len(strVal)) > maxNum {
					return &types.ValidationError{
						Field:   path,
						Message: fmt.Sprintf("string length %d greater than maxLength %v", len(strVal), maxLen),
					}
				}
			}
		}
	}

	// number 类型校验
	if num, ok := toFloat(value); ok {
		if minVal, ok := schema["minimum"]; ok {
			if minNum, ok := toFloat(minVal); ok {
				if num < minNum {
					return &types.ValidationError{
						Field:   path,
						Message: fmt.Sprintf("value %v less than minimum %v", value, minVal),
					}
				}
			}
		}
		if maxVal, ok := schema["maximum"]; ok {
			if maxNum, ok := toFloat(maxVal); ok {
				if num > maxNum {
					return &types.ValidationError{
						Field:   path,
						Message: fmt.Sprintf("value %v greater than maximum %v", value, maxVal),
					}
				}
			}
		}
	}

	return nil
}

// validateType 校验值类型
func validateType(value interface{}, expectedType string, path string) error {
	switch expectedType {
	case "string":
		if _, ok := value.(string); !ok {
			return &types.ValidationError{Field: path, Message: fmt.Sprintf("expected string, got %T", value)}
		}
	case "number":
		if _, ok := toFloat(value); !ok {
			return &types.ValidationError{Field: path, Message: fmt.Sprintf("expected number, got %T", value)}
		}
	case "integer":
		if f, ok := toFloat(value); ok {
			if f != float64(int64(f)) {
				return &types.ValidationError{Field: path, Message: "expected integer, got non-integer number"}
			}
		} else {
			return &types.ValidationError{Field: path, Message: fmt.Sprintf("expected integer, got %T", value)}
		}
	case "boolean":
		if _, ok := value.(bool); !ok {
			return &types.ValidationError{Field: path, Message: fmt.Sprintf("expected boolean, got %T", value)}
		}
	case "object":
		if _, ok := value.(map[string]interface{}); !ok {
			return &types.ValidationError{Field: path, Message: fmt.Sprintf("expected object, got %T", value)}
		}
	case "array":
		if _, ok := value.([]interface{}); !ok {
			return &types.ValidationError{Field: path, Message: fmt.Sprintf("expected array, got %T", value)}
		}
	case "null":
		if value != nil {
			return &types.ValidationError{Field: path, Message: "expected null"}
		}
	}
	return nil
}

// validateEnum 校验值在枚举列表中
func validateEnum(value interface{}, enumVal interface{}, path string) error {
	enumSlice, ok := enumVal.([]interface{})
	if !ok {
		return nil
	}
	for _, e := range enumSlice {
		if jsonEqual(e, value) {
			return nil
		}
	}
	return &types.ValidationError{
		Field:   path,
		Message: fmt.Sprintf("value %v not in enum", value),
	}
}

// validateRequired 校验 required 字段存在
func validateRequired(obj map[string]interface{}, reqVal interface{}, path string) error {
	reqSlice, ok := reqVal.([]interface{})
	if !ok {
		return nil
	}
	for _, r := range reqSlice {
		field, ok := r.(string)
		if !ok {
			continue
		}
		if _, exists := obj[field]; !exists {
			return &types.ValidationError{
				Field:   buildPath(path, field),
				Message: "required field is missing",
			}
		}
	}
	return nil
}

// buildPath 拼接字段路径
func buildPath(parent, field string) string {
	if parent == "" {
		return field
	}
	return parent + "." + field
}

// toFloat 尝试将 interface{} 转为 float64（JSON 数字统一解码为 float64）
func toFloat(v interface{}) (float64, bool) {
	switch n := v.(type) {
	case float64:
		return n, true
	case float32:
		return float64(n), true
	case int:
		return float64(n), true
	case int64:
		return float64(n), true
	case json.Number:
		f, err := n.Float64()
		return f, err == nil
	}
	return 0, false
}

// jsonEqual 简单比较两个 JSON 值
func jsonEqual(a, b interface{}) bool {
	aj, _ := json.Marshal(a)
	bj, _ := json.Marshal(b)
	return string(aj) == string(bj)
}

// ValidateSchema 验证 JSON 数据是否符合 JSON Schema。
// data 为 JSON 字符串，schema 为 JSON Schema 字符串。
// schema 为空时仅校验 data 是否为合法 JSON。
// 支持关键字：type, required, enum, properties, items, additionalProperties,
// minLength, maxLength, minimum, maximum, minItems。
func ValidateSchema(data string, schema string) error {
	if data == "" {
		return &types.ValidationError{Field: "data", Message: "data cannot be empty"}
	}

	// 真实 JSON 解码（支持多行）
	var value interface{}
	decoder := json.NewDecoder(strings.NewReader(data))
	decoder.UseNumber()
	if err := decoder.Decode(&value); err != nil {
		return &types.ValidationError{
			Field:   "data",
			Message: fmt.Sprintf("invalid JSON: %v", err),
		}
	}

	// 无 schema 时仅校验 JSON 合法性
	if schema == "" {
		return nil
	}

	// 解码 schema
	var schemaMapVal schemaMap
	decoder = json.NewDecoder(strings.NewReader(schema))
	decoder.UseNumber()
	if err := decoder.Decode(&schemaMapVal); err != nil {
		return &types.ValidationError{
			Field:   "schema",
			Message: fmt.Sprintf("invalid schema JSON: %v", err),
		}
	}

	// 递归校验
	return validateAgainstSchema(value, schemaMapVal, "")
}
