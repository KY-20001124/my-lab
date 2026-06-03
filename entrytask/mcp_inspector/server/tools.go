package server

import "encoding/json"

// Tool 所有MCP工具必须实现的核心接口（唯一定义）
type Tool interface {
	Name() string                                            // 工具名称（如http_health_check）
	Description() string                                     // 工具描述
	InputSchema() json.RawMessage                            // 输入参数JSON Schema
	OutputSchema() json.RawMessage                           // 输出结果JSON Schema
	Call(params map[string]interface{}) (interface{}, error) // 工具执行逻辑
}
