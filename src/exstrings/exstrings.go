package exstrings

import (
	"bytes"
	"strconv"
	"strings"
)

func Split(str, delimiters string) []string {
	return strings.FieldsFunc(str, func(r rune) bool {
		for i := 0; i < len(delimiters); i++ {
			if string(r) == delimiters[i:i+1] {
				return true
			}
		}
		return false
	})
}

func TrimMiddle(str string) string {
	var buffer bytes.Buffer
	for _, word := range Split(str, " \t") {
		buffer.WriteString(word + " ")
	}
	return buffer.String()
}

func Sort(s []string, callback func(a, b string) int) []string {
	sorted := false

	for ; sorted == false; {
		for i := 0; i < len(s) - 1; i++ {
			ret := callback(s[i], s[i+1])

			//	a < b, swap a -> b
			if ret < 0 {
				//fmt.Printf("%s > %s\n", s[i], s[i+1])
				t := s[i]
				s[i] = s[i+1]
				s[i+1] = t
			}
		}
		sorted = true
		for i := 0; i < len(s) - 1; i++ {
			if callback(s[i], s[i+1]) < 0 {
				sorted = false
			}
		}
	}
	return s
}

func Remove(str, delim string) string {
	for i := 0; i < len(delim); i++ {
		str = strings.Replace(str, delim[i:i+1], "", -1)
	}
	return str
}

func Bytetostring(b byte) (str string) {
	n1 := b & 0xF
	n2 := b & (0xF << 4) >> 4

	if n2 >= 0 && n2 <= 9 {
		str = string(n2 + '0')
	} else {
		str = strconv.FormatInt(int64(n2), 16)
	}
	if n1 >= 0 && n1 <= 9 {
		str += string(n1 + '0')
	} else {
		str += strconv.FormatInt(int64(n1), 16)
	}
	return strings.ToUpper(str)
}