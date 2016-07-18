package main

import "./map_testing"
import "fmt"
import "sync"

func main() {
    fmt.Printf("Goroutines: %v\nElements per Goroutine: %v\nTotal Elements: %v\nTrials: %v\n\n",
        map_testing.ROWS, map_testing.COLS, map_testing.ROWS * map_testing.COLS, map_testing.TESTS)
    
    m := make(map[int]int, 0, 1)
    m[0] = 0
    wg := sync.WaitGroup{}
    wg.Add(32)
    for i := 0; i < 32; i++ {
        go func() {
            for j := 0; j < 100000; j++ {
                sync.Interlocked m[0] {
                    m[0]++
                    // cmapassign(maptype, m, 0, n)
                }
            }
            wg.Done()
        }()
    }
    wg.Wait()
    fmt.Printf("%v", m[0])
    // map_testing.TestDefaultMap()
    // map_testing.TestConcurrentMap()
    // map_testing.TestRWLockMap()
}