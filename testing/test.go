package main

import "fmt"
import "time"
import "sync"
import "runtime"

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

func populate_map_sync(m map[point]point) {
	for i := 0; i < rows; i++ {
		go func(idx int) {
			for j := 0; j < cols; j++ {
				// val := fmt.Sprintf("{%v, %v}", idx, j)
				key, val := point{idx, j}, point{rows - idx, cols - j}
				mtx.Lock()
				m[key] = val
				mtx.Unlock()
			}
			c <- 0
		}(i)
	}
}

func populate_map(m map[point]point) {
	for i := 0; i < rows; i++ {
		go func(idx int) {
			for j := 0; j < cols; j++ {
				// val := fmt.Sprintf("{%v, %v}", idx, j)
				key, val := point{idx, j}, point{rows - idx, cols - j}
				m[key] = val
			}
			c <- 0
		}(i)
	}
}

func main() {
	if (rows * cols) > 1000000 {
		panic("Only a million elements may be added to the list to avoid resource exhaustion!")
	}

	m := make(map[point]point, 0, 1)
	n := make(map[point]point)
	c = make(chan int)

	start := time.Now()
	populate_map_sync(n)
	for i := 0; i < rows; i++ {
		<-c
	}
	elapsed := time.Since(start)
	fmt.Println("Normal Map Time: ", elapsed)
	n = nil
	runtime.GC()

	start = time.Now()
	populate_map(m)
	for i := 0; i < rows; i++ {
		<-c
	}
	elapsed = time.Since(start)
	fmt.Println("Concurrent Map Time: ", elapsed)

	badVal := false
	for i := 0; i < rows; i++ {
		for j := 0; j < cols; j++ {
			key := point{i, j}
			val := point{rows - i, cols - j}
			retval := m[key]
			if retval != val {
				badVal = true
				fmt.Printf("Key: %v does not match value %v;Got %v\n", key, val, retval)
			}
		}
	}

	if !badVal {
		fmt.Print("Concurrent Map successfully obtained key-value pairs!")
	}

	iterations := 0
	for k, v := range m {
		expected := point{rows - k.x, cols - k.y}
		if v != expected {
			panic(fmt.Sprintf("[Concurrent Map] Expected %v for key %v, but received %v", expected, k, v))
		}
		iterations++
	}

	if iterations != (rows * cols) {
		fmt.Printf("Expected %v iterations, but only iterated %v times!", rows*cols, iterations)
	}
}
