package util

import "github.com/bytedance/sonic"

func ToJSON[T any](data T) string {
	dataStr, err := sonic.MarshalString(data)
	if err != nil {
		return "{}"
	}
	return dataStr
}
