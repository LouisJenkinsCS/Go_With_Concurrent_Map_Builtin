package map_testing

import "sync"
import "log"
import "time"
import "../github.com/pkg/profile"

var mtx sync.Mutex

var c chan int

func populate_map_sync_struct(m map[point]point) {
	for i := 0; i < ROWS; i++ {
		go func (idx int) { 
			for j := 0; j < COLS; j++ {
				// val := fmt.Sprintf("{%v, %v}", idx, j)
				key, val := point{idx, j}, point{ROWS - idx, COLS - j}
				mtx.Lock()
				m[key] = val
				mtx.Unlock()
			}
			c <- 0
		}(i)
	}
}


func TestDefaultMap() {
	m := make(map[point]point)
	c = make(chan int)
	
	log.Println("Populating Default Map")
	// cpuProf := profile.Start(profile.CPUProfile, profile.ProfilePath("."));
	memProf := profile.Start(profile.MemProfile, profile.ProfilePath("."));
	start := time.Now()
	populate_map_sync_struct(m)
	for i := 0; i < ROWS; i++ {
		<- c
	}
	end := time.Since(start)
	// cpuProf.Stop()
	memProf.Stop()
	log.Println("Default Map Time: ", end)

	log.Println("Testing Default Map accuracy")
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