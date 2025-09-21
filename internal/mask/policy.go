// SPDX-FileCopyrightText: 2025 ChoreoAtlas contributors
// SPDX-License-Identifier: Apache-2.0
package mask

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

// Policy 定义脱敏策略的根结构
type Policy struct {
	Version int                        `yaml:"version"`
	Globals Globals                    `yaml:"globals"`
	Rules   []Rule                     `yaml:"rules"`
}

// Globals 定义全局策略配置
type Globals struct {
	Strategies map[string]Strategy `yaml:"strategies"`
}

// Rule 定义单个脱敏规则
type Rule struct {
	Match    Selector `yaml:"match"`
	Strategy any      `yaml:"strategy"` // 可以是字符串（引用全局策略）或内联 Strategy
}

// Selector 定义匹配条件
type Selector struct {
	Service   string   `yaml:"service,omitempty"`
	Operation string   `yaml:"operation,omitempty"`
	Paths     []string `yaml:"paths,omitempty"`
	RegexKeys []string `yaml:"regexKeys,omitempty"`
	Tag       string   `yaml:"tag,omitempty"`
}

// Strategy 定义脱敏策略
type Strategy struct {
	Type        string `yaml:"type"`        // redact, hash, null, keep-prefix, tokenize
	PrefixLen   int    `yaml:"prefixLen,omitempty"`
	Mask        string `yaml:"mask,omitempty"`
	Algo        string `yaml:"algo,omitempty"`
	Salt        string `yaml:"salt,omitempty"`
	TokenPrefix string `yaml:"tokenPrefix,omitempty"`
}

// CompiledSelector 编译后的选择器，用于高效匹配
type CompiledSelector struct {
	Service      string
	Operation    string
	Paths        []string
	RegexMatchers []*regexp.Regexp
	Tag          string
}

// CompiledRule 编译后的规则
type CompiledRule struct {
	Selector CompiledSelector
	Strategy Strategy
}

// CompiledPolicy 编译后的策略，用于高效执行
type CompiledPolicy struct {
	Rules []CompiledRule
}

// LoadPolicy 从文件加载策略
func LoadPolicy(path string) (*Policy, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("读取策略文件失败: %w", err)
	}

	var policy Policy
	if err := yaml.Unmarshal(data, &policy); err != nil {
		return nil, fmt.Errorf("解析策略文件失败: %w", err)
	}

	// 验证策略版本
	if policy.Version != 1 {
		return nil, fmt.Errorf("不支持的策略版本: %d", policy.Version)
	}

	return &policy, nil
}

// Compile 编译策略以提高执行效率
func (p *Policy) Compile() (*CompiledPolicy, error) {
	compiled := &CompiledPolicy{
		Rules: make([]CompiledRule, 0, len(p.Rules)),
	}

	for i, rule := range p.Rules {
		// 编译选择器
		selector := CompiledSelector{
			Service:   rule.Match.Service,
			Operation: rule.Match.Operation,
			Paths:     rule.Match.Paths,
			Tag:       rule.Match.Tag,
		}

		// 编译正则表达式
		for _, regexStr := range rule.Match.RegexKeys {
			regex, err := regexp.Compile(regexStr)
			if err != nil {
				return nil, fmt.Errorf("规则 %d 正则表达式编译失败 '%s': %w", i, regexStr, err)
			}
			selector.RegexMatchers = append(selector.RegexMatchers, regex)
		}

		// 解析策略
		var strategy Strategy
		switch s := rule.Strategy.(type) {
		case string:
			// 引用全局策略
			globalStrategy, exists := p.Globals.Strategies[s]
			if !exists {
				return nil, fmt.Errorf("规则 %d 引用了不存在的全局策略: %s", i, s)
			}
			strategy = globalStrategy
		case map[string]any:
			// 内联策略，需要手动解析
			if err := parseInlineStrategy(s, &strategy); err != nil {
				return nil, fmt.Errorf("规则 %d 内联策略解析失败: %w", i, err)
			}
		default:
			return nil, fmt.Errorf("规则 %d 策略格式不正确", i)
		}

		// 验证策略
		if err := validateStrategy(strategy); err != nil {
			return nil, fmt.Errorf("规则 %d 策略验证失败: %w", i, err)
		}

		compiled.Rules = append(compiled.Rules, CompiledRule{
			Selector: selector,
			Strategy: strategy,
		})
	}

	return compiled, nil
}

