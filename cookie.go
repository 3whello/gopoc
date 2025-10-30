package sdk

import (
	"fmt"
	"regexp"
	"strings"
)

// CookieExtractor Cookie 提取器
type CookieExtractor struct {
	response *Response
}

// NewCookieExtractor 创建 Cookie 提取器
func NewCookieExtractor() *CookieExtractor {
	return &CookieExtractor{}
}

// ExtractCookie 根据表达式提取 Cookie
func (ce *CookieExtractor) ExtractCookie(expr string, response *Response) (string, error) {
	ce.response = response

	if response == nil {
		return "", fmt.Errorf("响应为空")
	}

	// 处理 response.headers.get('Set-Cookie')
	if strings.Contains(expr, "response.headers.get") {
		re := regexp.MustCompile(`response\.headers\.get\(['"]([^'"]+)['"]\)`)
		matches := re.FindStringSubmatch(expr)
		if len(matches) != 2 {
			return "", fmt.Errorf("无法解析 headers.get 表达式: %s", expr)
		}

		headerName := matches[1]
		headerNameLower := strings.ToLower(headerName)

		// 查找 Set-Cookie 头
		for k, v := range response.Headers {
			if strings.ToLower(k) == headerNameLower {
				if len(v) > 0 {
					// 如果有多个 Set-Cookie，合并它们
					return strings.Join(v, "; "), nil
				}
			}
		}

		// 也检查 Cookies 字段（http.Cookie）
		if len(response.Cookies) > 0 {
			var cookieParts []string
			for _, cookie := range response.Cookies {
				cookieParts = append(cookieParts, cookie.String())
			}
			return strings.Join(cookieParts, "; "), nil
		}

		return "", nil
	}

	// 处理 response.body.extract('pattern')
	if strings.Contains(expr, "response.body.extract") {
		re := regexp.MustCompile(`response\.body\.extract\(['"]([^'"]+)['"]\)`)
		matches := re.FindStringSubmatch(expr)
		if len(matches) != 2 {
			return "", fmt.Errorf("无法解析 body.extract 表达式: %s", expr)
		}

		pattern := matches[1]
		// 转换 Rust 正则语法到 Go（简化处理）
		pattern = convertRustRegex(pattern)

		regex, err := regexp.Compile(pattern)
		if err != nil {
			return "", fmt.Errorf("无效的正则表达式: %w", err)
		}

		match := regex.FindStringSubmatch(response.Body)
		if len(match) > 1 {
			return match[1], nil
		}

		return "", nil
	}

	// 直接使用提取的 Cookie 变量
	if expr == "response.extracted_cookie" {
		// 这个应该从上下文获取
		return "", fmt.Errorf("extracted_cookie 需要从执行上下文获取")
	}

	return "", fmt.Errorf("不支持的 Cookie 提取表达式: %s", expr)
}

// convertRustRegex 将 Rust 正则语法转换为 Go 正则语法
// 这里是一个简化版本，实际可能需要更复杂的转换
func convertRustRegex(pattern string) string {
	// Rust 使用 r'\d+' 格式，Go 使用 '\d+'
	// 移除 r'...' 包装（如果存在）
	pattern = strings.TrimPrefix(pattern, "r'")
	pattern = strings.TrimPrefix(pattern, "r\"")
	pattern = strings.TrimSuffix(pattern, "'")
	pattern = strings.TrimSuffix(pattern, "\"")

	// 转义处理（Rust 和 Go 的正则转义略有不同）
	return pattern
}

// ValidateCookie 验证 Cookie 表达式
func (ce *CookieExtractor) ValidateCookie(expr string, cookie string) (bool, error) {
	if expr == "" {
		return true, nil
	}

	evaluator := NewExpressionEvaluator()
	// 创建一个虚拟响应，因为 cookie_expression 主要操作 cookie
	dummyResponse := &Response{Status: 200, Body: "", Headers: make(map[string][]string)}
	return evaluator.Evaluate(expr, dummyResponse, cookie)
}

