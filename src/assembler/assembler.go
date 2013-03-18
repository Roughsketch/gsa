package assembler

import (
	"bytes"
	"encoding/gob"
	"exstrings"
	"fmt"
	"io/ioutil"
	"os"
	"regexp"
	"strconv"
	"strings"
)

type Instruction struct {
	Opcode 	byte
	Name   	string
	Params 	string
}

func Parse(file string) {
	/*Inst := InstructionSet()
	CreateInstructionSet(Inst, "65c816.dat")
	return*/
	Instructions, err := LoadInstructionSet("65c816.dat")
	if err != nil {
		fmt.Print("Could not load 65c816.dat")
		return
	}

	text, err := ioutil.ReadFile(file)
	if err != nil {
		panic(err)
	}

	lines := strings.Split(stripComments(string(text)), "\n")
	if err != nil {
		panic(err)
	}

	f, err := os.Create("out.log")
	defer f.Close()

	address := 0
	labels := make(map[string]int, strings.Count(strings.Join(lines, ""), ":"))

	//	Single pass to find label addresses and fix special cases
	for i, line := range lines {
		line = strings.TrimSpace(line)
		if len(line) != 0 {
			if line[len(line)-1:] == ":" {
				fmt.Printf("Found label %s at %d\n", line[0:len(line)-1], address)
				labels[line[0:len(line)-1]] = address
			} else {
				spl := strings.Split(line, " ")
				if len(spl) == 2 && strings.ToUpper(spl[1]) == "A" {
					lines[i] = strings.ToUpper(spl[0])
				}
				fmt.Println(line, countBytes(line))
				address += countBytes(line)
			}
		}
	}

	linetemp := strings.Join(lines, "\n")
	templabels := sortLabels(labels)

	for _, k := range templabels {
		straddr := strconv.Itoa(labels[k])
		linetemp = strings.Replace(linetemp, k, "~L"+straddr, -1)
		fmt.Printf("Replacing %s with %s\n", k, "~L"+straddr)
	}

	lines = strings.Split(linetemp, "\n")

	bin := make([]byte, address)
	//	Reset address for second run
	address = 0

	for _, line := range lines {
		pieces := exstrings.Split(line, " ,")

		if len(pieces) != 0 && strings.Contains(line, ":") == false{
			name := strings.ToUpper(pieces[0])

			if len(pieces) == 1 {
				b, err := searchInstruction(Instructions, name, "")
				bin[address] = b
				if err != nil {
					fmt.Println("Could not find " + name + " in set")
				} else {
					fmt.Printf("%4.2X ", b)
				}
				bin[address] = b
				address++
			} else {
				oper, err := getOpcodes(pieces, address, Instructions)

				if err != nil {
					fmt.Println("Could not find " + name)
				} else {
					for _, v := range oper {
						fmt.Printf("%4.2X ", v)
						bin[address] = v
						address++
					}
				}
				//bin[address:] = oper
				//address += countBytes(strings.Join(pieces, " "))
			}
		}
	}
	n, _ := f.Write(bin)
	if n != address {
		f.Close()
		os.Remove("out.log")
		f, _ = os.Create("out.log")
		defer f.Close()
		f.Write(bin[0:n - (n - address)])
	}

}

func getOpcodes(str []string, address int, inset []Instruction) (bin []byte, err error) {
	i := 0
	name := strings.ToUpper(str[0])
	bin = make([]byte, 4, 4)

	regnum, err := regexp.Compile("[0-9a-fA-F]{2}")
	reghash, err := regexp.Compile("#\\$[x]+")

	for k, v := range str[1:] {
		j := k + 1
		if len(v) > 2 && v[0:2] == "~L" {
			lbladdr, _ := strconv.Atoi(v[2:])
			if address > lbladdr {
				lbladdr -= address + countBytes(strings.Join(str, " "))
			}
			str[j] = getUint8(int64(lbladdr))
			if len(str[j]) % 2 != 0 {
				str[j] = "0" + str[j]
			}

			if name == "JSR" {
				if len(str[j]) <= 2 {
					str[j] = "00" + str[j]
				}
				if len(str[j]) == 3 || len(str[j]) == 5{
					str[j] = "0" + str[j]
				}
			}

			str[j] = "$" + str[j]
		}
	}
	params := strings.Join(str[1:], ",")
	params = regnum.ReplaceAllString(params, "xx")
	params = reghash.ReplaceAllString(params, "#")
	b, err := searchInstruction(inset, name, params)

	if err != nil {
		fmt.Println("Can't find", name, params)
		return nil, err
	}

	bin[i] = b
	i++
	
	for _, v := range str[1:] {
		index := strings.Index(v, "$")

		if index != -1 {
			max := (len(v[index:]) - 1)/2
			for m := max; m > 0; index += 2 {
				m--
				value, _ := strconv.ParseInt(v[index+1:index+3], 16, 0)
				bin[i+m] = byte(value)
			}
			i += max
		}
	}

	return bin[0:i], nil
}

