package main

import (
	"disassembler"
	"flag"
	"fmt"
	"os"
	"strconv"
)

func main() {
	f_rommode := flag.String("r", "", "Specifies which rommode to use (lo/hi).")
	f_amode := flag.Bool("a", false, "If true, a flag will start with 16-bit.")
	f_xmode := flag.Bool("x", false, "If true, x flag will start with 16-bit.")
	f_mode := flag.Bool("m", false, "If true, a and x flags will start with 16-bits.")
	s_start := flag.String("s", "0", "Offset to start disassembling from.")
	s_length := flag.String("l", "-1", "Maximum size to disassemble.")
	file := flag.String("f", "", "The file to read from.")
	outfile := flag.String("o", "", "File to store the disassembly info to.")
	prefix := flag.Bool("p", false, "Remove addressing prefix in output.")

	flag.Parse()

	if *file == "" {
		exitError(fmt.Errorf("Invalid file name. Must specify file with -f switch."))
	}

	if *outfile == "" {
		exitError(fmt.Errorf("Invalid output file name. Must specify file with -o switch."))
	}

	var mode disassembler.Mode
	var rmode int
	var start, length int64
	var err error

	if len(*s_start) > 1 && (*s_start)[0:1] == "$" {
		start, err = strconv.ParseInt((*s_start)[1:], 16, 64)
	} else if len(*s_start) > 2 && (*s_start)[0:2] == "0x" {
		start, err = strconv.ParseInt((*s_start)[2:], 16, 64)
	} else {
		start, err = strconv.ParseInt(*s_start, 10, 64)
	}
	if err != nil {
		exitError(err)
	}

	if len(*s_length) > 1 && (*s_length)[0:1] == "$" {
		length, err = strconv.ParseInt((*s_length)[1:], 16, 64)
	} else if len(*s_length) > 2 && (*s_length)[0:2] == "0x" {
		length, err = strconv.ParseInt((*s_length)[2:], 16, 64)
	} else {
		length, err = strconv.ParseInt((*s_length), 10, 64)
	}
	if err != nil {
		panic(err)
	}

	if *f_amode || *f_mode {
		mode.A = true
	}
	if *f_xmode || *f_mode {
		mode.X = true
	}

	if len(*f_rommode) == 2 {
		if *f_rommode == "lo" {
			rmode = disassembler.LoROM
		} else if *f_rommode == "hi" {
			rmode = disassembler.HiROM
		} else {
			rmode = disassembler.None
		}
	} else if len(*f_rommode) > 2 && (*f_rommode)[0:2] == "lo" {
		rmode = disassembler.LoROM
	} else if len(*f_rommode) > 2 && (*f_rommode)[0:2] == "hi" {
		rmode = disassembler.HiROM
	} else {
		rmode = disassembler.None
	}


	disassembler.Parse(*file, *outfile, mode, int(start), int(length), rmode, !*prefix)
}

func exitError(err error) {
	fmt.Println(err)
	os.Exit(1)
}
