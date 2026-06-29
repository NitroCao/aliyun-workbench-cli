package workbench

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

const (
	workbenchBase  = "https://ecs-workbench.aliyun.com"
	ecsConsoleBase = "https://ecsnew.console.aliyun.com"
)

type Client struct {
	ticket string
	http   *http.Client
}

func NewClient(ticket string) (*Client, error) {
	if strings.TrimSpace(ticket) == "" {
		return nil, errors.New("empty login_aliyunid_ticket")
	}
	return &Client{
		ticket: strings.TrimSpace(ticket),
		http: &http.Client{
			Timeout: 30 * time.Second,
		},
	}, nil
}

func (c *Client) ResourceList(ctx context.Context, region, instanceID string) ([]Resource, error) {
	endpoint := fmt.Sprintf("%s/instance/ecs/resource/list/%s/%s", workbenchBase, url.PathEscape(region), url.PathEscape(instanceID))
	form := url.Values{
		"resourceGroupId": {"-1"},
		"instanceType":    {"ecs"},
	}

	var out apiResponse[resourceListRoot]
	if err := c.postWorkbenchForm(ctx, endpoint, region, instanceID, form, "", &out); err != nil {
		return nil, err
	}
	if !out.Success {
		return nil, fmt.Errorf("resource list failed: %s", out.Message)
	}
	return out.Root.Resources, nil
}

func (c *Client) AcquireRequestToken(ctx context.Context, region, instanceID string) (string, error) {
	endpoint := workbenchBase + "/request/token/acquireRequestToken"
	var out apiResponse[string]
	if err := c.postWorkbenchForm(ctx, endpoint, region, instanceID, nil, "", &out); err != nil {
		return "", err
	}
	if !out.Success || out.Root == "" {
		return "", fmt.Errorf("acquire request token failed: %s", out.Message)
	}
	return out.Root, nil
}

func (c *Client) LoginInstance(ctx context.Context, region, instanceID, username string, resource Resource, requestToken string) (*LoginResult, error) {
	if username == "" {
		username = "root"
	}
	host, ipAddressType := resource.loginHost()
	if host == "" {
		return nil, fmt.Errorf("no usable IP address found for instance %s", instanceID)
	}
	osType := resource.OSType
	if osType == "" {
		osType = "Linux"
	}
	instanceName := resource.ResourceName
	if instanceName == "" {
		instanceName = instanceID
	}

	form := url.Values{
		"port":                     {"22"},
		"authenticationType":       {"temp_certificate"},
		"checkCertificate":         {"false"},
		"protocol":                 {"ssh"},
		"language":                 {""},
		"charset":                  {""},
		"certificate":              {""},
		"certificateName":          {""},
		"instanceType":             {"ecs"},
		"osType":                   {osType},
		"username":                 {username},
		"password":                 {""},
		"host":                     {host},
		"regionId":                 {region},
		"instanceName":             {instanceName},
		"instanceId":               {instanceID},
		"dockerContainerId":        {""},
		"dockerContainerName":      {""},
		"dockerExec":               {""},
		"dockerImageId":            {""},
		"passPhrase":               {""},
		"networkAccessMode":        {"classic"},
		"resourceGroupId":          {resource.ResourceGroupID},
		"resourceGroupName":        {resource.ResourceGroupName},
		"resourceGroupDisplayName": {resource.ResourceGroupDisplayName},
		"ipAddressType":            {ipAddressType},
		"credentialToken":          {""},
		"from":                     {"ecs"},
	}

	var out apiResponse[json.RawMessage]
	if err := c.postWorkbenchForm(ctx, workbenchBase+"/login/instance/single", region, instanceID, form, requestToken, &out); err != nil {
		return nil, err
	}
	if !out.Success {
		return nil, fmt.Errorf("login instance failed: %s", out.Message)
	}

	var info LoginInfo
	if err := json.Unmarshal(out.Root, &info); err != nil {
		return nil, err
	}
	var root map[string]any
	if err := json.Unmarshal(out.Root, &root); err != nil {
		return nil, err
	}
	if info.InstanceLoginToken == "" {
		return nil, errors.New("login response did not include instanceLoginToken")
	}
	if !info.LoginSuccess {
		return nil, fmt.Errorf("workbench login failed: %s", info.ErrorMessage)
	}

	return &LoginResult{Info: info, Root: root}, nil
}

func (c *Client) CheckInstanceLogin(ctx context.Context, region, instanceID, loginToken string) error {
	endpoint := fmt.Sprintf("%s/instance/%s/login/%s/checkInstanceLogin", workbenchBase, url.PathEscape(instanceID), url.PathEscape(loginToken))
	var out apiResponse[bool]
	if err := c.postWorkbenchForm(ctx, endpoint, region, instanceID, nil, "", &out); err != nil {
		return err
	}
	if !out.Success || !out.Root {
		return fmt.Errorf("check instance login failed: %s", out.Message)
	}
	return nil
}

