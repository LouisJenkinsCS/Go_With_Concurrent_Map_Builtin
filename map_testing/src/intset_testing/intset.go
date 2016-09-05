package intset_testing

import (
	"math/rand"
	"runtime"
	"settings"
	"sync"
	"testing"
	"time"

	cmap "github.com/streamrail/concurrent-map"
	"github.com/zond/gotomic"
)

import "strconv"

func BenchmarkConcurrentIntset(b *testing.B) {
	cmap := make(map[int64]settings.Unused, settings.COMBINED_KEY_RANGE, runtime.GOMAXPROCS(0))

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		rng := rand.New(rand.NewSource(time.Now().UTC().UnixNano()))
		maxElements := settings.INTSET_VALUE_RANGE / 4

		for pb.Next() {
			rngRatio := rng.Float64()
			randNum := rng.Int63n(int64(settings.INTSET_VALUE_RANGE))
			switch {
			// 0 <= rngRatio < LOOKUP -> Do Lookup
			case rngRatio <= .34:
				a := cmap[randNum]
				// Necessary instruction to prevent compiler flagging as unused, and prevent it from being compiled away
				a++
			// LOOKUP <= rngRatio < INSERT -> Do Insert
			case rngRatio <= .67:
				if uint64(len(cmap)) < maxElements {
					cmap[randNum] = settings.UNUSED
					continue
				}

				// If we have too many elements fall through to delete
				fallthrough
			// INSERT <= rngRatio <= 1 -> Do Remove
			default:
				delete(cmap, randNum)
			}
		}
	})
}

func BenchmarkConcurrentIntset_Bias(b *testing.B) {
	cmap := make(map[int64]settings.Unused, settings.COMBINED_KEY_RANGE, runtime.GOMAXPROCS(0))

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		rng := rand.New(rand.NewSource(time.Now().UTC().UnixNano()))
		maxElements := settings.INTSET_VALUE_RANGE / 4

		for pb.Next() {
			rngRatio := rng.Float64()
			randNum := rng.Int63n(int64(settings.INTSET_VALUE_RANGE))
			switch {
			// 0 <= rngRatio < LOOKUP -> Do Lookup
			case rngRatio <= .80:
				a := cmap[randNum]
				// Necessary instruction to prevent compiler flagging as unused, and prevent it from being compiled away
				a++
			// LOOKUP <= rngRatio < INSERT -> Do Insert
			case rngRatio <= .90:
				if uint64(len(cmap)) < maxElements {
					cmap[randNum] = settings.UNUSED
					continue
				}

				// If we have too many elements fall through to delete
				fallthrough
			// INSERT <= rngRatio <= 1 -> Do Remove
			default:
				delete(cmap, randNum)
			}
		}
	})
}

func BenchmarkStreamrailConcurrentIntset(b *testing.B) {
	scmap := cmap.New()

	b.RunParallel(func(pb *testing.PB) {
		rng := rand.New(rand.NewSource(time.Now().UTC().UnixNano()))
		maxElements := settings.INTSET_VALUE_RANGE / 4

		for pb.Next() {
			rngRatio := rng.Float64()
			randNum := rng.Int63n(int64(settings.INTSET_VALUE_RANGE))
			switch {
			// 0 <= rngRatio < LOOKUP -> Do Lookup
			case rngRatio <= .34:
				scmap.Get(strconv.FormatInt(randNum, 10))
			// LOOKUP <= rngRatio < INSERT -> Do Insert
			case rngRatio <= .67:
				if uint64(scmap.Count()) < maxElements {
					scmap.Set(strconv.FormatInt(randNum, 10), settings.UNUSED)
					continue
				}

				// If we have too many elements fall through to delete
				fallthrough
			// INSERT <= rngRatio <= 1 -> Do Remove
			default:
				scmap.Remove(strconv.FormatInt(randNum, 10))
			}
		}
	})
}

