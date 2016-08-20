package intset_testing

import "sync"
import "math/rand"
import "time"
import "settings"
import "testing"

func BenchmarkIntsetGenerate(b *testing.B) {
    cmap := make(map[int64]settings.Unused, 0, 1)

    // Initialize to half full
    for i := uint64(0); i < settings.INTSET_VALUE_RANGE; i++ {
        cmap[int64(i)] = settings.UNUSED
    }
    for i := uint64(0); i < settings.INTSET_VALUE_RANGE; i++ {
        delete(cmap, int64(i))
    }
    b.ResetTimer()

    b.RunParallel(func(pb *testing.PB) {
        rng := rand.New(rand.NewSource(time.Now().UTC().UnixNano()))
        maxElements := settings.INTSET_VALUE_RANGE / 4
        
        for pb.Next() {
            rngRatio := rng.Float64()
            randNum := rand.Int63n(int64(settings.INTSET_VALUE_RANGE))
            switch {
                // 0 <= rngRatio < LOOKUP -> Do Lookup
                case rngRatio < settings.INTSET_FAIR_LOOKUP_RATIO: 
                    a := cmap[randNum]
                    // Necessary instruction to prevent compiler flagging as unused, and prevent it from being compiled away
                    a++
                // LOOKUP <= rngRatio < INSERT -> Do Insert
                case rngRatio < settings.INTSET_FAIR_INSERT_RATIO + settings.INTSET_FAIR_LOOKUP_RATIO:
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

func BenchmarkIntsetGenerate_sync(b *testing.B) {
    smap := make(map[int64]settings.Unused)
    mtx := sync.Mutex{}

    // Initialize to half full
    for i := uint64(0); i < settings.INTSET_VALUE_RANGE; i++ {
        smap[int64(i)] = settings.UNUSED
    }
    for i := uint64(0); i < settings.INTSET_VALUE_RANGE; i++ {
        delete(smap, int64(i))
    }
    b.ResetTimer()

    b.RunParallel(func(pb *testing.PB) {
        rng := rand.New(rand.NewSource(time.Now().UTC().UnixNano()))
        maxElements := settings.INTSET_VALUE_RANGE / 4
        
        for pb.Next() {
            rngRatio := rng.Float64()
            randNum := rand.Int63n(int64(settings.INTSET_VALUE_RANGE))
            switch {
                // 0 <= rngRatio < LOOKUP -> Do Lookup
                case rngRatio < settings.INTSET_FAIR_LOOKUP_RATIO: 
                    mtx.Lock()
                    a := smap[randNum]
                    mtx.Unlock()
                    // Necessary instruction to prevent compiler flagging as unused, and prevent it from being compiled away
                    a++
                // LOOKUP <= rngRatio < INSERT -> Do Insert
                case rngRatio < settings.INTSET_FAIR_INSERT_RATIO + settings.INTSET_FAIR_LOOKUP_RATIO:
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

func BenchmarkIntsetGenerate_rw(b *testing.B) {
    rwmap := make(map[int64]settings.Unused)
    mtx := sync.RWMutex{}

    // Initialize to half full
    for i := uint64(0); i < settings.INTSET_VALUE_RANGE; i++ {
        rwmap[int64(i)] = settings.UNUSED
    }
    for i := uint64(0); i < settings.INTSET_VALUE_RANGE; i++ {
        delete(rwmap, int64(i))
    }
    b.ResetTimer()

    b.RunParallel(func(pb *testing.PB) {
        rng := rand.New(rand.NewSource(time.Now().UTC().UnixNano()))
        maxElements := settings.INTSET_VALUE_RANGE / 4
        
        for pb.Next() {
            rngRatio := rng.Float64()
            randNum := rand.Int63n(int64(settings.INTSET_VALUE_RANGE))
            switch {
                // 0 <= rngRatio < LOOKUP -> Do Lookup
                case rngRatio < settings.INTSET_FAIR_LOOKUP_RATIO: 
                    mtx.RLock()
                    a := rwmap[randNum]
                    mtx.RUnlock()
                    // Necessary instruction to prevent compiler flagging as unused, and prevent it from being compiled away
                    a++
                // LOOKUP <= rngRatio < INSERT -> Do Insert
                case rngRatio < settings.INTSET_FAIR_INSERT_RATIO + settings.INTSET_FAIR_LOOKUP_RATIO:
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