package main

import (
    "math/rand"
    "sync"
    "time"
    "fmt"
)

func cmap_genIntSet(cmap map[int]int, wg *sync.WaitGroup) {
    // Goroutine specific RNG for deciding when to do lookup, insertion, or removal
    rngSrc := rand.NewSource(time.Now().UTC().UnixNano())
    rng := rand.New(rngSrc)
    leftOverRatio := (1 - LOOKUP_RATIO) / 2
    insertRatio := leftOverRatio / 2

    for i := 0; i < OPS_PER_GOROUTINE; i++ {
        randNum := rand.Intn(GEN_RANGE)
        randRatio := rng.Float64()
        
        switch {
            // [0, LOOKUP_RATIO); Do Lookup
            case randRatio < LOOKUP_RATIO:
                a := cmap[randNum]
                a++

            // [LOOKUP_RATIO, INSERT_RATIO); Do Insert
            case randRatio < (LOOKUP_RATIO + insertRatio):
                cmap[randNum] = 1

            // [INSERT_RATIO, 1]; Do removal
            default:
                delete(cmap, randNum)
        }
    }
    wg.Done()
}

func cmap_runTest() {
    cmap := make(map[int]int, 0, 1)
    var wg sync.WaitGroup
    wg.Add(NUM_GOROUTINES)
    rand.Seed(GEN_SEED)

    // Initialize map to half-full; Eliminates overhead of resizing/rehashing
    nElems := (int)((NUM_GOROUTINES * OPS_PER_GOROUTINE) * ((1 - LOOKUP_RATIO) / 2))
    for i := 0; i < nElems; i++ {
        cmap[i] = 0
    }

    // Delete all elements to ensure no false-positive insertions
    for i := 0; i < nElems; i++ {
        delete(cmap, i)
    }

    start := time.Now()
    // Start Goroutines
    for i := 0; i < NUM_GOROUTINES; i++ {
        go cmap_genIntSet(cmap, &wg)
    }
    wg.Wait()
    time := time.Since(start)

    fmt.Printf("ConcurrentMap Time: %v\n", time)
}

