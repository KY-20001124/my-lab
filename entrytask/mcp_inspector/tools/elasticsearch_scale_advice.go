package tools

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"mcp_inspector/schemas"
)

// ElasticsearchScaleAdviceTool ES集群扩容建议工具
type ElasticsearchScaleAdviceTool struct{}

// NewElasticsearchScaleAdviceTool 创建工具实例
func NewElasticsearchScaleAdviceTool() *ElasticsearchScaleAdviceTool {
	return &ElasticsearchScaleAdviceTool{}
}

// Name 返回工具名称
func (t *ElasticsearchScaleAdviceTool) Name() string {
	return "elasticsearch_scale_advice"
}

// Description 返回工具描述
func (t *ElasticsearchScaleAdviceTool) Description() string {
	return "Elasticsearch 集群巡检与扩容建议工具，包含健康概览、分片建议、节点建议、操作步骤、优化效果"
}

// InputSchema 返回输入Schema
func (t *ElasticsearchScaleAdviceTool) InputSchema() json.RawMessage {
	return schemas.ElasticsearchScaleAdviceInputSchema
}

// OutputSchema 返回输出Schema
func (t *ElasticsearchScaleAdviceTool) OutputSchema() json.RawMessage {
	return schemas.ElasticsearchScaleAdviceOutputSchema
}

// 最终输出结构体（严格匹配你修改后的 schema）
type EsScaleResult struct {
	ClusterHealthOverview       string `json:"cluster_health_overview"`
	ShardScaleAdvice            string `json:"shard_scale_advice"`
	NodeScaleAdvice             string `json:"node_scale_advice"`
	OperationalSteps            string `json:"operational_steps"`
	EstimatedOptimizationEffect string `json:"estimated_optimization_effect"`
	Error                       string `json:"error,omitempty"`
}

// 内部结构体
type esHealthResp struct {
	ClusterName  string `json:"cluster_name"`
	Status       string `json:"status"`
	NodeTotal    int    `json:"number_of_nodes"`
	ActiveShards int    `json:"active_shards"`
}

type esNodesStatsResp struct {
	Nodes map[string]struct {
		Name string `json:"name"`
		OS   struct {
			CPU struct {
				Percent int `json:"percent"`
			} `json:"cpu"`
			Mem struct {
				UsedPercent int `json:"used_percent"`
			} `json:"mem"`
		} `json:"os"`
		FS struct {
			Total struct {
				UsedPercent int `json:"used_percent"`
			} `json:"total"`
		} `json:"fs"`
		Indices struct {
			Shards struct {
				Total int `json:"total"`
			} `json:"shards"`
		} `json:"indices"`
	} `json:"nodes"`
}

type esShardInfo struct {
	State string `json:"state"`
	Node  string `json:"node"`
}

type esShardsResp []esShardInfo

// ===================== 主逻辑 Call =====================
func (t *ElasticsearchScaleAdviceTool) Call(params map[string]interface{}) (interface{}, error) {
	// 解析参数
	esHosts, ok := params["es_hosts"].([]interface{})
	if !ok || len(esHosts) == 0 {
		return EsScaleResult{Error: "es_hosts 不能为空"}, nil
	}

	hosts := make([]string, 0, len(esHosts))
	for _, v := range esHosts {
		if s, ok := v.(string); ok {
			hosts = append(hosts, s)
		}
	}
	if len(hosts) == 0 {
		return EsScaleResult{Error: "es_hosts 不合法"}, nil
	}

	username, _ := params["username"].(string)
	password, _ := params["password"].(string)

	cpuThreshold := 80
	if v, ok := params["cpu_threshold"].(float64); ok {
		cpuThreshold = int(v)
	}
	memThreshold := 85
	if v, ok := params["memory_threshold"].(float64); ok {
		memThreshold = int(v)
	}
	diskThreshold := 80
	if v, ok := params["disk_threshold"].(float64); ok {
		diskThreshold = int(v)
	}

	timeout := 10
	if v, ok := params["timeout"].(float64); ok {
		timeout = int(v)
	}

	// HTTP 客户端
	client := &http.Client{
		Timeout: time.Duration(timeout) * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}

	// 1. 获取集群健康
	health, err := t.getHealth(client, hosts[0], username, password)
	if err != nil {
		return EsScaleResult{Error: "获取集群健康失败: " + err.Error()}, nil
	}

	// 2. 获取节点指标
	nodeStats, err := t.getNodeStats(client, hosts[0], username, password)
	if err != nil {
		return EsScaleResult{Error: "获取节点指标失败: " + err.Error()}, nil
	}

	// 3. 获取分片信息
	shards, err := t.getShards(client, hosts[0], username, password)
	if err != nil {
		return EsScaleResult{Error: "获取分片信息失败: " + err.Error()}, nil
	}

	// ===================== 生成5大模块内容 =====================
	overview := fmt.Sprintf("集群：%s | 节点数：%d | 总分片：%d | 健康：%s",
		health.ClusterName, health.NodeTotal, health.ActiveShards, health.Status)

	shardAdvice := t.buildShardAdvice(shards, health)
	nodeAdvice := t.buildNodeAdvice(nodeStats, cpuThreshold, memThreshold, diskThreshold)
	steps := t.buildOperationalSteps()
	effect := t.buildOptimizationEffect()

	return EsScaleResult{
		ClusterHealthOverview:       overview,
		ShardScaleAdvice:            shardAdvice,
		NodeScaleAdvice:             nodeAdvice,
		OperationalSteps:            steps,
		EstimatedOptimizationEffect: effect,
	}, nil
}

