package main

import "./map_testing"
import "fmt"

func main() {
    m := make(map[int]int)
    sync.Interlocked m {
        fmt.Printf("Inside sync.Interlocked!\n")
        m[0] = 1
    }
    for k, v := range sync.Interlocked m {
        fmt.Printf("Inside of range sync.Interlocked!Key: %v, Value: %v\n", k, v)
    }

    fmt.Printf("Goroutines: %v\nElements per Goroutine: %v\nTotal Elements: %v\nTrials: %v\n\n",
        map_testing.ROWS, map_testing.COLS, map_testing.ROWS * map_testing.COLS, map_testing.TESTS)

    map_testing.TestDefaultMap()
    map_testing.TestConcurrentMap()
    map_testing.TestRWLockMap()
}