// parseInlineStrategy 解析内联策略
func parseInlineStrategy(data map[string]any, strategy *Strategy) error {
	if typeVal, ok := data["type"].(string); ok {
		strategy.Type = typeVal
	} else {
		return fmt.Errorf("缺少必需的 'type' 字段")
	}

	if val, ok := data["prefixLen"]; ok {
		switch v := val.(type) {
		case int:
			strategy.PrefixLen = v
		case float64:
			strategy.PrefixLen = int(v)
		case string:
			if parsed, err := strconv.Atoi(v); err == nil {
				strategy.PrefixLen = parsed
			}
		}
	}

	if val, ok := data["mask"].(string); ok {
		strategy.Mask = val
	}
	if val, ok := data["algo"].(string); ok {
		strategy.Algo = val
	}
	if val, ok := data["salt"].(string); ok {
		strategy.Salt = val
	}
	if val, ok := data["tokenPrefix"].(string); ok {
		strategy.TokenPrefix = val
	}

	return nil
}

// validateStrategy 验证策略配置
func validateStrategy(s Strategy) error {
	switch s.Type {
	case "redact", "null":
		// 无额外参数要求
	case "keep-prefix":
		if s.PrefixLen <= 0 {
			return fmt.Errorf("keep-prefix 策略需要 prefixLen > 0")
		}
		if s.Mask == "" {
			return fmt.Errorf("keep-prefix 策略需要指定 mask")
		}
	case "hash":
		if s.Algo == "" {
			s.Algo = "sha256" // 默认值
		}
		if s.Algo != "sha256" {
			return fmt.Errorf("目前只支持 sha256 算法")
		}
	case "tokenize":
		if s.TokenPrefix == "" {
			return fmt.Errorf("tokenize 策略需要指定 tokenPrefix")
		}
	default:
		return fmt.Errorf("不支持的策略类型: %s", s.Type)
	}
	return nil
}

// ApplyStrategy 应用单个策略到值
func ApplyStrategy(value any, strategy Strategy) any {
	str, ok := value.(string)
	if !ok {
		// 非字符串值，根据策略类型处理
		switch strategy.Type {
		case "null":
			return nil
		case "redact":
			return "***REDACTED***"
		default:
			return value // 保持原值
		}
	}

	switch strategy.Type {
	case "redact":
		return "***REDACTED***"
	case "null":
		return nil
	case "keep-prefix":
		if len(str) <= strategy.PrefixLen {
			return str // 字符串太短，不脱敏
		}
		prefix := str[:strategy.PrefixLen]
		masked := strings.Repeat(strategy.Mask, len(str)-strategy.PrefixLen)
		return prefix + masked
	case "hash":
		hasher := sha256.New()
		hasher.Write([]byte(str + strategy.Salt))
		return hex.EncodeToString(hasher.Sum(nil))
	case "tokenize":
		hasher := sha256.New()
		hasher.Write([]byte(str + strategy.Salt))
		hash := hex.EncodeToString(hasher.Sum(nil))[:8] // 取前8位
		return strategy.TokenPrefix + hash
	default:
		return value
	}
}

// GetDefaultPolicy 返回内置的默认 PII 策略
func GetDefaultPolicy() *Policy {
	return &Policy{
		Version: 1,
		Globals: Globals{
			Strategies: map[string]Strategy{
				"email": {
					Type:      "keep-prefix",
					PrefixLen: 2,
					Mask:      "*",
				},
				"phone": {
					Type: "redact",
				},
				"id": {
					Type: "hash",
					Algo: "sha256",
					Salt: "default-salt",
				},
				"secret": {
					Type: "redact",
				},
			},
		},
		Rules: []Rule{
			{
				Match: Selector{
					RegexKeys: []string{"(?i)email"},
				},
				Strategy: "email",
			},
			{
				Match: Selector{
					RegexKeys: []string{"(?i)phone", "(?i)mobile", "(?i)tel"},
				},
				Strategy: "phone",
			},
			{
				Match: Selector{
					RegexKeys: []string{"(?i)id$", "(?i).*id$"},
				},
				Strategy: "id",
			},
			{
				Match: Selector{
					RegexKeys: []string{"(?i)password", "(?i)secret", "(?i)token", "(?i)key"},
				},
				Strategy: "secret",
			},
		},
	}
}