package main

import "sync"
import "sync/atomic"
import "math/rand"
import "time"
import "fmt"

type iterElement struct {
    value uint64
}


func cmap_iter(m map[uint64]*iterElement, start *sync.WaitGroup, done *sync.WaitGroup, counter *uint64) {
    rngSrc := rand.NewSource(time.Now().UTC().UnixNano())
    rng := rand.New(rngSrc)
    start.Wait()

    for i := 1; i <= OPS_PER_GOROUTINE; i++ {
        // 66% ratio of inserts
        if (rng.Intn(3) % i) != 0 {
            // Obtain pooled element (saves time)
            m[atomic.AddUint64(counter, 1)] = &iterElement{}
        } else {
            for k, _ := range m {
                sync.Interlocked m[k] {
                    m[k].value++
                }
            }
        }
    }
    done.Done()
}

func cmap_iter_interlocked(m map[uint64]*iterElement, start *sync.WaitGroup, done *sync.WaitGroup) {
    start.Wait()

    for i := 0; i < OPS_PER_GOROUTINE; i++ {
        for _, v := range sync.Interlocked m {
            v.value++
        }
    }

    done.Done()
}

func cmap_runTest_iter() {
    cmap := make(map[uint64]*iterElement, 0, 1)
    var done sync.WaitGroup
    var start sync.WaitGroup
    var counter uint64
    start.Add(1)
    done.Add(NUM_GOROUTINES)

    // Half Goroutines will use default range with periodic inserts, other half will use sync.Interlocked range to increment a field
    for i := 0; i < NUM_GOROUTINES / 2; i++ {
        go cmap_iter(cmap, &start, &done, &counter)
    }

    for i := NUM_GOROUTINES / 2; i < NUM_GOROUTINES; i++ {
        go cmap_iter_interlocked(cmap, &start, &done)
    }

    t := time.Now()
    start.Done()
    done.Wait()
    end := time.Since(t)

    fmt.Printf("Concurrent Iteration Time: %v\n", end)
}