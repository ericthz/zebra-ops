package plan_execute_replan

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestFlexiblePlanUnmarshal(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want []string
	}{
		{"标准数组格式", `{"steps":["获取告警","查询文档"]}`, []string{"获取告警", "查询文档"}},
		{"字符串化数组格式", `{"steps":"[\"获取告警\",\"查询文档\"]"}`, []string{"获取告警", "查询文档"}},
		{"单字符串步骤", `{"steps":"仅一步"}`, []string{"仅一步"}},
	}
	for _, c := range cases {
		p := &flexiblePlan{}
		if err := json.Unmarshal([]byte(c.in), p); err != nil {
			t.Errorf("%s: 反序列化失败: %v", c.name, err)
			continue
		}
		if !reflect.DeepEqual(p.Steps, c.want) {
			t.Errorf("%s: got %v want %v", c.name, p.Steps, c.want)
		}
	}
}

func TestFlexiblePlanFirstStep(t *testing.T) {
	p := &flexiblePlan{Steps: []string{"第一步", "第二步"}}
	if got := p.FirstStep(); got != "第一步" {
		t.Errorf("FirstStep got %q want %q", got, "第一步")
	}
	empty := &flexiblePlan{}
	if got := empty.FirstStep(); got != "" {
		t.Errorf("empty FirstStep got %q want empty", got)
	}
}

func TestNormalizeMarkdown(t *testing.T) {
	in := "## # 告警处理详情\n### ## 活跃告警清单\n## 正常标题\n正文内容\n## # 告警根因分析N(第N个告警)"
	want := "## 告警处理详情\n### 活跃告警清单\n## 正常标题\n正文内容\n## 告警根因分析N(第N个告警)"
	if got := normalizeMarkdown(in); got != want {
		t.Errorf("got:\n%q\nwant:\n%q", got, want)
	}
}

func TestIsReportContent(t *testing.T) {
	if !isReportContent("## 告警运维分析报告\n内容") {
		t.Error("应识别 告警运维分析报告")
	}
	if !isReportContent("# 告警处理详情") {
		t.Error("应识别 告警处理详情")
	}
	if isReportContent("普通文本") {
		t.Error("不应识别普通文本")
	}
}

func TestIsCompleteReport(t *testing.T) {
	// 短报告不完整
	if isCompleteReport("短") {
		t.Error("短报告不应视为完整")
	}
	// 长且含处理方案
	report := "# 告警运维分析报告\n" + repeat("# 告警处理详情\n", 60) + "## 处理方案执行 1\n步骤"
	if !isCompleteReport(report) {
		t.Error("长报告含处理方案应视为完整")
	}
	// 长但无处理方案
	longNoPlan := "# 告警运维分析报告\n" + repeat("内容\n", 100)
	if isCompleteReport(longNoPlan) {
		t.Error("无处理方案的长报告不应视为完整")
	}
}

func TestExtractRespond(t *testing.T) {
	if got := extractRespond(`{"response":"最终报告"}`); got != "最终报告" {
		t.Errorf("extractRespond got %q", got)
	}
	if got := extractRespond("普通文本"); got != "" {
		t.Errorf("普通文本应返回空, got %q", got)
	}
}

func repeat(s string, n int) string {
	out := ""
	for i := 0; i < n; i++ {
		out += s
	}
	return out
}
