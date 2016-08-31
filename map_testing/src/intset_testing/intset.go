package intset_testing

import "sync"
import "math/rand"
import "time"
import "settings"
import "strconv"
import cmap "github.com/streamrail/concurrent-map"
import "github.com/zond/gotomic"

func ConcurrentIntset(nGoroutines int64) int64 {
	cmap := make(map[int64]settings.Unused, settings.INTSET_VALUE_RANGE, nGoroutines)

	return settings.ParallelTest(int(nGoroutines), func() {
		rng := rand.New(rand.NewSource(time.Now().UTC().UnixNano()))
		maxElements := settings.INTSET_VALUE_RANGE / 4

		for i := uint64(0); i < settings.INTSET_OPS_PER_GOROUTINE; i++ {
			rngRatio := rng.Float64()
			randNum := rng.Int63n(int64(settings.INTSET_VALUE_RANGE))
			switch {
			// 0 <= rngRatio < LOOKUP -> Do Lookup
			case rngRatio < settings.INTSET_FAIR_LOOKUP_RATIO:
				a := cmap[randNum]
				// Necessary instruction to prevent compiler flagging as unused, and prevent it from being compiled away
				a++
			// LOOKUP <= rngRatio < INSERT -> Do Insert
			case rngRatio < settings.INTSET_FAIR_INSERT_RATIO+settings.INTSET_FAIR_LOOKUP_RATIO:
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
	}).Nanoseconds() / int64(settings.INTSET_OPS_PER_GOROUTINE*uint64(nGoroutines))
}

func StreamrailConcurrentIntset(nGoroutines int64) int64 {
	scmap := cmap.New()

	return settings.ParallelTest(int(nGoroutines), func() {
		rng := rand.New(rand.NewSource(time.Now().UTC().UnixNano()))
		maxElements := settings.INTSET_VALUE_RANGE / 4

		for i := uint64(0); i < settings.INTSET_OPS_PER_GOROUTINE; i++ {
			rngRatio := rng.Float64()
			randNum := rng.Int63n(int64(settings.INTSET_VALUE_RANGE))
			switch {
			// 0 <= rngRatio < LOOKUP -> Do Lookup
			case rngRatio < settings.INTSET_FAIR_LOOKUP_RATIO:
				scmap.Get(strconv.FormatInt(randNum, 10))
			// LOOKUP <= rngRatio < INSERT -> Do Insert
			case rngRatio < settings.INTSET_FAIR_INSERT_RATIO+settings.INTSET_FAIR_LOOKUP_RATIO:
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
	}).Nanoseconds() / int64(settings.INTSET_OPS_PER_GOROUTINE*uint64(nGoroutines))
}

func GotomicConcurrentIntset(nGoroutines int64) int64 {
	gcmap := gotomic.NewHash()

	return settings.ParallelTest(int(nGoroutines), func() {
		rng := rand.New(rand.NewSource(time.Now().UTC().UnixNano()))
		maxElements := settings.INTSET_VALUE_RANGE / 4

		for i := uint64(0); i < settings.INTSET_OPS_PER_GOROUTINE; i++ {
			rngRatio := rng.Float64()
			randNum := rng.Int63n(int64(settings.INTSET_VALUE_RANGE))
			switch {
			// 0 <= rngRatio < LOOKUP -> Do Lookup
			case rngRatio < settings.INTSET_FAIR_LOOKUP_RATIO:
				gcmap.Get(gotomic.IntKey(int(randNum)))
			// LOOKUP <= rngRatio < INSERT -> Do Insert
			case rngRatio < settings.INTSET_FAIR_INSERT_RATIO+settings.INTSET_FAIR_LOOKUP_RATIO:
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
	}).Nanoseconds() / int64(settings.INTSET_OPS_PER_GOROUTINE*uint64(nGoroutines))
}

func SynchronizedIntset(nGoroutines int64) int64 {
	smap := make(map[int64]settings.Unused, settings.INTSET_VALUE_RANGE)
	mtx := sync.Mutex{}

	return settings.ParallelTest(int(nGoroutines), func() {
		rng := rand.New(rand.NewSource(time.Now().UTC().UnixNano()))
		maxElements := settings.INTSET_VALUE_RANGE

		for i := uint64(0); i < settings.INTSET_OPS_PER_GOROUTINE; i++ {
			rngRatio := rng.Float64()
			randNum := rng.Int63n(int64(settings.INTSET_VALUE_RANGE))
			switch {
			// 0 <= rngRatio < LOOKUP -> Do Lookup
			case rngRatio < settings.INTSET_FAIR_LOOKUP_RATIO:
				mtx.Lock()
				a := smap[randNum]
				mtx.Unlock()
				// Necessary instruction to prevent compiler flagging as unused, and prevent it from being compiled away
				a++
			// LOOKUP <= rngRatio < INSERT -> Do Insert
			case rngRatio < settings.INTSET_FAIR_INSERT_RATIO+settings.INTSET_FAIR_LOOKUP_RATIO:
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
	}).Nanoseconds() / int64(settings.INTSET_OPS_PER_GOROUTINE*uint64(nGoroutines))
}

func ReaderWriterIntset(nGoroutines int64) int64 {
	rwmap := make(map[int64]settings.Unused, settings.INTSET_VALUE_RANGE)
	mtx := sync.RWMutex{}

	return settings.ParallelTest(int(nGoroutines), func() {
		rng := rand.New(rand.NewSource(time.Now().UTC().UnixNano()))
		maxElements := settings.INTSET_VALUE_RANGE / 4

		for i := uint64(0); i < settings.INTSET_OPS_PER_GOROUTINE; i++ {
			rngRatio := rng.Float64()
			randNum := rng.Int63n(int64(settings.INTSET_VALUE_RANGE))
			switch {
			// 0 <= rngRatio < LOOKUP -> Do Lookup
			case rngRatio < settings.INTSET_FAIR_LOOKUP_RATIO:
				mtx.RLock()
				a := rwmap[randNum]
				mtx.RUnlock()
				// Necessary instruction to prevent compiler flagging as unused, and prevent it from being compiled away
				a++
			// LOOKUP <= rngRatio < INSERT -> Do Insert
			case rngRatio < settings.INTSET_FAIR_INSERT_RATIO+settings.INTSET_FAIR_LOOKUP_RATIO:
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
	}).Nanoseconds() / int64(settings.INTSET_OPS_PER_GOROUTINE*uint64(nGoroutines))
}
