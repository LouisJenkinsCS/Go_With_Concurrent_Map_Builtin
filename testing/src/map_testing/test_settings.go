package map_testing

import "log"
import "time"

type point struct {
	x int
	y int
}

const (
    ROWS = 32
    COLS = 10000
    TESTS = 100
)

func test_map_insertion_accuracy(m map[point]point) {
    passed := true
    c := make(chan int)
	for i := 0; i < ROWS; i++ {
        go func (idx int) {
            for j := 0; j < COLS; j++ {
                key := point{idx, j}
                val := point{ROWS - idx, COLS - j}
                retval := m[key]
                if retval != val {
                    log.Printf("Key: %v;Expected: %v;Received: %v", key, val, retval)
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

func test_map_deletion_accuracy(m map[point]point) {
    passed := true
    c := make(chan int)
    for i := 0; i < ROWS; i++ {
        go func (idx int) {
            for j := 0; j < COLS; j++ {
                key := point{idx, j}
                val := point{0, 0}
                retval := m[key]
                if retval != val {
                    log.Printf("Key: %v;Expected: %v;Received: %v", key, val, retval)
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

func nopFunction(k, v interface{}, depth int) {
    if k != v {
        nopFunction(v, v, depth + 1)
    } else if depth < 100 {
        nopFunction(k, v, depth + 1)
    }
}

func averageTime(times []time.Duration) (time.Duration) {
    var avg int64 = 0
    for _, val := range times {
        t := int64(val)
        avg += t
    }
    avg /= int64(len(times))
    return time.Duration(avg)
}