package assembler

import (
	"encoding/gob"
	"exstrings"
	"fmt"
	"io/ioutil"
	"os"
	"regexp"
	"strings"
)

type Instruction struct {
	Opcode 	byte
	Name   	string
	Params 	string
}

func Parse(file string) {
	//	Load instructions for instructing things
	Instructions, err := LoadInstructionSet("65c816.dat")
	if err != nil {
		exitError(err)
	}

	bfile, err := ioutil.ReadFile(file)
	if err != nil {
		exitError(err)
	}

	text := string(bfile)

	//	Remove tabs because they're icky
	text = strings.Replace(text, "\t", " ", -1)

	//	Make sure all spaces are just a singls space
	text = exstrings.Compress(text, "  ", " ")

	//	Make sure carriage returns are put in their place
	strings.Replace(text, "\r\n", "\n", -1)

	err = sanityTest(text)
	if err != nil {
		exitError(err)
	}

	//	Prepend all newlines with a space so they don't get skipped on split
	text = strings.Replace(text, "\n", " \n", -1)

	lines := strings.Split(text, "\n")

	var commentBlock bool //, macroBlock bool
	address := 0 //, macroIndex := 0, 0
	bin := make([]byte, 1024)
	defines := make(map[string]string, exstrings.Countregex(text, "![a-zA-Z0-9] (=|equ) .*"))
	//macros := make([]string, exstrings.Countregex(exstrings.Stripquotes(text), "macro\\(.*\\)"))

	for number, line := range lines {
		number++	//	Starts at 0 index, so push it up one
		line = stripComments(strings.TrimSpace(line))
		noquotes := exstrings.Stripquotes(line)

		//	Line is empty or only a comment, skip before any processing
		if line == "" {
			continue
		}

		//	Check for start or end of a comment block
		if commentBlock {
			if strings.Contains(noquotes, "*/") {
				commentBlock = false
				line = line[strings.Index(line, "*/")+2:]
				if line != "" {
					fmt.Printf("Ending Comment: '%s'\n", line)
				}
			}
		} else if strings.Contains(noquotes, "/*") {
			line = line[0:strings.Index(line, "/*")]
			if line != "" {
				fmt.Printf("Starting comment: '%s'\n", line)
			}
			commentBlock = true
		}

		//	Empty after comment processing, skip
		if line == "" {
			continue
		}

		if strings.Contains(noquotes, "!") {
			line, err = parseDefine(line, defines)
			if err != nil {
				err = fmt.Errorf("%s on line %d", err, number)
				exitError(err)
			}
		}

		if line != "" {
			fmt.Printf("%d: '%s'\n", number, line)
		}
	}
	fmt.Println(Instructions[0])
	fmt.Println(bin[0])
	address++
}


func parseDefine(s string, def map[string] string) (string, error) {
	noquotes := exstrings.Stripquotes(s)
	regdef := regexp.MustCompile("(![a-zA-Z0-9]+) (=|equ) (.*)")
	regnum := regexp.MustCompile("([a-zA-Z0-9]{2})[ ]*\\$([a-zA-Z0-9]{2})")
	m := regdef.FindAllStringSubmatch(noquotes, -1)

	if m != nil {
		v := m[0]
		if def[v[1]] != "" {
			return "", fmt.Errorf("%s redefined", v[1])
		} 
		if strings.Contains(v[3], "!") {
			split := exstrings.Split(v[3], " $#,")
			for _, str := range split {
				if strings.Contains(str, "!") {
					str = str[strings.Index(str, "!"):]

					if def[str] == "" {
						return "", fmt.Errorf("%s not defined before use.", str)
					}
					//	Replace the !define with its value
					v[3] = strings.Replace(v[3], str, def[str], -1)

					//	Replace any $xx$xx with $xxxx
					v[3] = regnum.ReplaceAllString(v[3], "$1$2")
				}
			}
		}
		def[v[1]] = v[3]
		return "", nil
	} else {
		regsym := regexp.MustCompile("(![a-zA-Z0-9]+)")
		matches := regsym.FindAllStringSubmatch(s, -1)
		submatches := make([]string, len(matches))

		for k, v := range matches {
			submatches[k] = v[1]
		}

		sortDescending(submatches)

		for _, v := range submatches {
			s = strings.Replace(s, v, def[v], -1)
		}
		s_split := strings.Split(s, " ")
		s = s_split[0] + " " + regnum.ReplaceAllString(strings.Join(s_split[1:], " "), "$1$2")
	}
	return s, nil
}

func stripComments(str string) string {
	c_comments := regexp.MustCompile("(//.*|/\\*.*\\*/)")
	asm_comments := regexp.MustCompile(";.*")

	str = c_comments.ReplaceAllString(str, "")
	str = asm_comments.ReplaceAllString(str, "")
	return str
}

func sanityTest(s string) error {
	//	Remove strings in quotes to not catch "/*", etc
	noquotes := exstrings.Stripquotes(s)

	startblock := strings.Count(noquotes, "/*")
	endblock := strings.Count(noquotes, "*/")

	if startblock != endblock {
		//	Prepend all newlines with a space so they don't get skipped on split
		noquotes = strings.Replace(noquotes, "\n", " \n", -1)

		comment, last := 0, 0

		for k, v := range strings.Split(noquotes, "\n") {
			if strings.Contains(v, "/*") && strings.Contains(v, "*/") {
				comment += strings.Count(v, "/*")
				comment -= strings.Count(v, "*/")
				if comment <= 1 && comment >= -1 {
					last = k + 1
				}
			} else if strings.Contains(v, "/*") {
				comment++
				if comment <= 1 {
					last = k + 1
				}
			} else if strings.Contains(v, "*/") {
				comment--
				if comment >= -1 {
					last = k + 1
				}
			}

			if comment > 1 {
				return fmt.Errorf("Mismatching /* found on line %d", last)
			} else if comment < 0 {
				return fmt.Errorf("Mismatching */ found on line %d", last)
			}
		}
	} else {
		comment, last := 0, 0
		for k, v := range strings.Split(noquotes, "\n") {
			if strings.Contains(v, "/*") && strings.Contains(v, "*/") {
				comment += strings.Count(v, "/*")
				comment -= strings.Count(v, "*/")
			} else if strings.Contains(v, "/*") {
				comment++
				if comment <= 1 {
					last = k + 1
				}
			} else if strings.Contains(v, "*/") {
				comment--
				if comment >= -1 {
					last = k + 1
				}
			}

			if comment > 1 {
				return fmt.Errorf("Multi-level multiline comment found from line %d to %d", last, k + 1)
			}
		}
	}
	return nil
}

func sortDescending(str []string) []string {
	str = exstrings.Sort(str, func(a, b string) int {
			if len(a) > len(b) {
				return 1
			} else if len(a) < len(b) {
				return -1
			}
			return 0
		})
	return str
}

func LoadInstructionSet(file string) (inst []Instruction, err error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	dec := gob.NewDecoder(f)
	err = dec.Decode(&inst)
	if err != nil {
		return nil, err
	}
	return
}

//	Append taken from http://golang.org/doc/effective_go.html#slices
func Append(slice, data []byte) []byte {
    l := len(slice)
    if l + len(data) > cap(slice) {  // reallocate
        // Allocate double what's needed, for future growth.
        newSlice := make([]byte, (l+len(data))*2)
        // The copy function is predeclared and works for any slice type.
        copy(newSlice, slice)
        slice = newSlice
    }
    slice = slice[0:l+len(data)]
    for i, c := range data {
        slice[l+i] = c
    }
    return slice
}

func exitError(err error) {
	fmt.Println(err)
	os.Exit(1)
}
