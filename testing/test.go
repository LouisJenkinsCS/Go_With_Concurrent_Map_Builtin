package main

import "fmt"

func main() {
	m := make(map[int]int, 0, 1)
	m[1] = 1
	m[2] = 2
	m[2]++
	m[3] = m[1] + m[2]
	fmt.Print(m[1])
}