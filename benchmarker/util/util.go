package util

import (
	"crypto/md5"
	"fmt"
	"io"
	"io/ioutil"
)

func GetMD5(data []byte) string {
	return fmt.Sprintf("%x", md5.Sum(data))
}

func GetMD5ByIO(r io.Reader) string {
	bytes, _ := ioutil.ReadAll(r)
	return GetMD5(bytes)
}
