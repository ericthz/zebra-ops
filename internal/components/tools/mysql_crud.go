package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

// MysqlCrudInput MySQL 数据库操作的输入参数
type MysqlCrudInput struct {
	DSN         string `json:"dsn" jsonschema:"description=The Data Source Name for connecting to the MySQL database, including username, password, host, port, and database name"`
	SQL         string `json:"sql" jsonschema:"description=The SQL query to execute against the MySQL database"`
	OperateType string `json:"operate_type" jsonschema:"description=The type of SQL operation to perform: query, insert, update, or delete"`
}

// NewMysqlCrudTool 创建 MySQL 数据库增删改查工具，支持通过 DSN 连接数据库执行 SQL
func NewMysqlCrudTool() (tool.InvokableTool, error) {
	t, err := utils.InferOptionableTool(
		"mysql_crud",
		"Execute SQL queries against the MySQL database and return results in JSON format. Use this tool when you need to query, insert, update or delete data from the database. The results will be formatted as JSON for easy parsing. Only allow SELECT statements when operate_type is query.",
		func(ctx context.Context, input *MysqlCrudInput, opts ...tool.Option) (output string, err error) {
			// 1. 校验参数
			if input == nil || strings.TrimSpace(input.SQL) == "" {
				return "", fmt.Errorf("sql is required")
			}
			operateType := strings.ToLower(strings.TrimSpace(input.OperateType))
			if operateType == "" {
				// 根据 SQL 前缀推断操作类型
				sqlUpper := strings.ToUpper(strings.TrimSpace(input.SQL))
				switch {
				case strings.HasPrefix(sqlUpper, "SELECT"):
					operateType = "query"
				default:
					return "", fmt.Errorf("operate_type is required for non-SELECT statements")
				}
			}

			// 2. 建立数据库连接
			db, err := gorm.Open(mysql.Open(input.DSN), &gorm.Config{})
			if err != nil {
				return "", fmt.Errorf("open mysql connection failed: %w", err)
			}
			sqlDB, err := db.DB()
			if err != nil {
				return "", fmt.Errorf("get underlying sql.DB failed: %w", err)
			}
			defer sqlDB.Close()

			// 3. 执行 SQL
			if operateType == "query" {
				var results []map[string]interface{}
				err = db.Raw(input.SQL).Scan(&results).Error
				if err != nil {
					return "", fmt.Errorf("query failed: %w", err)
				}
				resBytes, err := json.Marshal(results)
				if err != nil {
					return "", fmt.Errorf("marshal query result failed: %w", err)
				}
				return string(resBytes), nil
			}

			err = db.Exec(input.SQL).Error
			if err != nil {
				return "", fmt.Errorf("exec %s failed: %w", operateType, err)
			}
			return fmt.Sprintf("ok, affected rows: %d", db.RowsAffected), nil
		})
	if err != nil {
		return nil, err
	}
	return t, nil
}
