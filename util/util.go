package util

import (
	"github.com/bytedance/sonic"
	"math/rand"
)

const RANDCHAR = "1234567890ABCDEFGHIJKLMNOPQRSTUVWXYZ"

func ToJSON[T any](data T) string {
	dataStr, err := sonic.MarshalString(data)
	if err != nil {
		return "{}"
	}
	return dataStr
}

func CreateRandString(n int) string {
	nonceStr := ""
	for i := 0; i < n; i++ {
		index := rand.Intn(len(RANDCHAR))
		nonceStr += string(RANDCHAR[index])
	}
	return nonceStr
}