func sortLabels(m map[string] int) []string {
	str := make([]string, len(m))
	i := 0

	for k, _ := range m {
		str[i] = k
		i++
	}

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

func countBytes(str string) int {
	spl := strings.Split(str, " ")
	if spl[0] == "JSR" {
		if spl[1][0:1] != "$" {
			return 3
		}
	}
	regnum, _ := regexp.Compile("[0-9a-fA-F]{2}")
	str = regnum.ReplaceAllString(strings.Join(exstrings.Split(str, " ,")[1:]," "), "x")
	return 1 + strings.Count(str, "x")
}
func searchInstruction(set []Instruction, inst, delim string) (byte, error) {
	for _, i := range set {
		if i.Name == inst && i.Params == delim {
			return i.Opcode, nil
		}
	}
	return 0, fmt.Errorf("Opcode not found")
}

func stripComments(str string) string {
	var buffer bytes.Buffer

	//	Split text into lines
	lines := exstrings.Split(str, "\r\n")

	for index, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" && line[0:1] != ";" {
			i := strings.Index(line, ";")
			if i != -1 {
				line = strings.TrimSpace(line[0:i])
			}

			if index == 0 {
				buffer.WriteString(line)
			} else {
				buffer.WriteString("\n" + line)
			}
		}
	}

	return buffer.String()
}

func getUint8(i int64) (str string) {
	num := i & (0xF << 4) >> 4
	if num < 10 {
		str += string(num+'0')
	} else {
		str += strconv.FormatInt(int64(num), 16)
	}
	num = i & 0xF
	if num < 10 {
		str += string(num+'0')
	} else {
		str += strconv.FormatInt(int64(num), 16)
	}
	return
}

