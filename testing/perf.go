package main

import "time"
import "fmt"
import "sync"

func main() {
	m := make(map[int]int, 1000000, 1)
	wg := sync.WaitGroup{}
	wg.Add(8)
	mtx := sync.Mutex{}
	for i := 0; i < 1000000; i++ {
		m[i] = i
	}

	nsPerOp := int64(0)
	fmt.Println("Map Assignment")
	start := time.Now()
	operations := int64(0)

	for i := int64(0); i < int64(100); i++ {
		for k := 0; k < 8; k++ {
			go func() {
				for j := 0; j < 10; j++ {
					mtx.Lock()
					for k, v := range m {
						m[k] = v + 1
						operations++
					}
					mtx.Unlock()
				}
				wg.Done()
			}()
		}
	}
	wg.Wait()
	nsPerOp += time.Since(start).Nanoseconds() / operations
	fmt.Printf("\tns/op: %v", nsPerOp)
}
