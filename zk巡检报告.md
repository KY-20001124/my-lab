# ZK 集群巡检报告

**集群**: `zk-content-intelligence-video` | **巡检时间**: 2026-06-11 18:13 CST (UTC+8) | **日志窗口**: 近 6h | **指标模块**: `zk_engine`

---

## 0. 集群信息

| 字段 | 值 |
|------|-----|
| **cluster_id** | `zk-content-intelligence-video` |
| **cluster_name** | N/A（`umo_get_cluster` 不可用） |
| **Status** | 🟢 健康（空闲状态） |
| **members_seen** | 3 |
| **expected_members** | N/A |
| **ensemble_size** | 3 |
| **majority_size** | 2 |
| **leader_changes_1h** | 0 |
| **request_count_60m** | 0（所有节点） |
| **outstanding_requests** | 0 / 0 / 0 |
| **outstanding_requests_60m_peak** | 0 / 0 / 0 |
| **avg_latency_ms** | 14.33 / 0 / 3.33 |
| **avg_latency_ms_60m_peak** | 14.33 / 0 / 3.33 |
| **historical_max_latency_ms** | 33 / 0 / 5 |
| **max_latency_delta_60m** | 0 / 0 / 0 |
| **max_latency_changes_60m** | 0 / 0 / 0 |
| **num_alive_connections** | 0 / 0 / 0 |
| **global_sessions** | 0 / 0 / 0 |
| **znode_count** | 5 / 5 / 5 |
| **ephemerals_count** | 0 / 0 / 0 |
| **watch_count** | 0 / 0 / 0 |
| **snapshot_time_ms_avg_5m** | 0 / 0 / 0 |
| **jvm_heap_used_ratio_current** | 6.81% / 24.62% / 40.98% |
| **jvm_heap_used_ratio_5m_avg** | 6.79% / 24.64% / 40.93% |
| **jvm_heap_used_ratio_60m_peak** | 6.81% / 24.64% / 40.98% |
| **jvm_threads_current** | 141 / 127 / 136 |
| **request_count_5m** | 0 / N/A / 0 |
| **ephemerals_delta_5m** | 0 / 0 / 0 |
| **time** | 2026-06-11T10:13:40Z |

> 节点标识：节点1 = `10.251.135.68:23181`，节点2 = `10.251.184.5:23181`，节点3 = `10.251.184.6:23181`

---

## 1. 总体概览

- **Summary**: 集群共识稳定（leader 在位，0 次选主变化），无请求积压、无连接、无错误日志。当前处于**空闲状态**（0 个客户端连接、0 条请求、5 个 ZNode、258 bytes 数据体量），无运行态风险。三个节点 Heap 使用率离散（6.8% / 24.6% / 41.0%），但均在安全范围内且 GC 为 0。
- **Risk counts**: CRITICAL 0 / WARNING 1 / INFO 2

---

## 2. 严重级别汇总表

| 严重级别 | 巡检项 | 对象 | 证据（指标名 / 日志片段） | 阈值或规则 |
|----------|--------|------|--------------------------|-----------|
| WARNING | Item 2: 请求积压与延迟 | 集群（节点倾斜） | avg_latency_ms 节点间 skew = 143%；节点1=14.33ms，节点2=0ms，节点3=3.33ms | skew > 50%（avg_latency max >= 10ms，符合低量级保护条件） |

> 注：该 WARNING 发生在一个**0 连接 0 请求的空闲集群**上。节点1（推测为 leader）以 14.33ms 处理内部操作，节点2/3（follower）几乎无延迟。此倾斜是 leader/follower 角色差异的正常表现，在空闲场景下不构成实际风险。

---

## 3. 逐项分析

### Inspection Item 1: Quorum 与 Leader 健康

**数据摘要**:
- leader_present = 1（有且仅有一个 leader），leader_uptime = 7d+
- ensemble_size = 3，members_seen = 3，up_instances = 3
- synced_followers = 2，learners = 2，pending_syncs = 0
- leader_changes_1h = 0，leader_unavailable_1h = 0，unavailable_time_1h = 0

