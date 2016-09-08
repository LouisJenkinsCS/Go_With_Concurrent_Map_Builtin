package combined_testing

import (
	"math/rand"
	"runtime"
	"settings"
	"strconv"
	"sync"
	"testing"
	"time"

	cmap "github.com/streamrail/concurrent-map"
	"github.com/zond/gotomic"
)

type T struct {
	_    uintptr
	iter uintptr
	_    uintptr
}

func BenchmarkConcurrentCombinedSkim_RW(b *testing.B) {
	cmap := make(map[int64]T, settings.COMBINED_KEY_RANGE, runtime.GOMAXPROCS(0))

	// Fill map to reduce overhead of resizing
	for i := int64(0); i < settings.COMBINED_KEY_RANGE; i++ {
		cmap[i] = T{}
	}
	for i := int64(0); i < settings.COMBINED_KEY_RANGE; i++ {
		delete(cmap, i)
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		rng := rand.New(rand.NewSource(time.Now().UTC().UnixNano()))

		for pb.Next() {
			randRatio := rng.Float64()
			randNum := rng.Int63n(settings.COMBINED_KEY_RANGE)
			switch {
			// 0 <= randRatio < .25 -> Insert
			case randRatio < settings.COMBINED_SKIM_NON_ITERATION_RATIO:
				cmap[randNum] = T{}

			// .25 <= randRatio < .5 -> Delete
			case randRatio < 2*settings.COMBINED_SKIM_NON_ITERATION_RATIO:
				delete(cmap, randNum)

			// .5 <= randRatio < .75 -> Lookup
			case randRatio < 3*settings.COMBINED_SKIM_NON_ITERATION_RATIO:
				tmp := cmap[randNum]
				tmp.iter++

			// .75 <= randRatio < 1 -> Iterate
			default:
				// Each iteration counts as an operation (as it calls mapiternext)
				for k, v := range cmap {
					v.iter++
					t := cmap[k]
					t.iter++
					cmap[k] = t

				}
			}
		}
	})
}

func BenchmarkConcurrentCombinedSkim_RO(b *testing.B) {
	cmap := make(map[int64]T, settings.COMBINED_KEY_RANGE, runtime.GOMAXPROCS(0))

	// Fill map to reduce overhead of resizing
	for i := int64(0); i < settings.COMBINED_KEY_RANGE; i++ {
		cmap[i] = T{}
	}
	for i := int64(0); i < settings.COMBINED_KEY_RANGE; i++ {
		delete(cmap, i)
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		rng := rand.New(rand.NewSource(time.Now().UTC().UnixNano()))

		for pb.Next() {
			randRatio := rng.Float64()
			randNum := rng.Int63n(settings.COMBINED_KEY_RANGE)
			switch {
			// 0 <= randRatio < .25 -> Insert
			case randRatio < settings.COMBINED_SKIM_NON_ITERATION_RATIO:
				cmap[randNum] = T{}

			// .25 <= randRatio < .5 -> Delete
			case randRatio < 2*settings.COMBINED_SKIM_NON_ITERATION_RATIO:
				delete(cmap, randNum)

			// .5 <= randRatio < .75 -> Lookup
			case randRatio < 3*settings.COMBINED_SKIM_NON_ITERATION_RATIO:
				tmp := cmap[randNum]
				tmp.iter++

			// .75 <= randRatio < 1 -> Iterate
			default:
				// Each iteration counts as an operation (as it calls mapiternext)
				for k, v := range cmap {
					k++
					v.iter++

				}
			}
		}
	})
}

func BenchmarkSynchronizedCombinedSkim_RW(b *testing.B) {
	smap := make(map[int64]T, settings.COMBINED_KEY_RANGE)
	var mtx sync.Mutex

	// Fill map to reduce overhead of resizing
	for i := int64(0); i < settings.COMBINED_KEY_RANGE; i++ {
		smap[i] = T{}
	}
	for i := int64(0); i < settings.COMBINED_KEY_RANGE; i++ {
		delete(smap, i)
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		rng := rand.New(rand.NewSource(time.Now().UTC().UnixNano()))

		for pb.Next() {
			randRatio := rng.Float64()
			randNum := rng.Int63n(settings.COMBINED_KEY_RANGE)
			switch {
			// 0 <= randRatio < .25 -> Insert
			case randRatio < settings.COMBINED_SKIM_NON_ITERATION_RATIO:
				mtx.Lock()
				smap[randNum] = T{}
				mtx.Unlock()
			// .25 <= randRatio < .5 -> Delete
			case randRatio < 2*settings.COMBINED_SKIM_NON_ITERATION_RATIO:
				mtx.Lock()
				delete(smap, randNum)
				mtx.Unlock()
			// .5 <= randRatio < .75 -> Lookup
			case randRatio < 3*settings.COMBINED_SKIM_NON_ITERATION_RATIO:
				mtx.Lock()
				tmp := smap[randNum]
				mtx.Unlock()
				tmp.iter++
			// .75 <= randRatio < 1 -> Iterate
			default:
				mtx.Lock()
				for k, v := range smap {
					v.iter++
					t := smap[k]
					t.iter++
					smap[k] = t
				}
				mtx.Unlock()
			}
		}
	})
}

