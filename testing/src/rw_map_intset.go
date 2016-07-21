package main

import (
    "math/rand"
    "sync"
    "time"
    "fmt"
)

func rw_genIntSet(cmap map[int]int, rw *sync.RWMutex, wg *sync.WaitGroup) {
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
                rw.RLock()
                a := cmap[randNum]
                rw.RUnlock()
                a++

            // [LOOKUP_RATIO, INSERT_RATIO); Do Insert
            case randRatio < (LOOKUP_RATIO + insertRatio):
                rw.Lock()
                cmap[randNum] = 1
                rw.Unlock()

            // [INSERT_RATIO, 1]; Do removal
            default:
                rw.Lock()
                delete(cmap, randNum)
                rw.Unlock()
        }
    }
    wg.Done()
}

func rw_runTest() {
    cmap := make(map[int]int, 0, 1)
    var wg sync.WaitGroup
    var rw sync.RWMutex
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
        go rw_genIntSet(cmap, &rw, &wg)
    }
    wg.Wait()
    time := time.Since(start)

    fmt.Printf("ReaderWrite Map Time: %v\n", time)
}

