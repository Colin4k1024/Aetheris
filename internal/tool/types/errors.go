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

package types

import (
	"context"
	"encoding/json"
)

// Endpoint HTTP 端点定义
type Endpoint struct {
	Path    string
	Method  string
	Handler func(ctx context.Context, params json.RawMessage) (interface{}, error)
}

// HTTPError HTTP 错误
type HTTPError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func (e *HTTPError) Error() string {
	return e.Message
}

// NewHTTPError 创建 HTTP 错误
func NewHTTPError(code int, message string) *HTTPError {
	return &HTTPError{Code: code, Message: message}
}

// ValidationError 参数验证错误
type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

func (e *ValidationError) Error() string {
	return e.Field + ": " + e.Message
}

// MissingParameterError 缺少参数错误
type MissingParameterError struct {
	Parameter string `json:"parameter"`
}

func (e *MissingParameterError) Error() string {
	return "missing required parameter: " + e.Parameter
}

// InvalidTypeError 类型错误
type InvalidTypeError struct {
	Parameter string
	Expected  string
	Actual    string
}

func (e *InvalidTypeError) Error() string {
	return "invalid type for parameter " + e.Parameter + ": expected " + e.Expected + ", got " + e.Actual
}

// NetworkError 网络错误
type NetworkError struct {
	URL    string
	Status int
	Body   string
}

func (e *NetworkError) Error() string {
	return "network error for " + e.URL + ": status " + string(rune(e.Status)) + ", body: " + e.Body
}
