package main

import "fmt"

func main() {
	m := make(map[int]int, 0, 1)
	m[1] = 1
	m[1] = m[1] + 1
	fmt.Print(m[1])

	for n, _ := range m {
		fmt.Println(n)
	}
}