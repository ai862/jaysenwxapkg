package appname

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// 提取小程序名称
func ExtractAppName(dir string) string {
	var foundName string
	bestScore := 0

	// 预编译正则表达式
	zPattern := regexp.MustCompile(`Z\(\[3,\s*'([^']+)'\]\)`)

	// 从所有JS文件中搜索小程序名称
	filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}

		// 只检查JS文件
		if info.IsDir() || !strings.HasSuffix(path, ".js") {
			return nil
		}

		// 跳过不需要的目录
		if strings.Contains(path, "node_modules") ||
			strings.Contains(path, "miniprogram_npm") ||
			strings.Contains(path, "decrypted.wxapkg") {
			return nil
		}

		content, err := os.ReadFile(path)
		if err != nil {
			return nil
		}

		text := string(content)

		// 搜索 Z([3,'xxx']) 模式 - 压缩代码中最常见的字符串格式
		matches := zPattern.FindAllStringSubmatch(text, -1)
		for _, match := range matches {
			if len(match) > 1 {
				name := strings.TrimSpace(match[1])
				score := getAppNameScore(name)

				// 如果得分更高，就更新
				if score > bestScore {
					bestScore = score
					foundName = name
				}
			}
		}

		return nil
	})

	// 只有得分足够高才返回
	if bestScore < 10 {
		return ""
	}

	return foundName
}

// 计算名称的得分
func getAppNameScore(name string) int {
	score := 0

	// 长度检查：6-50个字符
	if len(name) < 6 || len(name) > 50 {
		return 0
	}

	// 过滤掉包含转义字符的字符串
	if strings.Contains(name, "\\x") || strings.Contains(name, "\\n") {
		return 0
	}

	// 过滤掉包含 URL 或路径的字符串
	if strings.HasPrefix(name, "http://") || strings.HasPrefix(name, "https://") {
		return 0
	}
	if strings.Contains(name, ".com") || strings.Contains(name, ".cn") || strings.Contains(name, ".org") {
		return 0
	}

	// 过滤掉明显的代码关键字
	badKeywords := []string{
		"undefined", "null", "true", "false", "none",
		"function", "return", "var ", "let ", "const ",
		"class", "object", "array", "string", "number",
		"error", "Error", "success", "failed",
		"loading", "Loading", "click", "btn", "hover",
		"page", "logo", "icon", "img", "image",
		"desc", "Privacy", "Policy",
	}

	lowerName := strings.ToLower(name)
	for _, keyword := range badKeywords {
		if strings.Contains(lowerName, strings.ToLower(keyword)) {
			return 0
		}
	}

	// 必须包含中文或英文
	hasChinese := false
	hasEnglish := false

	for _, r := range name {
		if r >= 0x4e00 && r <= 0x9fa5 {
			hasChinese = true
		}
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') {
			hasEnglish = true
		}
	}

	if !hasChinese && !hasEnglish {
		return 0
	}

	// 计算得分 - 基于关键词匹配
	highValueKeywords := map[string]int{
		// 小程序主名称（高优先级）
		"小程序": 100, "微信": 100,
		
		// 出行旅游
		"同程": 50, "旅行": 30, "旅游": 30, "订票": 30,
		"酒店": 20, "机票": 20, "火车": 20, "汽车票": 20, "门票": 15,
		
		// 购物商城
		"外卖": 20, "购物": 20, "商城": 20, "优选": 15, "精选": 15, "超市": 15,
		
		// 企业福利
		"福鲤": 50, "瑞祥": 50, "食堂": 20, "充值": 15, "福利": 20, "卡包": 20, "钱包": 15,
		
		// 娱乐
		"电影": 20, "视频": 15, "音乐": 15, "阅读": 15, "游戏": 15, "小说": 15, "直播": 15,
		
		// 教育
		"教育": 20, "培训": 15, "课程": 15, "学习": 15, "学校": 15,
		
		// 健康
		"健康": 20, "医疗": 20, "挂号": 15, "买药": 15, "医院": 15,
		
		// 政务
		"政务": 20, "办事": 15, "服务": 15, "社保": 15, "公积金": 15,
		
		// 常用
		"同城": 20, "生活": 15, "社区": 15, "工作": 15, "招聘": 15,
		
		// 金融
		"银行": 20, "理财": 15, "保险": 15, "贷款": 15, "支付": 15,
		
		// 地区
		"国内": 5, "港澳台": 5, "国际": 5,
	}

	for keyword, weight := range highValueKeywords {
		if strings.Contains(name, keyword) {
			score += weight
		}
	}

	// 长度得分
	if len(name) >= 10 && len(name) <= 30 {
		score += 10
	}

	return score
}

// 打印小程序名称信息
func PrintAppName(appid, name string) {
	fmt.Println()
	fmt.Println("╔════════════════════════════════════════════════════════════╗")
	fmt.Println("║                    📱 小程序信息                           ║")
	fmt.Println("╠════════════════════════════════════════════════════════════╣")
	if name != "" {
		fmt.Printf("║  小程序名称 : %-40s ║\n", truncate(name, 40))
	} else {
		fmt.Printf("║  小程序名称 : %-40s ║\n", "（未在源码中找到）")
	}
	fmt.Printf("║  AppID      : %-40s ║\n", truncate(appid, 40))
	if name == "" {
		fmt.Println("║  💡 提示：部分小程序名称未硬编码在源码中")
		fmt.Println("║          可通过微信搜索 AppID 或查看小程序资料获取")
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
