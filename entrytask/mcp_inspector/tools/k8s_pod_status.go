package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"

	"mcp_inspector/server"
)

// K8sPodStatusTool K8s Pod状态巡检工具（实现server.Tool接口）
type K8sPodStatusTool struct{}

// NewK8sPodStatusTool 创建工具实例（供main注册）
func NewK8sPodStatusTool() server.Tool {
	return &K8sPodStatusTool{}
}

// Name 工具唯一标识
func (t *K8sPodStatusTool) Name() string {
	return "kubernetes_pod_status"
}

// Description 工具描述
func (t *K8sPodStatusTool) Description() string {
	return "巡检Kubernetes指定命名空间下的Pod状态，筛选非Running状态的Pod并返回详细信息"
}

// InputSchema 输入参数JSON Schema
func (t *K8sPodStatusTool) InputSchema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"namespace": {
				"type": "string",
				"description": "K8s命名空间，默认default"
			},
			"kubeconfig_path": {
				"type": "string",
				"description": "Kubeconfig文件路径，默认~/.kube/config"
			},
			"context": {
				"type": "string",
				"description": "K8s上下文名称，默认使用kubeconfig当前上下文"
			},
			"timeout": {
				"type": "integer",
				"description": "请求超时时间（秒），默认10秒"
			},
			"include_running": {
				"type": "boolean",
				"description": "是否包含Running状态的Pod，默认false"
			}
		}
	}`)
}

// OutputSchema 输出结果JSON Schema
func (t *K8sPodStatusTool) OutputSchema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"namespace": {
				"type": "string",
				"description": "巡检的命名空间"
			},
			"total_pods": {
				"type": "integer",
				"description": "命名空间下Pod总数"
			},
			"unhealthy_pods": {
				"type": "array",
				"items": {
					"type": "object",
					"properties": {
						"name": {
							"type": "string",
							"description": "Pod名称"
						},
						"status": {
							"type": "string",
							"description": "Pod状态（如Pending/CrashLoopBackOff）"
						},
						"reason": {
							"type": "string",
							"description": "状态原因"
						},
						"restart_count": {
							"type": "integer",
							"description": "重启次数"
						},
						"node_name": {
							"type": "string",
							"description": "运行节点名称"
						},
						"creation_time": {
							"type": "string",
							"description": "创建时间（RFC3339格式）"
						}
					}
				}
			},
			"error": {
				"type": "string",
				"description": "错误信息（如有）"
			}
		}
	}`)
}

// Call 工具核心执行逻辑
func (t *K8sPodStatusTool) Call(params map[string]interface{}) (interface{}, error) {
	// 1. 解析输入参数
	// 命名空间（默认default）
	namespace := "default"
	if nsVal, ok := params["namespace"].(string); ok && nsVal != "" { //转换为字符串类型
		namespace = nsVal
	}

	// Kubeconfig路径（默认~/.kube/config）
	kubeconfigPath := ""
	if kcVal, ok := params["kubeconfig_path"].(string); ok && kcVal != "" {
		kubeconfigPath = kcVal
	} else {
		if home := homedir.HomeDir(); home != "" { //homedir.HomeDir就是获取主目录
			kubeconfigPath = filepath.Join(home, ".kube", "config")
		}
	}

	// 上下文名称（可选）
	contextName := ""
	if ctxVal, ok := params["context"].(string); ok && ctxVal != "" {
		contextName = ctxVal
	}

	// 超时时间（默认10秒）
	timeoutSec := 10
	if timeoutVal, ok := params["timeout"].(float64); ok && timeoutVal > 0 {
		timeoutSec = int(timeoutVal)
	}

	// 是否包含Running状态Pod（默认false）
	includeRunning := false
	if irVal, ok := params["include_running"].(bool); ok {
		includeRunning = irVal
	}

	// 2. 构建K8s客户端配置，拿到集群地址和证书
	config, err := buildK8sConfig(kubeconfigPath, contextName)
	if err != nil {
		return buildErrorResult(namespace, fmt.Sprintf("构建K8s配置失败：%v", err)), nil
	}

	// 3. 创建K8s客户端，操纵集群的总入口
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return buildErrorResult(namespace, fmt.Sprintf("创建K8s客户端失败：%v", err)), nil
	}

	// 4. 设置请求超时，用于下一行的获取pod列表功能
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeoutSec)*time.Second)
	defer cancel() //call函数执行完运行，释放定时器资源

	// 5. 获取Pod列表，CoreV1是k8s里的核心资源组
	podList, err := clientset.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return buildErrorResult(namespace, fmt.Sprintf("获取Pod列表失败：%v", err)), nil
	}

	// 6. 解析Pod状态
	totalPods := len(podList.Items)
	unhealthyPods := make([]map[string]interface{}, 0)

	for _, pod := range podList.Items {
		podStatus := getPodStatus(&pod)
		podReason := getPodReason(&pod)

		// 筛选条件：非Running 或 强制包含Running
		if includeRunning || podStatus != "Running" {
			unhealthyPods = append(unhealthyPods, map[string]interface{}{
				"name":          pod.Name,
				"status":        podStatus,
				"reason":        podReason,
				"restart_count": getPodRestartCount(&pod),
				"node_name":     pod.Spec.NodeName,
				"creation_time": pod.CreationTimestamp.Format(time.RFC3339),
			})
		}
	}

	// 7. 构造返回结果
	result := map[string]interface{}{
		"namespace":      namespace,
		"total_pods":     totalPods,
		"unhealthy_pods": unhealthyPods,
		"error":          "",
	}

	return result, nil
}

