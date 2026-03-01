package main

import (
	"errors"
	"fmt"
	"os"
)

type Machine struct {
	mem []byte
	ptr int
}

func (m *Machine) checkLength(n int) {
	l := len(m.mem)
	if n >= l {
		newM := make([]byte, 2*l)
		copy(newM, m.mem)
		m.mem = newM
	}
}

func preProcess(program []byte) (map[int]int, error) {
	res := map[int]int{}
	stack := []int{}
	for p, c := range program {
		switch c {
		case '[':
			stack = append(stack, p)
		case ']':
			l := len(stack) - 1
			if l < 0 {
				return res, errors.New("Pair did't match, too many ']'")
			}
			res[stack[l]] = p
			res[p] = stack[l]
			stack = stack[:l]
		}
	}
	if len(stack) != 0 {
		return res, errors.New("Pairs didn't match, too many '['")
	} else {
		return res, nil
	}
}

func process(program []byte, mapPair map[int]int) error {
	m:=Machine{
		mem: []byte{0},
		ptr: 0,
	}
	l := len(program)
	p := 0
	for true {
		if p >= l {
			break
		}
		c := program[p]

		switch c {
		case '+':
 			m.mem[m.ptr]++
			p++
		case '-':
			m.mem[m.ptr]--
			p++
		case '.':
			fmt.Printf("%c", m.mem[m.ptr])
			p++
		case ',':
			fmt.Scanf("%c\n", &m.mem[m.ptr])
			p++
		case '>':
			m.ptr++
			m.checkLength(m.ptr)
			p++
		case '<':
			if m.ptr == 0 {
				return errors.New("pointer out of range")
			}
			m.ptr--
			p++
		case '[':
			if m.mem[m.ptr] == 0 {
				p = mapPair[p] + 1
			} else {
				p++
			}
		case ']':
			if m.mem[m.ptr] == 0 {
				p++
			} else {
				p = mapPair[p] + 1
			}
		default: p++
		}
	}
	return nil
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("usage: brainfuck <file.bf>")
		return
	}
	filename := os.Args[1]
	program, err := os.ReadFile(filename)
	if err != nil {
		fmt.Println("Failed to read: ", err)
		return
	}
	mapPair, err := preProcess(program)
	if err != nil {
		fmt.Println("Failed to creat paris: ", err)
		return
	}
	err = process(program, mapPair)
	if err != nil {
		fmt.Println("Failed process: ", err)
		return
	}
}
