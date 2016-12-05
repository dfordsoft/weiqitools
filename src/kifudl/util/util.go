package util

import (
	"bytes"
	"os"
)

func Exists(f string) bool {
	stat, err := os.Stat(f)
	if err == nil {
		if stat.Mode()&os.ModeType == 0 {
			return true
		}
		return false
	}
	if os.IsNotExist(err) {
		return false
	}
	return false
}

func InsertSlashNth(s string, n int) string {
	var buffer bytes.Buffer
	var n_1 = n - 1
	var l_1 = len(s) - 1
	for i, rune := range s {
		buffer.WriteRune(rune)
		if i%n == n_1 && i != l_1 {
			buffer.WriteRune('/')
		}
	}
	return buffer.String()
}
