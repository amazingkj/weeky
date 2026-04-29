package service

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/jiin/weeky/internal/model"
)

type JiraService struct {
	client *http.Client

	fieldMu        sync.Mutex
	fieldCacheKey  string // baseURL + email (사용자/인스턴스별 캐시 분리)
	solutionFieldID string // 솔루션명 customfield ID
	siteFieldID     string // 요청사이트 customfield ID
}

func NewJiraService() *JiraService {
	return &JiraService{
		client: &http.Client{Timeout: 30 * time.Second},
	}
}

type jiraSearchRequest struct {
	JQL        string   `json:"jql"`
	Fields     []string `json:"fields"`
	MaxResults int      `json:"maxResults"`
}

type jiraSearchResponse struct {
	Issues []json.RawMessage `json:"issues"`
	Total  int               `json:"total"`
}

type jiraField struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

func (s *JiraService) Sync(req model.JiraSyncRequest) (*model.SyncResult, error) {
	if err := ValidateExternalURL(req.BaseURL); err != nil {
		return nil, fmt.Errorf("invalid Jira URL: %w", err)
	}

	result := &model.SyncResult{
		Source:   "jira",
		Items:    []model.SyncItem{},
		SyncedAt: time.Now(),
	}

	auth := base64.StdEncoding.EncodeToString([]byte(req.Email + ":" + req.Token))

	// 솔루션명/요청사이트 customfield ID 자동 조회 (실패해도 동기화는 계속 — 그 경우 해당 값만 빈 문자열)
	solutionID, siteID := s.resolveCustomFields(req.BaseURL, req.Email, auth)

	searchURL := fmt.Sprintf("%s/rest/api/3/search/jql", req.BaseURL)
	jql := fmt.Sprintf(
		`assignee = currentUser() AND updated >= "%s" AND updated <= "%s" ORDER BY updated DESC`,
		req.StartDate, req.EndDate,
	)

	fields := []string{"summary", "status", "updated", "resolutiondate", "duedate"}
	if solutionID != "" {
		fields = append(fields, solutionID)
	}
	if siteID != "" {
		fields = append(fields, siteID)
	}

	rawIssues, err := s.fetchIssues(searchURL, auth, jql, fields)
	if err != nil {
		return nil, err
	}

	for _, raw := range rawIssues {
		item, ok := parseIssue(raw, req.BaseURL, solutionID, siteID)
		if !ok {
			continue
		}
		result.Items = append(result.Items, item)
	}

	return result, nil
}

func (s *JiraService) resolveCustomFields(baseURL, email, auth string) (solutionID, siteID string) {
	s.fieldMu.Lock()
	defer s.fieldMu.Unlock()

	cacheKey := baseURL + "|" + email
	if s.fieldCacheKey == cacheKey {
		return s.solutionFieldID, s.siteFieldID
	}

	httpReq, err := http.NewRequest("GET", baseURL+"/rest/api/3/field", nil)
	if err != nil {
		return "", ""
	}
	httpReq.Header.Set("Authorization", "Basic "+auth)
	httpReq.Header.Set("Accept", "application/json")

	resp, err := s.client.Do(httpReq)
	if err != nil {
		return "", ""
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", ""
	}

	var fields []jiraField
	if err := json.NewDecoder(resp.Body).Decode(&fields); err != nil {
		return "", ""
	}

	solutionID = findCustomFieldID(fields, "솔루션명")
	siteID = findCustomFieldID(fields, "요청사이트")

	s.fieldCacheKey = cacheKey
	s.solutionFieldID = solutionID
	s.siteFieldID = siteID
	return solutionID, siteID
}

func findCustomFieldID(fields []jiraField, name string) string {
	for _, f := range fields {
		if f.Name == name {
			return f.ID
		}
	}
	return ""
}

func (s *JiraService) fetchIssues(searchURL, auth, jql string, fields []string) ([]json.RawMessage, error) {
	body := jiraSearchRequest{
		JQL:        jql,
		Fields:     fields,
		MaxResults: 50,
	}
	bodyJSON, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequest("POST", searchURL, bytes.NewReader(bodyJSON))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Authorization", "Basic "+auth)
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json")

	resp, err := s.client.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Jira API returned status %d", resp.StatusCode)
	}

	var searchResp jiraSearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&searchResp); err != nil {
		return nil, err
	}

	return searchResp.Issues, nil
}

func parseIssue(raw json.RawMessage, baseURL, solutionID, siteID string) (model.SyncItem, bool) {
	var envelope struct {
		Key    string                     `json:"key"`
		Fields map[string]json.RawMessage `json:"fields"`
	}
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return model.SyncItem{}, false
	}

	summary := decodeString(envelope.Fields["summary"])
	updated := decodeString(envelope.Fields["updated"])
	resolutionDate := decodeString(envelope.Fields["resolutiondate"])
	dueDate := decodeString(envelope.Fields["duedate"])
	status := decodeStatusName(envelope.Fields["status"])

	date := ""
	if len(updated) >= 10 {
		date = updated[:10]
	}
	if len(resolutionDate) >= 10 {
		date = resolutionDate[:10]
	}

	solution := ""
	if solutionID != "" {
		solution = decodeFieldValue(envelope.Fields[solutionID])
	}
	site := ""
	if siteID != "" {
		site = decodeFieldValue(envelope.Fields[siteID])
	}

	return model.SyncItem{
		Title:    summary,
		Date:     date,
		URL:      fmt.Sprintf("%s/browse/%s", baseURL, envelope.Key),
		Type:     "issue",
		Content:  status,
		DueDate:  dueDate,
		Solution: solution,
		Site:     site,
	}, true
}

func decodeString(raw json.RawMessage) string {
	if len(raw) == 0 || string(raw) == "null" {
		return ""
	}
	var s string
	if err := json.Unmarshal(raw, &s); err != nil {
		return ""
	}
	return s
}

func decodeStatusName(raw json.RawMessage) string {
	if len(raw) == 0 || string(raw) == "null" {
		return ""
	}
	var st struct {
		Name string `json:"name"`
	}
	if err := json.Unmarshal(raw, &st); err != nil {
		return ""
	}
	return st.Name
}

// decodeFieldValue handles Jira customfield value shapes:
//   - string: "한화손해보험"
//   - {"value": "..."}: single-select option
//   - [{"value": "..."}, ...]: multi-select / labels-style — joined with ", "
//   - null / 그 외: ""
func decodeFieldValue(raw json.RawMessage) string {
	if len(raw) == 0 || string(raw) == "null" {
		return ""
	}
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		return s
	}
	var obj struct {
		Value string `json:"value"`
		Name  string `json:"name"`
	}
	if err := json.Unmarshal(raw, &obj); err == nil && (obj.Value != "" || obj.Name != "") {
		if obj.Value != "" {
			return obj.Value
		}
		return obj.Name
	}
	var arr []json.RawMessage
	if err := json.Unmarshal(raw, &arr); err == nil {
		parts := make([]string, 0, len(arr))
		for _, el := range arr {
			if v := decodeFieldValue(el); v != "" {
				parts = append(parts, v)
			}
		}
		return strings.Join(parts, ", ")
	}
	return ""
}