func BenchmarkStreamrailConcurrentIntset_Bias(b *testing.B) {
	scmap := cmap.New()

	b.RunParallel(func(pb *testing.PB) {
		rng := rand.New(rand.NewSource(time.Now().UTC().UnixNano()))
		maxElements := settings.INTSET_VALUE_RANGE / 4

		for pb.Next() {
			rngRatio := rng.Float64()
			randNum := rng.Int63n(int64(settings.INTSET_VALUE_RANGE))
			switch {
			// 0 <= rngRatio < LOOKUP -> Do Lookup
			case rngRatio <= .80:
				scmap.Get(strconv.FormatInt(randNum, 10))
			// LOOKUP <= rngRatio < INSERT -> Do Insert
			case rngRatio <= .90:
				if uint64(scmap.Count()) < maxElements {
					scmap.Set(strconv.FormatInt(randNum, 10), settings.UNUSED)
					continue
				}

				// If we have too many elements fall through to delete
				fallthrough
			// INSERT <= rngRatio <= 1 -> Do Remove
			default:
				scmap.Remove(strconv.FormatInt(randNum, 10))
			}
		}
	})
}

func BenchmarkGotomicConcurrentIntset(b *testing.B) {
	gcmap := gotomic.NewHash()

	b.RunParallel(func(pb *testing.PB) {
		rng := rand.New(rand.NewSource(time.Now().UTC().UnixNano()))
		maxElements := settings.INTSET_VALUE_RANGE / 4

		for pb.Next() {
			rngRatio := rng.Float64()
			randNum := rng.Int63n(int64(settings.INTSET_VALUE_RANGE))
			switch {
			// 0 <= rngRatio < LOOKUP -> Do Lookup
			case rngRatio <= .34:
				gcmap.Get(gotomic.IntKey(int(randNum)))
			// LOOKUP <= rngRatio < INSERT -> Do Insert
			case rngRatio <= .67:
				if uint64(gcmap.Size()) < maxElements {
					gcmap.Put(gotomic.IntKey(int(randNum)), settings.UNUSED)
					continue
				}

				// If we have too many elements fall through to delete
				fallthrough
			// INSERT <= rngRatio <= 1 -> Do Remove
			default:
				gcmap.Delete(gotomic.IntKey(int(randNum)))
			}
		}
	})
}

func BenchmarkGotomicConcurrentIntset_Bias(b *testing.B) {
	gcmap := gotomic.NewHash()

	b.RunParallel(func(pb *testing.PB) {
		rng := rand.New(rand.NewSource(time.Now().UTC().UnixNano()))
		maxElements := settings.INTSET_VALUE_RANGE / 4

		for pb.Next() {
			rngRatio := rng.Float64()
			randNum := rng.Int63n(int64(settings.INTSET_VALUE_RANGE))
			switch {
			// 0 <= rngRatio < LOOKUP -> Do Lookup
			case rngRatio <= .80:
				gcmap.Get(gotomic.IntKey(int(randNum)))
			// LOOKUP <= rngRatio < INSERT -> Do Insert
			case rngRatio <= .90:
				if uint64(gcmap.Size()) < maxElements {
					gcmap.Put(gotomic.IntKey(int(randNum)), settings.UNUSED)
					continue
				}

				// If we have too many elements fall through to delete
				fallthrough
			// INSERT <= rngRatio <= 1 -> Do Remove
			default:
				gcmap.Delete(gotomic.IntKey(int(randNum)))
			}
		}
	})
}

func BenchmarkSynchronizedIntset(b *testing.B) {
	smap := make(map[int64]settings.Unused, settings.INTSET_VALUE_RANGE)
	mtx := sync.Mutex{}

	b.RunParallel(func(pb *testing.PB) {
		rng := rand.New(rand.NewSource(time.Now().UTC().UnixNano()))
		maxElements := settings.INTSET_VALUE_RANGE

		for pb.Next() {
			rngRatio := rng.Float64()
			randNum := rng.Int63n(int64(settings.INTSET_VALUE_RANGE))
			switch {
			// 0 <= rngRatio < LOOKUP -> Do Lookup
			case rngRatio <= .34:
				mtx.Lock()
				a := smap[randNum]
				mtx.Unlock()
				// Necessary instruction to prevent compiler flagging as unused, and prevent it from being compiled away
				a++
			// LOOKUP <= rngRatio < INSERT -> Do Insert
			case rngRatio <= .67:
				mtx.Lock()
				if uint64(len(smap)) < maxElements {
					smap[randNum] = settings.UNUSED
					mtx.Unlock()
					continue
				}
				mtx.Unlock()

				// If we have too many elements fall through to delete
				fallthrough
			// INSERT <= rngRatio <= 1 -> Do Remove
			default:
				mtx.Lock()
				delete(smap, randNum)
				mtx.Unlock()
			}
		}
	})
}

