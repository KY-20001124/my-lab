package server

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// MCPServer MCP服务实例
type MCPServer struct {
	tools map[string]Tool // 注册的工具列表
}

// NewMCPServer 创建MCP服务实例
func NewMCPServer() *MCPServer {
	return &MCPServer{
		tools: make(map[string]Tool),
	}
}

// RegisterTool 注册工具（核心方法）
func (s *MCPServer) RegisterTool(tool Tool) {
	s.tools[tool.Name()] = tool
	fmt.Fprintf(os.Stderr, "[INFO] 已注册工具：%s\n", tool.Name())
}

// Run 启动MCP服务（监听stdin，处理请求）
func (s *MCPServer) Run() {
	scanner := bufio.NewScanner(os.Stdin)
	fmt.Fprintf(os.Stderr, "[INFO] MCP服务已启动，等待请求...\n")

	for scanner.Scan() {
		line := scanner.Text()
		if strings.TrimSpace(line) == "" {
			continue
		}

		// 1. 解析JSON-RPC请求
		var req Request
		if err := json.Unmarshal([]byte(line), &req); err != nil {
			s.sendErrorResponse(req.ID, -999, "请求格式错误："+err.Error())
			continue
		}

		// 2. 分发请求到对应方法
		switch req.Method {
		case "initialize":
			s.handleInitialize(req)
		case "initialized":
			// 可以忽略，但建议支持
		case "tools/call":
			s.handleToolCall(req) // 工具调用
		case "tools/list":
			s.handleToolList(req) // 工具列表查询
		default:
			s.sendErrorResponse(req.ID, -998, "未知方法："+req.Method)
		}
	}

	// 处理stdin读取错误
	if err := scanner.Err(); err != nil {
		fmt.Fprintf(os.Stderr, "[ERROR] 读取stdin失败：%v\n", err)
		os.Exit(1)
	}
}

func (s *MCPServer) handleInitialize(req Request) {
	resp := map[string]interface{}{
		"protocolVersion": "2024-11-05",
		"capabilities": map[string]interface{}{
			"tools": map[string]interface{}{},
		},
		"serverInfo": map[string]interface{}{
			"name":    "mcp-inspector-server",
			"version": "1.0.0",
		},
	}

	s.sendSuccessResponse(req.ID, resp)
}

// handleToolCall 处理工具调用请求
func (s *MCPServer) handleToolCall(req Request) {
	// 解析工具名称
	toolName, ok := req.Params["name"].(string)
	if !ok {
		s.sendErrorResponse(req.ID, -1, "缺少工具名称（name参数）")
		return
	}

	// 解析工具参数
	arguments, ok := req.Params["arguments"].(map[string]interface{})
	if !ok {
		arguments = make(map[string]interface{}) // 无参数时默认空map
	}

	// 查找已注册的工具
	tool, exists := s.tools[toolName]
	if !exists {
		s.sendErrorResponse(req.ID, -2, "工具不存在："+toolName)
		return
	}

	// 执行工具并返回结果
	result, err := tool.Call(arguments)
	if err != nil {
		s.sendErrorResponse(req.ID, -3, "工具执行失败："+err.Error())
		return
	}

	// 发送成功响应
	s.sendSuccessResponse(req.ID, result)
}

// handleToolList 处理工具列表查询
func (s *MCPServer) handleToolList(req Request) {
	var toolList []map[string]interface{}
	for _, tool := range s.tools {
		toolList = append(toolList, map[string]interface{}{
			"name":        tool.Name(),
			"description": tool.Description(),
			"inputSchema": tool.InputSchema(),
			// 👉 建议先去掉 outputSchema（避免兼容问题）
		})
	}
	// ✅ 关键：包一层 tools
	result := map[string]interface{}{
		"tools": toolList,
	}
	s.sendSuccessResponse(req.ID, result)
}

// sendSuccessResponse 发送成功响应到stdout
func (s *MCPServer) sendSuccessResponse(id interface{}, result interface{}) {
	res := Response{
		JSONRPC: "2.0",
		ID:      id,
		Result:  result,
	}
	s.sendResponse(res)
}

// sendErrorResponse 发送错误响应到stdout
func (s *MCPServer) sendErrorResponse(id interface{}, code int, msg string) {
	var errRes ErrorResponse
	errRes.JSONRPC = "2.0"
	errRes.ID = id
	errRes.Error.Code = code
	errRes.Error.Message = msg
	s.sendResponse(errRes)
}

// sendResponse 通用响应发送方法（序列化+输出到stdout）
func (s *MCPServer) sendResponse(response interface{}) {
	data, err := json.Marshal(response)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[ERROR] 序列化响应失败：%v\n", err)
		return
	}
	fmt.Println(string(data)) // MCP stdio规范：每行一个JSON
}
