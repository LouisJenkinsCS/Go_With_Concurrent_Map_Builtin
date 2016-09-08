package iterator_testing

import (
	"runtime"
	"settings"
	"strconv"
	"testing"

	cmap "github.com/streamrail/concurrent-map"
	"github.com/zond/gotomic"
)

func BenchmarkConcurrentIterator_RO(b *testing.B) {
	cmap := make(map[int64]settings.Unused, settings.ITERATOR_NUM_ELEMS, runtime.GOMAXPROCS(0))

	// Initialize the map with a fixed number of elements.
	for i := int64(0); i < settings.ITERATOR_NUM_ELEMS; i++ {
		cmap[i] = settings.UNUSED
	}

	// Begin iteration test
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			for k, v := range cmap {
				k++
				v++
			}
		}
	})
}

func BenchmarkStreamrailConcurrentIterator_RO(b *testing.B) {
	scmap := cmap.New()

	// Initialize the map with a fixed number of elements.
	for i := int64(0); i < settings.ITERATOR_NUM_ELEMS; i++ {
		scmap.Set(strconv.FormatInt(i, 10), settings.UNUSED)
	}

	// Begin iteration test
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			for item := range scmap.IterBuffered() {
				_ = item.Key
				_ = item.Val
			}
		}
	})
}

func BenchmarkGotomicConcurrentIterator_RO(b *testing.B) {
	gcmap := gotomic.NewHash()

	// Initialize the map with a fixed number of elements.
	for i := int64(0); i < settings.ITERATOR_NUM_ELEMS; i++ {
		gcmap.Put(gotomic.IntKey(int(i)), settings.UNUSED)
	}

	// Begin iteration test
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			gcmap.Each(func(k gotomic.Hashable, v gotomic.Thing) bool {
				_ = k
				_ = v
				return false
			})
		}
	})
}

func BenchmarkDefaultIterator_RO(b *testing.B) {
	smap := make(map[int64]settings.Unused, settings.ITERATOR_NUM_ELEMS)

	// Initialize the map with a fixed number of elements.
	for i := int64(0); i < settings.ITERATOR_NUM_ELEMS; i++ {
		smap[i] = settings.UNUSED
	}

	// Begin iteration test
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			for k, v := range smap {
				k++
				v++
			}
		}
	})
}
