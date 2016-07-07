package map_testing

import "sync"
import "fmt"
import "math/rand"
import "time"
import _ "../github.com/pkg/profile"

var rwlock sync.RWMutex

var rwc chan int

func populate_map_rw_struct(m map[point]point) {
	for i := 0; i < ROWS; i++ {
		go func (idx int) { 
			for j := 0; j < COLS; j++ {
				// val := fmt.Sprintf("{%v, %v}", idx, j)
				key, val := point{idx, j}, point{ROWS - idx, COLS - j}
				rwlock.Lock()
				// Adds a dummy value to test assignment
				m[key] = point{0, 0}
				// Adds real value to test update
				m[key] = val
				rwlock.Unlock()
			}
			c <- 0
		}(i)
	}
}

func delete_map_rw_struct(m map[point]point) {
	for i := 0; i < ROWS; i++ {
		go func (idx int) {
			for j := 0; j < COLS; j++ {
				key := point{idx, j}
				rwlock.Lock()
				delete(m, key)
				rwlock.Unlock()
			}
			c <- 0
		}(i)
	}
}

func iterate_map_rw_struct(m map[point]point) {
	for i := 0; i < ROWS; i++ {
		go func () {
			rwlock.RLock()
			for k, v := range m {
				expected := point{ROWS - k.x, COLS - k.y}
				if v != expected {
					panic(fmt.Sprintf("Expected %v for key %v, but received %v", expected, k, v))
				}
			}
			rwlock.RUnlock()
			c <- 0
		}()
	}
}

func test_map_insertion_accuracy_rw(m map[point]point) {
    passed := true
    c := make(chan int)
	for i := 0; i < ROWS; i++ {
        go func (idx int) {
            for j := 0; j < COLS; j++ {
                key := point{idx, j}
                val := point{ROWS - idx, COLS - j}
				rwlock.RLock()
                retval := m[key]
				rwlock.RUnlock()
                if retval != val {
                    fmt.Printf("Key: %v;Expected: %v;Received: %v", key, val, retval)
                    passed = false
                }
            }
            c <- 0
        }(i)
	}

    for i := 0; i < ROWS; i++ {
        <- c
    }

    if !passed {
        panic("Failed Insertion Accuracy Test!!!")
    }
}

func test_map_deletion_accuracy_rw(m map[point]point) {
    passed := true
    c := make(chan int)
    for i := 0; i < ROWS; i++ {
        go func (idx int) {
            for j := 0; j < COLS; j++ {
                key := point{idx, j}
                val := point{0, 0}
				rwlock.Lock()
                retval := m[key]
			    rwlock.Unlock()
                if retval != val {
                    fmt.Printf("Key: %v;Expected: %v;Received: %v", key, val, retval)
                    passed = false
                }
            }
            c <- 0
        }(i)
	}

    for i := 0; i < ROWS; i++ {
        <- c
    }

    if !passed {
        panic("Failed Deletion Accuracy Test!!!")
    }
}

