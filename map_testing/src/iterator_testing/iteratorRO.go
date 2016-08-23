package iterator_testing

import (
	"settings"
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
