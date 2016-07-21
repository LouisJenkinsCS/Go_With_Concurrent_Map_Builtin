package main

import (
    "math/rand"
    "time"
    "fmt"
    "sync"
     _ "github.com/pkg/profile"
)

var letterRunes = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

func randomWord(n int) string {
    word := make([]rune, n)
    for i := range word {
        word[i] = letterRunes[rand.Intn(len(letterRunes))]
    }
    return string(word)
}

func main() {
    rand.Seed(time.Now().UnixNano())
    m := make(map[string]string, 0, 1)
    wg := sync.WaitGroup{}
    producerDone := make(chan int)
    wg.Add(9)
    // prof := profile.Start(profile.CPUProfile, profile.ProfilePath("."), profile.NoShutdownHook)
    go func() {
        chars := make(map[rune]uint64, 0, 1)
        producersDone := 0
        consumerWg := sync.WaitGroup{}
        for {
            select {
                case <- producerDone:
                    producersDone++
                    if producersDone == 8 {
                        for k, v := range chars {
                            fmt.Printf("Rune: %v;Occurence: %v\n", string(k), v)
                        }
                        wg.Done()
                        return
                    }
                default:
                    break
            }
            
            for k, v := range m {
                kstr := []rune(k)
                vstr := []rune(v)

                goFunc := func(data []rune) { 
                            for _, c := range data {
                                sync.Interlocked chars[c] {
                                    chars[c] += 1
                                }
                            }

                            delete(m, string(data))
                            consumerWg.Done()
                        }
                
                consumerWg.Add(2)
                go goFunc(kstr)
                go goFunc(vstr)
            }

            consumerWg.Wait()
        }
    }()
    
    for i := 0; i < 8; i++ {
        go func() {
            var counter uint64 = 0
            for counter < 100000 {
                randKey := randomWord(rand.Intn(128))
                randVal := randomWord(rand.Intn(128))
                m[randKey] = randVal

                counter++
            }
            fmt.Printf("Finished processing: %v words\n", counter)
            wg.Done()
            producerDone <- 0
        }()
    }
    wg.Wait()
    // prof.Stop()
}