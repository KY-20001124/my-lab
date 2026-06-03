package tools

import (
	"encoding/json"
	"errors"
	"time"

	"mcp_inspector/server"

	"github.com/valyala/fasthttp"
)

// HTTPHealthCheckTool HTTP健康检查工具（实现server.Tool接口）
type HTTPHealthCheckTool struct{}

// NewHTTPHealthCheckTool 创建工具实例（供main注册使用）
func NewHTTPHealthCheckTool() server.Tool {
	return &HTTPHealthCheckTool{}
}

// Name 工具唯一标识名称
func (t *HTTPHealthCheckTool) Name() string {
	return "http_health_check"
}

// Description 工具功能描述
func (t *HTTPHealthCheckTool) Description() string {
	return "检查指定URL的HTTP/HTTPS健康状态，返回状态码、响应时间、健康状态等信息"
}

// InputSchema 输入参数的JSON Schema定义
func (t *HTTPHealthCheckTool) InputSchema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"url": {
				"type": "string",
				"description": "要检查的HTTP/HTTPS URL，必填"
			},
			"timeout": {
				"type": "integer",
				"description": "请求超时时间（秒），默认5秒，范围1-30",
				"default": 5
			},
			"method": {
				"type": "string",
				"description": "HTTP请求方法，支持GET/POST/HEAD，默认GET",
				"enum": ["GET", "POST", "HEAD"],
				"default": "GET"
			}
		},
		"required": ["url"]
	}`)
}

// OutputSchema 输出结果的JSON Schema定义
func (t *HTTPHealthCheckTool) OutputSchema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"url": {
				"type": "string",
				"description": "检查的目标URL"
			},
			"status_code": {
				"type": "integer",
				"description": "HTTP响应状态码（0表示请求失败）"
			},
			"response_time_ms": {
				"type": "number",
				"description": "请求响应时间（毫秒）"
			},
			"healthy": {
				"type": "boolean",
				"description": "是否健康（状态码2xx为true）"
			},
			"error": {
				"type": "string",
				"description": "错误信息（请求失败时非空）"
			}
		}
	}`)
}

// Call 工具核心执行逻辑
func (t *HTTPHealthCheckTool) Call(params map[string]interface{}) (interface{}, error) {
	// 1. 解析并校验输入参数
	// 必选参数：url
	url, ok := params["url"].(string)
	if !ok || url == "" {
		return nil, errors.New("参数错误：url不能为空且必须为字符串类型")
	}

	// 可选参数：timeout（默认5秒）
	timeoutSec := 5
	if timeoutVal, ok := params["timeout"].(float64); ok && timeoutVal > 0 {
		timeoutSec = int(timeoutVal)
	}
	if timeoutSec <= 0 || timeoutSec > 30 {
		return nil, errors.New("参数错误：timeout必须是1-30之间的整数")
	}

	// 可选参数：method（默认GET）
	method := "GET"
	if methodVal, ok := params["method"].(string); ok {
		switch methodVal {
		case "GET", "POST", "HEAD":
			method = methodVal
		default:
			return nil, errors.New("参数错误：method仅支持GET/POST/HEAD")
		}
	}

	// 2. 初始化fasthttp客户端（
	client := &fasthttp.Client{
		ReadTimeout:     time.Duration(timeoutSec) * time.Second, // 读取响应超时
		WriteTimeout:    time.Duration(timeoutSec) * time.Second, // 发送请求超时
		MaxConnsPerHost: 100,                                     // 可选：限制每个主机的最大连接数，避免资源耗尽
	}

	// 3. 构建HTTP请求
	req := fasthttp.AcquireRequest()
	defer fasthttp.ReleaseRequest(req) // 用完释放资源，避免内存泄漏
	req.SetRequestURI(url)
	req.Header.SetMethod(method)

	// 4. 接收HTTP响应
	resp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseResponse(resp)

	// 5. 执行请求并计时
	startTime := time.Now()
	err := client.Do(req, resp) // 基础Do方法，不跟随重定向（如需重定向用DoRedirects）
	responseTimeMs := time.Since(startTime).Milliseconds()

	// 6. 构造返回结果
	result := map[string]interface{}{
		"url":              url,
		"status_code":      0,
		"response_time_ms": float64(responseTimeMs),
		"healthy":          false,
		"error":            "",
	}

	// 处理请求错误（如超时、网络不可达、URL无效等）
	if err != nil {
		result["error"] = err.Error()
		// 区分超时错误，给出更友好的提示
		if err.Error() == "timeout" || err.Error() == "context deadline exceeded" {
			result["error"] = "请求超时（" + string(timeoutSec) + "秒）"
		}
		return result, nil // 非panic级错误，返回错误信息但不抛出异常
	}

	// 解析响应状态码并判断健康状态
	statusCode := resp.StatusCode()
	result["status_code"] = statusCode
	// 2xx状态码判定为健康
	if statusCode >= 200 && statusCode < 300 {
		result["healthy"] = true
	}

	return result, nil
}
