package sdk

import (
	"fmt"
	"regexp"
	"strings"
)

// Engine POC 执行引擎
type Engine struct {
	config       *POCConfig
	httpClient   *HTTPClient
	evaluator    *ExpressionEvaluator
	cookieExtractor *CookieExtractor
	ruleResults  map[string]bool // 存储规则执行结果
	verbose      bool
}

// NewEngine 创建执行引擎
func NewEngine(config *POCConfig, baseURL string) *Engine {
	client := NewHTTPClient(baseURL)
	return &Engine{
		config:        config,
		httpClient:   client,
		evaluator:    NewExpressionEvaluator(),
		cookieExtractor: NewCookieExtractor(),
		ruleResults:  make(map[string]bool),
		verbose:      false,
	}
}

// SetVerbose 设置详细输出模式
func (e *Engine) SetVerbose(verbose bool) {
	e.verbose = verbose
	e.httpClient.SetVerbose(verbose)
}

// Execute 执行整个 POC
func (e *Engine) Execute() (bool, error) {
	// 先执行所有规则
	for ruleName, rule := range e.config.Rules {
		success, err := e.executeRule(ruleName, rule)
		if err != nil {
			return false, fmt.Errorf("执行规则 %s 失败: %w", ruleName, err)
		}
		e.ruleResults[ruleName] = success
	}

	// 评估主表达式
	if e.config.Expression != "" {
		return e.evaluateMainExpression(e.config.Expression)
	}

	// 如果没有主表达式，检查所有规则是否都成功
	for _, success := range e.ruleResults {
		if !success {
			return false, nil
		}
	}

	return true, nil
}

// executeRule 执行单个规则
func (e *Engine) executeRule(ruleName string, rule *Rule) (bool, error) {
	// 准备请求选项
	opts := RequestOptions{
		Method:     rule.Method,
		Path:       rule.Path,
		Headers:    rule.Headers,
		Body:       rule.GetBody(),
		UseCookie:  rule.UseCookie,
		Timeout:    rule.GetTimeout(),
		RetryCount: rule.GetRetryCount(),
	}

	// 执行 HTTP 请求
	response, err := e.httpClient.ExecuteRequest(opts)
	if err != nil {
		return false, fmt.Errorf("HTTP 请求失败: %w", err)
	}

	// 提取 Cookie
	if rule.ExtractCookie != "" {
		cookie, err := e.cookieExtractor.ExtractCookie(rule.ExtractCookie, response)
		if err == nil && cookie != "" {
			e.httpClient.StoreCookie(cookie)
		}
	}

	// 验证 Cookie 表达式
	if rule.CookieExpression != "" {
		cookieToValidate := e.httpClient.GetStoredCookie()
		if rule.UseCookie != "" {
			// 如果规则指定了 use_cookie，使用它
			if rule.UseCookie != "response.extracted_cookie" {
				cookieToValidate = rule.UseCookie
			}
		}
		valid, err := e.cookieExtractor.ValidateCookie(rule.CookieExpression, cookieToValidate)
		if err != nil {
			return false, fmt.Errorf("Cookie 验证失败: %w", err)
		}
		if !valid {
			return false, fmt.Errorf("Cookie 验证不通过")
		}
	}

	// 评估规则表达式
	if rule.Expression != "" {
		cookieStr := e.httpClient.GetStoredCookie()
		valid, err := e.evaluator.Evaluate(rule.Expression, response, cookieStr)
		if err != nil {
			return false, fmt.Errorf("表达式评估失败: %w", err)
		}
		if !valid {
			return false, fmt.Errorf("规则表达式不满足: %s", rule.Expression)
		}
	}

	return true, nil
}

// evaluateMainExpression 评估主表达式（如 "r0() && r1() && r2()" 或 "r0 && r1"）
func (e *Engine) evaluateMainExpression(expr string) (bool, error) {
	// 移除注释
	expr = e.removeComments(expr)
	expr = strings.TrimSpace(expr)

	// 先替换规则调用（如 r0()）为规则结果
	re1 := regexp.MustCompile(`(\w+)\(\)`)
	expr = re1.ReplaceAllStringFunc(expr, func(match string) string {
		ruleName := strings.TrimSuffix(match, "()")
		if result, ok := e.ruleResults[ruleName]; ok {
			if result {
				return "true"
			}
			return "false"
		}
		return "false"
	})

	// 再处理简写格式（如 r0 或 r1）
	// 查找所有规则名（r 开头后跟数字），但要避免替换已替换的值
	re2 := regexp.MustCompile(`\b(r\d+)\b`)
	expr = re2.ReplaceAllStringFunc(expr, func(match string) string {
		if result, ok := e.ruleResults[match]; ok {
			if result {
				return "true"
			}
			return "false"
		}
		return "false"
	})

	// 评估简化后的表达式
	if strings.Contains(expr, "&&") {
		parts := strings.Split(expr, "&&")
		for _, part := range parts {
			part = strings.TrimSpace(part)
			if part == "false" {
				return false, nil
			}
		}
		return true, nil
	}

	if strings.Contains(expr, "||") {
		parts := strings.Split(expr, "||")
		for _, part := range parts {
			part = strings.TrimSpace(part)
			if part == "true" {
				return true, nil
			}
		}
		return false, nil
	}

	// 单个值
	return expr == "true", nil
}

func (e *Engine) removeComments(s string) string {
	idx := strings.Index(s, "#")
	if idx != -1 {
		return s[:idx]
	}
	return s
}


// GetRuleResult 获取规则执行结果
func (e *Engine) GetRuleResult(ruleName string) (bool, bool) {
	result, ok := e.ruleResults[ruleName]
	return result, ok
}

// GetAllRuleResults 获取所有规则执行结果
func (e *Engine) GetAllRuleResults() map[string]bool {
	return e.ruleResults
}

