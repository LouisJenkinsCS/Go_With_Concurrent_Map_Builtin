package main

import "fmt"
import "time"
import "sync"

type point struct {
	x int
	y int
}

const (
	cols = 1000
	rows = 1000
)

var mtx sync.Mutex

var c chan int

func populate_map_sync(m map[string]point) {
	for i := 0; i < rows; i++ {
		go func (idx int) { 
			for j := 0; j < cols; j++ {
				key := fmt.Sprintf("%v, %v", idx, j)
				val := point{idx, j}
				mtx.Lock()
				m[key] = val
				mtx.Unlock()
			}
			c <- 0
		}(i)
	}
}

func populate_map(m map[string]point) {
	for i := 0; i < rows; i++ {
		go func (idx int) { 
			for j := 0; j < cols; j++ {
				key := fmt.Sprintf("%v, %v", idx, j)
				val := point{idx, j}
				m[key] = val
			}
			c <- 0
		}(i)
	}
}


func main() {
	m := make(map[string]point, 0, 1)
	n := make(map[string]point)
	c = make(chan int)

	start := time.Now()
	populate_map_sync(n)
	for i := 0; i < rows; i++ {
		<- c
	}
	elapsed := time.Since(start)
	fmt.Println("Normal Map Time: ", elapsed)

	start = time.Now();
	populate_map(m)
	for i := 0; i < rows; i++ {
		<- c
	}
	elapsed = time.Since(start)
	fmt.Println("Concurrent Map Time: ", elapsed)

	// go func() {m[point{3, 3}] = point{0, 0}; fmt.Println(m[point{3, 3}])}()
	fmt.Println("\n\nMain Thread: ", m["3, 3"], n["3, 3"], "\n\n")
}