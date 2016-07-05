package map_testing

import "sync"
import "fmt"
import "time"
import _ "../github.com/pkg/profile"

var mtx sync.Mutex

var c chan int

func populate_map_sync_struct(m map[point]point) {
	for i := 0; i < ROWS; i++ {
		go func (idx int) { 
			for j := 0; j < COLS; j++ {
				// val := fmt.Sprintf("{%v, %v}", idx, j)
				key, val := point{idx, j}, point{ROWS - idx, COLS - j}
				mtx.Lock()
				// Adds a dummy value to test assignment
				m[key] = point{0, 0}
				// Adds real value to test update
				m[key] = val
				mtx.Unlock()
			}
			c <- 0
		}(i)
	}
}

func delete_map_sync_struct(m map[point]point) {
	for i := 0; i < ROWS; i++ {
		go func (idx int) {
			for j := 0; j < COLS; j++ {
				key := point{idx, j}
				mtx.Lock()
				delete(m, key)
				mtx.Unlock()
			}
			c <- 0
		}(i)
	}
}


func TestDefaultMap() {
	m := make(map[point]point)
	c = make(chan int)
	insertTime := make([]time.Duration, TESTS)
	deleteTime := make([]time.Duration, TESTS)
	retrieveTime := make([]time.Duration, TESTS * 2)
	
	for iterations := 0; iterations < TESTS; iterations++ {
		// log.Printf("[Synchronized Map] ~Trial #%v~ Populating with %v elements...", iterations + 1, ROWS * COLS)
		start := time.Now()
		populate_map_sync_struct(m)
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
		test_map_insertion_accuracy(m)
		end = time.Since(start)
		retrieveTime[iterations * 2] = end

		// log.Printf("[Synchronized Map] ~Trial #%v~ Deleting all %v elements...", iterations + 1, ROWS * COLS)
		start = time.Now()
		delete_map_sync_struct(m)
		for i := 0; i < ROWS; i++ {
			<- c
		}
		end = time.Since(start)
		deleteTime[iterations] = end

		// log.Printf("[Synchronized Map] ~Trial #%v~ Testing Deletion Accuracy...", iterations + 1)
		start = time.Now()
		test_map_deletion_accuracy(m)
		end = time.Since(start)
		retrieveTime[(iterations * 2) + 1] = end
	}
	fmt.Printf("[Synchronized Map] Average Insertion Time: %v\n", averageTime(insertTime))
	fmt.Printf("[Synchronized Map] Average Retrieval Time: %v\n", averageTime(retrieveTime))
	fmt.Printf("[Synchronized Map] Average Deletion Time: %v\n", averageTime(deleteTime))
}