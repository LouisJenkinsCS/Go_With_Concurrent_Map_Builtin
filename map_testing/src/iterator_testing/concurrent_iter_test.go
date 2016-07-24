package iterator_testing

import "sync/atomic"
import "math/rand"
import "time"
import "settings"
import "testing"

type T struct {
    iter uint64
}

func BenchmarkIteratorMutate(b *testing.B) {
    cmap := make(map[int64]*T, 0, 1)
    rand.Seed(time.Now().UTC().UnixNano())
    for i := int64(0); i < settings.ITERATOR_NUM_ELEMS; i++ {
        cmap[i] = &T{}
    }

    b.ResetTimer()
    b.RunParallel(func(pb *testing.PB) {
        for pb.Next() {
            for k, v := range sync.Interlocked cmap {
                rng := rand.Intn(int(settings.ITERATOR_RNG_INCREMENT))
                if (int64(rng) % k) == 0 {
                    v.iter++
                }
            }
        }
    })
}

func BenchmarkIteratorMutate_default(b *testing.B) {
    dmap := make(map[int64]*T)
    rand.Seed(time.Now().UTC().UnixNano())
    for i := int64(0); i < settings.ITERATOR_NUM_ELEMS; i++ {
        dmap[i] = &T{}
    }

    b.ResetTimer()
    b.RunParallel(func(pb *testing.PB) {
        for pb.Next() {
            for k, v := range dmap {
                rng := rand.Intn(int(settings.ITERATOR_RNG_INCREMENT))
                if (int64(rng) % k) == 0 {
                    atomic.AddUint64(&v.iter, 1)
                }
            }
        }
    })
}