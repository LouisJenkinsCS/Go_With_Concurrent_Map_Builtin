package main

import "./map_testing"
import "fmt"

const (
    NUM_GOROUTINES = 16
    OPS_PER_GOROUTINE = 1000
    LOOKUP_RATIO = .5
    GEN_RANGE = 1000
    GEN_SEED = 0x1BAD5EED
)

func stressTest() {
    fmt.Printf("Goroutines: %v\nElements per Goroutine: %v\nTotal Elements: %v\nTrials: %v\n\n",
        map_testing.ROWS, map_testing.COLS, map_testing.ROWS * map_testing.COLS, map_testing.TESTS)
    
    map_testing.TestDefaultMap()
    map_testing.TestConcurrentMap()
    map_testing.TestRWLockMap()
}

func intsetTest() {
    fmt.Printf("Goroutines: %v\nElements per Goroutine: %v\nTotal Operations: %v\nRatio: %v\nRange: %v\nSeed: %X\n\n", 
        NUM_GOROUTINES, OPS_PER_GOROUTINE, NUM_GOROUTINES * OPS_PER_GOROUTINE, LOOKUP_RATIO, GEN_RANGE, GEN_SEED)
    
    // sync_runTest()
    // rw_runTest()
    cmap_runTest()
}

func main() {
    cmap_runTest_iter()
    intsetTest()
}