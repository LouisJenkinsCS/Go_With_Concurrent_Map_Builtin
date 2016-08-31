package iterator_testing

import (
	"settings"
	"strconv"

	cmap "github.com/streamrail/concurrent-map"
	"github.com/zond/gotomic"
)

func ConcurrentIterator_RO(nGoroutines int64) int64 {
	cmap := make(map[int64]settings.Unused, settings.ITERATOR_NUM_ELEMS, nGoroutines)

	// Initialize the map with a fixed number of elements.
	for i := int64(0); i < settings.ITERATOR_NUM_ELEMS; i++ {
		cmap[i] = settings.UNUSED
	}

	// Begin iteration test
	return settings.ParallelTest(int(nGoroutines), func() {
		for i := uint64(0); i < settings.ITERATOR_NUM_ITERATIONS; i++ {
			for k, v := range cmap {
				k++
				v++
			}
		}
	}).Nanoseconds() / int64(int64(nGoroutines)*settings.ITERATOR_NUM_ELEMS*int64(settings.ITERATOR_NUM_ITERATIONS))
}

func StreamrailConcurrentIterator_RO(nGoroutines int64) int64 {
	scmap := cmap.New()

	// Initialize the map with a fixed number of elements.
	for i := int64(0); i < settings.ITERATOR_NUM_ELEMS; i++ {
		scmap.Set(strconv.FormatInt(i, 10), settings.UNUSED)
	}

	// Begin iteration test
	return settings.ParallelTest(int(nGoroutines), func() {
		for i := uint64(0); i < settings.ITERATOR_NUM_ITERATIONS; i++ {
			for item := range scmap.Iter() {
				_ = item.Key
				_ = item.Val
			}
		}
	}).Nanoseconds() / int64(int64(nGoroutines)*settings.ITERATOR_NUM_ELEMS*int64(settings.ITERATOR_NUM_ITERATIONS))
}

func GotomicConcurrentIterator_RO(nGoroutines int64) int64 {
	gcmap := gotomic.NewHash()

	// Initialize the map with a fixed number of elements.
	for i := int64(0); i < settings.ITERATOR_NUM_ELEMS; i++ {
		gcmap.Put(gotomic.IntKey(int(i)), settings.UNUSED)
	}

	// Begin iteration test
	return settings.ParallelTest(int(nGoroutines), func() {
		for i := uint64(0); i < settings.ITERATOR_NUM_ITERATIONS; i++ {
			gcmap.Each(func(k gotomic.Hashable, v gotomic.Thing) bool {
				_ = k
				_ = v
				return false
			})
		}
	}).Nanoseconds() / int64(int64(nGoroutines)*settings.ITERATOR_NUM_ELEMS*int64(settings.ITERATOR_NUM_ITERATIONS))
}

func DefaultIterator_RO(nGoroutines int64) int64 {
	smap := make(map[int64]settings.Unused, settings.ITERATOR_NUM_ELEMS)

	// Initialize the map with a fixed number of elements.
	for i := int64(0); i < settings.ITERATOR_NUM_ELEMS; i++ {
		smap[i] = settings.UNUSED
	}

	// Begin iteration test
	return settings.ParallelTest(int(nGoroutines), func() {
		for i := uint64(0); i < settings.ITERATOR_NUM_ITERATIONS; i++ {
			for k, v := range smap {
				k++
				v++
			}
		}
	}).Nanoseconds() / int64(int64(nGoroutines)*settings.ITERATOR_NUM_ELEMS*int64(settings.ITERATOR_NUM_ITERATIONS))
}
