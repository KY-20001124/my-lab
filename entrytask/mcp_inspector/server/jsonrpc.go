package server

// -------------------------- JSON-RPC 2.0 核心结构体 --------------------------

// Request MCP JSON-RPC 请求
type Request struct {
	JSONRPC string                 `json:"jsonrpc"` // 固定为2.0
	ID      interface{}            `json:"id"`
	Method  string                 `json:"method"` // 方法名
	Params  map[string]interface{} `json:"params"` // 方法参数
}

// Response 成功响应
type Response struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      interface{} `json:"id"`
	Result  interface{} `json:"result"`
}

// ErrorResponse 错误响应
type ErrorResponse struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      interface{} `json:"id"`
	Error   struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}
