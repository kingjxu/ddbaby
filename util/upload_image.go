package util

import (
	"bytes"
	"fmt"
	"github.com/bytedance/sonic"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
)

type UploadResponse struct {
	URL     string `json:"url"` // 上传后的图片直链
	Created int64  `json:"created"`
}

func UploadImage(filePath string) (string, error) {
	// 1. 打开本地图片文件
	file, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("打开文件失败：%v", err)
	}
	defer file.Close() // 延迟关闭文件

	// 2. 创建缓冲区，用于构建 multipart/form-data 格式的请求体
	var requestBody bytes.Buffer
	writer := multipart.NewWriter(&requestBody)

	// 3. 手动创建 file 字段，并指定正确的 Content-Type（核心修复点）
	fileName := filepath.Base(filePath)
	// 手动设置字段的 Content-Disposition 和 Content-Type
	formFile, err := writer.CreatePart(map[string][]string{
		"Content-Disposition": {fmt.Sprintf(`form-data; name="file"; filename="%s"`, fileName)},
		"Content-Type":        {"image/jpeg"},
	})
	if err != nil {
		return "", fmt.Errorf("创建表单文件字段失败：%v", err)
	}

	// 4. 将文件内容复制到表单字段中
	_, err = io.Copy(formFile, file)
	if err != nil {
		return "", fmt.Errorf("复制文件内容失败：%v", err)
	}

	// 5. 关闭 writer，完成请求体构建（必须关闭，否则请求体不完整）
	err = writer.Close()
	if err != nil {
		return "", fmt.Errorf("关闭表单写入器失败：%v", err)
	}

	// 6. 构建 HTTP POST 请求
	req, err := http.NewRequest("POST", "https://imageproxy.zhongzhuan.chat/api/upload", &requestBody)
	if err != nil {
		return "", fmt.Errorf("创建请求失败：%v", err)
	}

	// 7. 设置请求头（关键：Content-Type 必须与 multipart/form-data 匹配）
	req.Header.Set("Content-Type", writer.FormDataContentType())

	// 8. 发送请求（增加超时配置，避免卡死）
	client := &http.Client{
		Timeout: 10 * 1000 * 1000 * 1000, // 10秒超时
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("发送请求失败：%v", err)
	}
	defer resp.Body.Close() // 延迟关闭响应体

	// 9. 读取响应体
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("读取响应失败：%v", err)
	}

	// 10. 解析 JSON 响应
	var uploadResp UploadResponse
	err = sonic.Unmarshal(respBody, &uploadResp)
	if err != nil {
		return "", fmt.Errorf("解析响应失败：%v，原始响应：%s", err, string(respBody))
	}

	// 11. 检查接口返回状态
	if uploadResp.URL == "" {
		return "", fmt.Errorf("上传失败：%s，原始响应：%s", uploadResp.URL, string(respBody))
	}

	// 12. 返回图片直链
	return uploadResp.URL, nil
}
