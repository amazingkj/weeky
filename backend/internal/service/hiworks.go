package service

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
	"time"

	"github.com/jiin/weeky/internal/model"
)

type HiworksService struct {
	client   *http.Client
	cookies  []*http.Cookie
	officeID string
}

func NewHiworksService() *HiworksService {
	jar, _ := cookiejar.New(nil)
	return &HiworksService{
		client: &http.Client{
			Timeout: 60 * time.Second,
			Jar:     jar,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return nil // Follow redirects
			},
		},
	}
}

func (s *HiworksService) Sync(req model.HiworksSyncRequest) (*model.SyncResult, error) {
	result := &model.SyncResult{
		Source:   "hiworks",
		Items:    []model.SyncItem{},
		SyncedAt: time.Now(),
	}

	// Step 1: Login to Hiworks
	if err := s.login(req.OfficeID, req.UserID, req.Password); err != nil {
		return nil, fmt.Errorf("Hiworks 로그인 실패: %w", err)
	}

	// Step 2: Fetch sent mails
	mails, err := s.fetchSentMails(req.OfficeID, req.StartDate, req.EndDate)
	if err != nil {
		return nil, fmt.Errorf("보낸메일 조회 실패: %w", err)
	}

	result.Items = mails
	return result, nil
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen]
}

// Login request body
type hiworksLoginRequest struct {
	ID              string `json:"id"`
	Password        string `json:"password"`
	IPSecurityLevel string `json:"ip_security_level"`
}

func (s *HiworksService) login(officeID, userID, password string) error {
	s.officeID = officeID

	// Step 1: Visit login page first to get session cookie
	loginPageURL := fmt.Sprintf("https://login.office.hiworks.com/%s/", officeID)
	pageReq, _ := http.NewRequest("GET", loginPageURL, nil)
	pageReq.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")
	pageResp, err := s.client.Do(pageReq)
	if err != nil {
		return fmt.Errorf("로그인 페이지 접근 실패: %w", err)
	}
	pageResp.Body.Close()

	// Step 2: POST login credentials as JSON
	loginURL := "https://auth-api.office.hiworks.com/office-web/login"

	// Prepare login JSON body - ID format: userid@officeid
	// userID can be just "jikim" or full email "jikim@direa.co.kr"
	loginID := userID
	if !strings.Contains(userID, "@") {
		loginID = fmt.Sprintf("%s@%s", userID, officeID)
	}

	loginReq := hiworksLoginRequest{
		ID:              loginID,
		Password:        password,
		IPSecurityLevel: "-1",
	}

	jsonBody, _ := json.Marshal(loginReq)

	req, err := http.NewRequest("POST", loginURL, strings.NewReader(string(jsonBody)))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")
	req.Header.Set("Referer", fmt.Sprintf("https://login.office.hiworks.com/%s/", officeID))

	resp, err := s.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Check for login failure
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("로그인 실패: status %d, %s", resp.StatusCode, string(body))
	}

	// Step 3: Visit mail page to establish session there
	mailPageURL := "https://mails.office.hiworks.com/"
	mailReq, _ := http.NewRequest("GET", mailPageURL, nil)
	mailReq.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")
	mailResp, _ := s.client.Do(mailReq)
	if mailResp != nil {
		mailResp.Body.Close()
	}

	// Collect all cookies from multiple domains
	s.cookies = []*http.Cookie{}

	// From login.office.hiworks.com
	u1, _ := url.Parse("https://login.office.hiworks.com")
	s.cookies = append(s.cookies, s.client.Jar.Cookies(u1)...)

	// From office.hiworks.com
	u0, _ := url.Parse("https://office.hiworks.com")
	s.cookies = append(s.cookies, s.client.Jar.Cookies(u0)...)

	// From mails.office.hiworks.com
	u2, _ := url.Parse("https://mails.office.hiworks.com")
	s.cookies = append(s.cookies, s.client.Jar.Cookies(u2)...)

	// From mail-api.office.hiworks.com
	u3, _ := url.Parse("https://mail-api.office.hiworks.com")
	s.cookies = append(s.cookies, s.client.Jar.Cookies(u3)...)

	// Add h_officeid cookie
	s.cookies = append(s.cookies, &http.Cookie{
		Name:  "h_officeid",
		Value: officeID,
	})

	return nil
}