func (c *Client) ListECSInstances(ctx context.Context, region string, pageNumber, pageSize int) ([]ECSInstance, error) {
	if pageNumber <= 0 {
		pageNumber = 1
	}
	if pageSize <= 0 {
		pageSize = 100
	}
	u, _ := url.Parse(ecsConsoleBase + "/instance/instance/list.json")
	q := u.Query()
	q.Set("regionId", region)
	q.Set("pageNumber", strconv.Itoa(pageNumber))
	q.Set("pageSize", strconv.Itoa(pageSize))
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}
	c.setCookie(req)
	req.Header.Set("Origin", "https://ecs.console.aliyun.com")
	req.Header.Set("Referer", "https://ecs.console.aliyun.com/")
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Accept", "application/json, text/plain, */*")

	resp, err := c.do(req)
	if err != nil {
		return nil, err
	}
	defer closeAndLog(resp.Body)
	if !isSuccessStatus(resp.StatusCode) {
		return nil, fmt.Errorf("list instances failed: %s: %s", resp.Status, readErrorBody(resp.Body))
	}

	var out ecsListResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	if out.SuccessResponse != nil && !*out.SuccessResponse {
		return nil, fmt.Errorf("list instances failed: %s: %s", out.Code, out.Message)
	}
	return out.Data.Instances.Instance, nil
}

func (c *Client) postWorkbenchForm(ctx context.Context, endpoint, region, instanceID string, form url.Values, requestToken string, out any) error {
	var body io.Reader
	if form != nil {
		body = strings.NewReader(form.Encode())
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, body)
	if err != nil {
		return err
	}
	c.setCommonHeaders(req, region, instanceID)
	if form != nil {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded; charset=UTF-8")
	}
	if requestToken != "" {
		req.Header.Set("X-Request-Token", requestToken)
	}

	resp, err := c.do(req)
	if err != nil {
		return err
	}
	defer closeAndLog(resp.Body)
	if !isSuccessStatus(resp.StatusCode) {
		return fmt.Errorf("%s failed: %s: %s", endpoint, resp.Status, readErrorBody(resp.Body))
	}
	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return fmt.Errorf("%s decode failed: %w", endpoint, err)
	}
	return nil
}

func (c *Client) do(req *http.Request) (*http.Response, error) {
	start := time.Now()
	logrus.WithFields(logrus.Fields{
		"content_length": req.ContentLength,
		"method":         req.Method,
		"url":            req.URL.Redacted(),
	}).Debug("http request")

	resp, err := c.http.Do(req)
	if err != nil {
		logrus.WithError(err).WithFields(logrus.Fields{
			"duration": time.Since(start),
			"method":   req.Method,
			"url":      req.URL.Redacted(),
		}).Debug("http request failed")
		return nil, err
	}

	logrus.WithFields(logrus.Fields{
		"duration": time.Since(start),
		"method":   req.Method,
		"status":   resp.StatusCode,
		"url":      req.URL.Redacted(),
	}).Debug("http response")
	return resp, nil
}

func isSuccessStatus(code int) bool {
	return code >= http.StatusOK && code < http.StatusMultipleChoices
}

func readErrorBody(r io.Reader) string {
	const limit = 64 << 10
	body, err := io.ReadAll(io.LimitReader(r, limit+1))
	if err != nil {
		return fmt.Sprintf("<read error: %v>", err)
	}
	if len(body) > limit {
		return string(body[:limit]) + "...<truncated>"
	}
	return string(body)
}

func closeAndLog(closer io.Closer) {
	if err := closer.Close(); err != nil {
		logrus.WithError(err).Debug("close resource")
	}
}

func (c *Client) setCommonHeaders(req *http.Request, region, instanceID string) {
	c.setCookie(req)
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Accept", "application/json, text/javascript, */*; q=0.01")
	req.Header.Set("Origin", "https://ecs-workbench.aliyun.com")
	req.Header.Set("Referer", workbenchURL(region, instanceID))
	req.Header.Set("X-Requested-With", "XMLHttpRequest")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
}

func (c *Client) setCookie(req *http.Request) {
	req.Header.Set("Cookie", "login_aliyunid_ticket="+c.ticket)
}

func workbenchURL(region, instanceID string) string {
	q := url.Values{}
	q.Set("from", "ecs")
	q.Set("instanceType", "ecs")
	q.Set("regionId", region)
	q.Set("instanceId", instanceID)
	q.Set("resourceGroupId", "")
	q.Set("language", "zh-CN")
	return workbenchBase + "/?" + q.Encode()
}

const userAgent = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/146.0.0.0 Safari/537.36"
