package unpack

import (
	"encoding/binary"
	"errors"
	"os"
	"path/filepath"
	"github.com/jaysen13/jaysenwxapkg-go/formatter"
)

// 文件元信息
type FileMeta struct {
	Name   string
	Offset uint32
	Size   uint32
}

// 检查是否是有效的 wxapkg 文件
func IsValidWxapkg(filePath string) (bool, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return false, err
	}
	return isValidWxapkgData(data), nil
}

func isValidWxapkgData(data []byte) bool {
	if len(data) < 14 {
		return false
	}
	return data[0] == 0xBE && data[13] == 0xED
}

// 读取 uint32（大端序
func readUint32(data []byte) uint32 {
	return binary.BigEndian.Uint32(data)
}

// 解包 wxapkg 文件
func UnpackWxapkg(inputPath, outputDir string) ([]FileMeta, error) {
	data, err := os.ReadFile(inputPath)
	if err != nil {
		return nil, err
	}

	if !isValidWxapkgData(data) {
		return nil, errors.New("无效的 wxapkg 文件")
	}

	// 读取文件数量
	fileCount := readUint32(data[14:18])
	if fileCount == 0 || fileCount > 100000 {
		return nil, errors.New("文件数量异常")
	}

	// 解析文件元信息
	var fileList []FileMeta
	idx := 18
	for i := uint32(0); i < fileCount; i++ {
		// 文件名长度
		nameLen := readUint32(data[idx : idx+4])
		idx += 4

		// 文件名
		nameBytes := data[idx : idx+int(nameLen)]
		name := string(nameBytes)
		idx += int(nameLen)

		// 偏移和大小
		offset := readUint32(data[idx : idx+4])
		idx += 4
		size := readUint32(data[idx : idx+4])
		idx += 4

		fileList = append(fileList, FileMeta{
			Name:   name,
			Offset: offset,
			Size:   size,
		})
	}

	// 创建输出目录
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return nil, err
	}

	// 提取文件
	for _, meta := range fileList {
		outputPath := filepath.Join(outputDir, meta.Name)

		// 创建子目录
		if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
			return nil, err
		}

		// 读取文件内容
		fileData := data[meta.Offset : meta.Offset+meta.Size]
		
		// 根据文件类型进行格式化
		var formattedData []byte = fileData
		var err error
		
		if formatter.IsJSONFile(meta.Name) {
			formattedData, err = formatter.FormatJSON(fileData)
			if err != nil {
				// 格式化失败时使用原始数据
				formattedData = fileData
			}
		} else if formatter.IsJSFile(meta.Name) {
			formattedData, err = formatter.FormatJS(fileData)
			if err != nil {
				formattedData = fileData
			}
		}

		// 写入文件
		if err := os.WriteFile(outputPath, formattedData, 0644); err != nil {
			return nil, err
		}
	}

	return fileList, nil
}
