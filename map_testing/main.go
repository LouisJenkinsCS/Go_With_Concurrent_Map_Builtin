package main

import "fmt"
import "intset_testing"
import "iterator_testing"
import "combined_testing"

import "os"

func MillionOpsPerSecond(nGoroutines int, callback func(nGoroutines int) int64) float64 {
	nsOp := callback(nGoroutines)
	opS := float64(1000000000) / float64(nsOp)
	return opS / float64(1000000)
}

// All benchmarks and the filename all benchmarks will be written to.
type benchmarks struct {
	benchmarks []benchmark
	fileName   string
	csvHeader  string
	info       []benchmarkInfo
}

type benchmark struct {
	callback func(nGoroutines int64) int64
	rowName  string
}

type benchmarkInfo struct {
	nGoroutines int64
	trials      int64
}

func runBenchmark(barr []benchmarks) {
	for _, bm := range barr {
		// Open file for benchmarks
		file, err := os.Create(bm.fileName)
		if err != nil {
			panic(fmt.Sprintf("Could not open %v\n", bm.fileName))
		}

		fmt.Printf("Benchmark: %v\n", bm.csvHeader)

		// Header
		file.WriteString(bm.csvHeader)
		for _, info := range bm.info {
			file.WriteString(fmt.Sprintf(",%v", info.nGoroutines))
		}
		file.WriteString("\n")

		// Run each benchmark
		for _, b := range bm.benchmarks {
			// Name of row
			file.WriteString(b.rowName)
			fmt.Printf("Section: %v\n", b.rowName)
			// Run the benchmark for all Goroutines and trials.
			for _, info := range bm.info {
				nGoroutines := info.nGoroutines
				fmt.Printf("Goroutines: %v\n", nGoroutines)

				// Keep track of all benchmark results, as we'll be taking the average after all trials are ran.
				nsPerOp := int64(0)

				for i := int64(0); i < info.trials; i++ {
					fmt.Printf("\rTrial %v/%v", i+1, info.trials)
					nsPerOp += b.callback(nGoroutines)
					fmt.Printf("\tns/op: %v", nsPerOp/(i+1))
				}
				fmt.Println()

				// Average then convert to Million Operations per Second
				nsPerOp /= info.trials
				OPS := float64(1000000000) / float64(nsPerOp)
				MOPS := OPS / float64(1000000)

				file.WriteString(fmt.Sprintf(",%.2f", MOPS))
			}
			file.WriteString("\n")
		}

		// Finished, so close the file.
		file.Close()
	}
}

func main() {
	benchmarks := []benchmarks{
		// Intset
		benchmarks{
			[]benchmark{
				benchmark{
					intset_testing.ConcurrentIntset,
					"Concurrent Map",
				},
				benchmark{
					intset_testing.SynchronizedIntset,
					"Synchronized Map (Mutex)",
				},
				benchmark{
					intset_testing.ReaderWriterIntset,
					"ReaderWriter Map (RWMutex)",
				},
			},
			"intset.csv",
			"intset",
			[]benchmarkInfo{
				benchmarkInfo{int64(1), int64(3)},
				benchmarkInfo{int64(2), int64(3)},
				benchmarkInfo{int64(4), int64(3)},
				benchmarkInfo{int64(8), int64(3)},
				benchmarkInfo{int64(16), int64(3)},
				benchmarkInfo{int64(32), int64(3)},
			},
		},
		// Read-Only Iterator
		benchmarks{
			[]benchmark{
				benchmark{
					iterator_testing.ConcurrentIterator_Interlocked_RO,
					"Concurrent Map (Interlocked)",
				},
				benchmark{
					iterator_testing.DefaultIterator_RO,
					"Default Map (No Mutex)",
				},
			},
			"iteratorRO.csv",
			"iteratorRO",
			[]benchmarkInfo{
				benchmarkInfo{int64(1), int64(3)},
				benchmarkInfo{int64(2), int64(3)},
				benchmarkInfo{int64(4), int64(3)},
				benchmarkInfo{int64(8), int64(3)},
				benchmarkInfo{int64(16), int64(3)},
				benchmarkInfo{int64(32), int64(3)},
			},
		},
		// Read-Write Iterator
		benchmarks{
			[]benchmark{
				benchmark{
					iterator_testing.ConcurrentIterator_Interlocked_RW,
					"Concurrent Map (Interlocked)",
				},
				benchmark{
					iterator_testing.SynchronizedIterator_RW,
					"Synchronized Map (Mutex)",
				},
				benchmark{
					iterator_testing.ReaderWriterIterator_RW,
					"ReaderWriter Map (RWMutex)",
				},
			},
			"iteratorRW.csv",
			"iteratorRW",
			[]benchmarkInfo{
				benchmarkInfo{int64(1), int64(3)},
				benchmarkInfo{int64(2), int64(3)},
				benchmarkInfo{int64(4), int64(3)},
				benchmarkInfo{int64(8), int64(3)},
				benchmarkInfo{int64(16), int64(3)},
				benchmarkInfo{int64(32), int64(3)},
			},
		},
		// Combined
		benchmarks{
			[]benchmark{
				benchmark{
					combined_testing.ConcurrentCombined,
					"Concurrent Map",
				},
				benchmark{
					combined_testing.ConcurrentCombined_Interlocked,
					"Concurrent Map (Interlocked)",
				},
				benchmark{
					combined_testing.SynchronizedCombined,
					"Synchronized Map (Mutex)",
				},
				benchmark{
					combined_testing.ReaderWriterCombined,
					"ReaderWriter Map (RWMutex)",
				},
			},
			"combined.csv",
			"combined",
			[]benchmarkInfo{
				benchmarkInfo{int64(1), int64(3)},
				benchmarkInfo{int64(2), int64(3)},
				benchmarkInfo{int64(4), int64(3)},
				benchmarkInfo{int64(8), int64(3)},
				benchmarkInfo{int64(16), int64(3)},
				benchmarkInfo{int64(32), int64(3)},
			},
		},
		// Combined - Skim
		benchmarks{
			[]benchmark{
				benchmark{
					combined_testing.ConcurrentCombinedSkim,
					"Concurrent Map",
				},
				benchmark{
					combined_testing.ConcurrentCombinedSkim_Interlocked,
					"Concurrent Map (Interlocked)",
				},
				benchmark{
					combined_testing.SynchronizedCombinedSkim,
					"Synchronized Map (Mutex)",
				},
				benchmark{
					combined_testing.ReaderWriterCombinedSkim,
					"ReaderWriter Map (RWMutex)",
				},
			},
			"combinedSkim.csv",
			"combinedSkim",
			[]benchmarkInfo{
				benchmarkInfo{int64(1), int64(3)},
				benchmarkInfo{int64(2), int64(3)},
				benchmarkInfo{int64(4), int64(3)},
				benchmarkInfo{int64(8), int64(3)},
				benchmarkInfo{int64(16), int64(3)},
				benchmarkInfo{int64(32), int64(3)},
			},
		},
	}

	runBenchmark(benchmarks)
}
