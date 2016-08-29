package main

import "github.com/pkg/profile"
import "time"

func main() {
	p := profile.Start(profile.MemProfile, profile.ProfilePath("."), profile.NoShutdownHook)
	defer p.Stop()

	m := make(map[int]int, 0, 1)
	nElems := 10000000
	for i := 0; i < nElems; i++ {
		m[i] = i
	}

	time.Sleep(5 * time.Second)
}