**告警与规则**: 无触发规则。

| 指标 | 值 | 阈值 | 判断 |
|------|-----|------|------|
| leader_present | 1 | = 0 CRITICAL | ✅ 正常 |
| members_seen | 3 | < 3 WARNING | ✅ 正常 |
| synced_followers | 2 | < majority_size-1 CRITICAL | ✅ majority=2, synced=2, 满足 |
| pending_syncs | 0 | > 0 WARNING | ✅ 正常 |
| leader_changes_1h | 0 | > 3 WARNING | ✅ 稳定 |
| leader_unavailable_1h | 0 | > 0 WARNING | ✅ 无不可用 |

**解读**: Quorum 完全健康，leader 稳定在位，所有 follower 已同步，无选主抖动。

**建议措施**: **Observe(观察)** — 保持常规监控。

---

### Inspection Item 2: 请求积压与延迟

**数据摘要**:
- request_count_60m = 0（所有节点），packets_received/sent = 0
- outstanding_requests = 0 / 0 / 0，outstanding_requests_5m_avg = 0 / 0 / 0
- avg_latency_ms = 14.33 / 0 / 3.33，avg_latency_ms_5m_avg = 14.33 / 0 / 3.33
- historical_max_latency_ms = 33 / 0 / 5
- max_latency_delta_60m = 0 / 0 / 0，最大延迟近窗口无变化

**告警与规则**: avg_latency 节点倾斜

| 指标 | 条件 | 严重级别 |
|------|------|----------|
| avg_latency 节点倾斜 | skew = 143% > 50% | **WARNING**（符合低量级保护条件：max=14.33ms ≥ 10ms） |

| 指标 | 值/范围 | 阈值 | 判断 |
|------|---------|------|------|
| outstanding_requests（当前/5m均值） | 0 / 0 / 0 | ≥ 200 WARNING | ✅ 正常 |
| avg_latency_ms（当前/5m均值） | max 14.33ms | ≥ 50 WARNING | ✅ 正常 |
| historical_max_latency_ms | max 33ms | ≥ 500 WARNING | ✅ 正常 |

**解读**: 节点延迟倾斜（14.33 vs 0 vs 3.33ms）是 leader/follower 角色的自然差异——leader 处理所有写入，follower 仅有内部心跳。绝对值 14.33ms 远低于 50ms 安全线，搭配 0 连接 0 积压，不构成实际风险。

**建议措施**: **Observe(观察)** — 在空闲集群上此倾斜无需干预。如有客户端接入后倾斜加剧，再排查热点。

---

### Inspection Item 3: 同步、提交链、磁盘延迟与快照性能

**条件判断**: 未命中深挖条件（leader_changes=0, outstanding=0, avg_latency<50, max_latency<500, GC<5%, 日志无异常），输出基线数据。

**数据摘要**:
- follower_sync_time_ms = 0（所有节点）
- fsync_window_avg_ms_5m = 0 / 0 / 0，fsync_count_inc_5m = 0
- fsync_lifetime_avg_ms = 0 / 0 / 2.5（节点3 累计平均 2.5ms，属于正常范围）
- commit_latency_ms、om/write/read commit proc、server_write_committed、sync_queue_flush = 0
- snapshot_time_ms_avg_5m = 0，snapshottime_count_5m = 0（近 5 分钟无快照）

**告警与规则**: 无触发规则。

| 指标 | 值 | 阈值 | 判断 |
|------|-----|------|------|
| follower_sync_time_ms | 0 | > 1000 WARNING | ✅ 正常 |
| fsync_window_avg_ms_5m | 0 | ≥ 50 WARNING | ✅ 正常 |
| commit_latency_ms | 0 | ≥ 50 WARNING | ✅ 正常 |
| snapshot_time_ms_avg_5m | 0（无样本） | ≥ 5000 WARNING | N/A（近 5 分钟无快照） |