// Hiworks mail API response structure
type hiworksMailResponse struct {
	Data []struct {
		No           int64  `json:"no"`
		Subject      string `json:"subject"`
		ReceivedDate string `json:"received_date"`
		From         string `json:"from"`
		MailboxID    string `json:"mailbox_id"`
	} `json:"data"`
	Meta struct {
		Page struct {
			Total  int `json:"total"`
			Limit  int `json:"limit"`
			Offset int `json:"offset"`
		} `json:"page"`
	} `json:"meta"`
}

// Hiworks individual mail detail response
type hiworksMailDetailResponse struct {
	Data struct {
		No      int64  `json:"no"`
		Subject string `json:"subject"`
		Body    string `json:"body"`      // HTML body
		BodyTxt string `json:"body_txt"`  // Plain text body
	} `json:"data"`
}

func (s *HiworksService) fetchSentMails(officeID, startDate, endDate string) ([]model.SyncItem, error) {
	// Hiworks mail API - sent mailbox (b1 = sent)
	mailURL := "https://mail-api.office.hiworks.com/v2/mails?page[limit]=50&page[offset]=0&sort[received_date]=desc&filter[mailbox_id][eq]=b1"

	req, err := http.NewRequest("GET", mailURL, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")
	req.Header.Set("Accept", "application/json;charset=UTF-8")
	req.Header.Set("Cache-Control", "no-cache")

	// Add cookies from login session
	for _, cookie := range s.cookies {
		req.AddCookie(cookie)
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("메일 API 오류: status %d, body: %s", resp.StatusCode, string(body))
	}

	var mailResp hiworksMailResponse
	if err := json.NewDecoder(resp.Body).Decode(&mailResp); err != nil {
		return nil, fmt.Errorf("응답 파싱 실패: %w", err)
	}

	// Parse dates for filtering
	start, _ := time.Parse("2006-01-02", startDate)
	end, _ := time.Parse("2006-01-02", endDate)
	end = end.Add(24 * time.Hour) // Include end date

	var items []model.SyncItem
	for _, mail := range mailResp.Data {
		dateStr := mail.ReceivedDate

		// Parse mail date (format: 2026-01-26T07:25:06Z)
		var mailDate time.Time
		if len(dateStr) >= 10 {
			mailDate, _ = time.Parse("2006-01-02", dateStr[:10])
		}

		// Filter by date range
		if !mailDate.IsZero() && (mailDate.Before(start) || mailDate.After(end)) {
			continue
		}

		// Format date as YYYY-MM-DD
		formattedDate := ""
		if len(dateStr) >= 10 {
			formattedDate = dateStr[:10]
		}

		items = append(items, model.SyncItem{
			Title: mail.Subject,
			Date:  formattedDate,
			URL:   fmt.Sprintf("https://mails.office.hiworks.com/view/%d", mail.No),
			Type:  "email",
		})
	}

	// Fetch content for each mail (limit to first 10 to avoid too many requests)
	maxFetch := 10
	if len(items) < maxFetch {
		maxFetch = len(items)
	}
	for i := 0; i < maxFetch; i++ {
		// Extract mail ID from URL
		var mailNo int64
		fmt.Sscanf(items[i].URL, "https://mails.office.hiworks.com/view/%d", &mailNo)
		if mailNo > 0 {
			content, err := s.fetchMailContent(mailNo)
			if err == nil && content != "" {
				items[i].Content = content
			}
		}
	}

	return items, nil
}

func (s *HiworksService) fetchMailContent(mailNo int64) (string, error) {
	mailURL := fmt.Sprintf("https://mail-api.office.hiworks.com/v2/mails/%d", mailNo)

	req, err := http.NewRequest("GET", mailURL, nil)
	if err != nil {
		return "", err
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")
	req.Header.Set("Accept", "application/json;charset=UTF-8")

	// Add cookies from login session
	for _, cookie := range s.cookies {
		req.AddCookie(cookie)
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("메일 상세 조회 실패: status %d", resp.StatusCode)
	}

	var mailResp hiworksMailDetailResponse
	if err := json.NewDecoder(resp.Body).Decode(&mailResp); err != nil {
		return "", err
	}

	// Prefer plain text, fallback to HTML (truncated)
	content := mailResp.Data.BodyTxt
	if content == "" {
		content = mailResp.Data.Body
	}

	// Truncate to 500 chars to avoid too much data
	if len(content) > 500 {
		content = content[:500] + "..."
	}

	return content, nil
}
