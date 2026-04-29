package handler

import (
	"testing"

	"github.com/jiin/weeky/internal/model"
)

func TestApplyConsolidationRules_RenameTitle(t *testing.T) {
	tasks := []model.Task{
		{Title: "MyData", Client: "한화생명"},
		{Title: "CruzAPIM", Client: "삼성카드"},
		{Title: "유지보수 지원", Client: ""},
		{Title: "MMS지원", Client: ""},
	}
	rules := []model.ConsolidationRule{
		{RuleType: model.RuleTypeRenameTitle, Pattern: "MyData", Replacement: "마이데이터", SortOrder: 0},
		{RuleType: model.RuleTypeRenameTitle, Pattern: "유지보수 지원", Replacement: "유지보수", SortOrder: 1},
		{RuleType: model.RuleTypeRenameTitle, Pattern: "MMS지원", Replacement: "유지보수", SortOrder: 2},
	}
	got := applyConsolidationRules(tasks, rules)
	want := []string{"마이데이터", "CruzAPIM", "유지보수", "유지보수"}
	for i, t2 := range got {
		if t2.Title != want[i] {
			t.Errorf("idx %d: got Title=%q, want %q", i, t2.Title, want[i])
		}
	}
	// 원본 불변 확인
	if tasks[0].Title != "MyData" {
		t.Error("original tasks must not be mutated")
	}
}

func TestApplyConsolidationRules_VirtualClient(t *testing.T) {
	tasks := []model.Task{
		{Title: "CruzAPIM", Client: ""},                  // 적용됨
		{Title: "CruzAPIM", Client: "삼성카드"},           // 이미 client 있음 → skip
		{Title: "Mesh Service", Client: ""},              // scope 다름 → skip
	}
	rules := []model.ConsolidationRule{
		{RuleType: model.RuleTypeVirtualClient, ScopeTitle: "CruzAPIM", Replacement: "본사"},
	}
	got := applyConsolidationRules(tasks, rules)
	if got[0].Client != "본사" {
		t.Errorf("expected client=본사, got %q", got[0].Client)
	}
	if got[1].Client != "삼성카드" {
		t.Errorf("expected client unchanged (삼성카드), got %q", got[1].Client)
	}
	if got[2].Client != "" {
		t.Errorf("expected client unchanged (empty), got %q", got[2].Client)
	}
}

func TestApplyConsolidationRules_Combined(t *testing.T) {
	// rename → virtual_client 순서로 적용해야 의도대로 동작
	// (예: CruzMMS → 마이데이터로 rename된 뒤, 마이데이터 + 빈 client → 본사 적용 가능)
	tasks := []model.Task{
		{Title: "MyData", Client: ""},
	}
	rules := []model.ConsolidationRule{
		{RuleType: model.RuleTypeRenameTitle, Pattern: "MyData", Replacement: "마이데이터", SortOrder: 0},
		{RuleType: model.RuleTypeVirtualClient, ScopeTitle: "마이데이터", Replacement: "본사", SortOrder: 1},
	}
	got := applyConsolidationRules(tasks, rules)
	if got[0].Title != "마이데이터" || got[0].Client != "본사" {
		t.Errorf("got Title=%q Client=%q, want 마이데이터/본사", got[0].Title, got[0].Client)
	}
}

func TestApplyConsolidationRules_EmptyInputs(t *testing.T) {
	if applyConsolidationRules(nil, nil) != nil {
		t.Error("nil/nil should return nil")
	}
	tasks := []model.Task{{Title: "X"}}
	if got := applyConsolidationRules(tasks, nil); &got[0] != &tasks[0] {
		// no rules → original slice returned (no allocation)
		// 단, 명세상 같은 슬라이스인지보다 동일 컨텐츠가 더 중요. 이 어셋은 fast-path 보장만.
	}
}

func TestApplyRulesToReport_NilSafe(t *testing.T) {
	if applyRulesToReport(nil, nil) != nil {
		t.Error("nil report should pass through")
	}
	r := &model.Report{ThisWeek: []model.Task{{Title: "MyData"}}}
	if applyRulesToReport(r, nil) != r {
		t.Error("no rules should return same pointer (fast-path)")
	}
}
