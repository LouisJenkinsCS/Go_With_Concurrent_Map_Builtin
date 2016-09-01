package iterator_testing

import (
	"runtime"
	"settings"
	"sync"
	"testing"
)

type T struct {
	iter int
}

func BenchmarkConcurrentIterator_RW(b *testing.B) {
	cmap := make(map[int64]T, settings.ITERATOR_NUM_ELEMS, runtime.GOMAXPROCS(0))

	// Initialize the map with a fixed number of elements.
	for i := int64(0); i < settings.ITERATOR_NUM_ELEMS; i++ {
		cmap[i] = T{}
	}

	// Begin iteration test
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			for k, v := range cmap {
				v.iter++
				// Read-Modify-Write operation to increment struct field atomically
				t := cmap[k]
				t.iter++
				cmap[k] = t
			}
		}
	})
}

func BenchmarkSynchronizedIterator_RW(b *testing.B) {
	smap := make(map[int64]T, settings.ITERATOR_NUM_ELEMS)
	var mtx sync.Mutex

	// Initialize the map with a fixed number of elements.
	for i := int64(0); i < settings.ITERATOR_NUM_ELEMS; i++ {
		smap[i] = T{}
	}

	// Begin iteration test
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			// Required to lock before iteration begins... huge bottleneck
			mtx.Lock()
			for k, v := range smap {
				v.iter++

				// Read-Modify-Write operation to increment struct field atomically
				t := smap[k]
				t.iter++
				smap[k] = t
			}
			mtx.Unlock()
		}
	})
}

func BenchmarkReaderWriterIterator_RW(b *testing.B) {
	rwmap := make(map[int64]T, settings.ITERATOR_NUM_ELEMS)
	var mtx sync.RWMutex

	// Initialize the map with a fixed number of elements.
	for i := int64(0); i < settings.ITERATOR_NUM_ELEMS; i++ {
		rwmap[i] = T{}
	}

	// Begin iteration test
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			// Required to lock before iteration begins... huge bottleneck, plus no promotion from reader lock to write lock
			mtx.Lock()
			for k, v := range rwmap {
				v.iter++

				// Read-Modify-Write operation to increment struct field atomically
				t := rwmap[k]
				t.iter++
				rwmap[k] = t
			}
			mtx.Unlock()
		}
	})
}
