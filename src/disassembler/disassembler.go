package disassembler

import (
	"bufio"
	"encoding/gob"
	"exstrings"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
)

type Instruction struct {
	Opcode byte
	Name   string
	Params string
}

type Mode struct {
	a 	bool
	xy 	bool
}

func Parse(file string, m int) {
	Instructions, err := LoadInstructionSet("65c816.dat")
	if err != nil {
		panic(err)
	}

	var mode Mode
	if m == 8 {
		mode = Mode{false, false}
	} else if m == 16 {
		mode = Mode{ true, true }
	} else {
		fmt.Println("Invalid flag: mode=", m)
		return
	}

	f, err := os.Open(file)
	if err != nil {
		panic(err)
	}

	fr := bufio.NewReader(f)
	address := 0
	output := ""

	for {
		out := ""
		b, err := fr.ReadByte()
		if err != nil {
			break
		}

		n, p, err := searchOpcode(Instructions, b)
		if err != nil {
			panic(err)
		}

		if n == "SEP" || n == "REP" {
			b, err := fr.ReadByte()
			if err != nil {
				panic(err)
			}
			//	The following evaluate to true if the command is REP or false otherwise
			if b == 0x10 {
				mode.xy = n == "REP"
			} else if b == 0x20 {
				mode.a = n == "REP"
			} else if b == 0x30 {
				mode.a = n == "REP"
				mode.xy = n == "REP"
			} else {
				fmt.Printf("Invalid byte: %s #$%X\n", n, b)
				return
			}

			out = "#$" + exstrings.Bytetostring(b)
		} else if p != "" {
			parameters := strings.Split(p, ",")

			for _, v := range parameters {
				bytes := 0
				num := ""

				if p == "#" {
					bytes = getMode(mode, n)
				} else {
					bytes = strings.Count(v, "xx")
				}

				for tb := bytes; tb > 0; tb-- {
					cb, err := fr.ReadByte()
					if err != nil {
						panic(err)
					}
					num = exstrings.Bytetostring(cb) + num
				}

				if p == "#" {
					out = "#$" + num
				} else if out == "" {
					out += strings.Replace(v, strings.Repeat("xx", bytes), num, 1)
				} else {
					out += "," + strings.Replace(v, strings.Repeat("xx", bytes), num, 1)
				}
			}
		}
		address += countBytes(n + " " + out)

		if isBranch(n) {
			num, err := strconv.ParseInt(exstrings.Remove(out, "#$, "), 16, 16)
			if num > 127 {
				num -= 256	
			}
			
			if err != nil {
				panic(err)
			}
			out = ".label_" + strconv.FormatInt(num + int64(address), 16)
		}
		output += n + " " + out + "\n"
	}
	spl := strings.Split(output, "\n") 
	labels := make([]int64, strings.Count(output, ".label_"))
	index := 0
	for _, v := range spl {
		line := strings.Split(v, " ")
		if isBranch(line[0]) {
			labels[index], _ = strconv.ParseInt(strings.Replace(line[1], ".label_", "", 1), 16, 16)
			fmt.Println("Added", labels[index])
			index++
		}
	}

	address = 0
	output = ""
	for _, v := range spl {
		if containsValue(labels, int64(address)) {
			output += ".label_" + strconv.FormatInt(int64(address), 16) + ":\n"
		} 
		output += v + "\n"
		fmt.Println(v, countBytes(v))
		address += countBytes(v)
	}
	fmt.Print(output)
}

func containsValue(i []int64, c int64) bool {
	for _, v := range i {
		if v == c {
			return true
		}
	}
	return false
}

func isBranch(s string) bool {
	return 	s == "BCC" || s == "BCS" || s == "BEQ" ||
			s == "BMI" || s == "BNE" || s == "BPL" ||
			s == "BRA" || s == "BRL" || s == "BVC" || s == "BVS"
}
func countBytes(str string) int {
	if str == "" {
		return 0
	}
	spl := strings.Split(str, " ")
	if spl[0] == "JSR" || spl[0] == "JMP" {
		if spl[1][0:1] != "$" {
			return 3
		}
	} else if len(spl) == 2 && isBranch(spl[0]) && spl[1][0:1] == "." {
		if spl[0] == "BRL" {
			return 3
		} else {
			return 2
		}
	}
	regnum, _ := regexp.Compile("[0-9a-fA-F]{2}")
	str = regnum.ReplaceAllString(strings.Join(exstrings.Split(str, " ,")[1:]," "), "x")
	return 1 + strings.Count(str, "x")
}

func getMode(m Mode, str string) (i int) {
	if 	str == "ADC" || str == "ORA" || str == "LDA" ||
		str == "BIT" || str == "CMP" || str == "SBC" {

		if m.a {
			i = 2
		} else {
			i = 1
		}
	} else {
		if m.xy {
			i = 2
		} else {
			i = 1
		}
	}
	return
}

func searchOpcode(set []Instruction, op byte) (name, delim string, err error) {
	for _, i := range set {
		if i.Opcode == op {
			return i.Name, i.Params, nil
		}
	}
	return "", "", fmt.Errorf("Opcode not found: %X", op)
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
