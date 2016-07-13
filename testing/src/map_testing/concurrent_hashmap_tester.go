package map_testing

import _ "../github.com/pkg/profile"
import "fmt"
import "time"
import "math/rand"

func populate_map_struct(m map[point]point) {
	for i := 0; i < ROWS; i++ {
		go func (idx int) { 
			for j := 0; j < COLS; j++ {
				key, val := point{idx, j}, point{ROWS - idx, COLS - j}
				m[key] = point{0, 0}
				m[key] = val
			}
			c <- 0
		}(i)
	}
}

func delete_map_struct(m map[point]point) {
	for i := 0; i < ROWS; i++ {
		go func (idx int) {
			for j := 0; j < COLS; j++ {
				key := point{idx, j}
				delete(m, key)
			}
			c <- 0
		}(i)
	}
}

func iterate_map_struct(m map[point]point) {
	for i := 0; i < ROWS; i++ {
		go func () {
			// j := 0
			for k, v := range m {
				expected := point{ROWS - k.x, COLS - k.y}
				if v != expected {
					panic(fmt.Sprintf("[Concurrent Map] Expected %v for key %v, but received %v", expected, k, v))
				}
				// j++
			}
			// fmt.Printf("%v correct...", j)
			c <- 0
		}()
	}
}

func all_map_struct(m map[point]point) {
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
					for k, v := range m {
						expected := point{ROWS - k.x, COLS - k.y}
						if v != expected {
							panic(fmt.Sprintf("[Concurrent Map] Expected %v for key %v, but received %v", expected, k, v))
						}
					}
					timeIterating += time.Since(start)
					iterations++
				} else if r % retrieve_modulo == 0 {
					key := point{idx, lastAdded}
					expected := point{0, 0}
					retval := m[key]
					// -1 means uninitialized...
					if lastAdded != -1 {
						expected = point{ROWS - idx, COLS - lastAdded}
					}
					if retval != expected {
						panic(fmt.Sprintf("[Concurrent Map] Key: %v;Expected: %v;Found: %v\n", key, expected, retval))
					}
					timeRetrieving += time.Since(start)
					retrieves++
				} else if r % delete_modulo == 0 {
					if lastAdded != -1 {
						delete(m, point{ROWS - idx, COLS - lastAdded})
						deletes++
					}
					lastAdded = -1
					timeDeleting += time.Since(start)
				} else {
					k := point{idx, j}
					v := point{ROWS - idx, COLS - j}
					m[k] = v
					lastAdded = j
					timeAdding += time.Since(start)
					adds++
				}
			}
			fmt.Printf("[Concurrent Map]\nInsertion { Time: %v; Operations: %v }\nRetrieve { Time: %v; Operations: %v }\nDelete { Time: %v; Operations: %v }\nIteration { Time: %v; Operations: %v }\n\n\n",
				timeAdding, adds, timeRetrieving, retrieves, timeDeleting, deletes, timeIterating, iterations)
			c <- 0
		}(i)
	}
}

func TestConcurrentMap() {
	m := make(map[point]point, 0, 1)
	c = make(chan int)
	insertTime := make([]time.Duration, TESTS)
	deleteTime := make([]time.Duration, TESTS)
	retrieveTime := make([]time.Duration, TESTS * 2)
	iterationTime := make([]time.Duration, TESTS)
	rand.Seed(time.Now().UTC().UnixNano())

	// prof := profile.Start(profile.CPUProfile, profile.ProfilePath("."), profile.NoShutdownHook)
	all_map_struct(m)
    for i := 0; i < ROWS; i++ {
        <- c
    }
	fmt.Printf("[Concurrent Map] Successfully completed the all_map_struct test!\n")
	if TESTS == 0 {
		return
	}

	for iterations := 0; iterations < TESTS; iterations++ {
		// log.Printf("[Concurrent Map] ~Trial #%v~ Populating with %v elements...", iterations + 1, ROWS * COLS)
		start := time.Now()
		populate_map_struct(m)
		for i := 0; i < ROWS; i++ {
			<- c
		}

		end := time.Since(start)
		insertTime[iterations] = end
		// memProf.Stop()
		// prof.Stop()
		// fmt.Printf("[Concurrent Map] ~Trial #%v~ Insertion Time: %v\n", iterations + 1, end)

		// fmt.Printf("[Concurrent Map] ~Trial #%v~ Testing Insertion Accuracy...", iterations + 1)
		start = time.Now()
		test_map_insertion_accuracy(m)
		end = time.Since(start)
		retrieveTime[iterations * 2] = end

		start = time.Now()
		iterate_map_struct(m)
		for i := 0; i < ROWS; i++ {
			<- c
		}
		end = time.Since(start)
		iterationTime[iterations] = end

		// fmt.Printf("[Concurrent Map] ~Trial #%v~ Deleting all %v elements...", iterations + 1, ROWS * COLS)
		start = time.Now()
		delete_map_struct(m)
		for i := 0; i < ROWS; i++ {
			<- c
		}
		end = time.Since(start)
		deleteTime[iterations] = end

		// fmt.Printf("[Concurrent Map] ~Trial #%v~ Testing Deletion Accuracy...", iterations + 1)
		start = time.Now()
		test_map_deletion_accuracy(m)
		end = time.Since(start)
		retrieveTime[(iterations * 2) + 1] = end
	}
	// prof.Stop()
	fmt.Printf("[Concurrent Map] Average Insertion Time: %v\n", averageTime(insertTime))
	fmt.Printf("[Concurrent Map] Average Retrieval Time: %v\n", averageTime(retrieveTime))
	fmt.Printf("[Concurrent Map] Average Deletion Time: %v\n", averageTime(deleteTime))
	fmt.Printf("[Concurrent Map] Average Iteration Time: %v\n", averageTime(iterationTime))
}