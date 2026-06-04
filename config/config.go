package config

import (
	"regexp"
)

// 配置结构
type Config struct {
	APIPattern        string            `json:"api_pattern"`
	SensitivePatterns map[string]string `json:"sensitive_patterns"`
	SuffixBlacklist   []string          `json:"suffix_blacklist"`
	PrefixBlacklist   []string          `json:"prefix_blacklist"`
}

// 默认配置
func DefaultConfig() *Config {
	return &Config{
		APIPattern: `(?:"|')(((?:[a-zA-Z]{1,10}://|//)[^"'/]{1,}\.([a-zA-Z]{2,})[^"']{0,})|((?:/|\.\./|\./)[^"'<,;| *()(%%$^/\\\[\]][^"'<,;|()]{1,})|([a-zA-Z0-9_\-/]{1,}/[a-zA-Z0-9_\-/]{1,}\.(?:[a-zA-Z]{1,4}|action)(?:[\?|/][^"|']{0,}|))|([a-zA-Z0-9_\-]{1,}\.(?:php|asp|aspx|jsp|json|action|html|js|txt|xml)(?:\?[^"|']{0,}|)))(?:"|')`,
		SensitivePatterns: map[string]string{
			"微信小程序 session_key 泄露": `(?i)\bsession_key\b`,
			"AppSecret 泄露":             `(?i)\b\w*secret\b`,
			"手机号":                    `1[3-9]\d{9}`,
			"身份证号":                   `\b\d{17}([0-9]|X|x)\b`,
			"邮箱地址":                   `[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,4}`,
			"IP地址":                    `^(?:25[0-5]|2[0-4][0-9]|1[0-9][0-9]|[1-9]?[0-9])\.(?:25[0-5]|2[0-4][0-9]|1[0-9][0-9]|[1-9]?[0-9])\.(?:25[0-5]|2[0-4][0-9]|1[0-9][0-9]|[1-9]?[0-9])\.(?:25[0-5]|2[0-4][0-9]|1[0-9][0-9]|[1-9]?[0-9])$`,
		},
		SuffixBlacklist: []string{"js", "jpg", "png", "jpeg", "gif", "svg", "wxml", "wxss", "wxs", "woff", "woff2", "ttf", "eot", "otf", "ttc", "json", "html", "txt", "xml"},
		PrefixBlacklist: []string{"pages/", "components/", "static/", "uni_modules/", "uview-ui/", "uview-plus/", "package/", "utils/", "images/", "font/", "fonts/", "@babel/", "demo/", "locale/", "common/", "mixins/", "radio-group/", "steps/", "tabs/", "cell-group/", "tab-panel/", "chunk_", "../", "./"},
	}
}

// 编译后的正则
type CompiledConfig struct {
	APIPattern        *regexp.Regexp
	SensitivePatterns map[string]*regexp.Regexp
	SuffixBlacklist   map[string]bool
	PrefixBlacklist   []string
}

// 编译配置
func (c *Config) Compile() (*CompiledConfig, error) {
	apiRe, err := regexp.Compile(c.APIPattern)
	if err != nil {
		return nil, err
	}

	sensitiveRes := make(map[string]*regexp.Regexp)
	for name, pattern := range c.SensitivePatterns {
		re, err := regexp.Compile(pattern)
		if err != nil {
			return nil, err
		}
		sensitiveRes[name] = re
	}

	suffixSet := make(map[string]bool)
	for _, suffix := range c.SuffixBlacklist {
		suffixSet[suffix] = true
	}

	return &CompiledConfig{
		APIPattern:        apiRe,
		SensitivePatterns: sensitiveRes,
		SuffixBlacklist:   suffixSet,
		PrefixBlacklist:   c.PrefixBlacklist,
	}, nil
}
