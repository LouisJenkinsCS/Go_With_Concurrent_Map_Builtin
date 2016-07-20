package main

import "./map_testing"
import "fmt"
// import "sync"

func main() {
    fmt.Printf("Goroutines: %v\nElements per Goroutine: %v\nTotal Elements: %v\nTrials: %v\n\n",
        map_testing.ROWS, map_testing.COLS, map_testing.ROWS * map_testing.COLS, map_testing.TESTS)
    
    // m := make(map[int]int, 0, 1)
    // wg := sync.WaitGroup{}
    // wg.Add(32)
    // for i := 0; i < 32; i++ {
    //     go func() {
    //         for j := 0; j < 1000000; j++ {
    //             key := j % 100
    //             sync.Interlocked m[key] {
    //                 m[key]++
    //             }
    //         }
    //         wg.Done()
    //     }()
    // }
    // wg.Wait()
    // for i := 0; i < 100; i++ {
    //     fmt.Printf("m[%v]=%v\n", i, m[i])
    // }
    // fmt.Printf("Testing normal range iterator...\n")
    // for k, v := range m {
    //     fmt.Printf("Key: %v, Value: %v\n", k, v)
    // }
    // fmt.Printf("Testing sync.Interlocked range iterator...\n")
    // for k, v := range sync.Interlocked m {
    //     fmt.Printf("Key: %v, Value: %v\n", k, v)
    // }
    
    // map_testing.TestDefaultMap()
    map_testing.TestConcurrentMap()
    // map_testing.TestRWLockMap()
}