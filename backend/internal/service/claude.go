package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/jiin/weeky/internal/model"
)

type ClaudeService struct {
	client *http.Client
	apiKey string
}

func NewClaudeService(apiKey string) *ClaudeService {
	return &ClaudeService{
		client: &http.Client{Timeout: 60 * time.Second},
		apiKey: apiKey,
	}
}

type claudeRequest struct {
	Model     string           `json:"model"`
	MaxTokens int              `json:"max_tokens"`
	Messages  []claudeMessage  `json:"messages"`
}

type claudeMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type claudeResponse struct {
	Content []struct {
		Text string `json:"text"`
	} `json:"content"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error"`
}

type GenerateReportRequest struct {
	Items        []model.SyncItem `json:"items"`
	StartDate    string           `json:"start_date"`
	EndDate      string           `json:"end_date"`
	Style        string           `json:"style"` // "concise" or "detailed"
	ProjectNames []string         `json:"project_names,omitempty"`
}

type GenerateReportResponse struct {
	ThisWeek []model.Task `json:"this_week"`
	NextWeek []model.Task `json:"next_week"`
	Summary  string       `json:"summary"`
}

func (s *ClaudeService) GenerateReport(req GenerateReportRequest) (*GenerateReportResponse, error) {
	if s.apiKey == "" {
		return nil, fmt.Errorf("Claude API 키가 설정되지 않았습니다")
	}

	style := req.Style
	if style == "" {
		style = "concise"
	}
	prompt := buildPrompt(req.Items, req.StartDate, req.EndDate, style, req.ProjectNames)

	maxTokens := 2000
	if style == "detailed" {
		maxTokens = 4000
	} else if style == "very_detailed" {
		maxTokens = 6000
	}
	claudeReq := claudeRequest{
		Model:     "claude-sonnet-4-20250514",
		MaxTokens: maxTokens,
		Messages: []claudeMessage{
			{Role: "user", Content: prompt},
		},
	}

	jsonBody, err := json.Marshal(claudeReq)
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequest("POST", "https://api.anthropic.com/v1/messages", bytes.NewReader(jsonBody))
	if err != nil {
		return nil, err
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", s.apiKey)
	httpReq.Header.Set("anthropic-version", "2023-06-01")

	resp, err := s.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("Claude API 호출 실패: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("응답 읽기 실패: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Claude API 오류: status %d, %s", resp.StatusCode, string(body))
	}

	var claudeResp claudeResponse
	if err := json.Unmarshal(body, &claudeResp); err != nil {
		return nil, fmt.Errorf("응답 파싱 실패: %w", err)
	}

	if claudeResp.Error != nil {
		return nil, fmt.Errorf("Claude API 오류: %s", claudeResp.Error.Message)
	}

	if len(claudeResp.Content) == 0 {
		return nil, fmt.Errorf("Claude 응답이 비어있습니다")
	}

	return parseClaudeResponse(claudeResp.Content[0].Text)
}

func buildPrompt(items []model.SyncItem, startDate, endDate, style string, projectNames []string) string {
	var b strings.Builder

	fmt.Fprintf(&b, `당신은 주간 업무 보고서 작성을 돕는 어시스턴트입니다.

다음은 %s ~ %s 기간 동안의 업무 활동 목록입니다:

`, startDate, endDate)

	type sourceGroup struct {
		commits []model.SyncItem
		mrs     []model.SyncItem
	}
	gitlabByProject := make(map[string]*sourceGroup) // key: project source
	var projectOrder []string                          // preserve insertion order
	var issues, emails []model.SyncItem

	for _, item := range items {
		switch item.Type {
		case "commit":
			key := item.Source
			if key == "" {
				key = "기타"
			}
			if _, ok := gitlabByProject[key]; !ok {
				gitlabByProject[key] = &sourceGroup{}
				projectOrder = append(projectOrder, key)
			}
			gitlabByProject[key].commits = append(gitlabByProject[key].commits, item)
		case "mr", "pr":
			key := item.Source
			if key == "" {
				key = "기타"
			}
			if _, ok := gitlabByProject[key]; !ok {
				gitlabByProject[key] = &sourceGroup{}
				projectOrder = append(projectOrder, key)
			}
			gitlabByProject[key].mrs = append(gitlabByProject[key].mrs, item)
		case "issue_done", "issue_todo", "issue":
			issues = append(issues, item)
		case "email":
			emails = append(emails, item)
		}
	}

	for _, proj := range projectOrder {
		sg := gitlabByProject[proj]
		fmt.Fprintf(&b, "## GitLab 프로젝트: %s\n", proj)
		if len(sg.commits) > 0 {
			b.WriteString("### 커밋:\n")
			for _, c := range sg.commits {
				fmt.Fprintf(&b, "- [%s] %s\n", c.Date, c.Title)
			}
		}
		if len(sg.mrs) > 0 {
			b.WriteString("### Merge Requests:\n")
			for _, m := range sg.mrs {
				fmt.Fprintf(&b, "- [%s] %s\n", m.Date, m.Title)
			}
		}
		b.WriteString("\n")
	}

	if len(issues) > 0 {
		b.WriteString("## Jira 이슈:\n")
		for _, i := range issues {
			status := i.Content
			if status == "" {
				status = "Unknown"
			}
			fmt.Fprintf(&b, "- [%s] %s (상태: %s", i.Date, i.Title, status)
			if i.DueDate != "" {
				fmt.Fprintf(&b, ", 기한: %s", i.DueDate)
			}
			if i.Solution != "" {
				fmt.Fprintf(&b, ", 솔루션: %s", i.Solution)
			}
			if i.Site != "" {
				fmt.Fprintf(&b, ", 요청사이트: %s", i.Site)
			}
			b.WriteString(")\n")
		}
		b.WriteString("\n")
	}

	if len(emails) > 0 {
		b.WriteString("## 보낸 메일:\n")
		for _, e := range emails {
			fmt.Fprintf(&b, "- [%s] %s\n", e.Date, e.Title)
			if e.Content != "" {
				fmt.Fprintf(&b, "  내용: %s\n", e.Content)
			}
		}
		b.WriteString("\n")
	}

	if len(projectNames) > 0 {
		b.WriteString("이 팀에서 사용하는 프로젝트 목록입니다:\n")
		for _, name := range projectNames {
			fmt.Fprintf(&b, "- %s\n", name)
		}
		b.WriteString("\ntask의 title은 위 프로젝트 이름을 **정확히** 사용해주세요.\n")
		b.WriteString("위 목록에 해당하지 않는 새로운 프로젝트가 발견되면 적절한 이름으로 title을 지정해주세요.\n\n")
	}

	b.WriteString("위 활동들을 분석하여 주간 업무 보고서를 작성해주세요.\n\n")

	b.WriteString(`Jira 필드 매핑 규칙 (이 규칙은 어떤 스타일이든 동일하게 적용):
- 요청사이트가 있으면 해당 Task의 client에 그대로 사용 (다른 단서로 추정하지 말 것)
- 솔루션명이 있으면 title에 사용하되 버전 표기는 제거 (예: "CruzAPIM 1.5" → "CruzAPIM")
- 기한이 있으면 due_date에 그대로 사용 (임의 추정 금지)

`)

	if style == "very_detailed" {
		b.WriteString(`요구사항:
1. 프로젝트별로 업무를 그룹화
2. title: 짧은 프로젝트명 (예: "CruzAPIM", "Mesh", "마이데이터")
3. client: 해당 업무의 고객사명 (예: "삼성카드", "도로교통공단", "흥국화재"). 고객사가 없으면 빈 문자열
4. details: 해당 고객사에서 수행한 진행사항을 2~3줄로 구체적으로 작성
   - 예시: "모니모 APIM imanager 프로젝트 API 설계 및 구현, 인증 모듈 개발, QA 환경 배포"
5. description: 진행사항 **완전 상세내용**. 모든 세부 작업을 빠짐없이 "- " 접두사로 나열
   - 줄바꿈(\n)을 사용하여 각 항목 구분
   - 커밋 메시지, MR, Jira 이슈의 내용을 최대한 반영하여 구체적으로 기술
   - 각 항목을 기술적으로 상세하게 작성 (어떤 모듈, 어떤 기능, 어떤 환경 등)
6. due_date: YYYY-MM-DD 형식
7. 메일 제목/내용, 커밋 메시지, Jira 이슈를 분석해서 프로젝트와 고객사를 식별
8. "this_week" (금주실적): 커밋, MR, Jira 이슈, 메일 기반으로 해당 기간의 모든 업무를 빠짐없이 포함
9. "next_week" (차주계획): 미완료 Jira 이슈 중 다음 주에 이어질 작업 기반으로 구체적 계획 작성
10. 같은 title(프로젝트)에 여러 client(고객사)가 있을 수 있음. 각 고객사별로 별도의 Task를 생성
11. summary: 금주 전체 업무를 3~5줄로 요약 (주요 성과, 진행 현황, 특이사항 포함)
12. Jira 티켓 1개 = Task 1개. 각 Jira 티켓은 별도의 Task로 생성하고, title은 프로젝트명, details에 티켓 키와 요약 포함
13. Jira 티켓의 상태에 따라 progress를 추정하세요 (예: To Do→0%, In Progress→30~70%, In Review/QA→70~90%, Done→100%)
14. 커밋/MR 기반 Task의 progress는 100
`)
	} else if style == "detailed" {
		b.WriteString(`요구사항:
1. 프로젝트별로 업무를 그룹화
2. title: 짧은 프로젝트명 (예: "CruzAPIM", "Mesh", "마이데이터")
3. client: 해당 업무의 고객사명 (예: "삼성카드", "도로교통공단"). 고객사가 없으면 빈 문자열
4. details: 해당 고객사에서 수행한 진행사항 한 줄 요약
5. description: 진행사항 상세내용. 세부 작업을 "- " 접두사로 여러 줄 나열
   - 줄바꿈(\n)을 사용하여 각 항목 구분
6. due_date: YYYY-MM-DD 형식
7. 메일 제목/내용, 커밋 메시지, Jira 이슈를 분석해서 프로젝트와 고객사를 식별
8. "this_week" (금주실적): 커밋, MR, Jira 이슈, 메일 기반으로 해당 기간의 모든 업무를 포함
9. "next_week" (차주계획): 미완료 Jira 이슈 중 다음 주에 이어질 작업 기반
10. 같은 title(프로젝트)에 여러 client(고객사)가 있을 수 있음
11. Jira 티켓 1개 = Task 1개. 각 Jira 티켓은 별도의 Task로 생성하고, title은 프로젝트명, details에 티켓 키와 요약 포함
12. Jira 티켓의 상태에 따라 progress를 추정하세요 (예: To Do→0%, In Progress→30~70%, In Review/QA→70~90%, Done→100%)
13. 커밋/MR 기반 Task의 progress는 100
`)
	} else {
		b.WriteString(`요구사항:
1. 프로젝트별로 업무를 그룹화
2. title: 짧은 프로젝트명 (예: "CruzAPIM", "Mesh", "마이데이터")
3. client: 해당 업무의 고객사명 (예: "삼성카드", "도로교통공단"). 고객사가 없으면 빈 문자열
4. details: 해당 고객사에서 수행한 진행사항을 **한 줄로 간결하게** 작성
5. due_date: YYYY-MM-DD 형식
6. 메일 제목/내용, 커밋 메시지, Jira 이슈를 분석해서 프로젝트와 고객사를 식별
7. "this_week" (금주실적): 커밋, MR, Jira 이슈, 메일 기반으로 해당 기간의 모든 업무를 포함
8. "next_week" (차주계획): 미완료 Jira 이슈 중 다음 주에 이어질 작업 기반
9. 같은 title(프로젝트)에 여러 client(고객사)가 있을 수 있음
10. Jira 티켓 1개 = Task 1개. 각 Jira 티켓은 별도의 Task로 생성하고, title은 프로젝트명, details에 티켓 키와 요약 포함
11. Jira 티켓의 상태에 따라 progress를 추정하세요 (예: To Do→0%, In Progress→30~70%, In Review/QA→70~90%, Done→100%)
12. 커밋/MR 기반 Task의 progress는 100
`)
	}

	b.WriteString(`
다음 JSON 형식으로 응답해주세요:
{
  "this_week": [
    {
      "title": "CruzAPIM",
      "client": "삼성카드",
      "details": "[PROJ-101] 모니모 APIM imanager 프로젝트 진행",
      "description": "- API 설계 및 구현\n- 인증 모듈 JWT 토큰 검증 로직 추가",
      "due_date": "2026-01-24",
      "progress": 100
    },
    {
      "title": "CruzAPIM",
      "client": "삼성카드",
      "details": "[PROJ-102] API 에러 핸들링 개선",
      "description": "",
      "due_date": "2026-01-24",
      "progress": 50
    }
  ],
  "next_week": [
    {
      "title": "Mesh",
      "details": "[MESH-55] Backend 아키텍처 설계",
      "description": "",
      "due_date": "2026-01-31",
      "progress": 0
    }
  ],
  "summary": ""
}

description은 상세/완전상세 스타일일 때만 채워주고, 간결 스타일이면 빈 문자열로 두세요.
summary는 완전상세 스타일일 때만 채워주세요.
JSON만 응답하고 다른 텍스트는 포함하지 마세요.`)

	return b.String()
}

func parseClaudeResponse(text string) (*GenerateReportResponse, error) {
	type taskJSON struct {
		Title       string `json:"title"`
		Client      string `json:"client"`
		Details     string `json:"details"`
		Description string `json:"description"`
		DueDate     string `json:"due_date"`
		Progress    int    `json:"progress"`
	}

	var result struct {
		Tasks    []taskJSON `json:"tasks"`
		ThisWeek []taskJSON `json:"this_week"`
		NextWeek []taskJSON `json:"next_week"`
		Summary  string     `json:"summary"`
	}

	jsonStr := text
	if start := findJSONStart(text); start >= 0 {
		if end := findJSONEnd(text, start); end > start {
			jsonStr = text[start:end]
		}
	}

	if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
		return nil, fmt.Errorf("Claude 응답 파싱 실패: %w\n응답: %s", err, text)
	}

	thisWeekTasks := result.ThisWeek
	if len(thisWeekTasks) == 0 {
		thisWeekTasks = result.Tasks
	}

	toModelTasks := func(items []taskJSON) []model.Task {
		tasks := make([]model.Task, 0, len(items))
		for _, t := range items {
			tasks = append(tasks, model.Task{
				Title:       t.Title,
				Client:      t.Client,
				Details:     t.Details,
				Description: t.Description,
				DueDate:     t.DueDate,
				Progress:    t.Progress,
			})
		}
		return tasks
	}

	return &GenerateReportResponse{
		ThisWeek: toModelTasks(thisWeekTasks),
		NextWeek: toModelTasks(result.NextWeek),
		Summary:  result.Summary,
	}, nil
}

func findJSONStart(s string) int {
	for i, c := range s {
		if c == '{' {
			return i
		}
	}
	return -1
}

func findJSONEnd(s string, start int) int {
	depth := 0
	for i := start; i < len(s); i++ {
		switch s[i] {
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return i + 1
			}
		}
	}
	return len(s)
}