func BenchmarkSynchronizedCombinedSkim_RO(b *testing.B) {
	smap := make(map[int64]T, settings.COMBINED_KEY_RANGE)
	var mtx sync.Mutex

	// Fill map to reduce overhead of resizing
	for i := int64(0); i < settings.COMBINED_KEY_RANGE; i++ {
		smap[i] = T{}
	}
	for i := int64(0); i < settings.COMBINED_KEY_RANGE; i++ {
		delete(smap, i)
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		rng := rand.New(rand.NewSource(time.Now().UTC().UnixNano()))

		for pb.Next() {
			randRatio := rng.Float64()
			randNum := rng.Int63n(settings.COMBINED_KEY_RANGE)
			switch {
			// 0 <= randRatio < .25 -> Insert
			case randRatio < settings.COMBINED_SKIM_NON_ITERATION_RATIO:
				mtx.Lock()
				smap[randNum] = T{}
				mtx.Unlock()
			// .25 <= randRatio < .5 -> Delete
			case randRatio < 2*settings.COMBINED_SKIM_NON_ITERATION_RATIO:
				mtx.Lock()
				delete(smap, randNum)
				mtx.Unlock()
			// .5 <= randRatio < .75 -> Lookup
			case randRatio < 3*settings.COMBINED_SKIM_NON_ITERATION_RATIO:
				mtx.Lock()
				tmp := smap[randNum]
				mtx.Unlock()
				tmp.iter++
			// .75 <= randRatio < 1 -> Iterate
			default:
				mtx.Lock()
				for k, v := range smap {
					k++
					v.iter++
				}
				mtx.Unlock()
			}
		}
	})
}

func BenchmarkReaderWriterCombinedSkim_RW(b *testing.B) {
	rwmap := make(map[int64]T, settings.COMBINED_KEY_RANGE)
	var mtx sync.RWMutex

	// Fill map to reduce overhead of resizing
	for i := int64(0); i < settings.COMBINED_KEY_RANGE; i++ {
		rwmap[i] = T{}
	}
	for i := int64(0); i < settings.COMBINED_KEY_RANGE; i++ {
		delete(rwmap, i)
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		rng := rand.New(rand.NewSource(time.Now().UTC().UnixNano()))

		for pb.Next() {
			randRatio := rng.Float64()
			randNum := rng.Int63n(settings.COMBINED_KEY_RANGE)
			switch {
			// 0 <= randRatio < .25 -> Insert
			case randRatio < settings.COMBINED_SKIM_NON_ITERATION_RATIO:
				mtx.Lock()
				rwmap[randNum] = T{}
				mtx.Unlock()

			// .25 <= randRatio < .5 -> Delete
			case randRatio < 2*settings.COMBINED_SKIM_NON_ITERATION_RATIO:
				mtx.Lock()
				delete(rwmap, randNum)
				mtx.Unlock()

			// .5 <= randRatio < .75 -> Lookup
			case randRatio < 3*settings.COMBINED_SKIM_NON_ITERATION_RATIO:
				mtx.RLock()
				tmp := rwmap[randNum]
				mtx.RUnlock()
				tmp.iter++

			// .75 <= randRatio < 1 -> Iterate
			default:
				mtx.Lock()
				for k, v := range rwmap {
					v.iter++
					t := rwmap[k]
					t.iter++
					rwmap[k] = t

				}
				mtx.Unlock()
			}
		}
	})
}

