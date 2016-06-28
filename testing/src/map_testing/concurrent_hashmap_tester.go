package map_testing

import "../github.com/pkg/profile"
import "log"
import "time"

func populate_map_struct(m map[point]point) {
	for i := 0; i < ROWS; i++ {
		go func (idx int) { 
			for j := 0; j < COLS; j++ {
				key, val := point{idx, j}, point{ROWS - idx, COLS - j}
				m[key] = val
			}
			c <- 0
		}(i)
	}
}


func TestConcurrentMap() {
	m := make(map[point]point, 0, 1)
	c = make(chan int)
	log.Println("Populating Concurrent Map")

	prof := profile.Start(profile.CPUProfile, profile.ProfilePath("."));
	// memProf := profile.Start(profile.MemProfile, profile.ProfilePath("."));
	start := time.Now()
	populate_map_struct(m)
	for i := 0; i < ROWS; i++ {
		<- c
	}

	// memProf.Stop()
	end := time.Since(start)
	prof.Stop()
	log.Println("Concurrent Map Time: ", end)

    log.Println("Testing Concurrent Map accuracy")
	for i := 0; i < ROWS; i++ {
		for j := 0; j < COLS; j++ {
			key := point{i, j}
			val := point{ROWS - i, COLS - j}
			retval := m[key]
			if retval != val {
				log.Printf("Key: %v;Expected: %v;Received: %v", key, val, retval)
			}
		}
	}
}