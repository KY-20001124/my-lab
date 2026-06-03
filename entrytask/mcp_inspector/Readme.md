## 运行
go run main.go

## 测试
1. http_health.go
{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"http_health_check","arguments":{"url":"https://www.baidu.com","timeout":5,"method":"GET"}}}

2. k8s_pod_status.go
   {"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"kubernetes_pod_status","arguments":{"namespace":"default","kubeconfig_path":"/Users/yao.ke/.kube/config","include_running":true,"timeout":10}}}

3. elasticsearch_scale_advice.go
起1终端暴露接口kubectl port-forward svc/elasticsearch-nodeport 9200:9200
另起终端 测试：curl http://127.0.0.1:9200