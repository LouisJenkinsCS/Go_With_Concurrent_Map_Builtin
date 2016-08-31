package main

import "fmt"
import "intset_testing"
import "iterator_testing"
import "combined_testing"
import "strconv"
import "strings"
import "flag"

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

type trialType int64
type arrayFlags []int64

func (i *trialType) String() string {
	return "Nothing good"
}

func (i *trialType) Set(value string) error {
	v, _ := strconv.ParseInt(value, 10, 64)
	*i = trialType(v)
	return nil
}

func (i *arrayFlags) String() string {
	return "my string representation"
}

func (i *arrayFlags) Set(value string) error {
	str := strings.Split(value, ",")
	for _, s := range str {
		v, _ := strconv.ParseInt(s, 10, 64)
		*i = append(*i, v)
	}
	return nil
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
	var f arrayFlags
	var trials trialType
	flag.Var(&f, "nGoroutines", "Number of Goroutines to run the tests with; comma-separated")
	flag.Var(&trials, "nTrials", "Number of trials each benchmark is ran; we take the average")
	flag.Parse()

	// No arguments passed? Use default
	if len(f) == 0 {
		f = arrayFlags([]int64{1, 2, 4, 8, 16, 32})
	}

	if trials == trialType(0) {
		trials = trialType(3)
	}

	var info []benchmarkInfo
	for _, goroutines := range f {
		info = append(info, benchmarkInfo{goroutines, int64(trials)})
	}

	benchmarks := []benchmarks{
		// Intset
		benchmarks{
			[]benchmark{
				benchmark{
					intset_testing.ConcurrentIntset,
					"Concurrent Map",
				},
				benchmark{
					intset_testing.StreamrailConcurrentIntset,
					"Streamrail Concurrent Map",
				},
				benchmark{
					intset_testing.GotomicConcurrentIntset,
					"Gotomic Concurrent Map",
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
			info,
		},
		// Read-Only Iterator
		benchmarks{
			[]benchmark{
				benchmark{
					iterator_testing.ConcurrentIterator_RO,
					"Concurrent Map",
				},
				benchmark{
					iterator_testing.StreamrailConcurrentIterator_RO,
					"Streamrail Concurrent Map",
				},
				benchmark{
					iterator_testing.GotomicConcurrentIterator_RO,
					"Streamrail Concurrent Map",
				},
				benchmark{
					iterator_testing.DefaultIterator_RO,
					"Default Map (No Mutex)",
				},
			},
			"iteratorRO.csv",
			"iteratorRO",
			info,
		},
		// // Read-Write Iterator
		// benchmarks{
		// 	[]benchmark{
		// 		benchmark{
		// 			iterator_testing.ConcurrentIterator_RW,
		// 			"Concurrent Map",
		// 		},
		// 		benchmark{
		// 			iterator_testing.SynchronizedIterator_RW,
		// 			"Synchronized Map (Mutex)",
		// 		},
		// 		benchmark{
		// 			iterator_testing.ReaderWriterIterator_RW,
		// 			"ReaderWriter Map (RWMutex)",
		// 		},
		// 	},
		// 	"iteratorRW.csv",
		// 	"iteratorRW",
		// 	info,
		// },
		// Combined
		benchmarks{
			[]benchmark{
				benchmark{
					combined_testing.ConcurrentCombined,
					"Concurrent Map",
				},
				benchmark{
					combined_testing.StreamrailConcurrentCombined,
					"Streamrail Concurrent Map",
				},
				benchmark{
					combined_testing.GotomicConcurrentCombined,
					"Gotomic Concurrent Map",
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
			info,
		},
		// // Combined - Skim
		// benchmarks{
		// 	[]benchmark{
		// 		benchmark{
		// 			combined_testing.ConcurrentCombinedSkim,
		// 			"Concurrent Map",
		// 		},
		// 		benchmark{
		// 			combined_testing.SynchronizedCombinedSkim,
		// 			"Synchronized Map (Mutex)",
		// 		},
		// 		benchmark{
		// 			combined_testing.ReaderWriterCombinedSkim,
		// 			"ReaderWriter Map (RWMutex)",
		// 		},
		// 	},
		// 	"combinedSkim.csv",
		// 	"combinedSkim",
		// 	info,
		// },
	}

	runBenchmark(benchmarks)
}
