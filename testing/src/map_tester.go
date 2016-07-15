package main

import "./map_testing"
import "fmt"

func main() {
    fmt.Printf("Goroutines: %v\nElements per Goroutine: %v\nTotal Elements: %v\nTrials: %v\n\n",
        map_testing.ROWS, map_testing.COLS, map_testing.ROWS * map_testing.COLS, map_testing.TESTS)
    
    m := make(map[int]int, 0, 1)
    key := 0
    sync.Interlocked m {
        if key == 0 {
            m[5] = 1
        }
        delete(m, key)
        m[key]++
        m[1] = m[2] + m[3] + 1
    }
    // map_testing.TestDefaultMap()
    // map_testing.TestConcurrentMap()
    // map_testing.TestRWLockMap()
}