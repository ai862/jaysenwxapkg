# JaySenWxapkg Makefile
# 方便智能体和开发者快速使用

.PHONY: help build test clean install analyze batch report

# 工具路径
TOOL := ./jaysenwxapkg
PYTHON_HELPER := .claude/skills/wx_helper.py

# 默认输出目录
OUTPUT_DIR := ./output
BATCH_DIR := ./batch-output

## help - 显示帮助信息
help:
	@echo "JaySenWxapkg - 微信小程序分析工具"
	@echo ""
	@echo "使用方法:"
	@echo "  make analyze FILE=<wxapkg-file> APPID=<appid>    分析单个文件"
	@echo "  make batch DIR=<directory> APPID=<appid>        批量分析目录"
	@echo "  make report DIR=<output-dir>                     生成分析报告"
	@echo "  make test                                       测试工具"
	@echo "  make build                                      编译 Go 程序"
	@echo "  make clean                                      清理输出"
	@echo ""
	@echo "示例:"
	@echo "  make analyze FILE=test.wxapkg APPID=wx1234567890abcdef"
	@echo "  make batch DIR=./packages APPID=wx1234567890abcdef"
	@echo "  make report DIR=./output"

## build - 编译 Go 程序
build:
	@echo "编译 JaySenWxapkg..."
	@go build -o jaysenwxapkg .
	@echo "✓ 编译完成"

## test - 测试工具
test:
	@echo "检查工具状态..."
	@test -f $(TOOL) || (echo "✗ 工具不存在，请运行: make build" && exit 1)
	@$(TOOL) --help | head -5
	@echo "✓ 工具测试通过"

## install - 安装到系统
install:
	@echo "安装 JaySenWxapkg 到 /usr/local/bin/..."
	@sudo cp $(TOOL) /usr/local/bin/
	@sudo chmod +x /usr/local/bin/jaysenwxapkg
	@echo "✓ 安装完成"

## analyze - 分析单个文件
# 使用: make analyze FILE=app.wxapkg APPID=wx1234567890abcdef
analyze:
ifndef FILE
	$(error FILE 参数必需)
endif
	@test -f $(TOOL) || (echo "✗ 工具不存在，请运行: make build" && exit 1)
	@echo "分析文件: $(FILE)"
	@$(TOOL) process $(FILE) -o $(OUTPUT_DIR) -w $(APPID)
	@echo ""
	@echo "✓ 分析完成！"
	@echo "  输出目录: $(OUTPUT_DIR)"
	@echo "  API 列表: $(OUTPUT_DIR)/*/apis.txt"
	@echo ""
	@echo "查看 API:"
	@cat $(OUTPUT_DIR)/*/apis.txt 2>/dev/null | head -20 || echo "  无 API 数据"

## batch - 批量分析目录
# 使用: make batch DIR=./packages APPID=wx1234567890abcdef
batch:
ifndef DIR
	$(error DIR 参数必需)
endif
	@echo "批量分析目录: $(DIR)"
	@mkdir -p $(BATCH_DIR)
	@$(TOOL) batch $(DIR) -o $(BATCH_DIR) -w $(APPID)
	@echo ""
	@echo "✓ 批量分析完成！"
	@echo "  输出目录: $(BATCH_DIR)"
	@find $(BATCH_DIR) -name "apis.txt" -exec wc -l {} \; | awk '{sum+=$1} END {print "  总 API 数: " sum}'

## report - 生成报告
# 使用: make report DIR=./output
report:
ifndef DIR
	$(error DIR 参数必需)
endif
	@echo "生成分析报告..."
	@python3 $(PYTHON_HELPER) report $(DIR)

## extract-apis - 提取 API 列表
# 使用: make extract-apis DIR=./output
extract-apis:
ifndef DIR
	$(error DIR 参数必需)
endif
	@echo "提取 API 列表..."
	@python3 $(PYTHON_HELPER) extract-apis $(DIR)

## clean - 清理输出
clean:
	@echo "清理输出目录..."
	@rm -rf $(OUTPUT_DIR) $(BATCH_DIR)
	@rm -f apis.txt sensitive.txt report.md
	@echo "✓ 清理完成"

## demo - 运行演示
demo:
	@echo "运行演示..."
	@echo ""
	@echo "1. 检查工具..."
	@test -f $(TOOL) || (echo "  ✗ 工具不存在，运行: make build" && exit 1)
	@echo "  ✓ 工具存在"
	@echo ""
	@echo "2. 工具版本..."
	@$(TOOL) --version 2>/dev/null || $(TOOL) --help | head -1
	@echo ""
	@echo "3. 可用命令:"
	@$(TOOL) --help | grep -E "^  " | head -5
	@echo ""
	@echo "✓ 演示完成"

## quick-analyze - 快速分析（跳过详细输出）
# 使用: make quick-analyze FILE=app.wxapkg APPID=wx1234567890abcdef
quick-analyze:
ifndef FILE
	$(error FILE 参数必需)
endif
	@$(TOOL) process $(FILE) -o $(OUTPUT_DIR) -w $(APPID) --no-analysis 2>&1 | grep -E "(处理|解密|解包|分析|完成)" || true
	@echo "输出: $(OUTPUT_DIR)"

## security-audit - 安全审计流程
# 使用: make security-audit FILE=app.wxapkg APPID=wx1234567890abcdef
security-audit:
ifndef FILE
	$(error FILE 参数必需)
endif
	@TIMESTAMP=$(date +%Y%m%d-%H%M%S); \
	OUT_DIR="./audit-$$TIMESTAMP"; \
	echo "开始安全审计..."; \
	echo "输出目录: $$OUT_DIR"; \
	$(TOOL) process $(FILE) -o "$$OUT_DIR" -w $(APPID); \
	echo ""; \
	echo "=== API 接口 ==="; \
	cat "$$OUT_DIR"/*/apis.txt 2>/dev/null | sort -u | head -20; \
	echo ""; \
	echo "=== 敏感信息 ==="; \
	grep -r "1[3-9]\d\{9\}" "$$OUT_DIR"/*/ 2>/dev/null | wc -l | xargs echo "  手机号数量:"; \
	echo ""; \
	echo "✓ 审计完成: $$OUT_DIR"

# 开发命令
dev: build test demo

.DEFAULT_GOAL := help