func all_map_struct_rw(m map[point]point) {
	iteration_modulo, retrieve_modulo, delete_modulo := 103, 5, 7

	// We spawn ROWS Goroutines
	for i := 0; i < ROWS; i++ {
		// The current 'i' when the Goroutine is spawned determines the points it generates
		go func(idx int) {
			adds, retrieves, deletes, iterations := 0, 0, 0, 0
			var timeAdding, timeRetrieving, timeDeleting, timeIterating time.Duration
			lastAdded := -1
			for j := 0; j < COLS; j++ {
				r := rand.Int()
				// The operation is done at random
				start := time.Now()
				if r % iteration_modulo == 0 {
					rwlock.RLock()
					for k, v := range m {
						expected := point{ROWS - k.x, COLS - k.y}
						if v != expected {
							panic(fmt.Sprintf("[ReaderWriter Map] Expected %v for key %v, but received %v", expected, k, v))
						}
					}
					timeIterating += time.Since(start)
					iterations++
					rwlock.RUnlock()
				} else if r % retrieve_modulo == 0 {
					rwlock.RLock()
					key := point{idx, lastAdded}
					expected := point{0, 0}
					retval := m[key]
					// -1 means uninitialized...
					if lastAdded != -1 {
						expected = point{ROWS - idx, COLS - lastAdded}
					}
					if retval != expected {
						panic(fmt.Sprintf("[ReaderWriter Map] Key: %v;Expected: %v;Found: %v\n", key, expected, retval))
					}
					timeRetrieving += time.Since(start)
					retrieves++
					rwlock.RUnlock()
				} else if r % delete_modulo == 0 {
					rwlock.Lock()
					if lastAdded != -1 {
						delete(m, point{ROWS - idx, COLS - lastAdded})
						deletes++
					}
					timeDeleting += time.Since(start)
					lastAdded = -1
					rwlock.Unlock()
				} else {
					rwlock.Lock()
					k := point{idx, j}
					v := point{ROWS - idx, COLS - j}
					m[k] = v
					lastAdded = j
					timeAdding += time.Since(start)
					adds++
					rwlock.Unlock()
				}
			}
			fmt.Printf("[ReaderWriter Map]\nInsertion { Time: %v; Operations: %v }\nRetrieve { Time: %v; Operations: %v }\nDelete { Time: %v; Operations: %v }\nIteration { Time: %v; Operations: %v }\n\n\n",
				timeAdding, adds, timeRetrieving, retrieves, timeDeleting, deletes, timeIterating, iterations)
			c <- 0
		}(i)
	}
}

func TestRWLockMap() {
	m := make(map[point]point)
	c = make(chan int)
	insertTime := make([]time.Duration, TESTS)
	deleteTime := make([]time.Duration, TESTS)
	retrieveTime := make([]time.Duration, TESTS * 2)
	iterationTime := make([]time.Duration, TESTS)

	all_map_struct_rw(m)
    for i := 0; i < ROWS; i++ {
        <- c
    }
	fmt.Printf("[ReaderWriter Map] Successfully completed the all_map_struct test!")
	if TESTS == 0 {
		return
	}
	
	for iterations := 0; iterations < TESTS; iterations++ {
		// log.Printf("[ReaderWriter Map] ~Trial #%v~ Populating with %v elements...", iterations + 1, ROWS * COLS)
		start := time.Now()
		populate_map_rw_struct(m)
		for i := 0; i < ROWS; i++ {
			<- c
		}

		end := time.Since(start)
		insertTime[iterations] = end
		// memProf.Stop()
		// prof.Stop()
		// log.Printf("[ReaderWriter Map] ~Trial #%v~ Insertion Time: %v\n", iterations + 1, end)

		// log.Printf("[ReaderWriter Map] ~Trial #%v~ Testing Insertion Accuracy...", iterations + 1)
		start = time.Now()
		test_map_insertion_accuracy_rw(m)
		end = time.Since(start)
		retrieveTime[iterations * 2] = end

		start = time.Now()
		iterate_map_rw_struct(m)
		for i := 0; i < ROWS; i++ {
			<- c
		}
		end = time.Since(start)
		iterationTime[iterations] = end

		// log.Printf("[ReaderWriter Map] ~Trial #%v~ Deleting all %v elements...", iterations + 1, ROWS * COLS)
		start = time.Now()
		delete_map_rw_struct(m)
		for i := 0; i < ROWS; i++ {
			<- c
		}
		end = time.Since(start)
		deleteTime[iterations] = end

		// log.Printf("[ReaderWriter Map] ~Trial #%v~ Testing Deletion Accuracy...", iterations + 1)
		start = time.Now()
		test_map_deletion_accuracy_rw(m)
		end = time.Since(start)
		retrieveTime[(iterations * 2) + 1] = end
	}
	fmt.Printf("[ReaderWriter Map] Average Insertion Time: %v\n", averageTime(insertTime))
	fmt.Printf("[ReaderWriter Map] Average Retrieval Time: %v\n", averageTime(retrieveTime))
	fmt.Printf("[ReaderWriter Map] Average Deletion Time: %v\n", averageTime(deleteTime))
	fmt.Printf("[ReaderWriter Map] Average Iteration Time: %v\n", averageTime(iterationTime))
}