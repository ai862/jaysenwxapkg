# JaySenWxapkg Go 版本

微信小程序 wxapkg 解密、解包和分析工具的 Go CLI 版本。

## 功能特性

- 🔓 wxapkg 解密 - 自动识别加密文件，使用 AES-CBC + XOR 解密
- 📦 wxapkg 解包 - 提取小程序源码
- 🚪 API 提取 - 自动提取 API 接口，支持自定义正则
- 🔍 敏感信息检测 - 检测手机号、身份证号、密钥等敏感数据
- 📦 批量处理 - 支持批量扫描目录中的所有 wxapkg 文件

## 编译

```bash
go build -o jaysenwxapkg .
```

或者直接运行：

```bash
go run .
```

## 使用方法

### 1. 处理单个 wxapkg 文件（推荐）

自动解密（如果需要）、解包并分析：

```bash
./jaysenwxapkg process /path/to/app.wxapkg -o ./output
```

如果文件是加密的，可以指定 AppID：

```bash
./jaysenwxapkg process /path/to/app.wxapkg -o ./output -w wx1234567890abcdef
```

### 2. 批量处理

扫描目录中的所有 wxapkg 文件并批量处理：

```bash
./jaysenwxapkg batch /path/to/wxapkg/dir -o ./output
```

### 3. 仅解密

```bash
./jaysenwxapkg decrypt /path/to/encrypted.wxapkg /path/to/decrypted.wxapkg -w wx1234567890abcdef
```

### 4. 仅解包

```bash
./jaysenwxapkg unpack /path/to/app.wxapkg -o ./output
```

## 参数说明

| 参数 | 说明 |
|------|------|
| `-o, --output` | 输出目录，默认 `./output` |
| `-w, --wxid` | 小程序 AppID（用于解密加密文件） |
| `--api-pattern` | 自定义 API 提取正则 |
| `--no-analysis` | 跳过 API 和敏感信息分析 |

## 项目结构

```
jaysenwxapkg-go/
├── main.go              # 主程序入口
├── config/              # 配置模块
│   └── config.go
├── decrypt/             # 解密模块
│   └── decrypt.go
├── unpack/              # 解包模块
│   └── unpack.go
└── analyze/             # 分析模块
    └── analyze.go
```

## 与原版 Java 版本对比

| 特性 | Java 版本 | Go 版本 |
|------|-----------|---------|
| UI | Burp Suite 插件 | CLI 命令行 |
| 部署 | 需要 Burp | 单个二进制文件 |
| 性能 | 良好 | 优秀 |
| 跨平台 | Java 支持 | 原生支持 |

## 注意事项

⚠️ 此工具仅供授权的安全测试使用，请勿用于非法用途。

## License

MIT License
