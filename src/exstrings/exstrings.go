package exstrings

import (
	"bytes"
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