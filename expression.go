package sdk

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// ExpressionEvaluator 表达式评估器
type ExpressionEvaluator struct {
	response *Response
	cookie   string
	context  map[string]interface{} // 存储变量和提取的值
}

// NewExpressionEvaluator 创建表达式评估器
func NewExpressionEvaluator() *ExpressionEvaluator {
	return &ExpressionEvaluator{
		context: make(map[string]interface{}),
	}
}

// Evaluate 评估表达式
func (e *ExpressionEvaluator) Evaluate(expr string, response *Response, cookie string) (bool, error) {
	e.response = response
	e.cookie = cookie

	// 移除注释
	expr = removeComments(expr)
	expr = strings.TrimSpace(expr)

	// 处理逻辑运算符 && 和 ||
	// 注意：&& 优先级高于 ||，需要先处理 ||
	// 但为了简化，我们按出现顺序处理，复杂表达式建议使用括号
	if strings.Contains(expr, "||") {
		return e.evaluateOr(expr)
	}
	if strings.Contains(expr, "&&") {
		return e.evaluateAnd(expr)
	}

	return e.evaluateSingle(expr)
}

func removeComments(s string) string {
	idx := strings.Index(s, "#")
	if idx != -1 {
		return s[:idx]
	}
	return s
}

func (e *ExpressionEvaluator) evaluateAnd(expr string) (bool, error) {
	parts := strings.Split(expr, "&&")
	result := true
	for _, part := range parts {
		part = strings.TrimSpace(part)
		val, err := e.evaluateSingle(part)
		if err != nil {
			return false, err
		}
		result = result && val
	}
	return result, nil
}

func (e *ExpressionEvaluator) evaluateOr(expr string) (bool, error) {
	parts := strings.Split(expr, "||")
	result := false
	for _, part := range parts {
		part = strings.TrimSpace(part)
		val, err := e.evaluateSingle(part)
		if err != nil {
			return false, err
		}
		result = result || val
		if result {
			break
		}
	}
	return result, nil
}

func (e *ExpressionEvaluator) evaluateSingle(expr string) (bool, error) {
	expr = strings.TrimSpace(expr)

	// 处理括号表达式（简化处理）
	if strings.HasPrefix(expr, "(") && strings.HasSuffix(expr, ")") {
		expr = strings.Trim(expr, "()")
		return e.Evaluate(expr, e.response, e.cookie)
	}

	// 优先处理函数调用（返回布尔值的函数）
	if strings.Contains(expr, "response.body.contains") {
		return e.evaluateContains(expr)
	}
	if strings.Contains(expr, "cookie.contains") {
		return e.evaluateCookieContains(expr)
	}

	// 处理比较运算符: ==, !=, >=, <=, >, <
	if strings.Contains(expr, "==") {
		return e.evaluateComparison(expr, "==", func(a, b interface{}) bool {
			return fmt.Sprintf("%v", a) == fmt.Sprintf("%v", b)
		})
	}
	if strings.Contains(expr, "!=") {
		return e.evaluateComparison(expr, "!=", func(a, b interface{}) bool {
			return fmt.Sprintf("%v", a) != fmt.Sprintf("%v", b)
		})
	}
	if strings.Contains(expr, ">=") {
		return e.evaluateNumericComparison(expr, ">=")
	}
	if strings.Contains(expr, "<=") {
		return e.evaluateNumericComparison(expr, "<=")
	}
	if strings.Contains(expr, ">") && !strings.Contains(expr, ">=") {
		return e.evaluateNumericComparison(expr, ">")
	}
	if strings.Contains(expr, "<") && !strings.Contains(expr, "<=") {
		return e.evaluateNumericComparison(expr, "<")
	}

	return false, fmt.Errorf("不支持的表达式: %s", expr)
}

func (e *ExpressionEvaluator) evaluateComparison(expr, op string, compare func(interface{}, interface{}) bool) (bool, error) {
	parts := strings.Split(expr, op)
	if len(parts) != 2 {
		return false, fmt.Errorf("无效的比较表达式: %s", expr)
	}

	left := strings.TrimSpace(parts[0])
	right := strings.TrimSpace(parts[1])

	leftVal, err := e.evaluateValue(left)
	if err != nil {
		return false, err
	}

	rightVal, err := e.evaluateValue(right)
	if err != nil {
		return false, err
	}

	return compare(leftVal, rightVal), nil
}

