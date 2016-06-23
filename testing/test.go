package main

import "fmt"

type point struct {
	x int
	y int
}

func main() {
	m := make(map[point]point, 0, 1)
	m[point{1, 1}] = point{2,2}
	m[point{2, 2}] = point{3, 3}
	m[point{3, 3}] = point {4, 4}
	m[point{3, 3}] = point{5, 5}
	go func() {m[point{3, 3}] = point{0, 0}; fmt.Println(m[point{3, 3}])}()
	fmt.Println("\n\nMain Thread: ", m[point{3, 3}], m[point{2, 2}], "\n\n")
	for {}
}