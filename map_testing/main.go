package main

import (
	"flag"
	"fmt"
	"iterator_testing"
	"os"
	"runtime"
	"strconv"
	"strings"
	"testing"
)

func MillionOpsPerSecond(nGoroutines int, callback func(nGoroutines int) int64) float64 {
	nsOp := callback(nGoroutines)
	opS := float64(1000000000) / float64(nsOp)
	return opS / float64(1000000)
}

// All benchmarks and the filename all benchmarks will be written to.
type benchmarks struct {
	benchmarks  []benchmark
	fileName    string
	csvHeader   string
	isIteration bool
	info        []benchmarkInfo
}

type benchmark struct {
	callback func(b *testing.B)
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
			oldGMP := runtime.GOMAXPROCS(0)
			for _, info := range bm.info {
				nGoroutines := info.nGoroutines
				fmt.Printf("Goroutines: %v\n", nGoroutines)
				runtime.GOMAXPROCS(int(nGoroutines))

				// Keep track of all benchmark results, as we'll be taking the average after all trials are ran.
				nsPerOp := int64(0)

				for i := int64(0); i < info.trials; i++ {
					fmt.Printf("\rTrial %v/%v", i+1, info.trials)
					nsPerOp += testing.Benchmark(b.callback).NsPerOp()
					fmt.Printf("\tns/op: %v", nsPerOp/(i+1))
				}
				fmt.Println()

				// Average then convert to Million Operations per Second
				nsPerOp /= info.trials
				OPS := float64(1000000000) / float64(nsPerOp)

				if bm.isIteration {
					KOPS := OPS / float64(1000)
					file.WriteString(fmt.Sprintf(",%.2f", KOPS))
				} else {
					MOPS := OPS / float64(1000000)
					file.WriteString(fmt.Sprintf(",%.2f", MOPS))
				}

			}
			file.WriteString("\n")
			runtime.GOMAXPROCS(oldGMP)
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

	// oldGMP := runtime.GOMAXPROCS(0)
	// for _, newGMP := range ([]int64)(f) {
	// 	runtime.GOMAXPROCS(int(newGMP))
	// 	var t time.Duration
	// 	var n int64
	// 	for i := int64(0); i < int64(trials); i++ {
	// 		br := testing.Benchmark(intset_testing.ConcurrentIntset)
	// 		t += br.T
	// 		n += int64(br.N)
	// 		nsPerOp := t.Nanoseconds() / int64(n)
	// 		fmt.Printf("\rGoroutines: %v, Trial: %v/%v, ns/op: %v", newGMP, i+1, int64(trials), nsPerOp)
	// 	}
	// 	fmt.Println()
	// }
	// runtime.GOMAXPROCS(oldGMP)

	var info []benchmarkInfo
	for _, goroutines := range f {
		info = append(info, benchmarkInfo{goroutines, int64(trials)})
	}

	benchmarks := []benchmarks{
		// Intset
		benchmarks{
			[]benchmark{
				benchmark{
					intset_testing.BenchmarkConcurrentIntset,
					"Concurrent Map",
				},
				benchmark{
					intset_testing.BenchmarkStreamrailConcurrentIntset,
					"Streamrail Concurrent Map",
				},
				benchmark{
					intset_testing.BenchmarkGotomicConcurrentIntset,
					"Gotomic Concurrent Map",
				},
				benchmark{
					intset_testing.BenchmarkSynchronizedIntset,
					"Synchronized Map (Mutex)",
				},
				benchmark{
					intset_testing.BenchmarkReaderWriterIntset,
					"ReaderWriter Map (RWMutex)",
				},
			},
			"intset.csv",
			"intset",
			false,
			info,
		},
		// Read-Only Iterator
		benchmarks{
			[]benchmark{
				benchmark{
					iterator_testing.BenchmarkConcurrentIterator_RO,
					"Concurrent Map",
				},
				benchmark{
					iterator_testing.BenchmarkStreamrailConcurrentIterator_RO,
					"Streamrail Concurrent Map",
				},
				benchmark{
					iterator_testing.BenchmarkGotomicConcurrentIterator_RO,
					"Gotomic Concurrent Map",
				},
				benchmark{
					iterator_testing.BenchmarkDefaultIterator_RO,
					"Default Map (No Mutex)",
				},
			},
			"iteratorRO.csv",
			"iteratorRO",
			true,
			info,
		},
		// Read-Write Iterator
		benchmarks{
			[]benchmark{
				benchmark{
					iterator_testing.BenchmarkConcurrentIterator_RW,
					"Concurrent Map",
				},
				benchmark{
					iterator_testing.BenchmarkSynchronizedIterator_RW,
					"Synchronized Map (Mutex)",
				},
				benchmark{
					iterator_testing.BenchmarkReaderWriterIterator_RW,
					"ReaderWriter Map (RWMutex)",
				},
			},
			"iteratorRW.csv",
			"iteratorRW",
			true,
			info,
		},
		// Combined
		benchmarks{
			[]benchmark{
				benchmark{
					combined_testing.BenchmarkConcurrentCombined,
					"Concurrent Map",
				},
				benchmark{
					combined_testing.BenchmarkStreamrailConcurrentCombined,
					"Streamrail Concurrent Map",
				},
				benchmark{
					combined_testing.BenchmarkGotomicConcurrentCombined,
					"Gotomic Concurrent Map",
				},
				benchmark{
					combined_testing.BenchmarkSynchronizedCombined,
					"Synchronized Map (Mutex)",
				},
				benchmark{
					combined_testing.BenchmarkReaderWriterCombined,
					"ReaderWriter Map (RWMutex)",
				},
			},
			"combined.csv",
			"combined",
			true,
			info,
		},
		// Combined - Skim
		benchmarks{
			[]benchmark{
				benchmark{
					combined_testing.BenchmarkConcurrentCombinedSkim,
					"Concurrent Map",
				},
				benchmark{
					combined_testing.BenchmarkSynchronizedCombinedSkim,
					"Synchronized Map (Mutex)",
				},
				benchmark{
					combined_testing.BenchmarkReaderWriterCombinedSkim,
					"ReaderWriter Map (RWMutex)",
				},
			},
			"combinedSkim.csv",
			"combinedSkim",
			true,
			info,
		},
	}

	runBenchmark(benchmarks)
}
