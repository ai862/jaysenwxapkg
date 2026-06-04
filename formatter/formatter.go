package formatter

import (
	"encoding/json"
	"strings"
)

// FormatJSON 格式化 JSON 文件
func FormatJSON(data []byte) ([]byte, error) {
	var jsonObj interface{}
	if err := json.Unmarshal(data, &jsonObj); err != nil {
		return data, nil
	}
	return json.MarshalIndent(jsonObj, "", "  ")
}

// FormatJS 格式化 JavaScript 文件
// 对于压缩的JS，保持原样，但可以做基础清理
func FormatJS(data []byte) ([]byte, error) {
	// 对于压缩的JS代码，保持原样，不做过度格式化
	// 因为格式化压缩的混淆代码往往适得其反
	return data, nil
}

// IsJSFile 检查是否是 JS 文件
func IsJSFile(path string) bool {
	return strings.HasSuffix(strings.ToLower(path), ".js")
}

// IsJSONFile 检查是否是 JSON 文件
func IsJSONFile(path string) bool {
	return strings.HasSuffix(strings.ToLower(path), ".json")
}
