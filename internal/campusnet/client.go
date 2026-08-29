package campusnet

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"regexp"
	"strings"
	"time"
)

var portalHrefPattern = regexp.MustCompile(`href=['\"]([^'\"]+)['\"]`)

type Client struct {
	httpClient *http.Client
	userAgent  string
	timeout    time.Duration
}

func New(timeout time.Duration, userAgent string) *Client {
	jar, _ := cookiejar.New(nil)
	return &Client{
		httpClient: &http.Client{
			Timeout: timeout,
			Jar:     jar,
		},
		userAgent: userAgent,
		timeout:   timeout,
	}
}

func (c *Client) TestConnection(checkURL, checkHost string) bool {
	req, err := http.NewRequest(http.MethodGet, checkURL, nil)
	if err != nil {
		return false
	}
	req.Header.Set("User-Agent", c.userAgent)
	if checkHost != "" {
		req.Host = checkHost
	}

	checkClient := &http.Client{
		Timeout: c.timeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	resp, err := checkClient.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNoContent {
		return true
	}
	if resp.StatusCode != http.StatusOK {
		return false
	}
	if strings.Contains(strings.ToLower(checkURL), "generate_204") {
		return false
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1024))
	if err != nil {
		return false
	}
	text := strings.ToLower(string(body))
	blockedMarkers := []string{
		"login",
		"sso",
		"portal",
		"eportal",
		"认证",
		"登录",
	}
	for _, marker := range blockedMarkers {
		if strings.Contains(text, marker) {
			return false
		}
	}
	return true
}

func (c *Client) Login(username, password, service string) error {
	portalURL, err := c.GetPortalURL()
	if err != nil {
		return err
	}
	return c.OldLoginService(portalURL, username, password, service)
}

func (c *Client) GetPortalURL() (string, error) {
	req, err := http.NewRequest(http.MethodGet, "http://123.123.123.123", nil)
	if err != nil {
		return "", fmt.Errorf("create portal request: %w", err)
	}
	req.Header.Set("User-Agent", c.userAgent)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("get portal URL: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("get portal URL: %s", resp.Status)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read portal response: %w", err)
	}

	match := portalHrefPattern.FindStringSubmatch(string(body))
	if match == nil {
		return "", errors.New("get portal URL: cannot find portal href")
	}
	return match[1], nil
}

func (c *Client) OldLoginService(portalURL, userID, password, service string) error {
	host, queryString, err := splitPortalURL(portalURL)
	if err != nil {
		return err
	}

	// The campus portal expects the service value pre-escaped, matching the
	// established request format used by the legacy client.
	form := url.Values{
		"userId":          {userID},
		"password":        {password},
		"service":         {url.QueryEscape(service)},
		"queryString":     {queryString},
		"operatorPwd":     {""},
		"operatorUserId":  {""},
		"validcode":       {""},
		"passwordEncrypt": {"false"},
	}

	endpoint := fmt.Sprintf("http://%s/eportal/InterFace.do?method=login", host)
	req, err := http.NewRequest(http.MethodPost, endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return fmt.Errorf("create login request: %w", err)
	}
	req.Header.Set("User-Agent", c.userAgent)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("login service: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("login service: %s", resp.Status)
	}

	var result struct {
		Result  string `json:"result"`
		Message string `json:"message"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("decode login response: %w", err)
	}
	if result.Result != "success" {
		if result.Message == "" {
			result.Message = "unknown campusnet login failure"
		}
		return errors.New(result.Message)
	}
	return nil
}

func splitPortalURL(portalURL string) (string, string, error) {
	u, err := url.Parse(portalURL)
	if err != nil {
		return "", "", fmt.Errorf("parse portal URL: %w", err)
	}
	if u.Host == "" || u.RawQuery == "" {
		return "", "", fmt.Errorf("invalid portal URL: %s", portalURL)
	}
	return u.Host, u.RawQuery, nil
}
