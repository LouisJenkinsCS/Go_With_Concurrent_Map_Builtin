package map_testing

import "sync"
import "fmt"
import "math/rand"
import "time"
import _ "../github.com/pkg/profile"

var rwlock sync.RWMutex
var rw_map map[point]point

var rwc chan int

func populate_map_rw_struct() {
	for i := 0; i < ROWS; i++ {
		go func (idx int) { 
			for j := 0; j < COLS; j++ {
				// val := fmt.Sprintf("{%v, %v}", idx, j)
				key, val := point{idx, j}, point{ROWS - idx, COLS - j}
				rwlock.Lock()
				// Adds a dummy value to test assignment
				rw_map[key] = point{0, 0}
				// Adds real value to test update
				rw_map[key] = val
				rwlock.Unlock()
			}
			c <- 0
		}(i)
	}
}

func delete_map_rw_struct() {
	for i := 0; i < ROWS; i++ {
		go func (idx int) {
			for j := 0; j < COLS; j++ {
				key := point{idx, j}
				rwlock.Lock()
				delete(rw_map, key)
				rwlock.Unlock()
			}
			c <- 0
		}(i)
	}
}

func iterate_map_rw_struct() {
	for i := 0; i < ROWS; i++ {
		go func () {
			rwlock.RLock()
			for k, v := range rw_map {
				expected := point{ROWS - k.x, COLS - k.y}
				if v != expected {
					panic(fmt.Sprintf("Expected %v for key %v, but received %v", expected, k, v))
				}
				nopFunction(k, v, 1)
			}
			rwlock.RUnlock()
			c <- 0
		}()
	}
}

func test_map_insertion_accuracy_rw() {
    passed := true
    c := make(chan int)
	for i := 0; i < ROWS; i++ {
        go func (idx int) {
            for j := 0; j < COLS; j++ {
                key := point{idx, j}
                val := point{ROWS - idx, COLS - j}
				rwlock.RLock()
                retval := rw_map[key]
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

func test_map_deletion_accuracy_rw() {
    passed := true
    c := make(chan int)
    for i := 0; i < ROWS; i++ {
        go func (idx int) {
            for j := 0; j < COLS; j++ {
                key := point{idx, j}
                val := point{0, 0}
				rwlock.Lock()
                retval := rw_map[key]
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

func all_map_struct_rw() {
	rw_map = make(map[point]point)
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
					for k, v := range rw_map {
						expected := point{ROWS - k.x, COLS - k.y}
						if v != expected {
							panic(fmt.Sprintf("[ReaderWriter Map] Expected %v for key %v, but received %v", expected, k, v))
						}
					}
					rwlock.RUnlock()
					timeIterating += time.Since(start)
					iterations++
				} else if r % retrieve_modulo == 0 {
					rwlock.RLock()
					key := point{idx, lastAdded}
					expected := point{0, 0}
					retval := rw_map[key]
					// -1 means uninitialized...
					if lastAdded != -1 {
						expected = point{ROWS - idx, COLS - lastAdded}
					}
					if retval != expected {
						panic(fmt.Sprintf("[ReaderWriter Map] Key: %v;Expected: %v;Found: %v\n", key, expected, retval))
					}
					rwlock.RUnlock()
					timeRetrieving += time.Since(start)
					retrieves++
				} else if r % delete_modulo == 0 {
					rwlock.Lock()
					if lastAdded != -1 {
						delete(rw_map, point{ROWS - idx, COLS - lastAdded})
						deletes++
					}
					rwlock.Unlock()
					timeDeleting += time.Since(start)
					lastAdded = -1
				} else {
					rwlock.Lock()
					k := point{idx, j}
					v := point{ROWS - idx, COLS - j}
					rw_map[k] = v
					lastAdded = j
					rwlock.Unlock()
					timeAdding += time.Since(start)
					adds++
				}
			}
			fmt.Printf("[ReaderWriter Map]\nInsertion { Time: %v; Operations: %v }\nRetrieve { Time: %v; Operations: %v }\nDelete { Time: %v; Operations: %v }\nIteration { Time: %v; Operations: %v }\n\n\n",
				timeAdding, adds, timeRetrieving, retrieves, timeDeleting, deletes, timeIterating, iterations)
			c <- 0
		}(i)
	}
}

func TestRWLockMap() {
	c = make(chan int)
	insertTime := make([]time.Duration, TESTS)
	deleteTime := make([]time.Duration, TESTS)
	retrieveTime := make([]time.Duration, TESTS * 2)
	iterationTime := make([]time.Duration, TESTS)

	all_map_struct_rw()
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
		populate_map_rw_struct()
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
		test_map_insertion_accuracy_rw()
		end = time.Since(start)
		retrieveTime[iterations * 2] = end

		start = time.Now()
		iterate_map_rw_struct()
		for i := 0; i < ROWS; i++ {
			<- c
		}
		end = time.Since(start)
		iterationTime[iterations] = end

		// log.Printf("[ReaderWriter Map] ~Trial #%v~ Deleting all %v elements...", iterations + 1, ROWS * COLS)
		start = time.Now()
		delete_map_rw_struct()
		for i := 0; i < ROWS; i++ {
			<- c
		}
		end = time.Since(start)
		deleteTime[iterations] = end

		// log.Printf("[ReaderWriter Map] ~Trial #%v~ Testing Deletion Accuracy...", iterations + 1)
		start = time.Now()
		test_map_deletion_accuracy_rw()
		end = time.Since(start)
		retrieveTime[(iterations * 2) + 1] = end
	}
	fmt.Printf("[ReaderWriter Map] Average Insertion Time: %v\n", averageTime(insertTime))
	fmt.Printf("[ReaderWriter Map] Average Retrieval Time: %v\n", averageTime(retrieveTime))
	fmt.Printf("[ReaderWriter Map] Average Deletion Time: %v\n", averageTime(deleteTime))
	fmt.Printf("[ReaderWriter Map] Average Iteration Time: %v\n", averageTime(iterationTime))
}