// ===================== 内部工具方法 =====================
func (t *ElasticsearchScaleAdviceTool) getHealth(client *http.Client, host, user, pwd string) (*esHealthResp, error) {
	req, _ := http.NewRequest("GET", host+"/_cluster/health", nil)
	if user != "" && pwd != "" {
		req.SetBasicAuth(user, pwd)
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var out esHealthResp
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (t *ElasticsearchScaleAdviceTool) getNodeStats(client *http.Client, host, user, pwd string) (*esNodesStatsResp, error) {
	req, _ := http.NewRequest("GET", host+"/_nodes/stats/os,fs,indices", nil)
	if user != "" && pwd != "" {
		req.SetBasicAuth(user, pwd)
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var out esNodesStatsResp
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (t *ElasticsearchScaleAdviceTool) getShards(client *http.Client, host, user, pwd string) (esShardsResp, error) {
	req, _ := http.NewRequest("GET", host+"/_cat/shards?format=json", nil)
	if user != "" && pwd != "" {
		req.SetBasicAuth(user, pwd)
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var out esShardsResp
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	return out, nil
}

// ===================== 生成建议文本 =====================
func (t *ElasticsearchScaleAdviceTool) buildShardAdvice(shards esShardsResp, health *esHealthResp) string {
	unassigned := 0
	for _, s := range shards {
		if s.State == "UNASSIGNED" {
			unassigned++
		}
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("总分片数：%d\n", health.ActiveShards))
	sb.WriteString(fmt.Sprintf("未分配分片：%d\n", unassigned))

	if unassigned > 0 {
		sb.WriteString("存在未分配分片，可能导致集群 yellow/red，建议排查分配规则\n")
	} else {
		sb.WriteString("分片分配正常\n")
	}
	sb.WriteString("建议单分片大小控制在 30-50GB，避免过大影响恢复速度")
	return sb.String()
}

func (t *ElasticsearchScaleAdviceTool) buildNodeAdvice(ns *esNodesStatsResp, cpuT, memT, diskT int) string {
	var sb strings.Builder
	for id, node := range ns.Nodes {
		cpu := node.OS.CPU.Percent
		mem := node.OS.Mem.UsedPercent
		disk := node.FS.Total.UsedPercent

		sb.WriteString(fmt.Sprintf("节点 %s (%s):\n", node.Name, id))
		sb.WriteString(fmt.Sprintf("  CPU: %d%%\n", cpu))
		sb.WriteString(fmt.Sprintf("  内存: %d%%\n", mem))
		sb.WriteString(fmt.Sprintf("  磁盘: %d%%\n", disk))

		if cpu > cpuT {
			sb.WriteString("  ⚠️ CPU 超过阈值，建议扩容或减负\n")
		}
		if mem > memT {
			sb.WriteString("  ⚠️ 内存使用率过高，存在 OOM 风险\n")
		}
		if disk > diskT {
			sb.WriteString("  ❗ 磁盘使用率过高，需及时扩容\n")
		}
		sb.WriteString("\n")
	}
	return sb.String()
}

func (t *ElasticsearchScaleAdviceTool) buildOperationalSteps() string {
	return `1. 查看未分配分片原因：GET /_cluster/allocation/explain
2. 重试失败分片：POST /_cluster/reroute?retry_failed=true
3. 开启集群重平衡：PUT /_cluster/settings {"persistent":{"cluster.routing.rebalance.enable":"all"}}
4. 新增数据节点加入集群
5. 调整索引分片数与副本数`
}

func (t *ElasticsearchScaleAdviceTool) buildOptimizationEffect() string {
	return `1. 集群健康恢复至 green
2. 热点分片压力下降，查询延迟降低 20%~50%
3. 内存/CPU/磁盘负载降至安全区间
4. 集群可支撑更高写入与查询吞吐量
5. 减少宕机与数据丢失风险`
}
