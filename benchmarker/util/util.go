package util

import (
	"crypto/md5"
	"fmt"
	"io"
	mrand "math/rand/v2"
)

func GetMD5(data []byte) string {
	return fmt.Sprintf("%x", md5.Sum(data))
}

func GetMD5ByIO(r io.Reader) string {
	bytes, err := io.ReadAll(r)
	if err != nil {
		fmt.Println(err)
	}
	return GetMD5(bytes)
}

var (
	lunRunes = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")
)

func RandomNumber(max int) int {
	return mrand.Int() % max
}

func RandomNumberRange(min, max int) int {
	return mrand.Int()%(max-min+1) + min
}

func RandomLUNStr(n int) string {
	return randomStr(n, lunRunes)
}

func randomStr(n int, s []rune) string {
	buf := make([]byte, 0, n)
	for i := 0; i < n; i++ {
		buf = append(buf, byte(s[mrand.Int()%len(s)]))
	}
	return string(buf)
}
