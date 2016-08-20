package combined_testing

import (
    "time"
    "math/rand"
    "settings"
    "sync"
    "sync/atomic"
)

func ConcurrentCombined(nGoroutines int) int64 {
    cmap := make(map[int64]settings.Unused, settings.COMBINED_KEY_RANGE, nGoroutines)
    
    // Fill map to reduce overhead of resizing
    for i := int64(0); i < settings.COMBINED_KEY_RANGE; i++ {
        cmap[i] = settings.UNUSED
    }
    for i := int64(0); i < settings.COMBINED_KEY_RANGE; i++ {
        delete(cmap, i)
    }

    var totalOps int64
    return settings.ParallelTest(nGoroutines, func() {
        var nOps int64
        rng := rand.New(rand.NewSource(time.Now().UTC().UnixNano()))

        for i := int64(0); i < settings.COMBINED_OPS_PER_GOROUTINE; i++ {
            randRatio := rng.Float64()
            randNum := rng.Int63n(settings.COMBINED_KEY_RANGE)
            switch {
                // 0 <= randRatio < .25 -> Insert
                case randRatio < settings.COMBINED_FAIR_RATIO:
                    cmap[randNum] = settings.UNUSED
                    nOps++
                // .25 <= randRatio < .5 -> Delete
                case randRatio < 2 * settings.COMBINED_FAIR_RATIO:
                    delete(cmap, randNum)
                    nOps++
                // .5 <= randRatio < .75 -> Lookup
                case randRatio < 3 * settings.COMBINED_FAIR_RATIO:
                    tmp := cmap[randNum]
                    tmp++
                    nOps++
                // .75 <= randRatio < 1 -> Iterate
                default:
                    // Each iteration counts as an operation (as it calls mapiternext)
                    for k, v := range cmap {
                        k++
                        v++
                        nOps++
                    }
            }
        }
        
        // Add our number of operations to the total number of Operations
        atomic.AddInt64(&totalOps, nOps)
    }).Nanoseconds() / totalOps
}

func ConcurrentCombined_Interlocked(nGoroutines int) int64 {
    cmap := make(map[int64]settings.Unused, settings.COMBINED_KEY_RANGE, nGoroutines)
    
    // Fill map to reduce overhead of resizing
    for i := int64(0); i < settings.COMBINED_KEY_RANGE; i++ {
        cmap[i] = settings.UNUSED
    }
    for i := int64(0); i < settings.COMBINED_KEY_RANGE; i++ {
        delete(cmap, i)
    }

    var totalOps int64
    return settings.ParallelTest(nGoroutines, func() {
        var nOps int64
        rng := rand.New(rand.NewSource(time.Now().UTC().UnixNano()))

        for i := int64(0); i < settings.COMBINED_OPS_PER_GOROUTINE; i++ {
            randRatio := rng.Float64()
            randNum := rng.Int63n(settings.COMBINED_KEY_RANGE)
            switch {
                // 0 <= randRatio < .25 -> Insert
                case randRatio < settings.COMBINED_FAIR_RATIO:
                    cmap[randNum] = settings.UNUSED
                    nOps++
                // .25 <= randRatio < .5 -> Delete
                case randRatio < 2 * settings.COMBINED_FAIR_RATIO:
                    delete(cmap, randNum)
                    nOps++
                // .5 <= randRatio < .75 -> Lookup
                case randRatio < 3 * settings.COMBINED_FAIR_RATIO:
                    tmp := cmap[randNum]
                    tmp++
                    nOps++
                // .75 <= randRatio < 1 -> Iterate
                default:
                    for k, v := range sync.Interlocked cmap {
                        k++
                        v++
                        nOps++
                    }
            }
        }
        
        // Add our number of operations to the total number of Operations
        atomic.AddInt64(&totalOps, nOps)
    }).Nanoseconds() / totalOps
}

