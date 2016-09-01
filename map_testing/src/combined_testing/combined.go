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
	gotomic "github.com/zond/gotomic"
)

func BenchmarkConcurrentCombined(b *testing.B) {
	cmap := make(map[int64]settings.Unused, settings.COMBINED_KEY_RANGE, runtime.GOMAXPROCS(0))

	// Fill map to reduce overhead of resizing
	for i := int64(0); i < settings.COMBINED_KEY_RANGE; i++ {
		cmap[i] = settings.UNUSED
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
			case randRatio < settings.COMBINED_FAIR_RATIO:
				cmap[randNum] = settings.UNUSED
			// .25 <= randRatio < .5 -> Delete
			case randRatio < 2*settings.COMBINED_FAIR_RATIO:
				delete(cmap, randNum)
			// .5 <= randRatio < .75 -> Lookup
			case randRatio < 3*settings.COMBINED_FAIR_RATIO:
				tmp := cmap[randNum]
				tmp++
			// .75 <= randRatio < 1 -> Iterate
			default:
				// Each iteration counts as an operation (as it calls mapiternext)
				for k, v := range cmap {
					k++
					v++
				}
			}
		}
	})
	b.StopTimer()
}

func BenchmarkStreamrailConcurrentCombined(b *testing.B) {
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
			case randRatio < settings.COMBINED_FAIR_RATIO:
				scmap.Set(strconv.FormatInt(randNum, 10), settings.UNUSED)
			// .25 <= randRatio < .5 -> Delete
			case randRatio < 2*settings.COMBINED_FAIR_RATIO:
				scmap.Remove(strconv.FormatInt(randNum, 10))
			// .5 <= randRatio < .75 -> Lookup
			case randRatio < 3*settings.COMBINED_FAIR_RATIO:
				scmap.Get(strconv.FormatInt(randNum, 10))
			// .75 <= randRatio < 1 -> Iterate
			default:
				// Each iteration counts as an operation (as it calls mapiternext)
				for item := range scmap.Iter() {
					_ = item.Key
					_ = item.Val
				}
			}
		}
	})
}

func BenchmarkGotomicConcurrentCombined(b *testing.B) {
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
			case randRatio < settings.COMBINED_FAIR_RATIO:
				gcmap.Put(gotomic.IntKey(int(randNum)), settings.UNUSED)
			// .25 <= randRatio < .5 -> Delete
			case randRatio < 2*settings.COMBINED_FAIR_RATIO:
				gcmap.Delete(gotomic.IntKey(int(randNum)))
			// .5 <= randRatio < .75 -> Lookup
			case randRatio < 3*settings.COMBINED_FAIR_RATIO:
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

func BenchmarkSynchronizedCombined(b *testing.B) {
	smap := make(map[int64]settings.Unused, settings.COMBINED_KEY_RANGE)
	var mtx sync.Mutex

	// Fill map to reduce overhead of resizing
	for i := int64(0); i < settings.COMBINED_KEY_RANGE; i++ {
		smap[i] = settings.UNUSED
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
			case randRatio < settings.COMBINED_FAIR_RATIO:
				mtx.Lock()
				smap[randNum] = settings.UNUSED
				mtx.Unlock()
			// .25 <= randRatio < .5 -> Delete
			case randRatio < 2*settings.COMBINED_FAIR_RATIO:
				mtx.Lock()
				delete(smap, randNum)
				mtx.Unlock()
			// .5 <= randRatio < .75 -> Lookup
			case randRatio < 3*settings.COMBINED_FAIR_RATIO:
				mtx.Lock()
				tmp := smap[randNum]
				mtx.Unlock()
				tmp++
			// .75 <= randRatio < 1 -> Iterate
			default:
				mtx.Lock()
				for k, v := range smap {
					k++
					v++
				}
				mtx.Unlock()
			}
		}
	})
}

func BenchmarkReaderWriterCombined(b *testing.B) {
	rwmap := make(map[int64]settings.Unused, settings.COMBINED_KEY_RANGE)
	var mtx sync.RWMutex

	// Fill map to reduce overhead of resizing
	for i := int64(0); i < settings.COMBINED_KEY_RANGE; i++ {
		rwmap[i] = settings.UNUSED
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
			case randRatio < settings.COMBINED_FAIR_RATIO:
				mtx.Lock()
				rwmap[randNum] = settings.UNUSED
				mtx.Unlock()
			// .25 <= randRatio < .5 -> Delete
			case randRatio < 2*settings.COMBINED_FAIR_RATIO:
				mtx.Lock()
				delete(rwmap, randNum)
				mtx.Unlock()
			// .5 <= randRatio < .75 -> Lookup
			case randRatio < 3*settings.COMBINED_FAIR_RATIO:
				mtx.RLock()
				tmp := rwmap[randNum]
				mtx.RUnlock()
				tmp++
			// .75 <= randRatio < 1 -> Iterate
			default:
				mtx.RLock()
				for k, v := range rwmap {
					k++
					v++
				}
				mtx.RUnlock()
			}
		}
	})
}
