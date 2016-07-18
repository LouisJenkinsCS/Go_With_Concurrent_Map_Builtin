package main

import "./map_testing"
import "fmt"

func main() {
    fmt.Printf("Goroutines: %v\nElements per Goroutine: %v\nTotal Elements: %v\nTrials: %v\n\n",
        map_testing.ROWS, map_testing.COLS, map_testing.ROWS * map_testing.COLS, map_testing.TESTS)
    
    salaries := make(map[int]int, 0, 1)
    // Imagine salaries already populated
    salaries[10] = 1
    sync.Interlocked num := salaries[10] {
        fmt.Printf("%v\n", num)
        num++
        fmt.Printf("%v\n", num)
    }

    // map_testing.TestDefaultMap()
    // map_testing.TestConcurrentMap()
    // map_testing.TestRWLockMap()
}