func BenchmarkSynchronizedIntset_Bias(b *testing.B) {
	smap := make(map[int64]settings.Unused, settings.INTSET_VALUE_RANGE)
	mtx := sync.Mutex{}

	b.RunParallel(func(pb *testing.PB) {
		rng := rand.New(rand.NewSource(time.Now().UTC().UnixNano()))
		maxElements := settings.INTSET_VALUE_RANGE

		for pb.Next() {
			rngRatio := rng.Float64()
			randNum := rng.Int63n(int64(settings.INTSET_VALUE_RANGE))
			switch {
			// 0 <= rngRatio < LOOKUP -> Do Lookup
			case rngRatio <= .80:
				mtx.Lock()
				a := smap[randNum]
				mtx.Unlock()
				// Necessary instruction to prevent compiler flagging as unused, and prevent it from being compiled away
				a++
			// LOOKUP <= rngRatio < INSERT -> Do Insert
			case rngRatio <= .90:
				mtx.Lock()
				if uint64(len(smap)) < maxElements {
					smap[randNum] = settings.UNUSED
					mtx.Unlock()
					continue
				}
				mtx.Unlock()

				// If we have too many elements fall through to delete
				fallthrough
			// INSERT <= rngRatio <= 1 -> Do Remove
			default:
				mtx.Lock()
				delete(smap, randNum)
				mtx.Unlock()
			}
		}
	})
}

func BenchmarkReaderWriterIntset(b *testing.B) {
	rwmap := make(map[int64]settings.Unused, settings.INTSET_VALUE_RANGE)
	mtx := sync.RWMutex{}

	b.RunParallel(func(pb *testing.PB) {
		rng := rand.New(rand.NewSource(time.Now().UTC().UnixNano()))
		maxElements := settings.INTSET_VALUE_RANGE / 4

		for pb.Next() {
			rngRatio := rng.Float64()
			randNum := rng.Int63n(int64(settings.INTSET_VALUE_RANGE))
			switch {
			// 0 <= rngRatio < LOOKUP -> Do Lookup
			case rngRatio <= .34:
				mtx.RLock()
				a := rwmap[randNum]
				mtx.RUnlock()
				// Necessary instruction to prevent compiler flagging as unused, and prevent it from being compiled away
				a++
			// LOOKUP <= rngRatio < INSERT -> Do Insert
			case rngRatio <= .67:
				mtx.Lock()
				if uint64(len(rwmap)) < maxElements {
					rwmap[randNum] = settings.UNUSED
					mtx.Unlock()
					continue
				}
				mtx.Unlock()

				// If we have too many elements fall through to delete
				fallthrough
			// INSERT <= rngRatio <= 1 -> Do Remove
			default:
				mtx.Lock()
				delete(rwmap, randNum)
				mtx.Unlock()
			}
		}
	})
}

func BenchmarkReaderWriterIntset_Bias(b *testing.B) {
	rwmap := make(map[int64]settings.Unused, settings.INTSET_VALUE_RANGE)
	mtx := sync.RWMutex{}

	b.RunParallel(func(pb *testing.PB) {
		rng := rand.New(rand.NewSource(time.Now().UTC().UnixNano()))
		maxElements := settings.INTSET_VALUE_RANGE / 4

		for pb.Next() {
			rngRatio := rng.Float64()
			randNum := rng.Int63n(int64(settings.INTSET_VALUE_RANGE))
			switch {
			// 0 <= rngRatio < LOOKUP -> Do Lookup
			case rngRatio <= .80:
				mtx.RLock()
				a := rwmap[randNum]
				mtx.RUnlock()
				// Necessary instruction to prevent compiler flagging as unused, and prevent it from being compiled away
				a++
			// LOOKUP <= rngRatio < INSERT -> Do Insert
			case rngRatio <= .90:
				mtx.Lock()
				if uint64(len(rwmap)) < maxElements {
					rwmap[randNum] = settings.UNUSED
					mtx.Unlock()
					continue
				}
				mtx.Unlock()

				// If we have too many elements fall through to delete
				fallthrough
			// INSERT <= rngRatio <= 1 -> Do Remove
			default:
				mtx.Lock()
				delete(rwmap, randNum)
				mtx.Unlock()
			}
		}
	})
}
