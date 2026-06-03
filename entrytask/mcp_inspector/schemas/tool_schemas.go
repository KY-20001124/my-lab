package schemas

import "encoding/json"

// -------------------------- HTTP 健康检查 Schema --------------------------
// HTTPHealthCheckInputSchema HTTP健康检查输入Schema
// 用于定义HTTP/HTTPS URL健康检查的入参规范
var HTTPHealthCheckInputSchema = json.RawMessage(`{
	"type": "object",
	"properties": {
		"url": {
			"type": "string",
			"description": "要检查的HTTP/HTTPS URL",
			"examples": ["https://api.example.com/health"]
		},
		"timeout": {
			"type": "integer",
			"description": "超时时间（秒），默认5秒",
			"default": 5,
			"minimum": 1,
			"maximum": 30
		},
		"method": {
			"type": "string",
			"description": "HTTP方法，默认GET",
			"enum": ["GET", "POST", "HEAD"],
			"default": "GET"
		}
	},
	"required": ["url"]
}`)

// HTTPHealthCheckOutputSchema HTTP健康检查输出Schema
// 用于定义HTTP健康检查的返回结果规范
var HTTPHealthCheckOutputSchema = json.RawMessage(`{
	"type": "object",
	"properties": {
		"url": {
			"type": "string",
			"description": "检查的URL"
		},
		"status_code": {
			"type": "integer",
			"description": "HTTP状态码"
		},
		"response_time_ms": {
			"type": "number",
			"description": "响应时间（毫秒）"
		},
		"healthy": {
			"type": "boolean",
			"description": "是否健康（状态码2xx）"
		},
		"error": {
			"type": "string",
			"description": "错误信息（如有）"
		}
	}
}`)

// -------------------------- K8s Pod 巡检 Schema --------------------------
// K8sPodStatusInputSchema K8s Pod状态巡检输入Schema
// 用于定义查询K8s集群Pod状态的入参规范
var K8sPodStatusInputSchema = json.RawMessage(`{
	"type": "object",
	"properties": {
		"namespace": {
			"type": "string",
			"description": "要查询的命名空间",
			"default": "default",
			"examples": ["default", "kube-system", "prod"]
		},
		"kubeconfig_path": {
			"type": "string",
			"description": "kubeconfig 文件路径，可选（默认读取~/.kube/config）",
			"examples": ["/root/.kube/config", "/home/user/.kube/config"]
		},
		"include_running": {
			"type": "boolean",
			"description": "是否包含运行中的 Pod",
			"default": true
		},
		"timeout": {
			"type": "integer",
			"description": "请求超时时间（秒）",
			"default": 10,
			"minimum": 1,
			"maximum": 60
		}
	},
	"required": ["namespace"]
}`)

// K8sPodStatusOutputSchema K8s Pod状态巡检输出Schema
// 用于定义K8s Pod状态巡检的返回结果规范
var K8sPodStatusOutputSchema = json.RawMessage(`{
	"type": "object",
	"properties": {
		"namespace": {
			"type": "string",
			"description": "查询的命名空间"
		},
		"total_pods": {
			"type": "integer",
			"description": "Pod总数"
		},
		"unhealthy_pods": {
			"type": "array",
			"items": {
				"type": "object",
				"properties": {
					"pod_name": {
						"type": "string",
						"description": "Pod名称"
					},
					"status": {
						"type": "string",
						"description": "Pod状态"
					},
					"reason": {
						"type": "string",
						"description": "异常原因（如有）"
					},
					"restart_count": {
						"type": "integer",
						"description": "重启次数"
					}
				}
			},
			"description": "不健康的Pod列表"
		},
		"error": {
			"type": "string",
			"description": "错误信息（如有）"
		}
	}
}`)

// -------------------------- ES 扩容建议 Schema --------------------------
// ElasticsearchScaleAdviceInputSchema ES集群扩容建议输入Schema
// 用于定义分析ES集群状态、生成扩容建议的入参规范
// -------------------------- ES 扩容建议 Schema --------------------------
// ElasticsearchScaleAdviceInputSchema ES集群扩容建议输入Schema
// 用于定义分析ES集群状态、生成扩容建议的入参规范
var ElasticsearchScaleAdviceInputSchema = json.RawMessage(`{
	"type": "object",
	"properties": {
		"es_hosts": {
			"type": "array",
			"items": {
				"type": "string"
			},
			"description": "ES集群地址列表",
			"examples": ["http://127.0.0.1:9200", "http://192.168.1.100:9200"],
			"minItems": 1,
			"uniqueItems": true
		},
		"username": {
			"type": "string",
			"description": "ES认证用户名，可选（无认证则留空）"
		},
		"password": {
			"type": "string",
			"description": "ES认证密码，可选（无认证则留空）"
		},
		"cpu_threshold": {
			"type": "integer",
			"description": "CPU使用率阈值（%）",
			"default": 80,
			"minimum": 50,
			"maximum": 95
		},
		"memory_threshold": {
			"type": "integer",
			"description": "内存使用率阈值（%）",
			"default": 85,
			"minimum": 50,
			"maximum": 95
		},
		"disk_threshold": {
			"type": "integer",
			"description": "磁盘使用率阈值（%）",
			"default": 80,
			"minimum": 50,
			"maximum": 95
		},
		"timeout": {
			"type": "integer",
			"description": "请求超时时间（秒）",
			"default": 10,
			"minimum": 1,
			"maximum": 60
		}
	},
	"required": ["es_hosts"]
}`)

// ElasticsearchScaleAdviceOutputSchema ES集群扩容建议输出Schema
// 用于定义ES集群扩容建议的返回结果规范
var ElasticsearchScaleAdviceOutputSchema = json.RawMessage(`{
	"type": "object",
	"properties": {
		"cluster_health_overview": {
			"type": "string",
			"description": "集群健康状态概览（包含集群名称、节点数、总分片数、健康状态）"
		},
		"shard_scale_advice": {
			"type": "string",
			"description": "分片扩容建议（分片大小、未分配分片、热点分片明细）"
		},
		"node_scale_advice": {
			"type": "string",
			"description": "节点扩容建议（CPU/内存/磁盘使用率 breakdown）"
		},
		"operational_steps": {
			"type": "string",
			"description": "具体可操作的命令/步骤（如分片重平衡、新增节点等）"
		},
		"estimated_optimization_effect": {
			"type": "string",
			"description": "预估优化效果（如吞吐量提升、磁盘使用率下降等）"
		},
		"error": {
			"type": "string",
			"description": "错误信息（如有）"
		}
	}
}`)

// -------------------------- 工具Schema映射（统一入口） --------------------------
// ToolInputSchemas 所有工具的输入Schema映射
// key：工具名称（需和server/tools.go中注册的工具名一致）
// value：对应的输入Schema
var ToolInputSchemas = map[string]json.RawMessage{
	"http_health_check":          HTTPHealthCheckInputSchema,
	"kubernetes_pod_status":      K8sPodStatusInputSchema,
	"elasticsearch_scale_advice": ElasticsearchScaleAdviceInputSchema,
}

// ToolOutputSchemas 所有工具的输出Schema映射
// key：工具名称（需和server/tools.go中注册的工具名一致）
// value：对应的输出Schema
var ToolOutputSchemas = map[string]json.RawMessage{
	"http_health_check":          HTTPHealthCheckOutputSchema,
	"kubernetes_pod_status":      K8sPodStatusOutputSchema,
	"elasticsearch_scale_advice": ElasticsearchScaleAdviceOutputSchema,
}