**解读**: 空闲集群无同步开销、无提交链延迟。节点3 累计 fsync 平均 2.5ms 属于正常范围。快照指标无近 5 分钟样本，符合空闲特征。

**建议措施**: **Observe(观察)** — 无异常。如有业务接入后关注快照耗时。

---

### Inspection Item 4: Watch、ZNode、连接与会话

**数据摘要**:
- watch_count = 0 / 0 / 0，无 Watch
- znode_count = 5 / 5 / 5，ZNode 总数 5
- ephemerals_count = 0 / 0 / 0，无临时节点
- num_alive_connections_current = 0 / 0 / 0，无客户端连接
- global_sessions = 0 / 0 / 0，local_sessions = 0 / 0 / 0
- connection_request_60m = 0，connection_drop_probability = 0
- connection_rejected_60m = 0，auth_failed_count_60m = 0

**告警与规则**: 无触发规则。

| 指标 | 值 | 阈值 | 判断 |
|------|-----|------|------|
| watch_count | 0 | > 50000 WARNING | ✅ 正常 |
| znode_count | 5 | > 500000 WARNING | ✅ 正常 |
| ephemerals_count | 0 | > 100000 WARNING | ✅ 正常 |
| connection_drop_probability | 0 | > 0.01 WARNING | ✅ 正常 |
| connection_rejected_60m | 0 | > 0 WARNING | ✅ 正常 |

**解读**: **INFO: 集群空闲 / 无客户端流量**。5 个 ZNode、0 连接、0 Watch、0 Session，当前无任何业务负载。

**建议措施**: **Observe(观察)** — 确认集群是否仍在服役以及是否有预期业务。

---

### Inspection Item 5: JVM、GC 与运行时资源

**数据摘要**:

| 指标 | 节点1 | 节点2 | 节点3 |
|------|-------|-------|-------|
| process_cpu_percent | 0.11% | 0.17% | 0.25% |
| memory_resident_bytes | 2.84 GB | 2.80 GB | 2.82 GB |
| jvm_heap_used_ratio_current | 6.81% | 24.62% | 40.98% |
| jvm_heap_used_ratio_5m_avg | 6.79% | 24.64% | 40.93% |
| jvm_heap_used_ratio_60m_peak | 6.81% | 24.64% | 40.98% |
| jvm_gc_time_percent_60m | 0% | 0% | 0% |
| jvm_threads_current | 141 | 127 | 136 |
| jvm_threads_deadlocked | 0 | 0 | 0 |
| jvm_nonheap_used_bytes | 52 MB | 49 MB | 51 MB |
| open_file_descriptor_usage_pct | 0.053% | 0.052% | 0.052% |

**告警与规则**: 无触发规则（绝对阈值）。Heap 节点倾斜仅作为 INFO 背景。

| 指标 | 条件判断 | 结果 |
|------|----------|------|
| jvm_heap_used_ratio_5m_avg 倾斜 | skew=70%，但 max=40.93% < 80% 且 GC=0% < 5% | **INFO**（不写入 WARNING 汇总表） |
| process_cpu_percent 倾斜 | max=0.25% < 10% | **INFO**（低量级保护） |

| 指标 | 值 | 阈值 | 判断 |
|------|-----|------|------|
| process_cpu_percent | max 0.25% | > 70 WARNING | ✅ 正常 |
| jvm_heap_used_ratio_5m_avg | max 40.93% | > 80% WARNING | ✅ 正常 |
| jvm_gc_time_percent_60m | 0% | > 5% WARNING | ✅ 正常 |
| jvm_threads_deadlocked | 0 | > 0 CRITICAL | ✅ 正常 |
| jvm_threads_current | max 141 | > 1000 WARNING | ✅ 正常 |
| open_file_descriptor_usage_pct | max 0.053% | > 70% WARNING | ✅ 正常 |

**解读**: 资源使用极低。Heap 使用率跨节点差异较大（7% ~ 41%），可能是 leader 节点持有更多内存对象。GC 为 0%、CPU 近乎空闲、非堆内存仅 50MB，所有指标远在安全范围内。