func CreateInstructionSet(set []Instruction, file string) error {
	buffer := new(bytes.Buffer)
	enc := gob.NewEncoder(buffer)
	err := enc.Encode(set)
	if err != nil {
		return err
	}

	f, err := os.OpenFile(file, os.O_CREATE|os.O_WRONLY, 0666)
	defer f.Close()

	if err != nil {
		return err
	}

	_, err = f.Write(buffer.Bytes())
	if err != nil {
		return err
	}

	return nil
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


func InstructionSet() []Instruction {
	return []Instruction{
		{ 0x00, "BRK", "" 		}, { 0x01, "ORA", "($xx,x)" 	},
		{ 0x02, "COP", "#" 		}, { 0x03, "ORA", "$xx,s" 	},
		{ 0x04, "TSB", "$xx" 	}, { 0x05, "ORA", "$xx" 		},
		{ 0x06, "ASL", "$xx" 	}, { 0x07, "ORA", "[$xx]" 	},
		{ 0x08, "PHP", "" 		}, { 0x09, "ORA", "#" 		},
		{ 0x0A, "ASL", "" 		}, { 0x0B, "PHD", "" 		},
		{ 0x0C, "TSB", "$xxxx" 	}, { 0x0D, "ORA", "$xxxx" 		},
		{ 0x0E, "ASL", "$xxxx" 	}, { 0x0F, "ORA", "$xxxxxx" 		},
		{ 0x10, "BPL", "$xx" 	}, { 0x11, "ORA", "($xx),y" 	},
		{ 0x12, "ORA", "($xx)" 	}, { 0x13, "ORA", "($xx,s),y"},
		{ 0x14, "TRB", "$xx" 	}, { 0x15, "ORA", "$xx,x" 	},
		{ 0x16, "ASL", "$xx,x" 	}, { 0x17, "ORA", "[$xx],y" 	},
		{ 0x18, "CLC", "" 		}, { 0x19, "ORA", "$xxxx,y" 	},
		{ 0x1A, "INC", "" 		}, { 0x1B, "TCS", "" 		},
		{ 0x1C, "TRB", "$xxxx" 	}, { 0x1D, "ORA", "$xxxx,x" 	},
		{ 0x1E, "ASL", "$xxxx,x" 	}, { 0x1F, "ORA", "$xxxxxx,x" 	},
		{ 0x20, "JSR", "$xxxx" 	}, { 0x21, "AND", "($xx,x)" 	},
		{ 0x22, "JSR", "$xxxxxx" 	}, { 0x23, "AND", "$xx,s" 	},
		{ 0x24, "BIT", "$xx" 	}, { 0x25, "AND", "$xx" 		},
		{ 0x26, "ROL", "$xx" 	}, { 0x27, "AND", "[$xx]" 	},
		{ 0x28, "PLP", "" 		}, { 0x29, "AND", "#" 		},
		{ 0x2A, "ROL", "" 		}, { 0x2B, "PLD", "" 		},
		{ 0x2C, "BIT", "$xxxx" 	}, { 0x2D, "AND", "$xxxx" 		},
		{ 0x2E, "ROL", "$xxxx" 	}, { 0x2F, "AND", "$xxxxxx" 		},
		{ 0x30, "BMI", "$xx" 	}, { 0x31, "AND", "($xx),y" 	},
		{ 0x32, "AND", "($xx)" 	}, { 0x33, "AND", "($xx,s),y"},
		{ 0x34, "BIT", "$xx,x" 	}, { 0x35, "AND", "$xx,x" 	},
		{ 0x36, "ROL", "$xx,x" 	}, { 0x37, "AND", "[$xx],y" 	},
		{ 0x38, "SEC", "" 		}, { 0x39, "AND", "$xxxx,y" 	},
		{ 0x3A, "DEC", "" 		}, { 0x3B, "TSC", "" 		},
		{ 0x3C, "BIT", "$xxxx,x" 	}, { 0x3D, "AND", "$xxxx,x" 	},
		{ 0x3E, "ROL", "addr,x" }, { 0x3F, "AND", "$xxxxxx,x" 	},
		{ 0x40, "RTI", "" 		}, { 0x41, "EOR", "($xx,x)" 	},
		{ 0x42, "WDM", "" 		}, { 0x43, "EOR", "$xx,s" 	},
		{ 0x44, "MVN", "$00,$00"}, { 0x45, "EOR", "$xx" 		},
		{ 0x46, "LSR", "$xx" 	}, { 0x47, "EOR", "[$xx]" 	},
		{ 0x48, "PHA", "" 		}, { 0x49, "EOR", "#" 		},
		{ 0x4A, "LSR", "" 		}, { 0x4B, "PHK", "" 		},
		{ 0x4C, "JMP", "$xxxx" 	}, { 0x4D, "EOR", "$xxxx" 		},
		{ 0x4E, "LSR", "$xxxx" 	}, { 0x4F, "EOR", "$xxxxxx" 		},
		{ 0x50, "BVC", "$xx" 	}, { 0x51, "EOR", "($xx),y" 	},
		{ 0x52, "EOR", "($xx)" 	}, { 0x53, "EOR", "($xx,s),y"},
		{ 0x54, "MVN", "$00,$00"}, { 0x55, "EOR", "$xx,x" 	},
		{ 0x56, "LSR", "$xx,x" 	}, { 0x57, "EOR", "[$xx],y" 	},
		{ 0x58, "CLI", "" 		}, { 0x59, "EOR", "$xxxx,y" 	},
		{ 0x5A, "PHY", "" 		}, { 0x5B, "TCD", "" 		},
		{ 0x5C, "JMP", "$xxxxxx" 	}, { 0x5D, "EOR", "$xxxx,x" 	},
		{ 0x5E, "LSR", "$xxxx,x" 	}, { 0x5F, "EOR", "$xxxxxx,x" 	},
		{ 0x60, "RTS", "" 		}, { 0x61, "ADC", "($xx,x)" 	},
		{ 0x62, "PER", "$xxxx" 	}, { 0x63, "ADC", "$xx,s" 	},
		{ 0x64, "STZ", "$xx" 	}, { 0x65, "ADC", "$xx" 		},
		{ 0x66, "ROR", "$xx" 	}, { 0x67, "ADC", "[$xx]" 	},
		{ 0x68, "PLA", "" 		}, { 0x69, "ADC", "#" 		},
		{ 0x6A, "ROR", "" 		}, { 0x6B, "RTL", "" 		},
		{ 0x6C, "JMP", "($xxxx)" 	}, { 0x6D, "ADC", "$xxxx" 		},
		{ 0x6E, "ROR", "$xxxx" 	}, { 0x6F, "ADC", "$xxxxxx" 		},
		{ 0x70, "BVS", "$xx" 	}, { 0x71, "ADC", "($xx),y" 	},
		{ 0x72, "ADC", "($xx)" 	}, { 0x73, "ADC", "($xx,s),y"},
		{ 0x74, "STZ", "$xx,x" 	}, { 0x75, "ADC", "$xx,x" 	},
		{ 0x76, "ROR", "$xx,x" 	}, { 0x77, "ADC", "[$xx],y" 	},
		{ 0x78, "SEI", "" 		}, { 0x79, "ADC", "$xxxx,y" 	},
		{ 0x7A, "PLY", "" 		}, { 0x7B, "TDC", "" 		},
		{ 0x7C, "JMP", "($xxxx,x)" }, { 0x7D, "ADC", "$xxxx,x" 	},
		{ 0x7E, "ROR", "$xxxx,x" 	}, { 0x7F, "ADC", "$xxxxxx,x" 	},
		{ 0x80, "BRA", "$xx" 	}, { 0x81, "STA", "($xx,x)" 	},
		{ 0x82, "BRL", "$xxxx" 	}, { 0x83, "STA", "$xx,s" 	},
		{ 0x84, "STY", "$xx" 	}, { 0x85, "STA", "$xx" 		},
		{ 0x86, "STX", "$xx" 	}, { 0x87, "STA", "[$xx]" 	},
		{ 0x88, "DEY", "" 		}, { 0x89, "BIT", "#" 		},
		{ 0x8A, "TXA", "" 		}, { 0x8B, "PHB", "" 		},
		{ 0x8C, "STY", "$xxxx" 	}, { 0x8D, "STA", "$xxxx" 		},
		{ 0x8E, "STX", "$xxxx" 	}, { 0x8F, "STA", "$xxxxxx" 		},
		{ 0x90, "BCC", "$xx" 	}, { 0x91, "STA", "($xx),y" 	},
		{ 0x92, "STA", "($xx)" 	}, { 0x93, "STA", "($xx,s),y"},
		{ 0x94, "STY", "$xx,x" 	}, { 0x95, "STA", "$xx,x" 	},
		{ 0x96, "STX", "$xx,y" 	}, { 0x97, "STA", "[$xx],y" 	},
		{ 0x98, "TYA", "" 		}, { 0x99, "STA", "$xxxx,y" 	},
		{ 0x9A, "TXS", "" 		}, { 0x9B, "TXY", "" 		},
		{ 0x9C, "STZ", "$xxxx" 	}, { 0x9D, "STA", "$xxxx,x" 	},
		{ 0x9E, "STZ", "$xxxx,x" 	}, { 0x9F, "STA", "$xxxxxx,x" 	},
		{ 0xA0, "LDY", "#" 		}, { 0xA1, "LDA", "($xx,x)" 	},
		{ 0xA2, "LDX", "#" 		}, { 0xA3, "LDA", "$xx,s" 	},
		{ 0xA4, "LDY", "$xx" 	}, { 0xA5, "LDA", "$xx" 		},
		{ 0xA6, "LDX", "$xx" 	}, { 0xA7, "LDA", "[$xx]" 	},
		{ 0xA8, "TAY", "" 		}, { 0xA9, "LDA", "#" 		},
		{ 0xAA, "TAX", "" 		}, { 0xAB, "PLB", "" 		},
		{ 0xAC, "LDY", "$xxxx" 	}, { 0xAD, "LDA", "$xxxx" 		},
		{ 0xAE, "LDX", "$xxxx" 	}, { 0xAF, "LDA", "$xxxxxx" 		},
		{ 0xB0, "BCS", "$xx" 	}, { 0xB1, "LDA", "($xx),y" 	},
		{ 0xB2, "LDA", "($xx)" 	}, { 0xB3, "LDA", "($xx,s),y"},
		{ 0xB4, "LDY", "$xx,x" 	}, { 0xB5, "LDA", "$xx,x" 	},
		{ 0xB6, "LDX", "$xx,y" 	}, { 0xB7, "LDA", "[$xx],y" 	},
		{ 0xB8, "CLV", "" 		}, { 0xB9, "LDA", "$xxxx,y" 	},
		{ 0xBA, "TSX", "" 		}, { 0xBB, "TYX", "" 		},
		{ 0xBC, "LDY", "$xxxx,x" 	}, { 0xBD, "LDA", "$xxxx,x" 	},
		{ 0xBE, "LDX", "$xxxx,y" 	}, { 0xBF, "LDA", "$xxxxxx,x" 	},
		{ 0xC0, "CPY", "#" 		}, { 0xC1, "CMP", "($xx,x)" 	},
		{ 0xC2, "REP", "#" 		}, { 0xC3, "CMP", "$xx,s" 	},
		{ 0xC4, "CPY", "$xx" 	}, { 0xC5, "CMP", "$xx" 		},
		{ 0xC6, "DEC", "$xx" 	}, { 0xC7, "CMP", "[$xx]" 	},
		{ 0xC8, "INY", "" 		}, { 0xC9, "CMP", "#" 		},
		{ 0xCA, "DEX", "" 		}, { 0xCB, "WAI", "" 		},
		{ 0xCC, "CPY", "$xxxx" 	}, { 0xCD, "CMP", "$xxxx" 		},
		{ 0xCE, "DEC", "$xxxx" 	}, { 0xCF, "CMP", "$xxxxxx" 		},
		{ 0xD0, "BNE", "$xx" 	}, { 0xD1, "CMP", "($xx),y" 	},
		{ 0xD2, "CMP", "($xx)" 	}, { 0xD3, "CMP", "($xx,s),y"},
		{ 0xD4, "PEI", "($xx)" 	}, { 0xD5, "CMP", "$xx,x" 	},
		{ 0xD6, "DEC", "$xx,x" 	}, { 0xD7, "CMP", "[$xx],y" 	},
		{ 0xD8, "CLD", "" 		}, { 0xD9, "CMP", "$xxxx,y" 	},
		{ 0xDA, "PHX", "" 		}, { 0xDB, "STP", "" 		},
		{ 0xDC, "JMP", "[$xxxx]" 	}, { 0xDD, "CMP", "$xxxx,x" 	},
		{ 0xDE, "DEC", "$xxxx,x" 	}, { 0xDF, "CMP", "$xxxxxx,x" 	},
		{ 0xE0, "CPX", "#" 		}, { 0xE1, "SBC", "($xx,x)" 	},
		{ 0xE2, "SEP", "#" 		}, { 0xE3, "SBC", "$xx,s" 	},
		{ 0xE4, "CPX", "$xx" 	}, { 0xE5, "SBC", "$xx" 		},
		{ 0xE6, "INC", "$xx" 	}, { 0xE7, "SBC", "[$xx]" 	},
		{ 0xE8, "INX", "" 		}, { 0xE9, "SBC", "#" 		},
		{ 0xEA, "NOP", "" 		}, { 0xEB, "XBA", "" 		},
		{ 0xEC, "CPX", "$xxxx" 	}, { 0xED, "SBC", "$xxxx" 		},
		{ 0xEE, "INC", "$xxxx" 	}, { 0xEF, "SBC", "$xxxxxx" 		},
		{ 0xF0, "BEQ", "$xx" 	}, { 0xF1, "SBC", "($xx),y" 	},
		{ 0xF2, "SBC", "($xx)" 	}, { 0xF3, "SBC", "($xx,s),y"},
		{ 0xF4, "PEA", "$xxxx" 	}, { 0xF5, "SBC", "$xx,x" 	},
		{ 0xF6, "INC", "$xx,x" 	}, { 0xF7, "SBC", "[$xx],y" 	},
		{ 0xF8, "SED", "" 		}, { 0xF9, "SBC", "$xxxx,y" 	},
		{ 0xFA, "PLX", "" 		}, { 0xFB, "XCE", "" 		},
		{ 0xFC, "JSR", "($xxxx,x)" }, { 0xFD, "SBC", "$xxxx,x" 	},
		{ 0xFE, "INC", "$xxxx,x" 	}, { 0xFF, "SBC", "$xxxxxx,x" 	}}
}