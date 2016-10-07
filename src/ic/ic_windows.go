// +build windows
package ic

import "fmt"

func Convert(from string, to string, src []byte) []byte {
	if to == "utf-8" {
		if out, e := ToUTF8(from, src); e == nil {
			return out
		} else {
			fmt.Printf("converting from %s to UTF-8 failed: %v", from, e)
			return src
		}
	}

	if from == "utf-8" {
		if out, e := FromUTF8(to, src); e != nil {
			return out
		} else {
			fmt.Printf("converting from UTF-8 to %s failed: %v", to, e)
			return src
		}
	}
	fmt.Println("only converting between CJK encodings and UTF-8 is supported")
	return src
}

func ConvertString(from string, to string, src string) string {
	return string(Convert(from, to, []byte(src)))
}
