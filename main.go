package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/jaysen13/jaysenwxapkg-go/analyze"
	"github.com/jaysen13/jaysenwxapkg-go/appname"
	"github.com/jaysen13/jaysenwxapkg-go/config"
	"github.com/jaysen13/jaysenwxapkg-go/decrypt"
	"github.com/jaysen13/jaysenwxapkg-go/unpack"
	"github.com/spf13/cobra"
)

var (
	outputDir    string
	wxid         string
	apiPattern   string
	noAnalysis   bool
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "jaysenwxapkg",
		Short: "微信小程序 wxapkg 解密、解包和分析工具",
		Long: `JaySenWxapkg - 微信小程序渗透测试工具

功能：
  🔓 wxapkg文件解密    - 自动识别加密文件，使用AES-CBC+XOR算法解密
  📦 批量解包         - 递归扫描目录，解包主包和分包
  🚪 API提取          - 从源代码中提取API接口地址
  🔍 敏感信息检测     - 识别手机号、身份证号、密钥、session_key等
  📱 小程序名称提取    - 从源码中自动识别小程序名称

获取帮助：
  jaysenwxapkg process --help    # 查看单个文件处理帮助
  jaysenwxapkg batch --help     # 查看批量处理帮助
`,
		Example: `
  # 处理单个 wxapkg 文件
  jaysenwxapkg process ./app.wxapkg -o ./output -w wx1234567890abcdef

  # 批量处理整个目录
  jaysenwxapkg batch ./wxpackages/ -o ./output

  # 仅解密文件
  jaysenwxapkg decrypt encrypted.wxapkg decrypted.wxapkg -w wx1234567890abcdef

  # 仅解包（不需要AppID）
  jaysenwxapkg unpack app.wxapkg -o ./extracted`,
	}

	// 单个文件处理命令
	processCmd := &cobra.Command{
		Use:   "process <wxapkg-file>",
		Short: "处理单个 wxapkg 文件（解密+解包+分析）",
		Long: `处理单个 wxapkg 文件的完整流程：
1. 自动识别文件是否加密
2. 解密加密的 wxapkg 文件（需要提供 AppID）
3. 解包文件并提取内容
4. 分析源代码，提取 API 接口
5. 检测敏感信息泄露
6. 提取小程序名称

📂 文件路径说明：
  <wxapkg-file> 支持以下路径格式：
    - 相对路径：./app.wxapkg、app.wxapkg、../dir/app.wxapkg
    - 绝对路径：/home/user/wxapkg/app.wxapkg
    - 包含 AppID 的路径：./wx2b4c74af074f2085/app.wxapkg
      （工具会自动从路径中提取 AppID，无需额外指定 -w 参数）

🔑 AppID 说明：
  - 微信小程序的 AppID 通常以 "wx" 开头，后跟 16 位字母数字
  - 例如：wx2b4c74af074f2085、wx336dcaf6a1ecf632
  - 如果路径中包含 AppID，工具会自动提取
  - 如果路径中没有包含 AppID，请使用 -w 参数指定

📁 输出目录：
  - 使用 -o 参数指定输出目录，默认为 ./output
  - 每个 wxapkg 文件会输出到单独的子目录中

示例：
  # 处理当前目录下的 wxapkg 文件（路径包含 AppID，自动识别）
  jaysenwxapkg process ./wx2b4c74af074f2085/app.wxapkg

  # 处理文件，手动指定 AppID
  jaysenwxapkg process ./app.wxapkg -w wx2b4c74af074f2085

  # 指定输出目录
  jaysenwxapkg process ./app.wxapkg -o /tmp/wxoutput

  # 处理后跳过分析（仅解密+解包）
  jaysenwxapkg process ./app.wxapkg --no-analysis`,
		Args: cobra.ExactArgs(1),
		Run:  runProcess,
	}
	processCmd.Flags().StringVarP(&outputDir, "output", "o", "./output", "输出目录路径")
	processCmd.Flags().StringVarP(&wxid, "wxid", "w", "", "小程序 AppID（用于解密加密文件，可自动从路径提取）")
	processCmd.Flags().StringVar(&apiPattern, "api-pattern", "", "自定义 API 提取正则表达式")
	processCmd.Flags().BoolVar(&noAnalysis, "no-analysis", false, "跳过 API 和敏感信息分析")

	// 批量处理命令
	batchCmd := &cobra.Command{
		Use:   "batch <directory>",
		Short: "批量处理目录中的所有 wxapkg 文件",
		Long: `批量处理目录中的所有 wxapkg 文件：
1. 递归扫描指定目录及其所有子目录
2. 查找所有 .wxapkg 文件
3. 依次处理每个文件（解密+解包+分析）
4. 每个文件输出到单独的子目录

📂 目录路径说明：
  <directory> 支持以下路径格式：
    - 当前目录：. 或不指定
    - 相对路径：./wxpackages、../outputdir
    - 绝对路径：/home/user/wxapkg_files
    - 用户目录：~/wxapkg（会展开为 /home/user/wxapkg）

📦 目录结构示例：
  批量处理会递归查找所有 .wxapkg 文件，例如：
  .
  └── wxpackages/
      ├── wx2b4c74af074f2085/
      │   └── app.wxapkg        # 会被处理
      └── wx336dcaf6a1ecf632/
          ├── app.wxapkg        # 会被处理
          └── pkg-1.wxapkg      # 分包，也会被处理

🔑 AppID 说明：
  - 批量处理时，工具会从每个文件的路径中自动提取 AppID
  - 如果路径中没有包含 AppID，请使用 -w 参数统一指定
  - 示例路径：./wx2b4c74af074f2085/app.wxapkg

📁 输出目录：
  - 使用 -o 参数指定输出目录，默认为 ./output
  - 每个 wxapkg 文件会输出到单独的子目录：
    ./output/wx2b4c74af074f2085/
    ./output/wx336dcaf6a1ecf632/

示例：
  # 批量处理当前目录（递归查找所有 wxapkg 文件）
  jaysenwxapkg batch .

  # 批量处理指定目录
  jaysenwxapkg batch ./wxpackages/ -o ./output

  # 批量处理，手动指定 AppID（用于解密所有文件）
  jaysenwxapkg batch ./wxpackages/ -w wx2b4c74af074f2085

  # 跳过分析（仅解密+解包，加快处理速度）
  jaysenwxapkg batch ./wxpackages/ --no-analysis`,
		Args: cobra.ExactArgs(1),
		Run:  runBatch,
	}
	batchCmd.Flags().StringVarP(&outputDir, "output", "o", "./output", "输出目录路径")
	batchCmd.Flags().StringVarP(&wxid, "wxid", "w", "", "小程序 AppID（用于解密所有加密文件）")
	batchCmd.Flags().StringVar(&apiPattern, "api-pattern", "", "自定义 API 提取正则表达式")
	batchCmd.Flags().BoolVar(&noAnalysis, "no-analysis", false, "跳过 API 和敏感信息分析")

	// 解密命令
	decryptCmd := &cobra.Command{
		Use:   "decrypt <encrypted-file> <output-file>",
		Short: "仅解密 wxapkg 文件",
		Long: `仅解密 wxapkg 文件（不进行解包和分析）：
- 自动使用 AES-CBC+XOR 算法解密
- 需要提供正确的 AppID
- 解密后的文件可以直接用 unpack 命令解包

📂 文件路径说明：
  <encrypted-file>  加密的 wxapkg 文件路径
    - 相对路径：./encrypted.wxapkg
    - 绝对路径：/home/user/encrypted.wxapkg
    - 如果路径包含 AppID，会自动提取，无需 -w 参数

  <output-file>     解密后的输出文件路径
    - 建议命名：decrypted.wxapkg、app_decrypted.wxapkg

🔑 AppID 说明（必须提供）：
  - 微信小程序的 AppID 通常以 "wx" 开头，后跟 16 位字母数字
  - 使用 -w 参数指定：jaysenwxapkg decrypt input.wxapkg output.wxapkg -w wx2b4c74af074f2085
  - 或从路径自动提取：./wx2b4c74af074f2085/encrypted.wxapkg

示例：
  # 解密文件（手动指定 AppID）
  jaysenwxapkg decrypt ./encrypted.wxapkg ./decrypted.wxapkg -w wx2b4c74af074f2085

  # 解密文件（路径包含 AppID，自动提取）
  jaysenwxapkg decrypt ./wx2b4c74af074f2085/encrypted.wxapkg ./decrypted.wxapkg`,
		Args: cobra.ExactArgs(2),
		Run:  runDecrypt,
	}
	decryptCmd.Flags().StringVarP(&wxid, "wxid", "w", "", "小程序 AppID（用于解密加密文件）")

	// 解包命令
	unpackCmd := &cobra.Command{
		Use:   "unpack <wxapkg-file>",
		Short: "仅解包 wxapkg 文件",
		Long: `仅解包 wxapkg 文件（不进行解密和分析）：
- 提取 wxapkg 文件内容
- 生成完整的目录结构
- 适用于已解密的 wxapkg 文件

📂 文件路径说明：
  <wxapkg-file> 支持以下路径格式：
    - 相对路径：./app.wxapkg、app.wxapkg
    - 绝对路径：/home/user/wxapkg/app.wxapkg
    - 支持加密或已解密的文件

📁 输出目录：
  - 使用 -o 参数指定输出目录，默认为 ./output
  - 会直接输出到指定的输出目录，不创建子目录

示例：
  # 解包当前目录下的 wxapkg 文件
  jaysenwxapkg unpack ./app.wxapkg -o ./extracted

  # 解包到指定目录
  jaysenwxapkg unpack ./folder/app.wxapkg -o /tmp/output

  # 解包已解密的文件
  jaysenwxapkg unpack ./decrypted.wxapkg -o ./content`,
		Args: cobra.ExactArgs(1),
		Run:  runUnpack,
	}
	unpackCmd.Flags().StringVarP(&outputDir, "output", "o", "./output", "输出目录路径")

	rootCmd.AddCommand(processCmd, batchCmd, decryptCmd, unpackCmd)

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