// -------------------------- 辅助函数 --------------------------

// buildK8sConfig 构建K8s客户端配置（兼容本地/集群内运行）
// buildK8sConfig 构建K8s客户端配置（兼容本地/集群内运行）
func buildK8sConfig(kubeconfigPath, contextName string) (*rest.Config, error) {
	// 集群内运行：使用in-cluster配置（无kubeconfig文件）
	if kubeconfigPath == "" {
		if home := homedir.HomeDir(); home != "" {
			kubeconfigPath = filepath.Join(home, ".kube", "config")
		}
	}

	// 检查kubeconfig文件是否存在
	if _, err := os.Stat(kubeconfigPath); os.IsNotExist(err) {
		// 尝试集群内配置
		return clientcmd.BuildConfigFromFlags("", "")
	}

	// 加载kubeconfig并指定上下文
	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	loadingRules.ExplicitPath = kubeconfigPath
	configOverrides := &clientcmd.ConfigOverrides{}
	if contextName != "" {
		configOverrides.CurrentContext = contextName
	}

	clientConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, configOverrides)
	return clientConfig.ClientConfig()
}

// getPodStatus 获取Pod核心状态（优先Phase，再容器状态）
func getPodStatus(pod *corev1.Pod) string {
	// 优先取Pod的Phase（Pending/Running/Succeeded/Failed/Unknown）
	if pod.Status.Phase != "" {
		return string(pod.Status.Phase)
	}

	// 取容器的详细状态（如CrashLoopBackOff）
	for _, containerStatus := range pod.Status.ContainerStatuses {
		if containerStatus.State.Waiting != nil && containerStatus.State.Waiting.Reason != "" {
			return containerStatus.State.Waiting.Reason
		}
		if containerStatus.State.Terminated != nil && containerStatus.State.Terminated.Reason != "" {
			return containerStatus.State.Terminated.Reason
		}
	}

	return "Unknown"
}

// getPodReason 获取Pod状态原因
func getPodReason(pod *corev1.Pod) string {
	// 取Pod条件的原因
	for _, condition := range pod.Status.Conditions {
		if condition.Reason != "" {
			return condition.Reason
		}
	}

	// 取容器等待原因
	for _, containerStatus := range pod.Status.ContainerStatuses {
		if containerStatus.State.Waiting != nil && containerStatus.State.Waiting.Message != "" {
			return containerStatus.State.Waiting.Message
		}
	}

	return "No reason provided"
}

// getPodRestartCount 获取Pod总重启次数
func getPodRestartCount(pod *corev1.Pod) int {
	total := 0
	for _, containerStatus := range pod.Status.ContainerStatuses {
		total += int(containerStatus.RestartCount)
	}
	return total
}

// buildErrorResult 构建错误结果
func buildErrorResult(namespace string, errMsg string) map[string]interface{} {
	return map[string]interface{}{
		"namespace":      namespace,
		"total_pods":     0,
		"unhealthy_pods": []interface{}{},
		"error":          errMsg,
	}
}
