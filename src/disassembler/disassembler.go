package disassembler

import (
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
	Opcode byte
	Name   string
	Params string
}

type Mode struct {
    A 	bool
    X 	bool
}

var None 	int = 0
var LoROM 	int = 1
var HiROM 	int = 2

func Parse(file, outfile string, mode Mode, s, l, r int, prefixing bool) {
	Instructions, err := LoadInstructionSet("65c816.dat")
	if err != nil {
		exitError(err)
	}

	f, err := os.Open(file)
	if err != nil {
		exitError(err)
	}
	defer f.Close();

	ROMInfo, err := f.Stat()
	if err != nil {
		exitError(err)
	}
	filesize := ROMInfo.Size()
	var header bool

	if filesize & 0x7FFF == 0x200 {
		header = true
	} 

	offset := convertSNESAddress(s, r, header)
	f.Seek(int64(offset), 0)

	fr, err := ioutil.ReadFile(file);
	if err != nil {
		exitError(err)
	}
	frindex := offset
	address := 0
	testindex := 0
	testoutlen := 4096
	testout := make([]string, 4096)

	for ; l < 0 || address < l; {
		out := ""
		b, err := getNextByte(fr, &frindex)
		if err != nil {
			break
		}

		n, p, err := searchOpcode(Instructions, b)
		if err != nil {
			exitError(err)
		}

		if n == "SEP" || n == "REP" {
			b, err := getNextByte(fr, &frindex)
			if err != nil {
				break
			}
			//	The following evaluate to true if the command is REP or false otherwise
			if b & 0x10 == 0x10 {
				mode.X = n == "REP"
			}
			if b & 0x20 == 0x20 {
				mode.A 	= n == "REP"
			}

			out = "#$" + exstrings.Bytetostring(b)
		} else if n == "BRK" {
			b, err := getNextByte(fr, &frindex)
			if err != nil {
				break
			}
			out = "#$" + exstrings.Bytetostring(b)

			if b == 0 {
				count := 0
				for cb := byte(0); cb == 0; count++ {
					cb = fr[frindex + count]
				}

				if count > 1 {
					if count % 2 == 1 {
						count--
					}

					out += strings.Repeat("\nBRK #$00", count/2)
					frindex += count
				}
			}
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
					cb, err := getNextByte(fr, &frindex)
					if err != nil {
						break
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
			num, err := strconv.ParseInt(exstrings.Remove(out, "$, "), 16, 32)
			if err != nil {
				exitError(err)
			}

			if num > 127 {
				num -= 256	
			}

			out = ".LABEL_" + strconv.FormatInt(num + int64(address), 16)
		}
		if testindex == testoutlen {
			testout = Append(testout, n + " " + out)
			testoutlen *= 2
		} else {
			testout[testindex] = n + " " + out
		}
		testindex++
		//fmt.Println(testout[testindex - 1])
	}
	//spl := strings.Split(output, "\n")
	labels := make([]int64, 1024)
	index := 0
	for _, v := range testout {
		line := strings.Split(v, " ")
		if isBranch(line[0]) {
			//labels[index], _ = strconv.ParseInt(strings.Replace(line[1], ".LABEL_", "", 1), 16, 16)
			temp, _ := strconv.ParseInt(strings.Replace(line[1], ".LABEL_", "", 1), 16, 16)
			
			labels = Appendint64(labels, temp)
			//fmt.Println("Added", labels[index], temp)
			index++
		}
	}

	of, err := os.Create(outfile)
	if err != nil {
		exitError(err)
	}
	defer of.Close()

	//fmt.Println("Starting to write")
	address = 0
	prefix := ".PC_"
	currentaddress := ""

	if r == LoROM {
		prefix = ".Lo_"
	} else if r == HiROM {
		prefix = ".Hi_"
	}

	for i, v := range testout {
		if i >= testindex || (address >= l && l >= 0) {
			break
		}
		//fmt.Println("Address:", address, v)
		if containsValue(labels, int64(address)) {
			//output += ".LABEL_" + strconv.FormatInt(int64(address), 16) + ":\n"
			_, err := of.WriteString("\t\t" + ".LABEL_" + strconv.FormatInt(int64(address), 16) + "\n")
			if err != nil {
				exitError(err)
			}
		}
		if prefixing {
			currentaddress = strconv.FormatInt(int64(convertPCAddress(offset + address, r, header)), 16)
			for ; len(currentaddress) < 6 ; {
				currentaddress = "0" + currentaddress
			}
			if strings.Count(v, "\n") > 1 {		
				for _, v2 := range strings.Split(v, "\n") {
					if (address >= l && l >= 0) {
						break
					}
					if strings.TrimSpace(v2) != "" {
						_, err := of.WriteString(prefix + currentaddress + "\t" + v2 + "\n")
						if err != nil {
							exitError(err)
						}
					}
					address += countBytes(v2)

					currentaddress = strconv.FormatInt(int64(convertPCAddress(offset + address, r, header)), 16)
					for ; len(currentaddress) < 6 ; {
						currentaddress = "0" + currentaddress
					}
				}
			} else {
				if strings.TrimSpace(v) != "" {
					_, err := of.WriteString(prefix + currentaddress + "\t" + v + "\n")
					if err != nil {
						exitError(err)
					}
				}
				address += countBytes(v)
			}
		} else {
			if strings.Count(v, "\n") > 1 {		
				for _, v2 := range strings.Split(v, "\n") {
					if (address >= l && l >= 0) {
						break
					}
					if strings.TrimSpace(v2) != "" {
						_, err := of.WriteString(v2 + "\n")
						if err != nil {
							exitError(err)
						}
					}
					address += countBytes(v2)
				}
			} else {
				if strings.TrimSpace(v) != "" {
					_, err := of.WriteString(v + "\n")
					if err != nil {
						exitError(err)
					}
				}
				address += countBytes(v)
			}
		}
	}
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
	} else if len(spl) == 2 && isBranch(spl[0]) && len(spl[1]) > 5 && spl[1][0:6] == ".LABEL" {
		if spl[0] == "BRL" {
			return 3
		} else {
			return 2
		}
	} else if spl[0] == "BRK" && len(spl) > 2 {
		return strings.Count(str, "BRK") * 2
	}
	regnum, _ := regexp.Compile("[0-9a-fA-F]{2}")
	str = regnum.ReplaceAllString(strings.Join(exstrings.Split(str, " ,")[1:]," "), "x")
	return 1 + strings.Count(str, "x")
}

