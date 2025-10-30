package sdk

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"
)

// HTTPClient HTTP 客户端包装
type HTTPClient struct {
	client       *http.Client
	baseURL      string
	cookies      map[string]string // 存储提取的 Cookie
	skipTLSVerify bool             // 跳过 TLS 验证（仅用于测试）
	verbose      bool               // 详细输出
}

// NewHTTPClient 创建新的 HTTP 客户端
func NewHTTPClient(baseURL string) *HTTPClient {
	// 配置 TLS，默认跳过验证（仅用于测试环境）
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}

	return &HTTPClient{
		client: &http.Client{
			Timeout:   60 * time.Second, // 增加默认超时到 60 秒
			Transport: tr,
		},
		baseURL:       baseURL,
		cookies:       make(map[string]string),
		skipTLSVerify: true, // 默认跳过 TLS 验证
		verbose:       false,
	}
}

// SetVerbose 设置详细输出模式
func (c *HTTPClient) SetVerbose(verbose bool) {
	c.verbose = verbose
}

// Response 响应结构
type Response struct {
	Status  int
	Headers map[string][]string
	Body    string
	Cookies []*http.Cookie
}

// RequestOptions 请求选项
type RequestOptions struct {
	Method      string
	Path        string
	Headers     map[string]string
	Body        string
	UseCookie   string
	Timeout     time.Duration
	RetryCount  int
}

// ExecuteRequest 执行 HTTP 请求
func (c *HTTPClient) ExecuteRequest(opts RequestOptions) (*Response, error) {
	var lastErr error
	
	// 处理 URL 拼接
	url := c.baseURL
	// 移除 baseURL 末尾的斜杠
	url = strings.TrimSuffix(url, "/")
	// 确保 path 以 / 开头
	path := opts.Path
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	url += path

	// 确保超时时间至少 60 秒（用于 HTTPS/TLS）
	if opts.Timeout < 60*time.Second {
		opts.Timeout = 60 * time.Second
	}

	if c.verbose {
		log.Printf("[请求] %s %s (超时: %v, 重试: %d)", opts.Method, url, opts.Timeout, opts.RetryCount)
	}

	// 创建带 TLS 配置和超时的传输层
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: c.skipTLSVerify,
		},
	}

	for i := 0; i <= opts.RetryCount; i++ {
		if i > 0 {
			delay := time.Second * time.Duration(i*2) // 递增重试延迟
			if c.verbose {
				log.Printf("[重试] 等待 %v 后重试 (第 %d/%d 次)", delay, i, opts.RetryCount)
			}
			time.Sleep(delay)
		}

		// 创建请求体
		var bodyReader io.Reader
		if opts.Body != "" {
			bodyReader = bytes.NewBufferString(opts.Body)
		}

		// 创建请求
		req, err := http.NewRequest(opts.Method, url, bodyReader)
		if err != nil {
			lastErr = fmt.Errorf("创建请求失败: %w", err)
			if c.verbose {
				log.Printf("[错误] %v", lastErr)
			}
			continue
		}

		// 设置请求头
		for k, v := range opts.Headers {
			req.Header.Set(k, v)
		}

		// 处理 Cookie
		if opts.UseCookie != "" {
			// 如果 use_cookie 是特殊标识，使用提取的 Cookie
			if opts.UseCookie == "response.extracted_cookie" {
				// 使用存储的 Cookie
				cookieStr := c.GetStoredCookie()
				if cookieStr != "" {
					req.Header.Set("Cookie", cookieStr)
				}
			} else {
				// 直接使用提供的 Cookie 字符串
				req.Header.Set("Cookie", opts.UseCookie)
			}
		}

		// 创建带超时和 TLS 配置的客户端
		client := &http.Client{
			Timeout:   opts.Timeout,
			Transport: tr,
		}

		// 执行请求
		startTime := time.Now()
		if c.verbose {
			log.Printf("[发送] 开始发送请求到 %s", url)
		}

		resp, err := client.Do(req)
		duration := time.Since(startTime)

		if err != nil {
			lastErr = fmt.Errorf("请求失败 (耗时: %v): %w", duration, err)
			if c.verbose {
				log.Printf("[错误] %v", lastErr)
			}
			continue
		}
		defer resp.Body.Close()

		if c.verbose {
			log.Printf("[响应] 状态码: %d, 耗时: %v", resp.StatusCode, duration)
		}

		// 读取响应体
		bodyBytes, err := io.ReadAll(resp.Body)
		if err != nil {
			lastErr = fmt.Errorf("读取响应体失败: %w", err)
			if c.verbose {
				log.Printf("[错误] %v", lastErr)
			}
			continue
		}

		if c.verbose {
			log.Printf("[响应] 响应体大小: %d 字节", len(bodyBytes))
		}

		response := &Response{
			Status:  resp.StatusCode,
			Headers: resp.Header,
			Body:    string(bodyBytes),
			Cookies: resp.Cookies(),
		}

		return response, nil
	}

	return nil, fmt.Errorf("请求失败，已重试 %d 次: %w", opts.RetryCount, lastErr)
}

// StoreCookie 存储提取的 Cookie
func (c *HTTPClient) StoreCookie(cookieStr string) {
	// 简单存储，实际可能需要解析多个 Cookie
	c.cookies["extracted"] = cookieStr
}

// GetStoredCookie 获取存储的 Cookie
func (c *HTTPClient) GetStoredCookie() string {
	return c.cookies["extracted"]
}

