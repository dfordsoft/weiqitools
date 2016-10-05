// +build windows
package ic

import "fmt"

func Convert(from string, to string, src []byte) []byte {
	fmt.Println("iconv not supported on Windows")
	return src
}

func Convert(from string, to string, src string) string {
	fmt.Println("iconv not supported on Windows")
	return src
}