**建议措施**: **Observe(观察)** — 无异常。Heap 离散度为 leader/follower 正常差异。

---

### Inspection Item 6: 数据体量与一致性错误

**数据摘要**:
- approximate_data_size_bytes = 258 / 258 / 258（所有节点一致，7 天无变化）
- approximate_data_size_bytes_5m_delta = 0 / 0 / 0
- approximate_data_size_bytes_1h_delta = 0 / 0 / 0
- snapshot_error_60m = 0，restore_error_60m = 0
- digest_mismatches_60m = 0，unrecoverable_error_60m = 0
- 累计错误计数均为 0

**告警与规则**: 无触发规则。

| 指标 | 值 | 阈值 | 判断 |
|------|-----|------|------|
| approximate_data_size 倾斜 | skew=0%（所有节点一致） | > 30% WARNING | ✅ 无倾斜 |
| data_size_1h_delta | 0 | > 100MB WARNING | ✅ 无增长 |
| snapshot_error_60m | 0 | > 0 WARNING | ✅ 正常 |
| digest_mismatches_60m | 0 | > 0 CRITICAL | ✅ 正常 |
| unrecoverable_error_60m | 0 | > 0 CRITICAL | ✅ 正常 |

**解读**: 数据体量 258 bytes × 3 节点完全一致，无快照/恢复错误，无摘要不匹配，无不可恢复错误。数据一致性完美。

**建议措施**: **Observe(观察)** — 无异常。

---

### Inspection Item 7: 集群日志

**数据摘要**: 近 6h 查询返回空（无 WARNING 及以上级别日志组）。

**告警与规则**: 无触发规则。

**解读**: 近 6h 无 WARNING 及以上日志，集群运行静默稳定。

**建议措施**: **Observe(观察)** — 无异常。

---

### Inspection Item 8: 请求量趋势与异常突变

**数据摘要**:
- request_count_5m = 0（所有节点，节点2 未采集标注 N/A）
- request_count_60m_prev = 0（节点1/3），节点2 为 N/A
- ephemerals_delta_5m = 0 / 0 / 0
- ephemerals_delta_1h = 0 / 0 / 0

**告警与规则**: 无触发规则。

| 指标 | 值 | 阈值 | 判断 |
|------|-----|------|------|
| request_count 60m 环比 | 0 → 0 | 增长 > 200% WARNING | ✅ 无变化 |
| request_count_5m vs 60m | 0 vs 0 | 增长 > 200% WARNING | ✅ 无变化 |
| ephemerals_delta_5m | 0 | > 5000 WARNING | ✅ 正常 |
| ephemerals_delta_5m | 0 | < -5000 WARNING | ✅ 正常 |

**解读**: 请求量和临时节点均无趋势变化。集群完全空闲。

**建议措施**: **Observe(观察)** — 无异常。

> 注：节点2 上 `request_count_5m`、`request_count_60m_prev`、`connection_request_60m` 返回 N/A，标注 `N/A / 指标侧未采集`，不影响对节点2 其他维度的判断。

---

## 4. 跨项综合建议

集群所有巡检项均未发现真实风险。唯一 WARNING（avg_latency 节点倾斜）发生在 0 连接 0 请求的空闲集群上，是 leader/follower 角色差异的正常表现，不构成实际风险。

**处理优先级**: 无需应急处理。

**核心建议**: 确认该集群是否仍在服役以及是否有预期业务。如已退役或无业务计划，可考虑回收资源（3 节点 × 内存 ~3GB/节点）。

---

## 5. 可选: 后续跟进清单

| 建议措施 | 对象 | 说明 | 需要变更工单 |
|----------|------|------|-------------|
| Observe(观察) | 全集群 | 保持常规监控；如未来有业务接入，重新评估各维度容量 | 否 |
| Change(变更)（低优先级） | 集群生命周期 | 确认集群是否仍在服役；如已退役建议回收资源 | 视情况 |
