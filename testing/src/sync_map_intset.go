package main

import (
    "math/rand"
    "sync"
    "time"
    "fmt"
)

func sync_genIntSet(cmap map[int]int, mtx *sync.Mutex, wg *sync.WaitGroup) {
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
                mtx.Lock()
                a := cmap[randNum]
                mtx.Unlock()
                a++

            // [LOOKUP_RATIO, INSERT_RATIO); Do Insert
            case randRatio < (LOOKUP_RATIO + insertRatio):
                mtx.Lock()
                cmap[randNum] = 1
                mtx.Unlock()

            // [INSERT_RATIO, 1]; Do removal
            default:
                mtx.Lock()
                delete(cmap, randNum)
                mtx.Unlock()
        }
    }
    wg.Done()
}

func sync_runTest() {
    cmap := make(map[int]int, 0, 1)
    var wg sync.WaitGroup
    var mtx sync.Mutex
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
        go sync_genIntSet(cmap, &mtx, &wg)
    }
    wg.Wait()
    time := time.Since(start)

    fmt.Printf("SynchronizedMap Time: %v\n", time)
}

