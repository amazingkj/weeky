package handler

import "github.com/jiin/weeky/internal/model"

// applyConsolidationRules transforms each task's title/client according to the team's rules.
// Rules are applied in sort_order. Currently supported:
//   - RuleTypeRenameTitle: task.Title == pattern → replacement
//   - RuleTypeVirtualClient: task.Title == scope_title AND task.Client is empty → client = replacement
//
// Pure function: original tasks are not mutated; a new transformed slice is returned.
func applyConsolidationRules(tasks []model.Task, rules []model.ConsolidationRule) []model.Task {
	if len(tasks) == 0 || len(rules) == 0 {
		return tasks
	}
	out := make([]model.Task, len(tasks))
	for i, t := range tasks {
		for _, r := range rules {
			switch r.RuleType {
			case model.RuleTypeRenameTitle:
				if t.Title == r.Pattern {
					t.Title = r.Replacement
				}
			case model.RuleTypeVirtualClient:
				if t.Title == r.ScopeTitle && t.Client == "" {
					t.Client = r.Replacement
				}
			}
		}
		out[i] = t
	}
	return out
}

// applyRulesToReport returns a copy of report with rules applied to both this_week and next_week tasks.
// nil report passes through unchanged.
func applyRulesToReport(report *model.Report, rules []model.ConsolidationRule) *model.Report {
	if report == nil || len(rules) == 0 {
		return report
	}
	clone := *report
	clone.ThisWeek = applyConsolidationRules(report.ThisWeek, rules)
	clone.NextWeek = applyConsolidationRules(report.NextWeek, rules)
	return &clone
}
