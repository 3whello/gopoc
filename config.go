package sdk

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

// POCConfig POC 配置结构
type POCConfig struct {
	Name      string            `yaml:"name"`
	Author    string            `yaml:"author"`
	Category  string            `yaml:"category"`
	CVEID     string            `yaml:"cve_id"`
	Level     string            `yaml:"level"`
	Source    string            `yaml:"source"`
	S1        string            `yaml:"s1"`
	Rules     map[string]*Rule  `yaml:"rules"`
	Expression string           `yaml:"expression"`
}

// Rule 单个规则定义
type Rule struct {
	Method          string            `yaml:"method"`
	Path            string            `yaml:"path"`
	Timeout         int               `yaml:"timeout"`
	RetryCount      int               `yaml:"retry_count"`
	Headers         map[string]string `yaml:"headers"`
	Body            []string          `yaml:"body"`
	ExtractCookie   string            `yaml:"extract_cookie"`
	UseCookie       string            `yaml:"use_cookie"`
	CookieExpression string           `yaml:"cookie_expression"`
	Expression      string            `yaml:"expression"`
}

// LoadConfig 从文件加载 POC 配置
func LoadConfig(filePath string) (*POCConfig, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("读取配置文件失败: %w", err)
	}

	config := &POCConfig{}
	if err := yaml.Unmarshal(data, config); err != nil {
		return nil, fmt.Errorf("解析 YAML 配置失败: %w", err)
	}

	return config, nil
}

// GetTimeout 获取超时时间（秒转 Duration）
// 最小超时时间为 60 秒，避免 TLS 握手超时（HTTPS 需要更长时间）
func (r *Rule) GetTimeout() time.Duration {
	if r.Timeout <= 0 {
		return 60 * time.Second
	}
	timeout := time.Duration(r.Timeout) * time.Second
	if timeout < 60*time.Second {
		return 60 * time.Second
	}
	return timeout
}

// GetRetryCount 获取重试次数
func (r *Rule) GetRetryCount() int {
	if r.RetryCount <= 0 {
		return 1
	}
	return r.RetryCount
}

// GetBody 获取请求体字符串
func (r *Rule) GetBody() string {
	if len(r.Body) == 0 {
		return ""
	}
	return r.Body[0]
}

