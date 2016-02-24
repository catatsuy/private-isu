package util

import (
	"crypto/md5"
	crand "crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"io/ioutil"
	mrand "math/rand"
	"time"
)

func GetMD5(data []byte) string {
	return fmt.Sprintf("%x", md5.Sum(data))
}

func GetMD5ByIO(r io.Reader) string {
	bytes, err := ioutil.ReadAll(r)
	if err != nil {
		fmt.Println(err)
	}
	return GetMD5(bytes)
}

var (
	lunRunes = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")
	lnRunes  = []rune("abcdefghijklmnopqrstuvwxyz0123456789")
	random   = mrand.New(mrand.NewSource(time.Now().UnixNano()))
)

func RandomLUNStr(n int) string {
	return randomStr(n, lunRunes)
}

func RandomLNStr(n int) string {
	return randomStr(n, lnRunes)
}

func randomStr(n int, s []rune) string {
	buf := make([]byte, 0, n)
	for i := 0; i < n; i++ {
		buf = append(buf, byte(s[random.Int()%len(s)]))
	}
	return string(buf)
}

func SecureRandomStr(n int) string {
	k := make([]byte, n)
	if _, err := io.ReadFull(crand.Reader, k); err != nil {
		return ""
	}
	return base64.StdEncoding.EncodeToString(k)
}