// 提取 AppID 从路径
func extractWxidFromPath(path string) string {
	re := regexp.MustCompile(`wx[a-f0-9]{16}`)
	match := re.FindString(path)
	return match
}

// 处理单个文件
func runProcess(cmd *cobra.Command, args []string) {
	inputPath := args[0]

	// 打印欢迎信息
	fmt.Println()
	fmt.Println("╔════════════════════════════════════════════════════════════╗")
	fmt.Println("║                🔓 JaySenWxapkg - 解密工具                    ║")
	fmt.Println("╚════════════════════════════════════════════════════════════╝")
	fmt.Println()

	// 如果没有指定 wxid，尝试从路径提取
	if wxid == "" {
		wxid = extractWxidFromPath(inputPath)
		if wxid != "" {
			fmt.Printf("📱 从路径提取到 AppID: %s\n", wxid)
		}
	}

	// 创建输出目录 — 优先使用 wxid（路径提取或 -w 指定），回退到文件名
	dirName := wxid
	if dirName == "" {
		dirName = strings.TrimSuffix(filepath.Base(inputPath), ".wxapkg")
	}
	fileOutputDir := filepath.Join(outputDir, dirName+"_out")
	if err := os.MkdirAll(fileOutputDir, 0755); err != nil {
		fmt.Printf("❌ 创建输出目录失败: %v\n", err)
		return
	}

	fmt.Printf("📦 处理文件: %s\n", inputPath)
	fmt.Printf("📂 输出目录: %s\n", fileOutputDir)

	// 检查是否是加密文件
	encrypted, err := decrypt.IsEncryptedWxapkg(inputPath)
	if err != nil {
		fmt.Printf("❌ 检查文件状态失败: %v\n", err)
		return
	}

	processPath := inputPath

	if encrypted {
		if wxid == "" {
			fmt.Println("⚠️ 文件是加密的，但没有提供 AppID")
			return
		}

		fmt.Println("🔐 检测到加密文件，开始解密...")
		// 解密到临时文件，解包后自动清理
		tmpFile, err := os.CreateTemp("", "decrypted-*.wxapkg")
		if err != nil {
			fmt.Printf("❌ 创建临时文件失败: %v\n", err)
			return
		}
		decryptedPath := tmpFile.Name()
		tmpFile.Close()
		defer os.Remove(decryptedPath)

		if err := decrypt.DecryptWxapkg(wxid, inputPath, decryptedPath); err != nil {
			fmt.Printf("❌ 解密失败: %v\n", err)
			return
		}
		fmt.Println("✅ 解密成功")
		processPath = decryptedPath
	}

	// 解包
	fmt.Println()
	fmt.Println("📦 开始解包...")
	files, err := unpack.UnpackWxapkg(processPath, fileOutputDir)
	if err != nil {
		fmt.Printf("❌ 解包失败: %v\n", err)
		return
	}
	fmt.Printf("✅ 解包完成，共 %d 个文件\n", len(files))

	// 提取小程序名称
	if wxid != "" {
		fmt.Println()
		fmt.Println("🔍 提取小程序名称...")
		appName := appname.ExtractAppName(fileOutputDir)
		appname.PrintAppName(wxid, appName)
	}

	// 分析
	if !noAnalysis {
		fmt.Println()
		fmt.Println("🔍 开始分析源代码...")
		cfg := config.DefaultConfig()
		if apiPattern != "" {
			cfg.APIPattern = apiPattern
		}
		compiledCfg, err := cfg.Compile()
		if err != nil {
			fmt.Printf("❌ 编译配置失败: %v\n", err)
			return
		}

		result, err := analyze.AnalyzeDirectory(fileOutputDir, compiledCfg)
		if err != nil {
			fmt.Printf("❌ 分析失败: %v\n", err)
			return
		}

		result.Print()

		// 保存 API 列表
		uniqueAPIs := result.GetUniqueAPIs()
		if len(uniqueAPIs) > 0 {
			apiFile := filepath.Join(fileOutputDir, "apis.txt")
			apiContent := strings.Join(uniqueAPIs, "\n")
			if err := os.WriteFile(apiFile, []byte(apiContent), 0644); err != nil {
				fmt.Printf("⚠️ 保存 API 列表失败: %v\n", err)
			} else {
				fmt.Printf("📝 API 列表已保存到: %s\n", apiFile)
			}
		}

		// 保存敏感信息
		sensitiveContent := result.GetFormattedSensitive()
		if sensitiveContent != "" {
			sensitiveFile := filepath.Join(fileOutputDir, "sensitive.txt")
			if err := os.WriteFile(sensitiveFile, []byte(sensitiveContent), 0644); err != nil {
				fmt.Printf("⚠️ 保存敏感信息失败: %v\n", err)
			} else {
				fmt.Printf("🔍 敏感信息已保存到: %s\n", sensitiveFile)
			}
		}
	}

	fmt.Println()
	fmt.Println("╔════════════════════════════════════════════════════════════╗")
	fmt.Println("║                       ✅ 处理完成！                          ║")
	fmt.Printf("║  输出目录: %-45s ║\n", truncateForOutput(fileOutputDir, 45))
	fmt.Println("╚════════════════════════════════════════════════════════════╝")
	fmt.Println()
}