func BenchmarkReaderWriterCombinedSkim_RO(b *testing.B) {
	rwmap := make(map[int64]T, settings.COMBINED_KEY_RANGE)
	var mtx sync.RWMutex

	// Fill map to reduce overhead of resizing
	for i := int64(0); i < settings.COMBINED_KEY_RANGE; i++ {
		rwmap[i] = T{}
	}
	for i := int64(0); i < settings.COMBINED_KEY_RANGE; i++ {
		delete(rwmap, i)
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		rng := rand.New(rand.NewSource(time.Now().UTC().UnixNano()))

		for pb.Next() {
			randRatio := rng.Float64()
			randNum := rng.Int63n(settings.COMBINED_KEY_RANGE)
			switch {
			// 0 <= randRatio < .25 -> Insert
			case randRatio < settings.COMBINED_SKIM_NON_ITERATION_RATIO:
				mtx.Lock()
				rwmap[randNum] = T{}
				mtx.Unlock()

			// .25 <= randRatio < .5 -> Delete
			case randRatio < 2*settings.COMBINED_SKIM_NON_ITERATION_RATIO:
				mtx.Lock()
				delete(rwmap, randNum)
				mtx.Unlock()

			// .5 <= randRatio < .75 -> Lookup
			case randRatio < 3*settings.COMBINED_SKIM_NON_ITERATION_RATIO:
				mtx.RLock()
				tmp := rwmap[randNum]
				mtx.RUnlock()
				tmp.iter++

			// .75 <= randRatio < 1 -> Iterate
			default:
				mtx.Lock()
				for k, v := range rwmap {
					k++
					v.iter++
				}
				mtx.Unlock()
			}
		}
	})
}

func BenchmarkStreamrailConcurrentCombinedSkim_RO(b *testing.B) {
	scmap := cmap.New()

	// Fill map to reduce overhead of resizing
	for i := int64(0); i < settings.COMBINED_KEY_RANGE; i++ {
		scmap.Set(strconv.FormatInt(i, 10), settings.UNUSED)
	}
	for i := int64(0); i < settings.COMBINED_KEY_RANGE; i++ {
		scmap.Remove(strconv.FormatInt(i, 10))
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		rng := rand.New(rand.NewSource(time.Now().UTC().UnixNano()))

		for pb.Next() {
			randRatio := rng.Float64()
			randNum := rng.Int63n(settings.COMBINED_KEY_RANGE)
			switch {
			// 0 <= randRatio < .25 -> Insert
			case randRatio < settings.COMBINED_SKIM_NON_ITERATION_RATIO:
				scmap.Set(strconv.FormatInt(randNum, 10), settings.UNUSED)
			// .25 <= randRatio < .5 -> Delete
			case randRatio < 2*settings.COMBINED_SKIM_NON_ITERATION_RATIO:
				scmap.Remove(strconv.FormatInt(randNum, 10))
			// .5 <= randRatio < .75 -> Lookup
			case randRatio < 3*settings.COMBINED_SKIM_NON_ITERATION_RATIO:
				scmap.Get(strconv.FormatInt(randNum, 10))
			// .75 <= randRatio < 1 -> Iterate
			default:
				// Each iteration counts as an operation (as it calls mapiternext)
				for item := range scmap.IterBuffered() {
					_ = item.Key
					_ = item.Val
				}
			}
		}
	})
}

func BenchmarkGotomicConcurrentCombinedSkim_RO(b *testing.B) {
	gcmap := gotomic.NewHash()

	// Fill map to reduce overhead of resizing
	for i := int64(0); i < settings.COMBINED_KEY_RANGE; i++ {
		gcmap.Put(gotomic.IntKey(int(i)), settings.UNUSED)
	}
	for i := int64(0); i < settings.COMBINED_KEY_RANGE; i++ {
		gcmap.Delete(gotomic.IntKey(int(i)))
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		rng := rand.New(rand.NewSource(time.Now().UTC().UnixNano()))

		for pb.Next() {
			randRatio := rng.Float64()
			randNum := rng.Int63n(settings.COMBINED_KEY_RANGE)
			switch {
			// 0 <= randRatio < .25 -> Insert
			case randRatio < settings.COMBINED_SKIM_NON_ITERATION_RATIO:
				gcmap.Put(gotomic.IntKey(int(randNum)), settings.UNUSED)
			// .25 <= randRatio < .5 -> Delete
			case randRatio < 2*settings.COMBINED_SKIM_NON_ITERATION_RATIO:
				gcmap.Delete(gotomic.IntKey(int(randNum)))
			// .5 <= randRatio < .75 -> Lookup
			case randRatio < 3*settings.COMBINED_SKIM_NON_ITERATION_RATIO:
				gcmap.Get(gotomic.IntKey(int(randNum)))
			// .75 <= randRatio < 1 -> Iterate
			default:
				// Each iteration counts as an operation (as it calls mapiternext)
				gcmap.Each(func(k gotomic.Hashable, v gotomic.Thing) bool {
					_ = k
					_ = v
					return false
				})
			}
		}
	})
}
