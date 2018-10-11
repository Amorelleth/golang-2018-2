package main

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

var idx int = 0
var stack = map[int]int{}

func pop() int {
	idx--
	return stack[idx]
}

func push(val int) {
	defer func() {
		idx++
	}()

	stack[idx] = val
}

func clear() {
	stack = map[int]int{}
	idx = 0
}

func Calculate(line string) (res int, err error) {
	lineSplitted := strings.Split(line, " ")

	for _, val := range lineSplitted {
		switch val {
		case "+":
			push(pop() + pop())
		case "-":
			push(-pop() + pop())
		case "*":
			push(pop() * pop())
		default:
			strNumToInt, convertionErr := strconv.Atoi(val)

			if convertionErr == nil {
				push(strNumToInt)
			}
		}
		if (idx - 1) < 0 {
			err = fmt.Errorf("incorrect input")
			return
		}
	}

	if err == nil {
		res = pop()
	}

	clear()

	return
}

func main() {
	in := bufio.NewScanner(os.Stdin)

	for in.Scan() {
		line := in.Text()
		res, err := Calculate(line)

		fmt.Println(res, err)
	}
}