// 截断字符串用于输出
func truncateForOutput(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return "..." + s[len(s)-max+3:]
}

// 批量处理
func runBatch(cmd *cobra.Command, args []string) {
	dir := args[0]

	fmt.Println()
	fmt.Println("╔════════════════════════════════════════════════════════════╗")
	fmt.Println("║              🔓 JaySenWxapkg - 批量处理                    ║")
	fmt.Println("╚════════════════════════════════════════════════════════════╝")
	fmt.Println()

	// 查找所有 wxapkg 文件
	var wxapkgFiles []string
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if !info.IsDir() && strings.HasSuffix(path, ".wxapkg") {
			wxapkgFiles = append(wxapkgFiles, path)
		}
		return nil
	})

	if err != nil {
		fmt.Printf("❌ 扫描目录失败: %v\n", err)
		return
	}

	if len(wxapkgFiles) == 0 {
		fmt.Println("⚠️ 未找到 wxapkg 文件")
		return
	}

	fmt.Printf("📦 找到 %d 个 wxapkg 文件\n", len(wxapkgFiles))

	// 处理每个文件
	for i, file := range wxapkgFiles {
		fmt.Printf("\n[%d/%d] 处理: %s\n", i+1, len(wxapkgFiles), file)
		runProcess(cmd, []string{file})
	}

	fmt.Println()
	fmt.Printf("✅ 批量处理完成！共处理 %d 个文件\n", len(wxapkgFiles))
	fmt.Printf("📂 输出目录: %s\n", outputDir)
}

