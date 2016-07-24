package main

import "fmt"
import "intset_testing"

func main() {
    // Header
    fmt.Println("Map,MOPS-1,MOPS-2,MOPS-4,MOPS-8,MOPS-16,MOPS-32")
    
    // Concurrent Map
    fmt.Printf("ConcurrentMap")
    for i := 1; i <= 32; i = i << 1 {
        fmt.Printf(",")
        nsOp := intset_testing.ConcurrentIntset(i)
        opS := float64(1000000000) / float64(nsOp)
        mOps := opS / float64(1000000)
        fmt.Printf("%.2f", mOps)
    }
    fmt.Println()

    // Synchronized Map
    fmt.Printf("SynchronizedMap")
    for i := 1; i <= 32; i = i << 1 {
        fmt.Printf(",")
        nsOp := intset_testing.SynchronizedIntset(i)
        opS := float64(1000000000) / float64(nsOp)
        mOps := opS / float64(1000000)
        fmt.Printf("%.2f", mOps)
    }
    fmt.Println()

    // ReaderWriterMap
    fmt.Printf("ReaderWriterMap")
    for i := 1; i <= 32; i = i << 1 {
        fmt.Printf(",")
        nsOp := intset_testing.ReaderWriterIntset(i)
        opS := float64(1000000000) / float64(nsOp)
        mOps := opS / float64(1000000)
        fmt.Printf("%.2f", mOps)
    }
    fmt.Println()
}