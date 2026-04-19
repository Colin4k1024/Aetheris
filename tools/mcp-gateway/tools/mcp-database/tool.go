// Copyright 2026 Aetheris Team
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

package mcp

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/Colin4k1024/Aetheris/v2/internal/tool"
	_ "github.com/go-sql-driver/mysql"
	_ "github.com/jackc/pgx/v5/stdlib"
)

// DatabaseConfig 数据库工具配置
type DatabaseConfig struct {
	// DSN 数据库连接字符串
	DSN string
	// Driver 数据库驱动: postgres, mysql
	Driver string
	// Timeout 查询超时时间
	Timeout time.Duration
	// MaxRows 最大返回行数
	MaxRows int
	// AllowedTables 允许访问的表（为空则允许所有表）
	AllowedTables []string
}

// DatabaseTool 数据库 MCP 工具
// 提供数据库查询和操作能力，支持 PostgreSQL 和 MySQL
type DatabaseTool struct {
	db     *sql.DB
	config *DatabaseConfig
}

// NewDatabaseTool 创建数据库工具实例
func NewDatabaseTool(config *DatabaseConfig) (*DatabaseTool, error) {
	if config == nil {
		config = &DatabaseConfig{}
	}
	if config.Driver == "" {
		config.Driver = "postgres"
	}
	if config.Timeout == 0 {
		config.Timeout = 30 * time.Second
	}
	if config.MaxRows == 0 {
		config.MaxRows = 1000
	}

	// 从环境变量获取 DSN（如果未提供）
	dsn := config.DSN
	if dsn == "" {
		switch config.Driver {
		case "postgres":
			dsn = os.Getenv("POSTGRES_DSN")
			if dsn == "" {
				dsn = os.Getenv("DATABASE_URL")
			}
		case "mysql":
			dsn = os.Getenv("MYSQL_DSN")
		}
	}

	if dsn == "" {
		return nil, fmt.Errorf("database DSN is required")
	}

	db, err := sql.Open(config.Driver, dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// 配置连接池
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	// 验证连接
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &DatabaseTool{
		db:     db,
		config: config,
	}, nil
}

// Name 返回工具名称
func (t *DatabaseTool) Name() string {
	return "mcp-database"
}

// Description 返回工具描述
func (t *DatabaseTool) Description() string {
	return "Database query tool supporting PostgreSQL and MySQL"
}

// Schema 返回参数 Schema
func (t *DatabaseTool) Schema() tool.Schema {
	return tool.Schema{
		Type: "object",
		Properties: map[string]tool.SchemaProperty{
			"action": {
				Type:        "string",
				Description: "Action to perform: query, execute, list_tables, describe_table",
			},
			"sql": {
				Type:        "string",
				Description: "SQL query (for query and execute actions)",
			},
			"table": {
				Type:        "string",
				Description: "Table name (for describe_table action)",
			},
			"limit": {
				Type:        "integer",
				Description: "Maximum number of rows to return (default: 100)",
			},
			"params": {
				Type:        "array",
				Description: "Query parameters for parameterized queries",
			},
		},
		Required: []string{"action"},
	}
}

// Execute 执行数据库操作
func (t *DatabaseTool) Execute(ctx context.Context, input map[string]any) (tool.ToolResult, error) {
	action, _ := input["action"].(string)

	if action == "" {
		return tool.ToolResult{Err: "action is required"}, nil
	}

	// 创建带超时的 context
	ctx, cancel := context.WithTimeout(ctx, t.config.Timeout)
	defer cancel()

	var result string
	var err error

	switch action {
	case "query":
		result, err = t.query(ctx, input)
	case "execute":
		result, err = t.execute(ctx, input)
	case "list_tables":
		result, err = t.listTables(ctx)
	case "describe_table":
		result, err = t.describeTable(ctx, input)
	default:
		return tool.ToolResult{Err: fmt.Sprintf("unknown action: %s", action)}, nil
	}

	if err != nil {
		return tool.ToolResult{Err: err.Error()}, nil
	}

	return tool.ToolResult{Content: result}, nil
}

// validateSQL 验证 SQL 安全性
func (t *DatabaseTool) validateSQL(sqlStr string) error {
	sqlStr = strings.TrimSpace(strings.ToLower(sqlStr))

	// 禁止的 SQL 关键字
	forbidden := []string{
		"drop ", "truncate ", "alter ", "create ", "grant ", "revoke ",
		"shutdown", "pg_", "information_schema", "mysql.", "sys.",
	}

	for _, kw := range forbidden {
		if strings.Contains(sqlStr, kw) {
			return fmt.Errorf("forbidden SQL keyword: %s", kw)
		}
	}

	// 验证表访问权限
	if t.config.AllowedTables != nil && len(t.config.AllowedTables) > 0 {
		// 简单检查 FROM/INTO/JOIN 子句中的表名
		tables := extractTables(sqlStr)
		for _, table := range tables {
			allowed := false
			for _, allowedTable := range t.config.AllowedTables {
				if table == allowedTable {
					allowed = true
					break
				}
			}
			if !allowed {
				return fmt.Errorf("table %s is not in allowed list", table)
			}
		}
	}

	return nil
}

// extractTables 从 SQL 中提取表名（简单实现）
func extractTables(sqlStr string) []string {
	var tables []string

	// 匹配 FROM xxx 和 JOIN xxx
	patterns := []string{
		`\bfrom\s+(\w+)`,
		`\binto\s+(\w+)`,
		`\bjoin\s+(\w+)`,
		`\bupdate\s+(\w+)`,
	}

	for _, pattern := range patterns {
		idx := strings.Index(strings.ToLower(sqlStr), strings.Replace(pattern, `\b`, "", 1))
		if idx >= 0 {
			rest := sqlStr[idx:]
			words := strings.Fields(rest)
			if len(words) >= 2 {
				table := strings.Trim(words[1], ",;()")
				if table != "" {
					tables = append(tables, table)
				}
			}
		}
	}

	return tables
}

// query 执行查询
func (t *DatabaseTool) query(ctx context.Context, input map[string]any) (string, error) {
	sqlStr, _ := input["sql"].(string)
	if sqlStr == "" {
		return "", fmt.Errorf("sql is required for query action")
	}

	// 验证 SQL 安全性
	if err := t.validateSQL(sqlStr); err != nil {
		return "", fmt.Errorf("sql validation failed: %w", err)
	}

	limit := t.config.MaxRows
	if l, ok := input["limit"].(float64); ok {
		limit = int(l)
		if limit <= 0 {
			limit = t.config.MaxRows
		}
		if limit > t.config.MaxRows {
			limit = t.config.MaxRows
		}
	}

	// 提取查询参数
	var params []any
	if p, ok := input["params"].([]any); ok {
		params = p
	}

	// 执行查询
	var rows *sql.Rows
	var err error

	if len(params) > 0 {
		rows, err = t.db.QueryContext(ctx, sqlStr, params...)
	} else {
		rows, err = t.db.QueryContext(ctx, sqlStr)
	}

	if err != nil {
		return "", fmt.Errorf("query failed: %w", err)
	}
	defer rows.Close()

	// 获取列信息
	columns, err := rows.Columns()
	if err != nil {
		return "", fmt.Errorf("failed to get columns: %w", err)
	}

	// 准备结果
	var results []map[string]any
	rowCount := 0

	for rows.Next() {
		if rowCount >= limit {
			break
		}

		// 创建列值的容器
		values := make([]any, len(columns))
		valuePtrs := make([]any, len(columns))
		for i := range values {
			valuePtrs[i] = &values[i]
		}

		// 扫描行
		if err := rows.Scan(valuePtrs...); err != nil {
			return "", fmt.Errorf("failed to scan row: %w", err)
		}

		// 转换为 map
		row := make(map[string]any)
		for i, col := range columns {
			val := values[i]
			// 处理字节切片（常见于数据库驱动）
			if b, ok := val.([]byte); ok {
				row[col] = string(b)
			} else {
				row[col] = val
			}
		}

		results = append(results, row)
		rowCount++
	}

	if err := rows.Err(); err != nil {
		return "", fmt.Errorf("rows error: %w", err)
	}

	output := map[string]any{
		"columns":   columns,
		"rows":      results,
		"count":     len(results),
		"truncated": rowCount >= limit,
	}

	data, _ := json.Marshal(output)
	return string(data), nil
}

// execute 执行 DML/DDL
func (t *DatabaseTool) execute(ctx context.Context, input map[string]any) (string, error) {
	sqlStr, _ := input["sql"].(string)
	if sqlStr == "" {
		return "", fmt.Errorf("sql is required for execute action")
	}

	sqlStr = strings.TrimSpace(strings.ToLower(sqlStr))

	// 验证 SQL 类型（只允许 INSERT, UPDATE, DELETE）
	allowedPrefixes := []string{"insert ", "update ", "delete ", "set "}
	allowed := false
	for _, prefix := range allowedPrefixes {
		if strings.HasPrefix(sqlStr, prefix) {
			allowed = true
			break
		}
	}
	if !allowed {
		return "", fmt.Errorf("only INSERT, UPDATE, DELETE, and SET statements are allowed for execute action")
	}

	// 提取查询参数
	var params []any
	if p, ok := input["params"].([]any); ok {
		params = p
	}

	// 执行
	var result sql.Result
	var err error

	if len(params) > 0 {
		result, err = t.db.ExecContext(ctx, sqlStr, params...)
	} else {
		result, err = t.db.ExecContext(ctx, sqlStr)
	}

	if err != nil {
		return "", fmt.Errorf("execute failed: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	lastInsertId, _ := result.LastInsertId()

	output := map[string]any{
		"success":       true,
		"rows_affected": rowsAffected,
	}
	if lastInsertId > 0 {
		output["last_insert_id"] = lastInsertId
	}

	data, _ := json.Marshal(output)
	return string(data), nil
}

// listTables 列出所有表
func (t *DatabaseTool) listTables(ctx context.Context) (string, error) {
	var sqlStr string

	switch t.config.Driver {
	case "postgres":
		sqlStr = `SELECT table_name FROM information_schema.tables WHERE table_schema = 'public' ORDER BY table_name`
	case "mysql":
		sqlStr = `SHOW TABLES`
	default:
		return "", fmt.Errorf("unsupported driver: %s", t.config.Driver)
	}

	rows, err := t.db.QueryContext(ctx, sqlStr)
	if err != nil {
		return "", fmt.Errorf("list tables failed: %w", err)
	}
	defer rows.Close()

	var tables []string
	for rows.Next() {
		var table string
		if err := rows.Scan(&table); err != nil {
			continue
		}
		tables = append(tables, table)
	}

	output := map[string]any{
		"tables": tables,
		"count":  len(tables),
	}

	data, _ := json.Marshal(output)
	return string(data), nil
}

// describeTable 描述表结构
func (t *DatabaseTool) describeTable(ctx context.Context, input map[string]any) (string, error) {
	table, _ := input["table"].(string)
	if table == "" {
		return "", fmt.Errorf("table is required for describe_table action")
	}

	// 验证表权限
	if t.config.AllowedTables != nil && len(t.config.AllowedTables) > 0 {
		allowed := false
		for _, allowedTable := range t.config.AllowedTables {
			if table == allowedTable {
				allowed = true
				break
			}
		}
		if !allowed {
			return "", fmt.Errorf("table %s is not in allowed list", table)
		}
	}

	var sqlStr string
	var args []any

	switch t.config.Driver {
	case "postgres":
		sqlStr = `
			SELECT column_name, data_type, is_nullable, column_default
			FROM information_schema.columns
			WHERE table_name = $1
			ORDER BY ordinal_position`
		args = []any{table}
	case "mysql":
		// Use information_schema with parameterized query to prevent SQL injection
		sqlStr = `
			SELECT column_name, data_type, is_nullable, column_default
			FROM information_schema.columns
			WHERE table_name = ?
			ORDER BY ordinal_position`
		args = []any{table}
	default:
		return "", fmt.Errorf("unsupported driver: %s", t.config.Driver)
	}

	rows, err := t.db.QueryContext(ctx, sqlStr, args...)
	if err != nil {
		return "", fmt.Errorf("describe table failed: %w", err)
	}
	defer rows.Close()

	var columns []map[string]any
	for rows.Next() {
		row := make(map[string]any)

		switch t.config.Driver {
		case "postgres":
			var colName, dataType, isNullable, defaultVal sql.NullString
			if err := rows.Scan(&colName, &dataType, &isNullable, &defaultVal); err != nil {
				continue
			}
			row["name"] = colName.String
			row["type"] = dataType.String
			row["nullable"] = isNullable.String == "YES"
			row["default"] = defaultVal.String
		case "mysql":
			var field, typ, null, key, extra sql.NullString
			var defaultVal sql.NullString
			if err := rows.Scan(&field, &typ, &null, &key, &defaultVal, &extra); err != nil {
				continue
			}
			row["field"] = field.String
			row["type"] = typ.String
			row["null"] = null.String
			row["key"] = key.String
			row["default"] = defaultVal.String
			row["extra"] = extra.String
		}

		columns = append(columns, row)
	}

	output := map[string]any{
		"table":   table,
		"columns": columns,
		"count":   len(columns),
	}

	data, _ := json.Marshal(output)
	return string(data), nil
}

// Close 关闭数据库连接
func (t *DatabaseTool) Close() error {
	if t.db != nil {
		return t.db.Close()
	}
	return nil
}
