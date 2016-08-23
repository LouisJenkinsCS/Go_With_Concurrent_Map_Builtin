package combined_testing

import (
	"math/rand"
	"settings"
	"sync"
	"sync/atomic"
	"time"
)

type T struct {
	_    uintptr
	iter uintptr
	_    uintptr
}

func ConcurrentCombinedSkim(nGoroutines int64) int64 {
	cmap := make(map[int64]T, settings.COMBINED_KEY_RANGE, nGoroutines)

	// Fill map to reduce overhead of resizing
	for i := int64(0); i < settings.COMBINED_KEY_RANGE; i++ {
		cmap[i] = T{}
	}
	for i := int64(0); i < settings.COMBINED_KEY_RANGE; i++ {
		delete(cmap, i)
	}

	var totalOps int64
	return settings.ParallelTest(int(nGoroutines), func() {
		var nOps int64
		rng := rand.New(rand.NewSource(time.Now().UTC().UnixNano()))

		for i := int64(0); i < settings.COMBINED_OPS_PER_GOROUTINE; i++ {
			randRatio := rng.Float64()
			randNum := rng.Int63n(settings.COMBINED_KEY_RANGE)
			switch {
			// 0 <= randRatio < .25 -> Insert
			case randRatio < settings.COMBINED_SKIM_NON_ITERATION_RATIO:
				cmap[randNum] = T{}
				nOps++
			// .25 <= randRatio < .5 -> Delete
			case randRatio < 2*settings.COMBINED_SKIM_NON_ITERATION_RATIO:
				delete(cmap, randNum)
				nOps++
			// .5 <= randRatio < .75 -> Lookup
			case randRatio < 3*settings.COMBINED_SKIM_NON_ITERATION_RATIO:
				tmp := cmap[randNum]
				tmp.iter++
				nOps++
			// .75 <= randRatio < 1 -> Iterate
			default:
				// Each iteration counts as an operation (as it calls mapiternext)
				for k, v := range cmap {
					v.iter++
					t := cmap[k]
					t.iter++
					cmap[k] = t
					nOps++
				}
			}
		}

		// Add our number of operations to the total number of Operations
		atomic.AddInt64(&totalOps, nOps)
	}).Nanoseconds() / totalOps
}

func SynchronizedCombinedSkim(nGoroutines int64) int64 {
	smap := make(map[int64]T, settings.COMBINED_KEY_RANGE)
	var mtx sync.Mutex

	// Fill map to reduce overhead of resizing
	for i := int64(0); i < settings.COMBINED_KEY_RANGE; i++ {
		smap[i] = T{}
	}
	for i := int64(0); i < settings.COMBINED_KEY_RANGE; i++ {
		delete(smap, i)
	}

	var totalOps int64
	return settings.ParallelTest(int(nGoroutines), func() {
		var nOps int64
		rng := rand.New(rand.NewSource(time.Now().UTC().UnixNano()))

		for i := int64(0); i < settings.COMBINED_OPS_PER_GOROUTINE; i++ {
			randRatio := rng.Float64()
			randNum := rng.Int63n(settings.COMBINED_KEY_RANGE)
			switch {
			// 0 <= randRatio < .25 -> Insert
			case randRatio < settings.COMBINED_SKIM_NON_ITERATION_RATIO:
				mtx.Lock()
				smap[randNum] = T{}
				mtx.Unlock()
				nOps++
			// .25 <= randRatio < .5 -> Delete
			case randRatio < 2*settings.COMBINED_SKIM_NON_ITERATION_RATIO:
				mtx.Lock()
				delete(smap, randNum)
				mtx.Unlock()
				nOps++
			// .5 <= randRatio < .75 -> Lookup
			case randRatio < 3*settings.COMBINED_SKIM_NON_ITERATION_RATIO:
				mtx.Lock()
				tmp := smap[randNum]
				mtx.Unlock()
				tmp.iter++
				nOps++
			// .75 <= randRatio < 1 -> Iterate
			default:
				mtx.Lock()
				for k, v := range smap {
					v.iter++
					t := smap[k]
					t.iter++
					smap[k] = t
					nOps++
				}
				mtx.Unlock()
			}
		}

		// Add our number of operations to the total number of Operations
		atomic.AddInt64(&totalOps, nOps)
	}).Nanoseconds() / totalOps
}

func ReaderWriterCombinedSkim(nGoroutines int64) int64 {
	rwmap := make(map[int64]T, settings.COMBINED_KEY_RANGE)
	var mtx sync.RWMutex

	// Fill map to reduce overhead of resizing
	for i := int64(0); i < settings.COMBINED_KEY_RANGE; i++ {
		rwmap[i] = T{}
	}
	for i := int64(0); i < settings.COMBINED_KEY_RANGE; i++ {
		delete(rwmap, i)
	}

	var totalOps int64
	return settings.ParallelTest(int(nGoroutines), func() {
		var nOps int64
		rng := rand.New(rand.NewSource(time.Now().UTC().UnixNano()))

		for i := int64(0); i < settings.COMBINED_OPS_PER_GOROUTINE; i++ {
			randRatio := rng.Float64()
			randNum := rng.Int63n(settings.COMBINED_KEY_RANGE)
			switch {
			// 0 <= randRatio < .25 -> Insert
			case randRatio < settings.COMBINED_SKIM_NON_ITERATION_RATIO:
				mtx.Lock()
				rwmap[randNum] = T{}
				mtx.Unlock()
				nOps++
			// .25 <= randRatio < .5 -> Delete
			case randRatio < 2*settings.COMBINED_SKIM_NON_ITERATION_RATIO:
				mtx.Lock()
				delete(rwmap, randNum)
				mtx.Unlock()
				nOps++
			// .5 <= randRatio < .75 -> Lookup
			case randRatio < 3*settings.COMBINED_SKIM_NON_ITERATION_RATIO:
				mtx.RLock()
				tmp := rwmap[randNum]
				mtx.RUnlock()
				tmp.iter++
				nOps++
			// .75 <= randRatio < 1 -> Iterate
			default:
				mtx.Lock()
				for k, v := range rwmap {
					v.iter++
					t := rwmap[k]
					t.iter++
					rwmap[k] = t
					nOps++
				}
				mtx.Unlock()
			}
		}

		// Add our number of operations to the total number of Operations
		atomic.AddInt64(&totalOps, nOps)
	}).Nanoseconds() / totalOps
}
