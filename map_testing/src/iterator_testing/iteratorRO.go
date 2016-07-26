package iterator_testing

import (
    "settings"
)

func ConcurrentIterator_RO(nGoroutines int) int64 {
    cmap := make(map[int64]settings.Unused, 0, 1)
    
    // Initialize the map with a fixed number of elements.
    for i := int64(0); i < settings.ITERATOR_NUM_ELEMS; i++ {
        cmap[i] = settings.UNUSED
    }

    // Begin iteration test
    return settings.ParallelTest(nGoroutines, func() {
        for k, v := range cmap {
            k++
            v++
        }
    }).Nanoseconds() / int64(settings.ITERATOR_NUM_ELEMS)
}

func ConcurrentIterator_Interlocked_RO(nGoroutines int) int64 {
    cmap := make(map[int64]settings.Unused, 0, 1)
    
    // Initialize the map with a fixed number of elements.
    for i := int64(0); i < settings.ITERATOR_NUM_ELEMS; i++ {
        cmap[i] = settings.UNUSED
    }

    // Begin iteration test
    return settings.ParallelTest(nGoroutines, func() {
        for k, v := range sync.Interlocked cmap {
            k++
            v++
        }
    }).Nanoseconds() / int64(settings.ITERATOR_NUM_ELEMS)
}

func DefaultIterator_RO(nGoroutines int) int64 {
    smap := make(map[int64]settings.Unused)

    
    // Initialize the map with a fixed number of elements.
    for i := int64(0); i < settings.ITERATOR_NUM_ELEMS; i++ {
        smap[i] = settings.UNUSED
    }

    // Begin iteration test
    return settings.ParallelTest(nGoroutines, func() {
        for k, v := range smap {
            k++
            v++
        }
    }).Nanoseconds() / int64(settings.ITERATOR_NUM_ELEMS)
}