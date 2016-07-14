package main

import "./map_testing"
import "fmt"

func main() {
    fmt.Printf("Goroutines: %v\nElements per Goroutine: %v\nTotal Elements: %v\nTrials: %v\n\n",
        map_testing.ROWS, map_testing.COLS, map_testing.ROWS * map_testing.COLS, map_testing.TESTS)
    
    m := make(map[int]int, 0, 1)
    sync.Interlocked m {
        delete(m, 0)
    }
    // map_testing.TestDefaultMap()
    // map_testing.TestConcurrentMap()
    // map_testing.TestRWLockMap()
}