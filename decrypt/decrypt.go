package decrypt

import (
	"crypto/aes"
	"crypto/cipher"
	"errors"
	"golang.org/x/crypto/pbkdf2"
	"crypto/sha1"
	"os"
)

const (
	wxapkgFlag = "V1MMWX"
	defaultIV  = "the iv: 16 bytes"
	defaultSalt = "saltiest"
	pbkdf2Iterations = 1000
	aesKeySize = 32
)

// 检查文件是否是加密的 wxapkg
func IsEncryptedWxapkg(filePath string) (bool, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return false, err
	}

	if len(data) < len(wxapkgFlag) {
		return false, nil
	}

	return string(data[:len(wxapkgFlag)]) == wxapkgFlag, nil
}

// 解密 wxapkg 文件
func DecryptWxapkg(wxid, inputPath, outputPath string) error {
	return DecryptWxapkgWithIVSalt(wxid, defaultIV, defaultSalt, inputPath, outputPath)
}

// 使用自定义 IV 和 Salt 解密
func DecryptWxapkgWithIVSalt(wxid, ivStr, saltStr, inputPath, outputPath string) error {
	// 读取加密文件
	data, err := os.ReadFile(inputPath)
	if err != nil {
		return err
	}

	// 检查文件头
	if len(data) < len(wxapkgFlag) || string(data[:len(wxapkgFlag)]) != wxapkgFlag {
		return errors.New("不是加密的 wxapkg 文件")
	}

	// PBKDF2 生成 AES 密钥
	key := pbkdf2.Key([]byte(wxid), []byte(saltStr), pbkdf2Iterations, aesKeySize, sha1.New)

	// AES-CBC 解密前 1024 字节
	block, err := aes.NewCipher(key)
	if err != nil {
		return err
	}

	iv := []byte(ivStr)
	mode := cipher.NewCBCDecrypter(block, iv)

	encryptedHead := data[len(wxapkgFlag) : len(wxapkgFlag)+1024]
	originHead := make([]byte, len(encryptedHead))
	mode.CryptBlocks(originHead, encryptedHead)

	// 去除 PKCS7 填充
	originHead, err = pkcs7Unpad(originHead, aes.BlockSize)
	if err != nil {
		return err
	}

	// 计算 XOR 密钥
	xorKey := byte(0x66)
	if len(wxid) >= 2 {
		xorKey = wxid[len(wxid)-2]
	}

	// XOR 解密剩余部分
	afData := data[len(wxapkgFlag)+1024:]
	xorData := make([]byte, len(afData))
	for i := range afData {
		xorData[i] = afData[i] ^ xorKey
	}

	// 拼接结果
	result := make([]byte, len(originHead)+len(xorData))
	copy(result, originHead)
	copy(result[len(originHead):], xorData)

	// 写入输出文件
	return os.WriteFile(outputPath, result, 0644)
}

// PKCS7 去除填充
func pkcs7Unpad(data []byte, blockSize int) ([]byte, error) {
	if len(data) == 0 {
		return nil, errors.New("数据为空")
	}
	if len(data)%blockSize != 0 {
		return nil, errors.New("数据长度不是块大小的整数倍")
	}
	padding := int(data[len(data)-1])
	if padding == 0 || padding > blockSize {
		return nil, errors.New("无效的填充")
	}
	for i := len(data) - padding; i < len(data); i++ {
		if data[i] != byte(padding) {
			return nil, errors.New("无效的填充")
		}
	}
	return data[:len(data)-padding], nil
}
