package main

import "fmt"

func main() {
	m := make(map[int]string, 0, 1)
	m[1] = "Hello "
	m[2] = "World"
	m[2] = "dlroW"
	fmt.Println(m[1], m[2])
}