package map_testing

import "sync"
import "fmt"
import "time"
import "math/rand"
import _ "../github.com/pkg/profile"

var mtx sync.Mutex
var default_map map[point]point

var c chan int

func populate_map_sync_struct() {
	for i := 0; i < ROWS; i++ {
		go func (idx int) { 
			for j := 0; j < COLS; j++ {
				// val := fmt.Sprintf("{%v, %v}", idx, j)
				key, val := point{idx, j}, point{ROWS - idx, COLS - j}
				mtx.Lock()
				// Adds a dummy value to test assignment
				default_map[key] = point{0, 0}
				// Adds real value to test update
				default_map[key] = val
				mtx.Unlock()
			}
			c <- 0
		}(i)
	}
}

func delete_map_sync_struct() {
	for i := 0; i < ROWS; i++ {
		go func (idx int) {
			for j := 0; j < COLS; j++ {
				key := point{idx, j}
				mtx.Lock()
				delete(default_map, key)
				mtx.Unlock()
			}
			c <- 0
		}(i)
	}
}

func iterate_map_sync_struct() {
	for i := 0; i < ROWS; i++ {
		go func () {
			mtx.Lock()
			for k, v := range default_map {
				expected := point{ROWS - k.x, COLS - k.y}
				if v != expected {
					panic(fmt.Sprintf("Expected %v for key %v, but received %v", expected, k, v))
				}
			}
			mtx.Unlock()
			c <- 0
		}()
	}
}

func test_map_insertion_accuracy_sync() {
    passed := true
    c := make(chan int)
	for i := 0; i < ROWS; i++ {
        go func (idx int) {
            for j := 0; j < COLS; j++ {
                key := point{idx, j}
                val := point{ROWS - idx, COLS - j}
				mtx.Lock()
                retval := default_map[key]
				mtx.Unlock()
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

func test_map_deletion_accuracy_sync() {
    passed := true
    c := make(chan int)
    for i := 0; i < ROWS; i++ {
        go func (idx int) {
            for j := 0; j < COLS; j++ {
                key := point{idx, j}
                val := point{0, 0}
				mtx.Lock()
                retval := default_map[key]
				mtx.Unlock()
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

func all_map_struct_sync() {
	default_map = make(map[point]point)
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
				mtx.Lock()
				if r % iteration_modulo == 0 {
					for k, v := range default_map {
						expected := point{ROWS - k.x, COLS - k.y}
						if v != expected {
							panic(fmt.Sprintf("[Synchronized Map] Expected %v for key %v, but received %v", expected, k, v))
						}
						nopFunction(k, v, 1)
					}
					mtx.Unlock()
					timeIterating += time.Since(start)
					iterations++
				} else if r % retrieve_modulo == 0 {
					key := point{idx, lastAdded}
					expected := point{0, 0}
					retval := default_map[key]
					// -1 means uninitialized...
					if lastAdded != -1 {
						expected = point{ROWS - idx, COLS - lastAdded}
					}
					if retval != expected {
						panic(fmt.Sprintf("[Synchronized Map] Key: %v;Expected: %v;Found: %v\n", key, expected, retval))
					}
					mtx.Unlock()
					timeRetrieving += time.Since(start)
					retrieves++
				} else if r % delete_modulo == 0 {
					if lastAdded != -1 {
						delete(default_map, point{ROWS - idx, COLS - lastAdded})
						deletes++
					}
					mtx.Unlock()
					timeDeleting += time.Since(start)
					lastAdded = -1
				} else {
					k := point{idx, j}
					v := point{ROWS - idx, COLS - j}
					default_map[k] = v
					lastAdded = j
					mtx.Unlock()
					timeAdding += time.Since(start)
					adds++
				}
			}
			fmt.Printf("[Synchronized Map]\nInsertion { Time: %v; Operations: %v }\nRetrieve { Time: %v; Operations: %v }\nDelete { Time: %v; Operations: %v }\nIteration { Time: %v; Operations: %v }\n\n\n",
				timeAdding, adds, timeRetrieving, retrieves, timeDeleting, deletes, timeIterating, iterations)
			c <- 0
		}(i)
	}
}

func TestDefaultMap() {
	c = make(chan int)
	insertTime := make([]time.Duration, TESTS)
	deleteTime := make([]time.Duration, TESTS)
	retrieveTime := make([]time.Duration, TESTS * 2)
	iterationTime := make([]time.Duration, TESTS)

	all_map_struct_sync()
    for i := 0; i < ROWS; i++ {
        <- c
    }
	fmt.Printf("[Synchronized Map] Successfully completed the all_map_struct test!")
	if TESTS == 0 {
		return
	}
	
	for iterations := 0; iterations < TESTS; iterations++ {
		// log.Printf("[Synchronized Map] ~Trial #%v~ Populating with %v elements...", iterations + 1, ROWS * COLS)
		start := time.Now()
		populate_map_sync_struct()
		for i := 0; i < ROWS; i++ {
			<- c
		}

		end := time.Since(start)
		insertTime[iterations] = end
		// memProf.Stop()
		// prof.Stop()
		// log.Printf("[Synchronized Map] ~Trial #%v~ Insertion Time: %v\n", iterations + 1, end)

		// log.Printf("[Synchronized Map] ~Trial #%v~ Testing Insertion Accuracy...", iterations + 1)
		start = time.Now()
		test_map_insertion_accuracy_sync()
		end = time.Since(start)
		retrieveTime[iterations * 2] = end

		start = time.Now()
		iterate_map_sync_struct()
		for i := 0; i < ROWS; i++ {
			<- c
		}
		end = time.Since(start)
		iterationTime[iterations] = end

		// log.Printf("[Synchronized Map] ~Trial #%v~ Deleting all %v elements...", iterations + 1, ROWS * COLS)
		start = time.Now()
		delete_map_sync_struct()
		for i := 0; i < ROWS; i++ {
			<- c
		}
		end = time.Since(start)
		deleteTime[iterations] = end

		// log.Printf("[Synchronized Map] ~Trial #%v~ Testing Deletion Accuracy...", iterations + 1)
		start = time.Now()
		test_map_deletion_accuracy_sync()
		end = time.Since(start)
		retrieveTime[(iterations * 2) + 1] = end
	}
	fmt.Printf("[Synchronized Map] Average Insertion Time: %v\n", averageTime(insertTime))
	fmt.Printf("[Synchronized Map] Average Retrieval Time: %v\n", averageTime(retrieveTime))
	fmt.Printf("[Synchronized Map] Average Deletion Time: %v\n", averageTime(deleteTime))
	fmt.Printf("[Synchronized Map] Average Iteration Time: %v\n", averageTime(iterationTime))
}