// 仅解密
func runDecrypt(cmd *cobra.Command, args []string) {
	inputPath := args[0]
	outputPath := args[1]

	fmt.Println()
	fmt.Println("╔════════════════════════════════════════════════════════════╗")
	fmt.Println("║                    🔓 JaySenWxapkg - 解密                  ║")
	fmt.Println("╚════════════════════════════════════════════════════════════╝")
	fmt.Println()

	if wxid == "" {
		wxid = extractWxidFromPath(inputPath)
		if wxid == "" {
			fmt.Println("❌ 未提供 AppID")
			return
		}
	}

	fmt.Printf("📦 输入文件: %s\n", inputPath)
	fmt.Printf("📝 输出文件: %s\n", outputPath)
	fmt.Printf("🔑 AppID: %s\n", wxid)

	err := decrypt.DecryptWxapkg(wxid, inputPath, outputPath)
	if err != nil {
		fmt.Printf("❌ 解密失败: %v\n", err)
		return
	}

	fmt.Println()
	fmt.Println("╔════════════════════════════════════════════════════════════╗")
	fmt.Println("║                       ✅ 解密成功！                          ║")
	fmt.Printf("║  输出文件: %-45s ║\n", truncateForOutput(outputPath, 45))
	fmt.Println("╚════════════════════════════════════════════════════════════╝")
}

// 仅解包
func runUnpack(cmd *cobra.Command, args []string) {
	inputPath := args[0]

	fmt.Println()
	fmt.Println("╔════════════════════════════════════════════════════════════╗")
	fmt.Println("║                    🔓 JaySenWxapkg - 解包                  ║")
	fmt.Println("╚════════════════════════════════════════════════════════════╝")
	fmt.Println()

	fmt.Printf("📦 输入文件: %s\n", inputPath)
	fmt.Printf("📂 输出目录: %s\n", outputDir)

	// 创建输出目录
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		fmt.Printf("❌ 创建输出目录失败: %v\n", err)
		return
	}

	// 解包
	files, err := unpack.UnpackWxapkg(inputPath, outputDir)
	if err != nil {
		fmt.Printf("❌ 解包失败: %v\n", err)
		return
	}

	fmt.Println()
	fmt.Println("╔════════════════════════════════════════════════════════════╗")
	fmt.Println("║                       ✅ 解包成功！                          ║")
	fmt.Printf("║  输出目录: %-45s ║\n", truncateForOutput(outputDir, 45))
	fmt.Printf("║  文件数量: %-45d ║\n", len(files))
	fmt.Println("╚════════════════════════════════════════════════════════════╝")
}
