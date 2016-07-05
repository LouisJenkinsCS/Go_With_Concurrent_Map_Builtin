package map_testing

import _ "../github.com/pkg/profile"
import "fmt"
import "time"

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

func TestConcurrentMap() {
	m := make(map[point]point, 0, 1)
	c = make(chan int)
	insertTime := make([]time.Duration, TESTS)
	deleteTime := make([]time.Duration, TESTS)
	retrieveTime := make([]time.Duration, TESTS * 2)

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
		// log.Printf("[Concurrent Map] ~Trial #%v~ Insertion Time: %v\n", iterations + 1, end)

		// log.Printf("[Concurrent Map] ~Trial #%v~ Testing Insertion Accuracy...", iterations + 1)
		start = time.Now()
		test_map_insertion_accuracy(m)
		end = time.Since(start)
		retrieveTime[iterations * 2] = end

		// log.Printf("[Concurrent Map] ~Trial #%v~ Deleting all %v elements...", iterations + 1, ROWS * COLS)
		start = time.Now()
		delete_map_struct(m)
		for i := 0; i < ROWS; i++ {
			<- c
		}
		end = time.Since(start)
		deleteTime[iterations] = end

		// log.Printf("[Concurrent Map] ~Trial #%v~ Testing Deletion Accuracy...", iterations + 1)
		start = time.Now()
		test_map_deletion_accuracy(m)
		end = time.Since(start)
		retrieveTime[(iterations * 2) + 1] = end
	}
	fmt.Printf("[Concurrent Map] Average Insertion Time: %v\n", averageTime(insertTime))
	fmt.Printf("[Concurrent Map] Average Retrieval Time: %v\n", averageTime(retrieveTime))
	fmt.Printf("[Concurrent Map] Average Deletion Time: %v\n", averageTime(deleteTime))
}