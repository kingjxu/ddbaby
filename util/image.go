package util

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
