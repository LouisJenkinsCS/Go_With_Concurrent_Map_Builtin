package settings

// Types used in tests

// Type used to indicate a particular key and/or value is unused for the sake of the test.
type Unused byte

// Variables used in tests (pseudo-constants, can be modified by command-line arguments)

// NUM_GOROUTINES is used to determine the maximum number of Goroutines used during the test.
// Normally, if none is specified by the user, it is initialized to GOMAXPROCS.
var NUM_GOROUTINES int

// ITERATOR_POOLED_ELEMENTS is used to determine, in iterator tests, how many elements should be allocated ahead of time
// to be used during the test. By default, this is 0, and hence, during benchmarks the overhead of allocation should be
// factored in
var ITERATOR_POOLED_ELEMENTS uint64

// ITERATOR_NUM_ELEMS is used to determine, in iterator tests, how many elements are added to the map to be iterated over
var ITERATOR_NUM_ELEMS int64 = 1000000

// ITERATOR_RNG_INCREMENT is used to determine, in iterator tests, the range of the randomized integer used to test for when
// the element's 'iter' field should be incremented
var ITERATOR_RNG_INCREMENT int64 = 10

// ITERATOR_NUM_ITERATIONS is used to determine, in iterator tests, how many full iterations through the map should be performed.
// By default, this value is 1000.
var ITERATOR_NUM_ITERATIONS uint64 = 1000

// INTSET_OPS_PER_GOROUTINE is used to determine just how many randomized operations are performed per Goroutine during an integer set test.
// By default, this value is 1000000.
var INTSET_OPS_PER_GOROUTINE uint64 = 10000000

// INTSET_RNG_SEED is used during the integer set test to randomly generate values. This is consistent during the test to test for correctness.
// By default, this value is set to 0x1BAD5EED (One Bad Seed) out of hilarity from the developer.
var INTSET_RNG_SEED int64 = 0x1BAD5EED

// INTSET_VALUE_RANGE is used during integer set test to determine the range of the randomly generated values.
// By default, this value is set to 1000000.
var INTSET_VALUE_RANGE uint64 = 1000000

// INTSET_FAIR_LOOKUP_RATIO is used during the integer set fair test to determine the ratio lookup operations are done.
// By default, this value is set to .5 to allow fairness in frequency of reads
var INTSET_FAIR_LOOKUP_RATIO float64 = .5

// INTSET_FAIR_INSERT_RATIO is used during the integer set fair test to determine the ratio lookup operations are done.
// By default, this value is set to .25 to allow fairness in frequency of writes
var INTSET_FAIR_INSERT_RATIO float64 = .25

// INTSET_FAIR_REMOVE_RATIO is used during the integer set fair test to determine the ratio lookup operations are done.
// By default, this value is set to .25 to allow fairness in frequency of writes
var INTSET_FAIR_REMOVE_RATIO float64 = .25

// INTSET_WRITE_BIAS_LOOKUP_RATIO is used during the integer set write-biased test to determine the ratio lookup operations are done.
// By default, this value is set to .33 to allow less bias in frequency of reads
var INTSET_WRITE_BIAS_LOOKUP_RATIO float64 = .33

// INTSET_WRITE_BIAS_INSERT_RATIO is used during the integer set write-biased test to determine the ratio lookup operations are done.
// By default, this value is set to .33 to allow more bias in frequency of writes
var INTSET_WRITE_BIAS_INSERT_RATIO float64 = .33

// INTSET_WRITE_BIAS_REMOVE_RATIO is used during the integer set write-biased test to determine the ratio lookup operations are done.
// By default, this value is set to .33 to allow more bias in frequency of writes
var INTSET_WRITE_BIAS_REMOVE_RATIO float64 = .33

// INTSET_PREEMPTIVE_FILL is used during the integer set test to initially fill the map to reduce overhead of expansion/resizing.
// By default, this value is set to .5.
var INTSET_PREEMPTIVE_FILL float64 = .5

// SWAP_FREQUENCY is used to during the sync.Interlocked swap test to determine how often elements are swapped.
// By default, this value is set to .33
var SWAP_FREQUENCY float64 = .33

var UNUSED Unused = '0'