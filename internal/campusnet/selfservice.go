package campusnet

import (
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"regexp"
	"strings"
	"time"
)

const (
	selfServiceHost          = "10.245.1.113:8080/selfservice/module"
	selfServiceFailureMarker = "操作执行失败"
)

var (
	selfServiceErrorPattern = regexp.MustCompile(`errorMsg=([^&]+)&`)
	onlineIPPattern         = regexp.MustCompile(`<input.*?id="userIp.*?value="(.*?)".*?>`)
)

type SelfService struct {
	httpClient *http.Client
	userAgent  string
}

func NewSelfService(timeout time.Duration, userAgent string) *SelfService {
	jar, _ := cookiejar.New(nil)
	return &SelfService{
		httpClient: &http.Client{Timeout: timeout, Jar: jar},
		userAgent:  userAgent,
	}
}

func NewSelfServiceOnInterface(timeout time.Duration, userAgent, interfaceName string) (*SelfService, error) {
	transport, err := newInterfaceTransport(timeout, interfaceName)
	if err != nil {
		return nil, err
	}
	jar, _ := cookiejar.New(nil)
	return &SelfService{
		httpClient: &http.Client{
			Timeout:   timeout,
			Jar:       jar,
			Transport: transport,
		},
		userAgent: userAgent,
	}, nil
}

func (s *SelfService) Login(username, password string) error {
	form := url.Values{
		"name":     {username},
		"password": {password},
	}
	endpoint := fmt.Sprintf("http://%s/scgroup/web/login_judge.jsf?mobileslef=true", selfServiceHost)
	req, err := http.NewRequest(http.MethodPost, endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return fmt.Errorf("create self-service login request: %w", err)
	}
	req.Header.Set("User-Agent", s.userAgent)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("self-service login: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read self-service login response: %w", err)
	}
	if match := selfServiceErrorPattern.FindStringSubmatch(string(body)); match != nil {
		return fmt.Errorf("self-service login error: %s", match[1])
	}
	return nil
}

func (s *SelfService) GetOnlines() ([]string, error) {
	endpoint := fmt.Sprintf("http://%s/webcontent/web/onlinedevice_list.jsf", selfServiceHost)
	req, err := http.NewRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("create online list request: %w", err)
	}
	req.Header.Set("User-Agent", s.userAgent)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("get online list: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read online list response: %w", err)
	}
	html := string(body)
	if strings.Contains(html, selfServiceFailureMarker) {
		return nil, fmt.Errorf("get online list: remote service returned failure")
	}

	matches := onlineIPPattern.FindAllStringSubmatch(html, -1)
	if len(matches) == 0 {
		return nil, fmt.Errorf("get online list: cannot find online IPs")
	}

	ips := make([]string, 0, len(matches))
	for _, match := range matches {
		ips = append(ips, match[1])
	}
	return ips, nil
}

func (s *SelfService) Logout(username, ip string) error {
	form := url.Values{
		"key": {username + ":" + ip},
	}
	endpoint := fmt.Sprintf("http://%s/userself/web/userself_ajax.jsf?methodName=indexBean.kickUserBySelfForAjax", selfServiceHost)
	req, err := http.NewRequest(http.MethodPost, endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return fmt.Errorf("create logout request: %w", err)
	}
	req.Header.Set("User-Agent", s.userAgent)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("logout %s: %w", ip, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read logout response: %w", err)
	}
	if strings.Contains(string(body), selfServiceFailureMarker) {
		return fmt.Errorf("logout %s: remote service returned failure", ip)
	}
	return nil
}
