package main

import "time"
import "reflect"

func main() {
	m := make(map[int]int, 0, 1)
	nElems := 10000000
	for i := 0; i < nElems; i++ {
		m[i] = i
	}

	reflect.ProfileMap(m)

	for k, _ := range m {
		delete(m, k)
	}

	reflect.ProfileMap(m)
	time.Sleep(5 * time.Second)
}