func SynchronizedCombined(nGoroutines int) int64 {
    smap := make(map[int64]settings.Unused, settings.COMBINED_KEY_RANGE)
    var mtx sync.Mutex
    
    // Fill map to reduce overhead of resizing
    for i := int64(0); i < settings.COMBINED_KEY_RANGE; i++ {
        smap[i] = settings.UNUSED
    }
    for i := int64(0); i < settings.COMBINED_KEY_RANGE; i++ {
        delete(smap, i)
    }

    var totalOps int64
    return settings.ParallelTest(nGoroutines, func() {
        var nOps int64
        rng := rand.New(rand.NewSource(time.Now().UTC().UnixNano()))

        for i := int64(0); i < settings.COMBINED_OPS_PER_GOROUTINE; i++ {
            randRatio := rng.Float64()
            randNum := rng.Int63n(settings.COMBINED_KEY_RANGE)
            switch {
                // 0 <= randRatio < .25 -> Insert
                case randRatio < settings.COMBINED_FAIR_RATIO:
                    mtx.Lock()
                    smap[randNum] = settings.UNUSED
                    mtx.Unlock()
                    nOps++
                // .25 <= randRatio < .5 -> Delete
                case randRatio < 2 * settings.COMBINED_FAIR_RATIO:
                    mtx.Lock()
                    delete(smap, randNum)
                    mtx.Unlock()
                    nOps++
                // .5 <= randRatio < .75 -> Lookup
                case randRatio < 3 * settings.COMBINED_FAIR_RATIO:
                    mtx.Lock()
                    tmp := smap[randNum]
                    mtx.Unlock()
                    tmp++
                    nOps++
                // .75 <= randRatio < 1 -> Iterate
                default:
                    mtx.Lock()
                    for k, v := range smap {
                        k++
                        v++
                        nOps++
                    }
                    mtx.Unlock()
            }
        }
        
        // Add our number of operations to the total number of Operations
        atomic.AddInt64(&totalOps, nOps)
    }).Nanoseconds() / totalOps
}

func ReaderWriterCombined(nGoroutines int) int64 {
    rwmap := make(map[int64]settings.Unused, settings.COMBINED_KEY_RANGE)
    var mtx sync.RWMutex
    
    // Fill map to reduce overhead of resizing
    for i := int64(0); i < settings.COMBINED_KEY_RANGE; i++ {
        rwmap[i] = settings.UNUSED
    }
    for i := int64(0); i < settings.COMBINED_KEY_RANGE; i++ {
        delete(rwmap, i)
    }

    var totalOps int64
    return settings.ParallelTest(nGoroutines, func() {
        var nOps int64
        rng := rand.New(rand.NewSource(time.Now().UTC().UnixNano()))

        for i := int64(0); i < settings.COMBINED_OPS_PER_GOROUTINE; i++ {
            randRatio := rng.Float64()
            randNum := rng.Int63n(settings.COMBINED_KEY_RANGE)
            switch {
                // 0 <= randRatio < .25 -> Insert
                case randRatio < settings.COMBINED_FAIR_RATIO:
                    mtx.Lock()
                    rwmap[randNum] = settings.UNUSED
                    mtx.Unlock()
                    nOps++
                // .25 <= randRatio < .5 -> Delete
                case randRatio < 2 * settings.COMBINED_FAIR_RATIO:
                    mtx.Lock()
                    delete(rwmap, randNum)
                    mtx.Unlock()
                    nOps++
                // .5 <= randRatio < .75 -> Lookup
                case randRatio < 3 * settings.COMBINED_FAIR_RATIO:
                    mtx.RLock()
                    tmp := rwmap[randNum]
                    mtx.RUnlock()
                    tmp++
                    nOps++
                // .75 <= randRatio < 1 -> Iterate
                default:
                    mtx.RLock()
                    for k, v := range rwmap {
                        k++
                        v++
                        nOps++
                    }
                    mtx.RUnlock()
            }
        }
        
        // Add our number of operations to the total number of Operations
        atomic.AddInt64(&totalOps, nOps)
    }).Nanoseconds() / totalOps
}