func getMode(m Mode, str string) (i int) {
	if 	str == "ADC" || str == "ORA" || str == "LDA" || str == "BIT" ||
		str == "CMP" || str == "SBC" || str == "AND" || str == "EOR" {
		if m.A {
			i = 2
		} else {
			i = 1
		}
	} else {
		if m.X {
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

func exitError(err error) {
	fmt.Println(err)
	os.Exit(1)
}

//	Append taken from http://golang.org/doc/effective_go.html#slices
func Append(slice []string, data string) []string {
    l := len(slice)
    if l == cap(slice) {  // reallocate
        // Allocate double what's needed, for future growth.
        newSlice := make([]string, l * 2)
        // The copy function is predeclared and works for any slice type.
        copy(newSlice, slice)
        slice = newSlice
    }
    slice = slice[0:l * 2]
    slice[l] = data
    return slice
}
//	Append taken from http://golang.org/doc/effective_go.html#slices
func Appendint64(slice []int64, data int64) []int64 {
    l := len(slice)
    if l == cap(slice) {  // reallocate
        // Allocate double what's needed, for future growth.
        newSlice := make([]int64, l * 2)
        // The copy function is predeclared and works for any slice type.
        copy(newSlice, slice)
        slice = newSlice
    }
    slice = slice[0:l+1]
    slice[l] = data
    return slice
}

func getNextByte(f []byte, index *int) (byte, error) {
	if *index >= len(f) {
		return 0, fmt.Errorf("Exceeded array length.")
	}
	*index++
	return f[*index-1], nil
}

func convertSNESAddress(addr, mode int, header bool) int {
    if addr < 0 || addr >= 0xFFFFFF {
    	return -1
    }

    if mode == LoROM {
        addr = ((addr & 0x7F0000) >> 1) + (addr & 0x7FFF)
    } else if mode == HiROM {
        addr = addr & 0x3FFFFF
    } else if mode == None {
        return addr
    }

    if header {
        addr += 0x200
    }

    return addr
}

func convertPCAddress(addr, mode int, header bool) int {
    if addr < 0 || addr > 0xFFFFFF || mode == None {
        return addr;	
    }

    if header {
        addr -= 0x200;
    }

    if mode == LoROM {
        addr = ((addr & 0x7F8000) << 1 ) + 0x8000 + (addr & 0x7FFF);
    } else if mode == HiROM {
        addr = 0xC00000 + (addr & 0x3fffff);
    } 

    return addr;
}
