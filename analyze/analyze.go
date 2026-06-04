package analyze

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode"

	"github.com/jaysen13/jaysenwxapkg-go/config"
)

// API 信息
type APIInfo struct {
	File string
	API  string
}

// 敏感信息
type SensitiveInfo struct {
	File string
	Type string
	Content string
}

// 分析结果
type AnalysisResult struct {
	APIs        []APIInfo
	Sensitive   []SensitiveInfo
}

// 判断是否为二进制文件（通过检查是否包含 null 字节）
func isBinaryFile(data []byte) bool {
	for i := 0; i < len(data) && i < 100; i++ {
		if data[i] == 0 {
			return true
		}
	}
	
	// 检查是否有足够的可打印字符
	printableCount := 0
	for i := 0; i < len(data) && i < 1000; i++ {
		if unicode.IsPrint(rune(data[i])) || unicode.IsSpace(rune(data[i])) {
			printableCount++
		}
	}
	
	return float64(printableCount)/float64(min(len(data), 1000)) < 0.7
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// 分析目录中的所有文件
func AnalyzeDirectory(dir string, compiledConfig *config.CompiledConfig) (*AnalysisResult, error) {
	result := &AnalysisResult{
		APIs:      []APIInfo{},
		Sensitive: []SensitiveInfo{},
	}

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		// 跳过 decrypted.wxapkg 文件
		if strings.HasSuffix(path, "decrypted.wxapkg") {
			return nil
		}

		// 读取文件内容
		content, err := os.ReadFile(path)
		if err != nil {
			return nil // 跳过读取失败的文件
		}

		// 检查是否为二进制文件，是的话跳过
		if isBinaryFile(content) {
			return nil
		}

		textContent := string(content)
		relPath, err := filepath.Rel(dir, path)
		if err != nil {
			relPath = path
		}

		// 提取 API（正则）
		apis := extractAPIs(textContent, relPath, compiledConfig)
		result.APIs = append(result.APIs, apis...)

		// 提取 API（AST 解析 — 仅对 JS 文件）
		if strings.HasSuffix(path, ".js") {
			astAPIs := ExtractAPIsFromJS(textContent)
			for _, u := range astAPIs {
				if !isValidAPI(u, compiledConfig) {
					continue
				}
				result.APIs = append(result.APIs, APIInfo{
					File: relPath,
					API:  u,
				})
			}
		}

		// 检测敏感信息
		sensitive := detectSensitive(textContent, relPath, compiledConfig)
		result.Sensitive = append(result.Sensitive, sensitive...)

		return nil
	})

	return result, err
}

