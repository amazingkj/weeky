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
	Items     []model.SyncItem `json:"items"`
	StartDate string           `json:"start_date"`
	EndDate   string           `json:"end_date"`
	Style     string           `json:"style"` // "concise" or "detailed"
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

	// Build prompt with synced items
	style := req.Style
	if style == "" {
		style = "concise"
	}
	prompt := buildPrompt(req.Items, req.StartDate, req.EndDate, style)

	// Call Claude API
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

	body, _ := io.ReadAll(resp.Body)

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

	// Parse Claude's response
	return parseClaudeResponse(claudeResp.Content[0].Text)
}

func buildPrompt(items []model.SyncItem, startDate, endDate, style string) string {
	var b strings.Builder

	fmt.Fprintf(&b, `당신은 주간 업무 보고서 작성을 돕는 어시스턴트입니다.

다음은 %s ~ %s 기간 동안의 업무 활동 목록입니다:

`, startDate, endDate)

	// Group items by type, then by source project
	type sourceGroup struct {
		commits []model.SyncItem
		mrs     []model.SyncItem
	}
	gitlabByProject := make(map[string]*sourceGroup) // key: project source
	var projectOrder []string                          // preserve insertion order
	var issuesDone, issuesTodo, emails []model.SyncItem

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
		case "issue_done", "issue":
			issuesDone = append(issuesDone, item)
		case "issue_todo":
			issuesTodo = append(issuesTodo, item)
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

	if len(issuesDone) > 0 {
		b.WriteString("## Jira 완료 이슈:\n")
		for _, i := range issuesDone {
			fmt.Fprintf(&b, "- [%s] %s\n", i.Date, i.Title)
		}
		b.WriteString("\n")
	}

	if len(issuesTodo) > 0 {
		b.WriteString("## Jira 미완료 이슈 (진행중/대기):\n")
		for _, i := range issuesTodo {
			fmt.Fprintf(&b, "- [%s] %s\n", i.Date, i.Title)
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

	b.WriteString("위 활동들을 분석하여 주간 업무 보고서를 작성해주세요.\n\n")

	if style == "very_detailed" {
		b.WriteString(`요구사항:
1. 프로젝트/고객사별로 업무를 그룹화
2. title: 짧은 프로젝트명 또는 고객사명 (예: "삼성카드", "Mesh", "기술검토")
3. details: 해당 프로젝트에서 수행한 진행사항을 2~3줄로 구체적으로 작성
   - 예시: "모니모 APIM imanager 프로젝트 API 설계 및 구현, 인증 모듈 개발, QA 환경 배포"
4. description: 진행사항 **완전 상세내용**. 모든 세부 작업을 빠짐없이 "- " 접두사로 나열
   - 줄바꿈(\n)을 사용하여 각 항목 구분
   - 커밋 메시지, MR, Jira 이슈의 내용을 최대한 반영하여 구체적으로 기술
   - 각 항목을 기술적으로 상세하게 작성 (어떤 모듈, 어떤 기능, 어떤 환경 등)
   - 예시: "- API Gateway 라우팅 설정 및 인증 미들웨어 구현\n- JWT 토큰 검증 로직 추가 (RS256 알고리즘)\n- 사용자 권한 체계 설계 (RBAC) 및 DB 스키마 반영\n- QA 환경 배포 및 통합 테스트 수행\n- API 문서 (Swagger) 업데이트"
5. due_date: YYYY-MM-DD 형식
6. 메일 제목/내용, 커밋 메시지, Jira 이슈를 분석해서 프로젝트를 식별
7. "this_week" (금주실적): 커밋, MR, 완료된 Jira 이슈, 메일 기반으로 해당 기간의 모든 업무를 빠짐없이 포함. progress는 100
8. "next_week" (차주계획): 미완료 Jira 이슈 기반으로 구체적 계획 작성. progress는 0
9. 하나의 프로젝트에 대해 세부 업무가 많으면 title이 같은 Task를 여러 개 생성하여 카테고리별로 분리
   - 예: title "삼성카드"로 "API 개발" Task와 "테스트/배포" Task를 분리
10. summary: 금주 전체 업무를 3~5줄로 요약 (주요 성과, 진행 현황, 특이사항 포함)
`)
	} else if style == "detailed" {
		b.WriteString(`요구사항:
1. 프로젝트/고객사별로 업무를 그룹화
2. title: 짧은 프로젝트명 또는 고객사명 (예: "삼성카드", "Mesh", "기술검토")
3. details: 해당 프로젝트에서 수행한 진행사항 한 줄 요약
   - 예시: "모니모 APIM imanager 프로젝트 진행"
4. description: 진행사항 상세내용. 세부 작업을 "- " 접두사로 여러 줄 나열
   - 줄바꿈(\n)을 사용하여 각 항목 구분
   - 예시: "- API 설계 및 구현\n- 인증 모듈 JWT 토큰 검증 로직 추가\n- QA 환경 배포 및 테스트 수행"
   - 예시: "- Backend 아키텍처 설계 문서 작성\n- DB 스키마 리뷰 및 인덱스 최적화"
5. due_date: YYYY-MM-DD 형식
6. 메일 제목/내용, 커밋 메시지, Jira 이슈를 분석해서 프로젝트를 식별
7. "this_week" (금주실적): 커밋, MR, 완료된 Jira 이슈, 메일 기반으로 해당 기간의 모든 업무를 포함. progress는 100
8. "next_week" (차주계획): 미완료 Jira 이슈 기반. progress는 0
`)
	} else {
		b.WriteString(`요구사항:
1. 프로젝트/고객사별로 업무를 그룹화
2. title: 짧은 프로젝트명 또는 고객사명 (예: "삼성카드", "Mesh", "기술검토")
3. details: 해당 프로젝트에서 수행한 진행사항을 **한 줄로 간결하게** 작성
   - 예시: "모니모 APIM imanager 프로젝트 진행"
   - 예시: "Backend 아키텍처 설계 및 문서화"
4. due_date: YYYY-MM-DD 형식
5. 메일 제목/내용, 커밋 메시지, Jira 이슈를 분석해서 프로젝트를 식별
6. "this_week" (금주실적): 커밋, MR, 완료된 Jira 이슈, 메일 기반으로 해당 기간의 모든 업무를 포함. progress는 100
7. "next_week" (차주계획): 미완료 Jira 이슈 기반. progress는 0
`)
	}

	b.WriteString(`
다음 JSON 형식으로 응답해주세요:
{
  "this_week": [
    {
      "title": "삼성카드",
      "details": "모니모 APIM imanager 프로젝트 진행",
      "description": "- API 설계 및 구현\n- 인증 모듈 JWT 토큰 검증 로직 추가",
      "due_date": "2026-01-24",
      "progress": 100
    }
  ],
  "next_week": [
    {
      "title": "Mesh",
      "details": "Backend 아키텍처 설계 계속 진행",
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

	// Find JSON in response (might be wrapped in markdown code blocks)
	jsonStr := text
	if start := findJSONStart(text); start >= 0 {
		if end := findJSONEnd(text, start); end > start {
			jsonStr = text[start:end]
		}
	}

	if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
		return nil, fmt.Errorf("Claude 응답 파싱 실패: %w\n응답: %s", err, text)
	}

	// Support both old "tasks" format and new "this_week"/"next_week" format
	thisWeekTasks := result.ThisWeek
	if len(thisWeekTasks) == 0 {
		thisWeekTasks = result.Tasks
	}

	toModelTasks := func(items []taskJSON) []model.Task {
		tasks := make([]model.Task, 0, len(items))
		for _, t := range items {
			tasks = append(tasks, model.Task{
				Title:       t.Title,
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
