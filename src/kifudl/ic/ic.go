// +build !windows

package ic

import (
	"log"

	"gopkg.in/iconv.v1"
)

func Convert(from string, to string, src []byte) []byte {
	cd, err := iconv.Open(to, from)
	if err != nil {
		log.Println("iconv.Open failed!")
		return src
	}
	defer cd.Close()

	outbuf := make([]byte, len(src)*4)
	out, _, _ := cd.Conv(src, outbuf)
	return out
}

func ConvertString(from string, to string, src string) string {
	cd, err := iconv.Open(to, from)
	if err != nil {
		log.Println("iconv.Open failed!")
		return src
	}
	defer cd.Close()
	return cd.ConvString(src)
}
