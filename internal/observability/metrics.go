// Package observability 提供极简 Prometheus 文本格式指标与健康检查。
// 自实现轻量注册表，避免为一个小型服务引入重依赖。
package observability

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

// DurationBuckets 直方图桶（单位：秒）
var DurationBuckets = []float64{0.01, 0.05, 0.1, 0.5, 1, 2, 5, 10}

// counter 计数器指标，存储标签和当前值
type counter struct {
	labels string // 形如 {method="GET",path="/api/chat"}
	value  float64
}

// histogram 直方图指标，存储标签、桶分布、计数和总和
type histogram struct {
	labels  string
	buckets map[string]float64
	count   float64
	sum     float64
}

// Metrics 线程安全的指标注册表，Render 输出 Prometheus 文本格式。
type Metrics struct {
	mu         sync.Mutex
	helps      map[string]string
	counters   map[string]*counter
	histograms map[string]*histogram
}

// New 创建指标注册表
func New() *Metrics {
	return &Metrics{
		helps:      map[string]string{},
		counters:   map[string]*counter{},
		histograms: map[string]*histogram{},
	}
}

// Default 全局默认注册表
var Default = New()

// labelSuffix 将标签 map 转换为 Prometheus 格式的标签后缀，如 {method="GET",path="/api/chat"}
func labelSuffix(labels map[string]string) string {
	keys := make([]string, 0, len(labels))
	for k := range labels {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		parts = append(parts, fmt.Sprintf("%s=%q", k, labels[k]))
	}
	return "{" + strings.Join(parts, ",") + "}"
}

// Inc 自增计数器（+1）
func (m *Metrics) Inc(name, help string, labels map[string]string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.helps[name] = help
	suffix := labelSuffix(labels)
	if c, ok := m.counters[name+suffix]; ok {
		c.value++
		return
	}
	m.counters[name+suffix] = &counter{labels: suffix, value: 1}
}

// Observe 记录一次耗时观测（直方图）
func (m *Metrics) Observe(name, help string, labels map[string]string, d time.Duration) {
	secs := d.Seconds()
	m.mu.Lock()
	defer m.mu.Unlock()
	m.helps[name] = help
	suffix := labelSuffix(labels)
	key := name + suffix
	h, ok := m.histograms[key]
	if !ok {
		h = &histogram{labels: suffix, buckets: map[string]float64{}}
		for _, b := range DurationBuckets {
			h.buckets[formatFloat(b)] = 0
		}
		m.histograms[key] = h
	}
	h.count++
	h.sum += secs
	for _, b := range DurationBuckets {
		if secs <= b {
			h.buckets[formatFloat(b)]++
		}
	}
}

// Render 输出 Prometheus 文本格式快照
func (m *Metrics) Render() string {
	m.mu.Lock()
	defer m.mu.Unlock()

	var b strings.Builder
	// 计数器按名称排序输出，保证确定性
	counterNames := map[string]bool{}
	for key := range m.counters {
		counterNames[key[:strings.Index(key, "{")]] = true
	}
	histNames := map[string]bool{}
	for key := range m.histograms {
		histNames[key[:strings.Index(key, "{")]] = true
	}

	for _, name := range sortedKeys(counterNames) {
		fmt.Fprintf(&b, "# HELP %s %s\n", name, m.helps[name])
		fmt.Fprintf(&b, "# TYPE %s counter\n", name)
		for _, key := range sortedKeysOf(m.counters, name) {
			c := m.counters[key]
			fmt.Fprintf(&b, "%s%s %s\n", name, c.labels, formatFloat(c.value))
		}
	}

	for _, name := range sortedKeys(histNames) {
		fmt.Fprintf(&b, "# HELP %s %s\n", name, m.helps[name])
		fmt.Fprintf(&b, "# TYPE %s histogram\n", name)
		for _, key := range sortedKeysOf(m.histograms, name) {
			h := m.histograms[key]
			for _, bucket := range DurationBuckets {
				le := formatFloat(bucket)
				fmt.Fprintf(&b, "%s%s,le=%q} %s\n", name, h.labels[:len(h.labels)-1], le, formatFloat(h.buckets[le]))
			}
			fmt.Fprintf(&b, "%s_sum%s %s\n", name, h.labels, formatFloat(h.sum))
			fmt.Fprintf(&b, "%s_count%s %s\n", name, h.labels, formatFloat(h.count))
		}
	}
	return b.String()
}

// sortedKeys 从 map 中提取所有 key 并按字母序排序返回
func sortedKeys(set map[string]bool) []string {
	keys := make([]string, 0, len(set))
	for k := range set {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// sortedKeysOf 从 map 中提取所有以指定前缀开头的 key 并按字母序排序返回
func sortedKeysOf[T any](m map[string]T, prefix string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		if strings.HasPrefix(k, prefix) {
			keys = append(keys, k)
		}
	}
	sort.Strings(keys)
	return keys
}

// formatFloat 将浮点数格式化为紧凑字符串表示
func formatFloat(f float64) string {
	return strconv.FormatFloat(f, 'g', -1, 64)
}
