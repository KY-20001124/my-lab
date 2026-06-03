package main

import (
	"mcp_inspector/server"
	"mcp_inspector/tools"
)

func main() {
	// 初始化 MCP 服务
	mcpServer := server.NewMCPServer()

	// 注册所有巡检工具
	mcpServer.RegisterTool(tools.NewHTTPHealthCheckTool())
	mcpServer.RegisterTool(tools.NewK8sPodStatusTool())
	mcpServer.RegisterTool(tools.NewElasticsearchScaleAdviceTool())

	// 启动 stdio 监听
	mcpServer.Run()
}
