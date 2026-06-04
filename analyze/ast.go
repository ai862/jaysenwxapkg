package analyze

import (
	"regexp"
	"strings"
)

// ExtractAPIsFromJS 从 JS 源码中提取字符串字面量、模板字符串里的 URL
// 使用自定义扫描器，不依赖外部 AST 库
func ExtractAPIsFromJS(content string) []string {
	seen := make(map[string]bool)
	var results []string

	addURL := func(s string) {
		s = strings.TrimSpace(s)
		if s == "" || seen[s] {
			return
		}
		seen[s] = true
		results = append(results, s)
	}

	// 提取所有 JS 字符串（单引号、双引号、模板字符串）
	strings := extractJSStrings(content)
	for _, s := range strings {
		for _, u := range extractURLs(s) {
			addURL(u)
		}
	}

	return results
}

// extractJSStrings 从 JS 源码中提取所有字符串字面量内容
func extractJSStrings(src string) []string {
	var results []string
	var i int
	n := len(src)

	for i < n {
		// 跳过注释
		if i+1 < n && src[i] == '/' {
			if src[i+1] == '/' {
				// 单行注释，跳过整行
				i += 2
				for i < n && src[i] != '\n' {
					i++
				}
				continue
			}
			if src[i+1] == '*' {
				// 多行注释，跳到 */
				i += 2
				for i+1 < n && !(src[i] == '*' && src[i+1] == '/') {
					i++
				}
				i += 2
				continue
			}
		}

		ch := src[i]

		if ch == '\'' {
			s := extractQuoted(src, i+1, '\'')
			if s != nil {
				results = append(results, *s)
				i = endOfLiteral(src, i+len(*s)+2)
				continue
			}
		}

		if ch == '"' {
			s := extractQuoted(src, i+1, '"')
			if s != nil {
				results = append(results, *s)
				i = endOfLiteral(src, i+len(*s)+2)
				continue
			}
		}

		if ch == '`' {
			s := extractTemplate(src, i+1)
			if s != nil {
				results = append(results, *s)
				i = endOfLiteral(src, i+len(*s)+2)
				continue
			}
		}

		i++
	}

	return results
}

// extractQuoted 提取引号内的字符串内容，处理转义
func extractQuoted(src string, start int, quote byte) *string {
	var buf strings.Builder
	for i := start; i < len(src); i++ {
		ch := src[i]
		if ch == '\\' && i+1 < len(src) {
			i++
			// 处理常见转义序列
			switch src[i] {
			case 'n':
				buf.WriteByte('\n')
			case 't':
				buf.WriteByte('\t')
			case 'r':
				buf.WriteByte('\r')
			case '\\':
				buf.WriteByte('\\')
			case '\'':
				buf.WriteByte('\'')
			case '"':
				buf.WriteByte('"')
			case 'u':
				// \uXXXX — 跳过，不解析
				if i+5 < len(src) {
					buf.WriteString(src[i-1 : i+5])
					i += 4
				}
			default:
				buf.WriteByte(src[i])
			}
			continue
		}
		if ch == quote {
			s := buf.String()
			return &s
		}
		if ch == '\n' || ch == '\r' {
			return nil // 字符串内不应有未转义换行
		}
		buf.WriteByte(ch)
	}
	return nil
}

// extractTemplate 提取模板字符串中的静态部分，跳过 ${...} 插值
func extractTemplate(src string, start int) *string {
	var buf strings.Builder
	for i := start; i < len(src); i++ {
		ch := src[i]
		if ch == '`' {
			s := buf.String()
			return &s
		}
		if ch == '$' && i+1 < len(src) && src[i+1] == '{' {
			// 跳过插值表达式
			buf.WriteString("${}")
			i += 2
			depth := 1
			for i < len(src) && depth > 0 {
				if src[i] == '{' {
					depth++
				} else if src[i] == '}' {
					depth--
				}
				i++
			}
			i-- // 回退一步，循环会 i++
			continue
		}
		if ch == '\\' && i+1 < len(src) {
			i++
			buf.WriteByte(src[i])
			continue
		}
		buf.WriteByte(ch)
	}
	return nil
}

// endOfLiteral 计算字符串字面量结束后的位置
func endOfLiteral(src string, offset int) int {
	if offset >= len(src) {
		return len(src)
	}
	return offset
}

// urlPattern 匹配 http/https/ftp/ws URL 和以 / 开头的路径
var urlPattern = regexp.MustCompile(`(?:https?|ftp|wss?)://[^\s"'<>]+|//[^\s"'<>]+|/[a-zA-Z0-9_\-./?&=%+#]+`)

// extractURLs 从字符串中提取所有 URL
func extractURLs(s string) []string {
	if len(s) < 3 {
		return nil
	}
	matches := urlPattern.FindAllString(s, -1)
	var results []string
	for _, m := range matches {
		m = strings.TrimRight(m, ".?&=/")
		if m != "" {
			results = append(results, m)
		}
	}
	return results
}
