package main

import "sync"
import "fmt"

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

type T struct {
	iter uint64
}

func interlockedFnc(val *T, pres bool) {
	fmt.Printf("%v", val.iter)
	val.iter = 1
	fmt.Printf("%v", val.iter)
}

func main() {
	m := make(map[int]T, 1000, 1)
	key := 1
	sync.Interlocked(m, key)
	v, pres := m[key]
	fmt.Printf("Val: %v,Present: %v\n", v, pres)
	m[key] = T{1}
	v, pres = m[key]
	fmt.Printf("Val: %v,Present: %v\n", v, pres)
	delete(m, key)
	v, pres = m[key]
	fmt.Printf("Val: %v,Present: %v\n", v, pres)
	sync.Release(m)
	v, pres = m[key]
	fmt.Printf("Val: %v,Present: %v\n", v, pres)
	for i := 0; i < 1000; i++ {
		m[i] = T{}
	}

	nGoroutines := 100
	fmt.Println("Finished adding elements...")

	var start, wg sync.WaitGroup
	wg.Add(nGoroutines)
	start.Add(1)

	for i := 0; i < nGoroutines; i++ {
		go func() {
			iterations := 0
			start.Wait()
			// for k, v := range m {
			// 	v.iter++
			// 	t := m[k]
			// 	t.iter = t.iter + 1
			// 	m[k] = t
			// 	iterations++
			// }
			for i := 0; i < 1000; i++ {
				sync.Interlocked(m, i)
				t := m[i]
				t.iter = t.iter + 1
				m[i] = t
				sync.Release(m)
				iterations++
			}
			fmt.Printf("Iterations: %v\n", iterations)
			wg.Done()
		}()
	}

	checkMap := make(map[int]bool)
	for i := 0; i < 1000; i++ {
		checkMap[i] = false
	}

	start.Done()
	wg.Wait()
	fmt.Println("Finished processing elements...")
	err := false
	for k, _ := range m {
		if checkMap[k] {
			panic(fmt.Sprintf("Key %v already marked off!", k))
		}
		checkMap[k] = true
		t := m[k]
		if t.iter != uint64(nGoroutines) {
			fmt.Printf("Key: %v;Expected: %v; Received %v\n", k, nGoroutines, t.iter)
			err = true
		}
		delete(m, k)
	}

	fmt.Println("Finished checking elements.")

	for k, v := range checkMap {
		if !v {
			fmt.Printf("Missed Key: %v\n", k)
			panic(fmt.Sprintf("CheckMap Not All True"))
		}
	}

	fmt.Printf("Error: %v\n", err)

	// if (rows * cols) > 1000000 {
	// 	panic("Only a million elements may be added to the list to avoid resource exhaustion!")
	// }

	// m := make(map[point]point, 0, 1)
	// n := make(map[point]point)
	// c = make(chan int)

	// start := time.Now()
	// populate_map_sync(n)
	// for i := 0; i < rows; i++ {
	// 	<-c
	// }
	// elapsed := time.Since(start)
	// fmt.Println("Normal Map Time: ", elapsed)
	// n = nil
	// runtime.GC()

	// start = time.Now()
	// populate_map(m)
	// for i := 0; i < rows; i++ {
	// 	<-c
	// }
	// elapsed = time.Since(start)
	// fmt.Println("Concurrent Map Time: ", elapsed)

	// badVal := false
	// for i := 0; i < rows; i++ {
	// 	for j := 0; j < cols; j++ {
	// 		key := point{i, j}
	// 		val := point{rows - i, cols - j}
	// 		retval := m[key]
	// 		if retval != val {
	// 			badVal = true
	// 			fmt.Printf("Key: %v does not match value %v;Got %v\n", key, val, retval)
	// 		}
	// 	}
	// }

	// if !badVal {
	// 	fmt.Print("Concurrent Map successfully obtained key-value pairs!")
	// }

	// iterations := 0
	// for k, v := range m {
	// 	expected := point{rows - k.x, cols - k.y}
	// 	if v != expected {
	// 		panic(fmt.Sprintf("[Concurrent Map] Expected %v for key %v, but received %v", expected, k, v))
	// 	}
	// 	iterations++
	// }

	// if iterations != (rows * cols) {
	// 	fmt.Printf("Expected %v iterations, but only iterated %v times!", rows*cols, iterations)
	// }
}
