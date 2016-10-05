package util

import "os"

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
