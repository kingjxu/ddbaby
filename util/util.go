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

func Ptr[T any](x T) *T {
	return &x
}

// Ternary 泛型三元运算符，类似于"cond ? a : b"的用法
func Ternary[T any](cond bool, t T, f T) T {
	if cond {
		return t
	}
	return f
}

func MarshalString(i interface{}) string {
	str, _ := sonic.MarshalString(i)
	return str
}

func Marshal(i interface{}) []byte {
	bytes, _ := sonic.Marshal(i)
	return bytes
}

func UnmarshalString[T any](str string) T {
	var res T
	_ = sonic.UnmarshalString(str, &res)
	return res
}
