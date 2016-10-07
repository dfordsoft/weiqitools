// +build windows
package ic

import "fmt"

func Convert(from string, to string, src []byte) []byte {
	if (from != "gbk" && from != "utf-8") || (to != "gbk" && to != "utf-8") {
		fmt.Println("only converting between GBK and UTF-8 is supported")
		return src
	}
	if from == "gbk" && to == "utf-8" {
		if out, e := GbkToUtf8(src); e == nil {
			return out
		} else {
			fmt.Println("converting from gbk to utf-8 failed", e)
		}
	}
	if to == "gbk" && from == "utf-8" {
		if out, e := Utf8ToGbk(src); e == nil {
			return out
		} else {
			fmt.Println("converting from utf-8 to gbk failed", e)
		}
	}
	return src
}

func ConvertString(from string, to string, src string) string {
	return string(Convert(from, to, []byte(src)))
}
