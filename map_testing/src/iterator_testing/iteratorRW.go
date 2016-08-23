package iterator_testing

import (
	"settings"
	"sync"
)

type T struct {
	iter int
}

func ConcurrentIterator_RW(nGoroutines int64) int64 {
	cmap := make(map[int64]T, settings.ITERATOR_NUM_ELEMS, nGoroutines)

	// Initialize the map with a fixed number of elements.
	for i := int64(0); i < settings.ITERATOR_NUM_ELEMS; i++ {
		cmap[i] = T{}
	}

	// Begin iteration test
	return settings.ParallelTest(int(nGoroutines), func() {
		for i := uint64(0); i < settings.ITERATOR_NUM_ITERATIONS; i++ {
			for k, v := range cmap {
				v.iter++
				// Read-Modify-Write operation to increment struct field atomically
				t := cmap[k]
				t.iter++
				cmap[k] = t
			}
		}
	}).Nanoseconds() / int64(int64(nGoroutines)*settings.ITERATOR_NUM_ELEMS*int64(settings.ITERATOR_NUM_ITERATIONS))
}

func SynchronizedIterator_RW(nGoroutines int64) int64 {
	smap := make(map[int64]T, settings.ITERATOR_NUM_ELEMS)
	var mtx sync.Mutex

	// Initialize the map with a fixed number of elements.
	for i := int64(0); i < settings.ITERATOR_NUM_ELEMS; i++ {
		smap[i] = T{}
	}

	// Begin iteration test
	return settings.ParallelTest(int(nGoroutines), func() {
		for i := uint64(0); i < settings.ITERATOR_NUM_ITERATIONS; i++ {
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
	}).Nanoseconds() / int64(int64(nGoroutines)*settings.ITERATOR_NUM_ELEMS*int64(settings.ITERATOR_NUM_ITERATIONS))
}

func ReaderWriterIterator_RW(nGoroutines int64) int64 {
	rwmap := make(map[int64]T, settings.ITERATOR_NUM_ELEMS)
	var mtx sync.RWMutex

	// Initialize the map with a fixed number of elements.
	for i := int64(0); i < settings.ITERATOR_NUM_ELEMS; i++ {
		rwmap[i] = T{}
	}

	// Begin iteration test
	return settings.ParallelTest(int(nGoroutines), func() {
		for i := uint64(0); i < settings.ITERATOR_NUM_ITERATIONS; i++ {
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
	}).Nanoseconds() / int64(int64(nGoroutines)*settings.ITERATOR_NUM_ELEMS*int64(settings.ITERATOR_NUM_ITERATIONS))
}

func ChannelIterator_RW(nGoroutines int64) int64 {
	chmap := make(map[int64]T, settings.ITERATOR_NUM_ELEMS)
	ch := make(chan int64, settings.ITERATOR_NUM_ELEMS)
	var done sync.WaitGroup
	done.Add(1)

	// Initialize the map with a fixed number of elements.
	for i := int64(0); i < settings.ITERATOR_NUM_ELEMS; i++ {
		chmap[i] = T{}
	}

	// Sentinel Goroutine
	go func() {
		workingGoroutines := nGoroutines
		for {
			select {
			case x := <-ch:
				if x == -1 {
					workingGoroutines--
					if workingGoroutines == 0 {
						done.Done()
						return
					}
				}

				// Read-Modify-Write operation to increment struct field atomically
				t := chmap[x]
				t.iter++
				chmap[x] = t
			}
		}
	}()

	// Begin iteration test
	return settings.ParallelTest(int(nGoroutines), func() {
		// Note that this is a special case wherein keys are simple integers, hence we are giving the map an unfair advantage here. Since we do not
		// need to iterate the map for all keys, we can generate each key easily in a function, then pass to the channel the keys to be incremented.
		// This is not realistic, however I deem it is necessary to prove how much better the concurrent map is over channels
		for i := int64(0); i < settings.ITERATOR_NUM_ELEMS; i++ {
			ch <- i
		}
		// Signify we are done (another special case)
		ch <- -1
		done.Wait()
	}).Nanoseconds() / int64(settings.ITERATOR_NUM_ELEMS*int64(settings.ITERATOR_NUM_ITERATIONS))
}
