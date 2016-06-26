package main

import "fmt"
import "time"
import "sync"

type point struct {
	x int
	y int
}

const (
	cols = 100
	rows = 100
)

var mtx sync.Mutex

var c chan int

func populate_map_sync(m map[point]string) {
	for i := 0; i < rows; i++ {
		go func (idx int) { 
			for j := 0; j < cols; j++ {
				val := fmt.Sprintf("{%v, %v}", idx, j)
				key := point{idx, j}
				mtx.Lock()
				m[key] = val
				mtx.Unlock()
			}
			c <- 0
		}(i)
	}
}

func populate_map(m map[point]string) {
	for i := 0; i < rows; i++ {
		go func (idx int) { 
			for j := 0; j < cols; j++ {
				val := fmt.Sprintf("{%v, %v}", idx, j)
				key := point{idx, j}
				m[key] = val
			}
			c <- 0
		}(i)
	}
}


func main() {
	m := make(map[point]string, 0, 1)
	n := make(map[point]string)
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

	// for i := 0; i < rows; i++ {
	// 	for j := 0; j < cols; j++ {
	// 		val := fmt.Sprintf("{%v, %v}", i, j)
	// 		key := point{i, j}
	// 		retval := m[key]
	// 		if retval != val {
	// 			fmt.Printf("Key: %v does not match value %v;Got %v\n", key, val, retval)
	// 		}
	// 	}
	// }

	// go func() {m[point{3, 3}] = point{0, 0}; fmt.Println(m[point{3, 3}])}()
	fmt.Println("\n\nMain Thread: ", m[point{1, 1}], n[point{1, 1}], "\n\n")
}