func (e *ExpressionEvaluator) evaluateNumericComparison(expr, op string) (bool, error) {
	parts := strings.Split(expr, op)
	if len(parts) != 2 {
		return false, fmt.Errorf("无效的数字比较表达式: %s", expr)
	}

	left := strings.TrimSpace(parts[0])
	right := strings.TrimSpace(parts[1])

	leftVal, err := e.evaluateNumericValue(left)
	if err != nil {
		return false, err
	}

	rightVal, err := e.evaluateNumericValue(right)
	if err != nil {
		return false, err
	}

	switch op {
	case ">=":
		return leftVal >= rightVal, nil
	case "<=":
		return leftVal <= rightVal, nil
	case ">":
		return leftVal > rightVal, nil
	case "<":
		return leftVal < rightVal, nil
	}

	return false, fmt.Errorf("不支持的运算符: %s", op)
}

func (e *ExpressionEvaluator) evaluateValue(expr string) (interface{}, error) {
	expr = strings.TrimSpace(expr)

	// 处理字符串字面量
	if strings.HasPrefix(expr, "'") && strings.HasSuffix(expr, "'") {
		return strings.Trim(expr, "'"), nil
	}
	if strings.HasPrefix(expr, "\"") && strings.HasSuffix(expr, "\"") {
		return strings.Trim(expr, "\""), nil
	}

	// 处理 response.status
	if expr == "response.status" {
		if e.response == nil {
			return 0, nil
		}
		return e.response.Status, nil
	}

	// 处理 response.body.contains()
	if strings.Contains(expr, "response.body.contains") {
		return e.evaluateContains(expr)
	}

	// 处理 cookie.contains()
	if strings.Contains(expr, "cookie.contains") {
		return e.evaluateCookieContains(expr)
	}

	// 处理 response.headers.get()
	if strings.Contains(expr, "response.headers.get") {
		return e.evaluateHeaderGet(expr)
	}

	// 处理数字
	if num, err := strconv.Atoi(expr); err == nil {
		return num, nil
	}

	return expr, nil
}

func (e *ExpressionEvaluator) evaluateContains(expr string) (bool, error) {
	// 解析 response.body.contains('text')
	re := regexp.MustCompile(`response\.body\.contains\(['"]([^'"]+)['"]\)`)
	matches := re.FindStringSubmatch(expr)
	if len(matches) != 2 {
		return false, fmt.Errorf("无法解析 contains 表达式: %s", expr)
	}

	if e.response == nil {
		return false, nil
	}

	return strings.Contains(e.response.Body, matches[1]), nil
}

func (e *ExpressionEvaluator) evaluateCookieContains(expr string) (bool, error) {
	// 解析 cookie.contains('text')
	re := regexp.MustCompile(`cookie\.contains\(['"]([^'"]+)['"]\)`)
	matches := re.FindStringSubmatch(expr)
	if len(matches) != 2 {
		return false, fmt.Errorf("无法解析 cookie.contains 表达式: %s", expr)
	}

	return strings.Contains(e.cookie, matches[1]), nil
}

func (e *ExpressionEvaluator) evaluateHeaderGet(expr string) (string, error) {
	// 解析 response.headers.get('header-name')
	re := regexp.MustCompile(`response\.headers\.get\(['"]([^'"]+)['"]\)`)
	matches := re.FindStringSubmatch(expr)
	if len(matches) != 2 {
		return "", fmt.Errorf("无法解析 headers.get 表达式: %s", expr)
	}

	if e.response == nil {
		return "", nil
	}

	headerName := matches[1]
	headerNameLower := strings.ToLower(headerName)
	
	// 查找响应头（不区分大小写）
	for k, v := range e.response.Headers {
		if strings.ToLower(k) == headerNameLower {
			if len(v) > 0 {
				return v[0], nil
			}
		}
	}

	return "", nil
}

func (e *ExpressionEvaluator) evaluateNumericValue(expr string) (int, error) {
	val, err := e.evaluateValue(expr)
	if err != nil {
		return 0, err
	}

	switch v := val.(type) {
	case int:
		return v, nil
	case string:
		return strconv.Atoi(v)
	default:
		return 0, fmt.Errorf("无法转换为数字: %v", val)
	}
}

