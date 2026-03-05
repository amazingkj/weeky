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

func (s *HiworksService) TestLogin(officeID, userID, password string) error {
	jar, _ := cookiejar.New(nil)
	fresh := &HiworksService{
		client: &http.Client{
			Timeout: 30 * time.Second,
			Jar:     jar,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return nil
			},
		},
	}
	return fresh.login(officeID, userID, password)
}

func (s *HiworksService) Sync(req model.HiworksSyncRequest) (*model.SyncResult, error) {
	result := &model.SyncResult{
		Source:   "hiworks",
		Items:    []model.SyncItem{},
		SyncedAt: time.Now(),
	}

	jar, _ := cookiejar.New(nil)
	fresh := &HiworksService{
		client: &http.Client{
			Timeout: 60 * time.Second,
			Jar:     jar,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return nil
			},
		},
	}

	if err := fresh.login(req.OfficeID, req.UserID, req.Password); err != nil {
		return nil, fmt.Errorf("Hiworks 로그인 실패: %w", err)
	}

	mails, err := fresh.fetchSentMails(req.OfficeID, req.StartDate, req.EndDate)
	if err != nil {
		return nil, fmt.Errorf("보낸메일 조회 실패: %w", err)
	}

	result.Items = mails
	return result, nil
}

type hiworksLoginRequest struct {
	ID              string `json:"id"`
	Password        string `json:"password"`
	OfficeID        string `json:"office_id"`
	IPSecurityLevel string `json:"ip_security_level"`
}

func (s *HiworksService) login(officeID, userID, password string) error {
	s.officeID = officeID

	loginPageURL := fmt.Sprintf("https://login.office.hiworks.com/%s/", officeID)
	pageReq, _ := http.NewRequest("GET", loginPageURL, nil)
	pageReq.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36")
	pageResp, err := s.client.Do(pageReq)
	if err != nil {
		return fmt.Errorf("로그인 페이지 접근 실패: %w", err)
	}
	pageResp.Body.Close()

	loginURL := "https://auth-api.office.hiworks.com/office-web/login"

	loginID := userID
	if !strings.Contains(userID, "@") {
		loginID = fmt.Sprintf("%s@%s", userID, officeID)
	}

	loginReq := hiworksLoginRequest{
		ID:              loginID,
		Password:        password,
		OfficeID:        officeID,
		IPSecurityLevel: "-1",
	}

	jsonBody, _ := json.Marshal(loginReq)

	req, err := http.NewRequest("POST", loginURL, strings.NewReader(string(jsonBody)))
	if err != nil {
		return err
	}

	req.AddCookie(&http.Cookie{Name: "h_officeid", Value: officeID})
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Origin", "https://login.office.hiworks.com")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36")
	req.Header.Set("Referer", fmt.Sprintf("https://login.office.hiworks.com/%s/", officeID))

	resp, err := s.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("로그인 실패: status %d, %s", resp.StatusCode, string(body))
	}

	mailPageURL := "https://mails.office.hiworks.com/"
	mailReq, _ := http.NewRequest("GET", mailPageURL, nil)
	mailReq.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")
	mailResp, _ := s.client.Do(mailReq)
	if mailResp != nil {
		mailResp.Body.Close()
	}

	s.cookies = []*http.Cookie{}
	for _, rawURL := range []string{
		"https://login.office.hiworks.com",
		"https://office.hiworks.com",
		"https://mails.office.hiworks.com",
		"https://mail-api.office.hiworks.com",
	} {
		u, _ := url.Parse(rawURL)
		s.cookies = append(s.cookies, s.client.Jar.Cookies(u)...)
	}
	s.cookies = append(s.cookies, &http.Cookie{
		Name:  "h_officeid",
		Value: officeID,
	})

	return nil
}

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

type hiworksMailDetailResponse struct {
	Data struct {
		No      int64  `json:"no"`
		Subject string `json:"subject"`
		Body    string `json:"body"`      // HTML body
		BodyTxt string `json:"body_txt"`  // Plain text body
	} `json:"data"`
}

func (s *HiworksService) fetchSentMails(officeID, startDate, endDate string) ([]model.SyncItem, error) {
	mailURL := "https://mail-api.office.hiworks.com/v2/mails?page[limit]=50&page[offset]=0&sort[received_date]=desc&filter[mailbox_id][eq]=b1"

	req, err := http.NewRequest("GET", mailURL, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")
	req.Header.Set("Accept", "application/json;charset=UTF-8")
	req.Header.Set("Cache-Control", "no-cache")

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

	start, _ := time.Parse("2006-01-02", startDate)
	end, _ := time.Parse("2006-01-02", endDate)
	end = end.Add(24 * time.Hour) // Include end date

	var items []model.SyncItem
	for _, mail := range mailResp.Data {
		dateStr := mail.ReceivedDate

		var mailDate time.Time
		if len(dateStr) >= 10 {
			mailDate, _ = time.Parse("2006-01-02", dateStr[:10])
		}

		if !mailDate.IsZero() && (mailDate.Before(start) || mailDate.After(end)) {
			continue
		}

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

	maxFetch := 10
	if len(items) < maxFetch {
		maxFetch = len(items)
	}
	for i := 0; i < maxFetch; i++ {
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

	content := mailResp.Data.BodyTxt
	if content == "" {
		content = mailResp.Data.Body
	}

	if len(content) > 500 {
		content = content[:500] + "..."
	}

	return content, nil
}
