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
				} else {
					switch spl[0] {
						case "BLT":
							lines[i] = "BCC " + strings.Join(spl[1:], " ")
						case "BGE":
							lines[i] = "BCS " + strings.Join(spl[1:], " ")
						case "DEA":
							lines[i] = "DEC"
						case "INA":
							lines[i] = "INC"
						case "JML":
							lines[i] = "JMP " + strings.Join(spl[1:], " ")
						case "JSL":
							lines[i] = "JSR " + strings.Join(spl[1:], " ")
					}
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
		pieces := exstrings.Split(line, " ,\t")

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

			if name == "JSR" || name == "BRL" {
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
	if spl[0] == "JSR" || spl[0] == "BRL" {
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
