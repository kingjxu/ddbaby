package util

import (
	"bytes"
	"encoding/base64"
	"errors"
	"fmt"
	"image"
	"os"

	"image/jpeg"
)

type ImageType string

const (
	ImageTypeJPG     ImageType = "JPG/JPEG"
	ImageTypePNG     ImageType = "PNG"
	ImageTypeGIF     ImageType = "GIF"
	ImageTypeBMP     ImageType = "BMP"
	ImageTypeWebP    ImageType = "WebP"
	ImageTypeUnknown ImageType = "未知（非图片）"
)

// DetectImageType 检测二进制数据是否为图片，并返回图片类型
func DetectImageType(data []byte) ImageType {
	// 空数据直接返回未知
	if len(data) == 0 {
		return ImageTypeUnknown
	}

	// 1. 检查 JPG/JPEG (开头: FF D8 FF, 结尾: FF D9)
	if len(data) >= 3 && data[0] == 0xFF && data[1] == 0xD8 && data[2] == 0xFF {
		// 可选：验证结尾（增强准确性）
		if len(data) >= 2 && data[len(data)-2] == 0xFF && data[len(data)-1] == 0xD9 {
			return ImageTypeJPG
		}
		// 即使结尾不匹配（比如截断的图片），开头匹配也判定为JPG
		return ImageTypeJPG
	}

	// 2. 检查 PNG (开头: 89 50 4E 47 0D 0A 1A 0A)
	if len(data) >= 8 &&
		data[0] == 0x89 && data[1] == 0x50 && data[2] == 0x4E && data[3] == 0x47 &&
		data[4] == 0x0D && data[5] == 0x0A && data[6] == 0x1A && data[7] == 0x0A {
		return ImageTypePNG
	}

	// 3. 检查 GIF (开头: 47 49 46 38)
	if len(data) >= 4 && data[0] == 0x47 && data[1] == 0x49 && data[2] == 0x46 && data[3] == 0x38 {
		return ImageTypeGIF
	}

	// 4. 检查 BMP (开头: 42 4D)
	if len(data) >= 2 && data[0] == 0x42 && data[1] == 0x4D {
		return ImageTypeBMP
	}

	// 5. 检查 WebP (开头: RIFF....WEBP，即 52 49 46 46 xx xx xx xx 57 45 42 50)
	if len(data) >= 12 &&
		data[0] == 0x52 && data[1] == 0x49 && data[2] == 0x46 && data[3] == 0x46 &&
		data[8] == 0x57 && data[9] == 0x45 && data[10] == 0x42 && data[11] == 0x50 {
		return ImageTypeWebP
	}

	// 都不匹配则为未知
	return ImageTypeUnknown
}

// Base64Decode 通用 Base64 解码函数（支持标准/URL 模式）
// input: Base64 编码字符串
// isURL: 是否为 URL 安全的 Base64 编码
// 返回：解码后的二进制数据、错误信息
func Base64Decode(input string, isURL bool) ([]byte, error) {
	// 空输入直接返回错误
	if input == "" {
		return nil, errors.New("Base64 编码字符串不能为空")
	}

	var decoder *base64.Encoding
	if isURL {
		decoder = base64.URLEncoding // URL 安全模式
	} else {
		decoder = base64.StdEncoding // 标准模式
	}

	// 执行解码（核心操作）
	data, err := decoder.DecodeString(input)
	if err != nil {
		return nil, fmt.Errorf("解码失败: %w", err) // 包装错误，保留原始信息
	}
	return data, nil
}

func WriteImageToFile(data []byte, filePath string) error {
	// 创建文件：
	// os.Create 会创建文件（如果不存在），如果存在则清空内容
	// 权限 0644 表示：所有者可读写，其他用户只读（通用文件权限）
	file, err := os.Create(filePath)
	if err != nil {
		return err // 返回创建文件的错误
	}
	// 延迟关闭文件，确保文件句柄被释放
	defer file.Close()

	// 将二进制数据写入文件
	_, err = file.Write(data)
	if err != nil {
		return err // 返回写入数据的错误
	}

	// 强制将缓冲区数据刷入磁盘（确保数据完整写入）
	err = file.Sync()
	if err != nil {
		return err
	}
	return nil
}

func CompressJPEGFromBytes(imgData []byte, quality int) ([]byte, error) {
	// 1. 从二进制数据解码图片
	img, _, err := image.Decode(bytes.NewReader(imgData))
	if err != nil {
		return nil, err
	}

	// 2. 创建缓冲区，把压缩后的图片写到内存
	buf := new(bytes.Buffer)

	// 3. JPEG 编码 + 压缩（核心）
	err = jpeg.Encode(buf, img, &jpeg.Options{
		Quality: quality, // 压缩质量
	})
	if err != nil {
		return nil, err
	}

	// 4. 返回压缩后的二进制
	return buf.Bytes(), nil
}

// ResizeJpegBytes
// 输入：原始 JPEG 二进制 []byte
// 输出：缩放后的 JPEG 二进制 []byte
// 功能：等比缩放，完整图片，不裁剪，不变形
func ResizeJpegBytes(original []byte, maxWidth, maxHeight int) ([]byte, error) {
	// 1. 从二进制解码图片
	img, _, err := image.Decode(bytes.NewReader(original))
	if err != nil {
		return nil, err
	}

	// 原图尺寸
	bounds := img.Bounds()
	origW := bounds.Dx()
	origH := bounds.Dy()

	// 2. 计算等比缩放尺寸（核心：完整显示，不裁剪）
	newW, newH := calcProportionalSize(origW, origH, maxWidth, maxHeight)

	// 3. 创建新画布
	dst := image.NewRGBA(image.Rect(0, 0, newW, newH))

	// 4. 逐像素缩放（纯标准库，真正缩放）
	for y := 0; y < newH; y++ {
		srcY := origH * y / newH
		for x := 0; x < newW; x++ {
			srcX := origW * x / newW
			dst.Set(x, y, img.At(srcX, srcY))
		}
	}

	// 5. 编码成 JPEG 二进制
	buf := new(bytes.Buffer)
	// Quality：1~100，可自行调整
	err = jpeg.Encode(buf, dst, &jpeg.Options{Quality: 85})
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// calcProportionalSize 等比计算宽高，保证完整显示
func calcProportionalSize(origW, origH, maxW, maxH int) (int, int) {
	wRatio := float64(maxW) / float64(origW)
	hRatio := float64(maxH) / float64(origH)

	// 取最小比例，确保图片完整不超出
	ratio := wRatio
	if hRatio < wRatio {
		ratio = hRatio
	}

	return int(float64(origW) * ratio), int(float64(origH) * ratio)
}