// isValidAPI 检查 API/URL 是否有效，过滤无意义的静态资源路径和垃圾数据
func isValidAPI(api string, cfg *config.CompiledConfig) bool {
	// 长度过滤：超过 300 字符的多为 base64/编码数据
	if len(api) > 300 {
		return false
	}

	// 高比例非 ASCII / 控制字符的字符串（编码数据、二进制片段）
	nonASCII := 0
	for _, r := range api {
		if r > 127 || (r < 32 && r != '\n' && r != '\r') {
			nonASCII++
		}
	}
	if len(api) > 20 && float64(nonASCII)/float64(len(api)) > 0.3 {
		return false
	}

	// 很多连续的大写字母/数字大概率是 base64
	if len(api) > 80 {
		upperOrDigit := 0
		for _, r := range api {
			if (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '/' || r == '+' || r == '=' {
				upperOrDigit++
			}
		}
		if float64(upperOrDigit)/float64(len(api)) > 0.8 {
			return false
		}
	}

	// 前缀黑名单
	for _, prefix := range cfg.PrefixBlacklist {
		if strings.Contains(api, prefix) {
			return false
		}
	}

	// 后缀黑名单（去掉查询参数和哈希后再检查）
	suffix := getURLSuffix(api)
	if cfg.SuffixBlacklist[suffix] {
		return false
	}

	return true
}

// 提取 API
func extractAPIs(content, filePath string, cfg *config.CompiledConfig) []APIInfo {
	var results []APIInfo

	matches := cfg.APIPattern.FindAllStringSubmatch(content, -1)
	for _, match := range matches {
		var api string
		for i := 1; i < len(match); i++ {
			if match[i] != "" {
				api = match[i]
				break
			}
		}

		if api == "" || !isValidAPI(api, cfg) {
			continue
		}

		results = append(results, APIInfo{
			File: filePath,
			API:  api,
		})
	}

	return results
}

// 检测敏感信息
func detectSensitive(content, filePath string, cfg *config.CompiledConfig) []SensitiveInfo {
	var results []SensitiveInfo

	for typeName, pattern := range cfg.SensitivePatterns {
		matches := pattern.FindAllString(content, -1)
		for _, match := range matches {
			results = append(results, SensitiveInfo{
				File:    filePath,
				Type:    typeName,
				Content: match,
			})
		}
	}

	return results
}

// 获取 URL 后缀
func getURLSuffix(url string) string {
	// 去除查询字符串和哈希
	cleanURL := strings.Split(url, "?")[0]
	cleanURL = strings.Split(cleanURL, "#")[0]

	// 获取最后一个点之后的内容
	lastDot := strings.LastIndex(cleanURL, ".")
	if lastDot == -1 || lastDot == len(cleanURL)-1 {
		return ""
	}

	return strings.ToLower(cleanURL[lastDot+1:])
}

// 打印结果
func (r *AnalysisResult) Print() {
	// 去重 API
	seenAPIs := make(map[string]bool)
	var uniqueAPIs []APIInfo
	for _, api := range r.APIs {
		if !seenAPIs[api.API] {
			seenAPIs[api.API] = true
			uniqueAPIs = append(uniqueAPIs, api)
		}
	}

	// 打印 API 接口
	fmt.Println()
	fmt.Println("╔════════════════════════════════════════════════════════════╗")
	fmt.Println("║                      🚪 API 接口列表                        ║")
	fmt.Println("╠════════════════════════════════════════════════════════════╣")
	if len(uniqueAPIs) > 0 {
		for i, api := range uniqueAPIs {
			fmt.Printf("║  %3d. %-50s ║\n", i+1, truncate(api.API, 50))
			fmt.Printf("║       文件: %-45s ║\n", truncate(api.File, 45))
		}
	} else {
		fmt.Println("║  未发现 API 接口")
	}
	fmt.Println("╚════════════════════════════════════════════════════════════╝")

	// 打印敏感信息 - 按类型分组并去重
	fmt.Println()
	fmt.Println("╔════════════════════════════════════════════════════════════╗")
	fmt.Println("║                     🔍 敏感信息检测结果                      ║")
	fmt.Println("╠════════════════════════════════════════════════════════════╣")
	
	if len(r.Sensitive) > 0 {
		// 按类型分组并去重
		type SensitiveGroup struct {
			Contents map[string][]string // content -> files
		}
		groups := make(map[string]*SensitiveGroup)
		
		for _, info := range r.Sensitive {
			if _, ok := groups[info.Type]; !ok {
				groups[info.Type] = &SensitiveGroup{
					Contents: make(map[string][]string),
				}
			}
			
			// 去重内容，并记录出现的文件
			contentGroup := groups[info.Type]
			if files, ok := contentGroup.Contents[info.Content]; ok {
				// 检查文件是否已经记录，避免重复
				fileExists := false
				for _, f := range files {
					if f == info.File {
						fileExists = true
						break
					}
				}
				if !fileExists {
					contentGroup.Contents[info.Content] = append(files, info.File)
				}
			} else {
				contentGroup.Contents[info.Content] = []string{info.File}
			}
		}
		
		// 打印分组后的敏感信息
		for typeName, group := range groups {
			fmt.Printf("║  ┌──────────────────────────────────────────────────────┐ ║\n")
			fmt.Printf("║  │ %-52s │ ║\n", "【"+typeName+"】")
			fmt.Printf("║  ├──────────────────────────────────────────────────────┤ ║\n")
			
			count := 0
			for content, files := range group.Contents {
				count++
				fmt.Printf("║  │ %3d. %-48s │ ║\n", count, truncate(content, 48))
				
				// 只显示前3个文件，避免太长
				if len(files) > 0 {
					fileList := ""
					for i, f := range files {
						if i > 2 {
							fileList += "…"
							break
						}
						if i > 0 {
							fileList += ", "
						}
						fileList += f
					}
					fmt.Printf("║  │      文件: %-40s │ ║\n", truncate(fileList, 40))
				}
			}
			fmt.Printf("║  └──────────────────────────────────────────────────────┘ ║\n")
		}
	} else {
		fmt.Println("║  未发现敏感信息")
	}
	fmt.Println("╚════════════════════════════════════════════════════════════╝")
	fmt.Println()
}

// 截断字符串
func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-3] + "..."
}

// 获取所有 API 列表（去重）
func (r *AnalysisResult) GetUniqueAPIs() []string {
	seen := make(map[string]bool)
	var apis []string

	for _, api := range r.APIs {
		if !seen[api.API] {
			seen[api.API] = true
			apis = append(apis, api.API)
		}
	}

	return apis
}

// GetFormattedSensitive 获取格式化的敏感信息文本（去重，适合写入文件）
func (r *AnalysisResult) GetFormattedSensitive() string {
	if len(r.Sensitive) == 0 {
		return ""
	}

	// 按类型分组去重: type -> { content -> [files] }
	type group struct {
		contents map[string][]string
	}
	groups := make(map[string]*group)

	for _, info := range r.Sensitive {
		g, ok := groups[info.Type]
		if !ok {
			g = &group{contents: make(map[string][]string)}
			groups[info.Type] = g
		}
		files := g.contents[info.Content]
		dup := false
		for _, f := range files {
			if f == info.File {
				dup = true
				break
			}
		}
		if !dup {
			g.contents[info.Content] = append(files, info.File)
		}
	}

	var sb strings.Builder
	for typeName, g := range groups {
		sb.WriteString(fmt.Sprintf("【%s】\n", typeName))
		for content, files := range g.contents {
			sb.WriteString(fmt.Sprintf("  %s\n", content))
			for _, f := range files {
				sb.WriteString(fmt.Sprintf("    文件: %s\n", f))
			}
		}
		sb.WriteString("\n")
	}
	return sb.String()
}
