package runtime

import (
	"runtime/internal/atomic"
	"runtime/internal/sys"
	"unsafe"
)

// Prime numbers pre-generated for the interlocked iterator to use when determining randomized start position
var primes = [...]int{
	2, 3, 5, 7, 11, 13, 17, 19, 23, 29,
	31, 37, 41, 43, 47, 53, 59, 61, 67, 71,
	73, 79, 83, 89, 97, 101, 103, 107, 109, 113,
	127, 131, 137, 139, 149, 151, 157, 163, 167, 173,
	179, 181, 191, 193, 197, 199, 211, 223, 227, 229,
	233, 239, 241, 251, 257, 263, 269, 271, 277, 281,
	283, 293, 307, 311, 313, 317, 331, 337, 347, 349,
	353, 359, 367, 373, 379, 383, 389, 397, 401, 409,
	419, 421, 431, 433, 439, 443, 449, 457, 461, 463,
	467, 479, 487, 491, 499, 503, 509, 521, 523, 541,
	547, 557, 563, 569, 571, 577, 587, 593, 599, 601,
	607, 613, 617, 619, 631, 641, 643, 647, 653, 659,
	661, 673, 677, 683, 691, 701, 709, 719, 727, 733,
	739, 743, 751, 757, 761, 769, 773, 787, 797, 809,
	811, 821, 823, 827, 829, 839, 853, 857, 859, 863,
	877, 881, 883, 887, 907, 911, 919, 929, 937, 941,
	947, 953, 967, 971, 977, 983, 991, 997, 1009, 1013,
	1019, 1021, 1031, 1033, 1039, 1049, 1051, 1061, 1063, 1069,
	1087, 1091, 1093, 1097, 1103, 1109, 1117, 1123, 1129, 1151,
	1153, 1163, 1171, 1181, 1187, 1193, 1201, 1213, 1217, 1223,
	1229, 1231, 1237, 1249, 1259, 1277, 1279, 1283, 1289, 1291,
	1297, 1301, 1303, 1307, 1319, 1321, 1327, 1361, 1367, 1373,
	1381, 1399, 1409, 1423, 1427, 1429, 1433, 1439, 1447, 1451,
	1453, 1459, 1471, 1481, 1483, 1487, 1489, 1493, 1499, 1511,
	1523, 1531, 1543, 1549, 1553, 1559, 1567, 1571, 1579, 1583,
	1597, 1601, 1607, 1609, 1613, 1619, 1621, 1627, 1637, 1657,
	1663, 1667, 1669, 1693, 1697, 1699, 1709, 1721, 1723, 1733,
	1741, 1747, 1753, 1759, 1777, 1783, 1787, 1789, 1801, 1811,
	1823, 1831, 1847, 1861, 1867, 1871, 1873, 1877, 1879, 1889,
	1901, 1907, 1913, 1931, 1933, 1949, 1951, 1973, 1979, 1987,
	1993, 1997, 1999, 2003, 2011, 2017, 2027, 2029, 2039, 2053,
	2063, 2069, 2081, 2083, 2087, 2089, 2099, 2111, 2113, 2129,
	2131, 2137, 2141, 2143, 2153, 2161, 2179, 2203, 2207, 2213,
	2221, 2237, 2239, 2243, 2251, 2267, 2269, 2273, 2281, 2287,
	2293, 2297, 2309, 2311, 2333, 2339, 2341, 2347, 2351, 2357,
	2371, 2377, 2381, 2383, 2389, 2393, 2399, 2411, 2417, 2423,
	2437, 2441, 2447, 2459, 2467, 2473, 2477, 2503, 2521, 2531,
	2539, 2543, 2549, 2551, 2557, 2579, 2591, 2593, 2609, 2617,
	2621, 2633, 2647, 2657, 2659, 2663, 2671, 2677, 2683, 2687,
	2689, 2693, 2699, 2707, 2711, 2713, 2719, 2729, 2731, 2741,
	2749, 2753, 2767, 2777, 2789, 2791, 2797, 2801, 2803, 2819,
	2833, 2837, 2843, 2851, 2857, 2861, 2879, 2887, 2897, 2903,
	2909, 2917, 2927, 2939, 2953, 2957, 2963, 2969, 2971, 2999,
	3001, 3011, 3019, 3023, 3037, 3041, 3049, 3061, 3067, 3079,
	3083, 3089, 3109, 3119, 3121, 3137, 3163, 3167, 3169, 3181,
	3187, 3191, 3203, 3209, 3217, 3221, 3229, 3251, 3253, 3257,
	3259, 3271, 3299, 3301, 3307, 3313, 3319, 3323, 3329, 3331,
	3343, 3347, 3359, 3361, 3371, 3373, 3389, 3391, 3407, 3413,
	3433, 3449, 3457, 3461, 3463, 3467, 3469, 3491, 3499, 3511,
	3517, 3527, 3529, 3533, 3539, 3541, 3547, 3557, 3559, 3571,
	3581, 3583, 3593, 3607, 3613, 3617, 3623, 3631, 3637, 3643,
	3659, 3671, 3673, 3677, 3691, 3697, 3701, 3709, 3719, 3727,
	3733, 3739, 3761, 3767, 3769, 3779, 3793, 3797, 3803, 3821,
	3823, 3833, 3847, 3851, 3853, 3863, 3877, 3881, 3889, 3907,
	3911, 3917, 3919, 3923, 3929, 3931, 3943, 3947, 3967, 3989,
	4001, 4003, 4007, 4013, 4019, 4021, 4027, 4049, 4051, 4057,
	4073, 4079, 4091, 4093, 4099, 4111, 4127, 4129, 4133, 4139,
	4153, 4157, 4159, 4177, 4201, 4211, 4217, 4219, 4229, 4231,
	4241, 4243, 4253, 4259, 4261, 4271, 4273, 4283, 4289, 4297,
	4327, 4337, 4339, 4349, 4357, 4363, 4373, 4391, 4397, 4409,
	4421, 4423, 4441, 4447, 4451, 4457, 4463, 4481, 4483, 4493,
	4507, 4513, 4517, 4519, 4523, 4547, 4549, 4561, 4567, 4583,
	4591, 4597, 4603, 4621, 4637, 4639, 4643, 4649, 4651, 4657,
	4663, 4673, 4679, 4691, 4703, 4721, 4723, 4729, 4733, 4751,
	4759, 4783, 4787, 4789, 4793, 4799, 4801, 4813, 4817, 4831,
	4861, 4871, 4877, 4889, 4903, 4909, 4919, 4931, 4933, 4937,
	4943, 4951, 4957, 4967, 4969, 4973, 4987, 4993, 4999, 5003,
	5009, 5011, 5021, 5023, 5039, 5051, 5059, 5077, 5081, 5087,
	5099, 5101, 5107, 5113, 5119, 5147, 5153, 5167, 5171, 5179,
	5189, 5197, 5209, 5227, 5231, 5233, 5237, 5261, 5273, 5279,
	5281, 5297, 5303, 5309, 5323, 5333, 5347, 5351, 5381, 5387,
	5393, 5399, 5407, 5413, 5417, 5419, 5431, 5437, 5441, 5443,
	5449, 5471, 5477, 5479, 5483, 5501, 5503, 5507, 5519, 5521,
	5527, 5531, 5557, 5563, 5569, 5573, 5581, 5591, 5623, 5639,
	5641, 5647, 5651, 5653, 5657, 5659, 5669, 5683, 5689, 5693,
	5701, 5711, 5717, 5737, 5741, 5743, 5749, 5779, 5783, 5791,
	5801, 5807, 5813, 5821, 5827, 5839, 5843, 5849, 5851, 5857,
	5861, 5867, 5869, 5879, 5881, 5897, 5903, 5923, 5927, 5939,
	5953, 5981, 5987, 6007, 6011, 6029, 6037, 6043, 6047, 6053,
	6067, 6073, 6079, 6089, 6091, 6101, 6113, 6121, 6131, 6133,
	6143, 6151, 6163, 6173, 6197, 6199, 6203, 6211, 6217, 6221,
	6229, 6247, 6257, 6263, 6269, 6271, 6277, 6287, 6299, 6301,
	6311, 6317, 6323, 6329, 6337, 6343, 6353, 6359, 6361, 6367,
	6373, 6379, 6389, 6397, 6421, 6427, 6449, 6451, 6469, 6473,
	6481, 6491, 6521, 6529, 6547, 6551, 6553, 6563, 6569, 6571,
	6577, 6581, 6599, 6607, 6619, 6637, 6653, 6659, 6661, 6673,
	6679, 6689, 6691, 6701, 6703, 6709, 6719, 6733, 6737, 6761,
	6763, 6779, 6781, 6791, 6793, 6803, 6823, 6827, 6829, 6833,
	6841, 6857, 6863, 6869, 6871, 6883, 6899, 6907, 6911, 6917,
	6947, 6949, 6959, 6961, 6967, 6971, 6977, 6983, 6991, 6997,
	7001, 7013, 7019, 7027, 7039, 7043, 7057, 7069, 7079, 7103,
	7109, 7121, 7127, 7129, 7151, 7159, 7177, 7187, 7193, 7207,
	7211, 7213, 7219, 7229, 7237, 7243, 7247, 7253, 7283, 7297,
	7307, 7309, 7321, 7331, 7333, 7349, 7351, 7369, 7393, 7411,
	7417, 7433, 7451, 7457, 7459, 7477, 7481, 7487, 7489, 7499,
	7507, 7517, 7523, 7529, 7537, 7541, 7547, 7549, 7559, 7561,
	7573, 7577, 7583, 7589, 7591, 7603, 7607, 7621, 7639, 7643,
	7649, 7669, 7673, 7681, 7687, 7691, 7699, 7703, 7717, 7723,
	7727, 7741, 7753, 7757, 7759, 7789, 7793, 7817, 7823, 7829,
	7841, 7853, 7867, 7873, 7877, 7879, 7883, 7901, 7907, 7919,
}

const (
	// must match value in ../cmd/compile/internal/gc/walk.go
	// MAXZERO is the size of the zero'd portion that must be returned when the requested element is not found in the map
	// yet it requires what is returned to be non-nil for compiler optimizations.
	MAXZERO = 1024

	// The number of buckets in the root bucketArray.
	DEFAULT_BUCKETS = 32
	// The number of slots (hash/key/value) in a bucketData.
	MAX_SLOTS = 8

	// The bucketHdr can safely be casted to a bucketArray
	ARRAY = 1 << 0
	// The bucketHdr is invalidated and needs to be reloaded (occurs during resizing or deletion)
	INVALID = 1 << 1

	// The bucketHdr lock is uncontested
	UNLOCKED = 0
	// Mask used to determine the lock-holder; Used when we are in a tight-loop, waiting for lock-holder to give up lock.
	// This will be mentioned once: We reset the backoff variables when the lock-holder relinquishes the lock to prevent excessive spinning.
	LOCKED_MASK = ^uintptr(0x3)

	// Hash value signifying that the hash is not in use.
	EMPTY = 0

	// After this many spins, we yield (remember Goroutine context switching only requires a switch in SP/PC and DX register and is lightning fast)
	GOSCHED_AFTER_SPINS = 1
	// After this many spins, we backoff (time.Sleep unfortunately has us park on a semaphore, but if we spin this many times, it's not a huge deal...)
	// Also as well, due to this, when deadlocks occur they are easier to identify since the CPU sinks to 0% rather than infinitely at 100%
	SLEEP_AFTER_SPINS = 5

	// Default backoff; 1 microsecond
	DEFAULT_BACKOFF = 1024
	// Maximum backoff; 33 milliseconds
	MAX_BACKOFF = 33000000

	// See hashmap.go, this obtains a properly aligned offset to the data
	// Is used to obtain the array of keys and values at the end of the runtime representation of the bucketData type.
	cdataOffset = unsafe.Offsetof(struct {
		b bucketData
		v int64
	}{}.v)
)

// Returned when no element is found, as we are allowed to return nil. See hashmap.go...
var DUMMY_RETVAL [MAXZERO]byte

/*
	bucketHdr is the header that serves three purposes. bucketHdr can be cast to and from bucketData and bucketArray.

	Descriptor:
		bucketHdr is a descriptor for the body of the bucketHdr, and can be casted to and from it's header to it's corresponding actual type
		based on the information it holds.
	Lock:
		It's lock, which doubles as the describer of what the body holds, is also used for mutual exclusion over the bucket.
	Counter:
		It's count keeps track of the actual number of elements, to save the effort of other concurrent accessors time before having to actually acquire
		the lock, further reducing contention.
*/
type bucketHdr struct {
	// INVALID | ARRAY | UNLOCKED | LOCKED
	lock uintptr
	// Number of elements in this bucketHdr
	count uintptr
	// Prevent false sharing
	_ [sys.CacheLineSize - 2*sys.PtrSize]byte
}

/*
   bucketArray is the body that keeps track of an array of bucketHdr's, that may point to either bucketData or even other bucketArrays.
   It's seed is unique relative to other bucketArray's to prevent excess collisions during hashing and reduce possible contention.
   It keeps track of the location of the bucketHdr that pointed to this, for O(1) navigation during iteration.
   Can be casted to and from bucketHdr.
*/
type bucketArray struct {
	// Fields embedded from bucketHdr; for complexity reasons, we can't actually embed the type in the runtime (because we would have to also do so in compiler)
	lock  uintptr
	count uintptr
	_     [sys.CacheLineSize - 2*sys.PtrSize]byte
	// Seed is different for each bucketArray to ensure that the re-hashing resolves to different indice
	seed uint32
	// Associated with the backlink (see below), used to signify what index the bucketHdr that pointed to this is
	backIdx uint32
	// Pointer to the previous bucketArray that pointed to this, if there is one.
	backLink *bucketArray
	// Slice of bucketHdr's. TODO: Make in fixed memory???
	buckets []*bucketHdr
}

/*
   bucketData is the actual bucket itself, containing MAX_SLOTS data slots (hash/key/value) for elements it holds.
   It can be casted to and from bucketHdr.
   It's key and value slots are only accessible through unsafe pointer arithmetic.
*/
type bucketData struct {
	// Fields embedded from bucketHdr
	lock  uintptr
	count uintptr
	_     [sys.CacheLineSize - 2*sys.PtrSize]byte
	// Hash of the key-value corresponding to this index. If it is 0, it is empty. Aligned to cache line (64 bytes)
	hash [MAX_SLOTS]uintptr
	// It's key and value slots are below, and would appear as such if the runtime supported generics...
	/*
	   key [MAX_SLOTS]keyType
	   val [MAX_SLOTS]valType
	*/
}

/*
	concurrentMap is the header which contains the root bucket which contains all data. It is the entry-point into the map.
*/
type concurrentMap struct {
	// Root bucket.
	root bucketArray
}

/*
	concurrentIterator is our version of the iterator header used in hashmap.go.

	It keeps track of where it is in the map, and where we should stop at/wrap to for randomized iteration.
	For snapshot iteration, the data field is used to iterate over a snapshot. For interlocked iteration, it's 'g' holds the locked bucket
	and is retrieved from that.

	The offset keeps track of it's current offset inside of the data it holds.
*/
type concurrentIterator struct {
	// The current index we are on
	idx uint32
	// Offset we are inside of data we are iterating over.
	offset uint32
	// The bucketArray we currently are working on
	arr *bucketArray
	// For the interlocked iterator, keeps track of randomized root we started at (and will wrap to)
	rootStartIdx uintptr
	// Cached 'g' for faster access; if interlocked iteration, data is the bucketHdr held.
	g *g
	// Snapshot if snapshot iteration used
	data bucketData
}

/*
	Obtains the pointer to the key slot at the requested offset.
*/
func (data *bucketData) key(t *maptype, idx uintptr) unsafe.Pointer {
	// Cast data to unsafe.Pointer to bypass Go's type system
	rawData := unsafe.Pointer(data)
	// The array of keys are located at the beginning of cdataOffset, and is contiguous up to MAX_SLOTS
	keyOffset := uintptr(rawData) + uintptr(cdataOffset)
	// Now the key at index 'idx' is located at idx * t.keysize
	ourKeyOffset := keyOffset + idx*uintptr(t.keysize)
	return unsafe.Pointer(ourKeyOffset)
}

/*
	Obtains the pointer to the value slot at the requested offset.
*/
func (data *bucketData) value(t *maptype, idx uintptr) unsafe.Pointer {
	// Cast data to unsafe.Pointer to bypass Go's type system
	rawData := unsafe.Pointer(data)
	// The array of keys are located at the beginning of cdataOffset, and is contiguous up to MAX_SLOTS
	keyOffset := uintptr(rawData) + uintptr(cdataOffset)
	// The array of values are located at the end of the array of keys, located at MAX_SLOTS * t.keysize
	valueOffset := keyOffset + MAX_SLOTS*uintptr(t.keysize)
	// Now the value at index 'idx' is located at idx * t.valuesize
	ourValueOffset := valueOffset + idx*uintptr(t.valuesize)
	return unsafe.Pointer(ourValueOffset)
}

/*
	Assigns into the data slot the passed information at the requested index (hash/key/value)
*/
func (data *bucketData) assign(t *maptype, idx, hash uintptr, key, value unsafe.Pointer) {
	k := data.key(t, idx)
	v := data.value(t, idx)

	if t.indirectkey {
		// println(".........g #", getg().goid, ": Key is indirect")
		kmem := newobject(t.key)
		*(*unsafe.Pointer)(k) = kmem
		k = kmem
	}
	if t.indirectvalue {
		// println(".........g #", getg().goid, ": Value is indirect")
		vmem := newobject(t.elem)
		*(*unsafe.Pointer)(v) = vmem
		v = vmem
	}

	// Copies memory in a way that it updates the GC to know objects pointed to by this copy should not be collected
	typedmemmove(t.key, k, key)
	typedmemmove(t.elem, v, value)

	data.hash[idx] = hash
}

/*
	Updates the requested key and value at the requested index. Note that it assumes that the index is correct and corresponds to the key passed.
*/
func (data *bucketData) update(t *maptype, idx uintptr, key, value unsafe.Pointer) {
	v := data.value(t, idx)

	// Indirect Key and Values need some indirection
	if t.indirectvalue {
		v = *(*unsafe.Pointer)(v)
	}

	// If we are required to update key, do so
	if t.needkeyupdate {
		k := data.key(t, idx)
		if t.indirectkey {
			k = *(*unsafe.Pointer)(k)
		}
		typedmemmove(t.key, k, key)
	}

	typedmemmove(t.elem, v, value)
}

/*
	Atomic snapshot initialization
*/
func cmapiterinit(t *maptype, h *hmap, it *hiter) {
	// Clear pointer fields so garbage collector does not complain.
	it.key = nil
	it.value = nil
	it.t = nil
	it.h = nil
	it.buckets = nil
	it.bptr = nil
	it.overflow[0] = nil
	it.overflow[1] = nil
	it.citerHdr = nil

	// You cannot iterate a nil or empty map
	if h == nil || atomic.Load((*uint32)(unsafe.Pointer(&h.count))) == 0 {
		it.key = nil
		it.value = nil
		return
	}

	it.t = t
	it.h = h

	cmap := (*concurrentMap)(h.chdr)
	root := &cmap.root
	citer := (*concurrentIterator)(newobject(t.concurrentiterator))
	it.citerHdr = unsafe.Pointer(citer)
	citer.arr = root

	// Cache the 'g' and used during locking
	citer.g = getg()

	// By setting offset to MAX_SLOTS, it allows it to bypass the findKeyValue portion without modification
	citer.offset = MAX_SLOTS

	cmapiternext(it)
}

/*
	Interlocked iteration initialization
*/
func cmapiterinit_interlocked(t *maptype, h *hmap, it *hiter) {
	// Clear pointer fields so garbage collector does not complain.
	it.key = nil
	it.value = nil
	it.t = nil
	it.h = nil
	it.buckets = nil
	it.bptr = nil
	it.overflow[0] = nil
	it.overflow[1] = nil
	it.citerHdr = nil

	// You cannot iterate a nil or empty map
	if h == nil || atomic.Load((*uint32)(unsafe.Pointer(&h.count))) == 0 {
		it.key = nil
		it.value = nil
		return
	}

	it.t = t
	it.h = h

	cmap := (*concurrentMap)(h.chdr)
	arr := &cmap.root
	citer := (*concurrentIterator)(newobject(t.concurrentiterator))
	it.citerHdr = unsafe.Pointer(citer)
	citer.arr = arr

	// By setting offset to MAX_SLOTS, it allows it to bypass the findKeyValue portion without modification
	citer.offset = MAX_SLOTS

	// Cache the 'g' and used during locking
	citer.g = getg()

	// Randomized root start index is a random prime, modulo the number of root buckets
	citer.rootStartIdx = uintptr(primes[fastrand1()%uint32(len(primes))] % DEFAULT_BUCKETS)
	citer.idx = uint32((citer.rootStartIdx + 1) % DEFAULT_BUCKETS)

	cmapiternext_interlocked(it)
}

/*
	Atomic snapshot iteration
*/
func cmapiternext(it *hiter) {
	citer := (*concurrentIterator)(it.citerHdr)
	data := &citer.data
	t := it.t
	var hdr *bucketHdr
	var key, value unsafe.Pointer
	spins := 0
	var backoff int64 = 1
	g := citer.g
	gptr := uintptr(unsafe.Pointer(g))

	// Find the next key-value element. It assumes that if citer.offset < MAX_SLOTS, that citer.data actually holds valid information.
	// This is jumped to during iteration when we find and make a snapshot of the bucket we need, and are iterating through it to find
	// the next element.
findKeyValue:
	offset := uintptr(citer.offset)
	citer.offset++

	// If there is more to find, do so
	if offset < MAX_SLOTS {
		// If this cell is empty, loop again
		if data.hash[offset] == EMPTY {
			goto findKeyValue
		}

		// The key and values are present, but perform necessary indirection
		key = citer.data.key(t, offset)
		if t.indirectkey {
			key = *(*unsafe.Pointer)(key)
		}
		value = citer.data.value(t, offset)
		if t.indirectvalue {
			value = *(*unsafe.Pointer)(value)
		}

		// Set the iterator's data and we're done
		it.key = key
		it.value = value
		return
	}

	// If the offset == MAX_SLOTS, then we exhausted this bucketData, reset offset for next one
	citer.offset = 0

	// Find the next bucketData snapshot if there is one.
next:
	// If we have hit the last cell (as in, finished processing this bucketArray)
	if citer.idx == uint32(len(citer.arr.buckets)) {
		// If this is the root, we are done
		if citer.arr.backLink == nil {
			it.key = nil
			it.value = nil
			return
		} else {
			// Go back one
			citer.idx = citer.arr.backIdx
			citer.arr = citer.arr.backLink

			// Increment idx by one to move on to next bucketHdr
			citer.idx++

			goto next
		}
	}

	// Obtain header (and forward index by one for next iteration)
	hdr = (*bucketHdr)(atomic.Loadp(unsafe.Pointer(&citer.arr.buckets[citer.idx])))
	citer.idx++

	// Read ahead of time if we should skip to the next. Any cold cache-misses should be resolved here as well.
	if hdr == nil || atomic.Loaduintptr(&hdr.count) == 0 {
		goto next
	}

	for {
		// Reset backoff variables
		spins = 0
		backoff = DEFAULT_BACKOFF

		lock := atomic.Loaduintptr(&hdr.lock)

		// If the state of the bucket is INVALID, then either it's been deleted or been converted into an ARRAY; Reload and try again
		if lock == INVALID {
			// Reload hdr, since what it was pointed to has changed; idx - 1 because we incremented above
			hdr = (*bucketHdr)(atomic.Loadp(unsafe.Pointer(&citer.arr.buckets[citer.idx-1])))
			// If the hdr was deleted, then the data we're trying to find isn't here anymore (if it was at all).
			// hdr.count == 0 iff another Goroutine has created a new bucketData during a 'mapassign' but has not yet finished it's assignment.
			// In this case, there's still nothing here for us.
			if hdr == nil || atomic.Loaduintptr(&hdr.count) == 0 {
				goto next
			}
			// Loop again.
			continue
		}

		// If it's recursive, recurse and find new bucket
		if lock == ARRAY {
			citer.arr = (*bucketArray)(unsafe.Pointer(hdr))
			citer.idx = 0

			goto next
		}

		// Acquire lock on bucket
		if lock == UNLOCKED {
			// Attempt to acquire
			if atomic.Casuintptr(&hdr.lock, UNLOCKED, gptr) {
				break
			}
			continue
		}

		// If we already own the lock, then we're iterating while in a sync.Interlocked block, which is forbidden (Don't know how we got here then)
		if lock == gptr {
			println("...g # ", g.goid, ": Recursive-owned lock while iterating forbidden!Potential iteration while sync.Interlocked?")
			break
		}

		// Keep track of the current lock-holder
		holder := lock & LOCKED_MASK

		// Tight-spin until the current lock-holder releases lock
		for {
			if spins > SLEEP_AFTER_SPINS {
				timeSleep(backoff)

				// ≈33ms
				if backoff < MAX_BACKOFF {
					backoff *= 2
				}
			} else if spins > GOSCHED_AFTER_SPINS {
				Gosched()
			}
			spins++

			// We test the lock on each iteration
			lock = atomic.Loaduintptr(&hdr.lock)
			// If the previous lock-holder released the lock, attempt to acquire again.
			if lock != holder {
				if spins > 20 {
					println("...g # ", g.goid, ": Spins:", spins, ", Backoff:", backoff)
				}
				break
			}
		}
	}

	// Atomic snapshot of the bucketData
	typedmemmove(t.bucketdata, unsafe.Pointer(&citer.data), unsafe.Pointer(hdr))

	// Release lock
	atomic.Storeuintptr(&hdr.lock, UNLOCKED)

	// We have the snapshot we want.
	goto findKeyValue
}

func cmapiternext_interlocked(it *hiter) {
	citer := (*concurrentIterator)(it.citerHdr)
	var data *bucketData
	t := it.t
	var hdr *bucketHdr
	var key, value unsafe.Pointer
	spins := 0
	var backoff int64
	g := citer.g
	gptr := uintptr(unsafe.Pointer(g))

	// Find the next key-value element. It assumes that if citer.offset < MAX_SLOTS, that citer.data actually holds valid information.
	// This is jumped to during iteration when we acquire a valid bucketHdr with the information we need, and assumes that g.releaseBucket
	// holds the pointer to that bucketHdr.
findKeyValue:
	offset := uintptr(citer.offset)
	citer.offset++

	// Note that if g.releaseBucket == nil, this will crash... not sure why I even have it like this truth be told.
	if g.releaseBucket != nil {
		data = (*bucketData)(g.releaseBucket)
	}

	// If there is more to find, do so
	if offset < MAX_SLOTS {
		// If this cell is empty, loop again
		if data.hash[offset] == EMPTY {
			goto findKeyValue
		}

		// The key and values are present, but perform necessary indirection
		key = data.key(t, offset)
		if t.indirectkey {
			key = *(*unsafe.Pointer)(key)
		}
		value = data.value(t, offset)
		if t.indirectvalue {
			value = *(*unsafe.Pointer)(value)
		}

		// Set the iterator's data and we're done
		it.key = key
		it.value = value
		return
	}

	// If the offset == MAX_SLOTS, then we exhausted this bucketData, reset offset for next one
	citer.offset = 0

	// Since for interlocked iteration, we hold on to the lock until we no longer have more to iterate over, we must release it before acquiring a new one
	maprelease()

	// Find the next bucketData if there is one
next:
	// If this is the root, we need to make sure we wrap to rootStartIdx
	if citer.arr.backLink == nil {
		// If we wrapped around, we are done
		if uintptr(citer.idx) == citer.rootStartIdx {
			it.key = nil
			it.value = nil
			return
		} else if citer.idx == uint32(len(citer.arr.buckets)) {
			// Wrap around
			citer.idx = 0
		}
	} else if citer.idx == uint32(len(citer.arr.buckets)) {
		// In this case, we have finished iterating through this nested bucketArray, so go back one
		citer.idx = citer.arr.backIdx
		citer.arr = citer.arr.backLink

		// Increment idx by one to move on to next bucketHdr
		citer.idx++

		goto next
	}

	// Obtain header (and forward index by one for next iteration)
	hdr = (*bucketHdr)(atomic.Loadp(unsafe.Pointer(&citer.arr.buckets[citer.idx])))
	citer.idx++

	// Read ahead of time if we should skip.
	if hdr == nil || atomic.Loaduintptr(&hdr.count) == 0 {
		goto next
	}

	for {
		// Reset backoff variables
		spins = 0
		backoff = DEFAULT_BACKOFF

		lock := atomic.Loaduintptr(&hdr.lock)

		// If the state of the bucket is INVALID, then either it's been deleted or been converted into an ARRAY; Reload and try again
		if lock == INVALID {
			// Reload hdr, since what it was pointed to has changed; idx - 1 because we incremented above
			hdr = (*bucketHdr)(atomic.Loadp(unsafe.Pointer(&citer.arr.buckets[citer.idx-1])))
			// If the hdr was deleted, then the data we're trying to find isn't here anymore (if it was at all).
			// hdr.count == 0 iff another Goroutine has created a new bucketData during a 'mapassign' but has not yet finished it's assignment.
			// In this case, there's still nothing here for us.
			if hdr == nil || atomic.Loaduintptr(&hdr.count) == 0 {
				goto next
			}
			// Loop again.
			continue
		}

		// If it's recursive, recurse and find new bucket
		if lock == ARRAY {
			citer.arr = (*bucketArray)(unsafe.Pointer(hdr))
			citer.idx = 0

			goto next
		}

		// Acquire lock on bucket
		if lock == UNLOCKED {
			// Attempt to acquire
			if atomic.Casuintptr(&hdr.lock, UNLOCKED, gptr) {
				g.releaseBucket = unsafe.Pointer(hdr)
				break
			}
			continue
		}

		// If we already own the lock, then we're iterating while in a sync.Interlocked block, which is forbidden (Don't know how we got here then)
		if lock == gptr {
			g.releaseBucket = unsafe.Pointer(hdr)
			println("...g # ", g.goid, ": Recursive-owned lock while iterating forbidden!Potential iteration while sync.Interlocked?")
			break
		}

		// Keep track of the current lock-holder
		holder := lock & LOCKED_MASK

		// Tight-spin until the current lock-holder releases lock
		for {
			if spins > SLEEP_AFTER_SPINS {
				timeSleep(backoff)

				// ≈33ms
				if backoff < MAX_BACKOFF {
					backoff *= 2
				}
			} else if spins > GOSCHED_AFTER_SPINS {
				Gosched()
			}
			spins++

			// We test the lock on each iteration
			lock = atomic.Loaduintptr(&hdr.lock)
			// If the previous lock-holder released the lock, attempt to acquire again.
			if lock != holder {
				if spins > 20 {
					println("...g # ", g.goid, ": Spins:", spins, ", Backoff:", backoff)
				}
				break
			}
		}
	}
	g.releaseDepth++

	// We have the data we are looking for.
	goto findKeyValue
}

func makecmap(t *maptype, hint int64, h *hmap, bucket unsafe.Pointer) *hmap {
	// Initialize the hashmap if needed
	if h == nil {
		h = (*hmap)(newobject(t.hmap))
	}

	// Initialize and allocate our concurrentMap
	cmap := (*concurrentMap)(newobject(t.concurrentmap))
	cmap.root.buckets = make([]*bucketHdr, DEFAULT_BUCKETS)
	cmap.root.seed = fastrand1()

	h.chdr = unsafe.Pointer(cmap)
	h.flags = 8 // CONCURRENT flag is non exported.

	return h
}

/*
	Insertion into map...

	map[key] = value
*/
func cmapassign1(t *maptype, h *hmap, key unsafe.Pointer, val unsafe.Pointer) {
	// g := getg().m.curg
	// println("g #", getg().goid, ": cmapassign1")
	cmap := (*concurrentMap)(h.chdr)
	arr := &cmap.root
	var hash, idx, spins uintptr
	var backoff int64
	var hdr *bucketHdr
	g := getg()
	gptr := uintptr(unsafe.Pointer(g))

	// Finds the bucket associated with the key's hash; if it is recursive we jump back to here.
next:
	// Obtain the hash, index, and bucketHdr.
	hash = t.key.alg.hash(key, uintptr(arr.seed))
	idx = hash % uintptr(len(arr.buckets))
	hdr = (*bucketHdr)(atomic.Loadp(unsafe.Pointer(&arr.buckets[idx])))

	// If hdr is nil, then no bucketData has been created yet. Do a wait-free creation of bucketData;
	for hdr == nil {
		// Note that bucketData has the same first 3 fields as bucketHdr, and can be safely casted
		newHdr := (*bucketHdr)(newobject(t.bucketdata))
		// Since we're setting it, may as well attempt to acquire lock
		newHdr.lock = gptr
		// If we fail, then some other Goroutine has already placed theirs.
		if atomic.Casp1((*unsafe.Pointer)(unsafe.Pointer(&arr.buckets[idx])), nil, unsafe.Pointer(newHdr)) {
			// If we succeed, then we own this bucket and need to keep track of it
			g.releaseBucket = unsafe.Pointer(newHdr)
			// Also increment count of buckets
			atomic.Xadduintptr(&arr.count, 1)
		}
		// Reload hdr
		hdr = (*bucketHdr)(atomic.Loadp(unsafe.Pointer(&arr.buckets[idx])))
	}

	// Attempt to acquire lock
	for {
		// Reset backoff variables
		spins = 0
		backoff = DEFAULT_BACKOFF

		lock := atomic.Loaduintptr(&hdr.lock)

		// If the state of the bucket is INVALID, then either it's been deleted or been converted into an ARRAY; Reload and try again
		if lock == INVALID {
			// Reload hdr, since what it was pointed to has changed
			hdr = (*bucketHdr)(atomic.Loadp(unsafe.Pointer(&arr.buckets[idx])))
			// If the hdr was deleted, then attempt to create a new one and try again
			if hdr == nil {
				// Note that bucketData has the same first 3 fields as bucketHdr, and can be safely casted
				newHdr := (*bucketHdr)(newobject(t.bucketdata))
				// Since we're setting it, may as well attempt to acquire lock
				newHdr.lock = gptr
				// If we fail, then some other Goroutine has already placed theirs.
				if atomic.Casp1((*unsafe.Pointer)(unsafe.Pointer(&arr.buckets[idx])), nil, unsafe.Pointer(newHdr)) {
					// If we succeed, then we own this bucket and need to keep track of it
					g.releaseBucket = unsafe.Pointer(newHdr)
					// Also increment count of buckets
					atomic.Xadduintptr(&arr.count, 1)
				}
				hdr = (*bucketHdr)(atomic.Loadp(unsafe.Pointer(&arr.buckets[idx])))
			}
			// Loop again.
			continue
		}

		// If it's recursive, try again on new bucket
		if lock == ARRAY {
			arr = (*bucketArray)(unsafe.Pointer(hdr))
			goto next
		}

		// If we hold the lock
		if lock == gptr {
			break
		}

		// If the lock is uncontested
		if lock == UNLOCKED {
			// Attempt to acquire
			if atomic.Casuintptr(&hdr.lock, UNLOCKED, gptr) {
				g.releaseBucket = unsafe.Pointer(hdr)
				// println("...g # ", g.goid, ": Acquired lock")
				break
			}
			continue
		}

		// Keep track of the current lock-holder
		holder := lock & LOCKED_MASK

		// Tight-spin until the current lock-holder releases lock
		for {
			if spins > SLEEP_AFTER_SPINS {
				timeSleep(backoff)

				// ≈33ms
				if backoff < MAX_BACKOFF {
					backoff *= 2
				}
			} else if spins > GOSCHED_AFTER_SPINS {
				Gosched()
			}
			spins++

			// We test the lock on each iteration
			lock = atomic.Loaduintptr(&hdr.lock)
			// If the previous lock-holder released the lock, attempt to acquire again.
			if lock != holder {
				if spins > 20 {
					println("...g # ", g.goid, ": Spins:", spins, ", Backoff:", backoff)
				}
				break
			}
		}
	}

	g.releaseDepth++

	data := (*bucketData)(unsafe.Pointer(hdr))
	firstEmpty := -1

	// In the special case that hdr.count == 0, then the bucketData is empty and we should just add the data immediately.
	if hdr.count == 0 {
		data.assign(t, 0, hash, key, val)
		atomic.Xadduintptr(&hdr.count, 1)
		atomic.Xadd((*uint32)(unsafe.Pointer(&h.count)), 1)
		return
	}

	// Otherwise, we must scan all hashes to find a matching hash; if they match, check if they are equal
	for i, count := 0, hdr.count; i < MAX_SLOTS && count > 0; i++ {
		currHash := data.hash[i]
		if currHash == EMPTY {
			// Keep track of the first empty so we know what to assign into if we do not find a match
			if firstEmpty == -1 {
				firstEmpty = i
			}
			continue
		}
		count--

		// If the hash matches, check to see if keys are equal
		if hash == data.hash[i] {
			otherKey := data.key(t, uintptr(i))

			// If they are equal, update...
			if t.key.alg.equal(key, otherKey) {
				data.update(t, uintptr(i), key, val)
				return
			}
		}
	}

	// If firstEmpty is still -1 and the bucket is full, that means we did not find any empty slots, and should convert immediate
	if firstEmpty == -1 && hdr.count == MAX_SLOTS {
		// println("g #", getg().goid, ": Resizing...")
		// Allocate and initialize
		newArr := (*bucketArray)(newobject(t.bucketarray))
		newArr.lock = ARRAY
		newArr.buckets = make([]*bucketHdr, len(arr.buckets)*2)
		newArr.seed = fastrand1()
		newArr.backLink = arr
		newArr.backIdx = uint32(idx)

		// Rehash and move all key-value pairs
		for i := 0; i < MAX_SLOTS; i++ {
			k := data.key(t, uintptr(i))
			v := data.value(t, uintptr(i))

			// Rehash the key to the new seed
			newHash := t.key.alg.hash(k, uintptr(newArr.seed))
			newIdx := newHash % uintptr(len(newArr.buckets))
			newHdr := newArr.buckets[newIdx]
			newData := (*bucketData)(unsafe.Pointer(newHdr))

			// Check if the bucket is nil, meaning we haven't allocated to it yet.
			if newData == nil {
				newArr.buckets[newIdx] = (*bucketHdr)(newobject(t.bucketdata))
				newArr.count++

				newData = (*bucketData)(unsafe.Pointer(newArr.buckets[newIdx]))
				newData.assign(t, 0, newHash, k, v)
				newData.count++

				continue
			}

			// If it is not nil, then we must scan for the first non-empty slot
			for j := 0; j < MAX_SLOTS; j++ {
				currHash := newData.hash[j]
				if currHash == EMPTY {
					newData.assign(t, uintptr(j), newHash, k, v)
					newData.count++
					break
				}
			}
		}

		// Now dispose of old data and update the header's bucket; We have to be careful to NOT overwrite the header portion
		memclr(add(unsafe.Pointer(data), unsafe.Sizeof(bucketHdr{})), uintptr(MAX_SLOTS)*(uintptr(sys.PtrSize)+uintptr(t.keysize)+uintptr(t.valuesize)))
		// Update and then invalidate to point to nested ARRAY
		// arr.buckets[idx] = (*bucketHdr)(unsafe.Pointer(newArr))
		sync_atomic_StorePointer((*unsafe.Pointer)(unsafe.Pointer(&arr.buckets[idx])), unsafe.Pointer(newArr))
		atomic.Storeuintptr(&hdr.lock, INVALID)

		// Now that we have converted the bucket successfully, we still haven't assigned nor found a spot for our current key-value.
		// In this case try again, to reduce contention and increase concurrency over the lock
		arr = newArr
		g.releaseDepth = 0
		g.releaseBucket = nil

		goto next
	}

	// At this point, if firstEmpty == -1, then we exceeded the count without first finding one empty.
	// Hence, the first empty is going to be at idx hdr.count because there is none empty before it.
	// TODO: Clarification
	if firstEmpty == -1 {
		firstEmpty = int(hdr.count)
	}
	// At this point, firstEmpty is guaranteed to be non-zero and within bounds, hence we can safely assign to it
	data.assign(t, uintptr(firstEmpty), hash, key, val)
	// Since we just assigned to a new empty slot, we need to increment count
	atomic.Xadduintptr(&hdr.count, 1)
	atomic.Xadd((*uint32)(unsafe.Pointer(&h.count)), 1)
}

/*
	Releases the current held bucketHdr lock iff g.releaseDepth - 1 == 0
*/
func maprelease() {
	g := getg()
	if g.releaseBucket != nil {
		// println("g #", g.goid, ": released lock")
		hdr := (*bucketHdr)(g.releaseBucket)

		g.releaseDepth--
		if g.releaseDepth == 0 {
			atomic.Storeuintptr(&hdr.lock, UNLOCKED)
			// println("...g # ", g.goid, ": Released lock")

			g.releaseBucket = nil
		}

	}
}

/*
   Concurrent hashmap_fast.go function implementations
*/
func cmapaccess1_fast32(t *maptype, h *hmap, key uint32) unsafe.Pointer {
	// g := getg().m.curg
	// println("g #", getg().goid, ": cmapaccess1_fast32!")
	retval, _ := cmapaccess2_fast32(t, h, key)
	return retval
}

func cmapaccess2_fast32(t *maptype, h *hmap, key uint32) (unsafe.Pointer, bool) {
	// g := getg().m.curg
	// println("g #", getg().goid, ": cmapaccess2_fast32!")
	return cmapaccess(t, h, noescape(unsafe.Pointer(&key)),
		func(k1, k2 unsafe.Pointer) bool {
			return *(*uint32)(k1) == *(*uint32)(k2)
		})
}

func cmapaccess1_fast64(t *maptype, h *hmap, key uint64) unsafe.Pointer {
	// g := getg().m.curg
	// println("g #", getg().goid, ": cmapaccess1_fast64")
	retval, _ := cmapaccess2_fast64(t, h, key)
	return retval
}

func cmapaccess2_fast64(t *maptype, h *hmap, key uint64) (unsafe.Pointer, bool) {
	// g := getg().m.curg
	// println("g #", getg().goid, ": cmapaccess2_fast64!")
	return cmapaccess(t, h, noescape(unsafe.Pointer(&key)),
		func(k1, k2 unsafe.Pointer) bool {
			return *(*uint64)(k1) == *(*uint64)(k2)
		})
}

func cmapaccess1_faststr(t *maptype, h *hmap, key string) unsafe.Pointer {
	// g := getg().m.curg
	// println("g #", getg().goid, ": cmapaccess1_faststr")
	retval, _ := cmapaccess2_faststr(t, h, key)
	return retval
}

func cmapaccess2_faststr(t *maptype, h *hmap, key string) (unsafe.Pointer, bool) {
	// g := getg().m.curg
	// println("g #", getg().goid, ": cmapaccess2_faststr")
	return cmapaccess(t, h, noescape(unsafe.Pointer(&key)),
		func(k1, k2 unsafe.Pointer) bool {
			sk1 := (*stringStruct)(k1)
			sk2 := (*stringStruct)(k2)
			return sk1.len == sk2.len &&
				(sk1.str == sk2.str || memequal(sk1.str, sk2.str, uintptr(sk1.len)))
		})
}

/*
	Lookup into the map.

	value := map[key]
*/
func cmapaccess(t *maptype, h *hmap, key unsafe.Pointer, equal func(k1, k2 unsafe.Pointer) bool) (unsafe.Pointer, bool) {
	// println("...g #", getg().goid, ": Searching for Key: ", toKeyType(key).x, "," ,  toKeyType(key).y)
	cmap := (*concurrentMap)(h.chdr)
	arr := &cmap.root
	var hash, idx, spins uintptr
	var backoff int64
	var hdr *bucketHdr
	g := getg()
	gptr := uintptr(unsafe.Pointer(g))

	// Finds the bucket associated with the key's hash; if it is recursive we jump back to here.
next:
	// Obtain the hash, index, and bucketHdr
	hash = t.key.alg.hash(key, uintptr(arr.seed))
	idx = hash % uintptr(len(arr.buckets))
	hdr = (*bucketHdr)(atomic.Loadp(unsafe.Pointer(&arr.buckets[idx])))

	// Save time by looking ahead of time (testing) if the header or if the bucket is empty
	if hdr == nil || atomic.Loaduintptr(&hdr.count) == 0 {
		// println("...g #", getg().goid, ": hdr == nil or hdr.count == 0...")
		return unsafe.Pointer(&DUMMY_RETVAL[0]), false
	}

	// Attempt to acquire lock
	for {
		// Reset backoff variables
		spins = 0
		backoff = DEFAULT_BACKOFF

		// Testing lock
		lock := atomic.Loaduintptr(&hdr.lock)

		// If the state of the bucket is INVALID, then either it's been deleted or been converted into an ARRAY; Reload and try again
		if lock == INVALID {
			// Reload hdr, since what it was pointed to has changed
			hdr = (*bucketHdr)(atomic.Loadp(unsafe.Pointer(&arr.buckets[idx])))
			// If the hdr was deleted, then the data we're trying to find isn't here anymore (if it was at all)
			// hdr.count == 0 if another Goroutine has created a new bucketData after the old became INVALID. In this
			// case, there's still nothing here for us.
			if hdr == nil || atomic.Loaduintptr(&hdr.count) == 0 {
				return unsafe.Pointer(&DUMMY_RETVAL[0]), false
			}
			// Loop again.
			continue
		}

		// If it's recursive, try again on new bucket
		if lock == ARRAY {
			arr = (*bucketArray)(unsafe.Pointer(hdr))
			goto next
		}

		// If we hold the lock
		if lock == gptr {
			break
		}

		// If the lock is uncontested
		if lock == UNLOCKED {
			// Attempt to acquire
			if atomic.Casuintptr(&hdr.lock, UNLOCKED, gptr) {
				g.releaseBucket = unsafe.Pointer(hdr)
				break
			}
			continue
		}

		// Keep track of the current lock-holder
		holder := lock & LOCKED_MASK

		// Tight-spin until the current lock-holder releases lock
		for {
			if spins > SLEEP_AFTER_SPINS {
				timeSleep(backoff)

				// ≈33ms
				if backoff < MAX_BACKOFF {
					backoff *= 2
				}
			} else if spins > GOSCHED_AFTER_SPINS {
				Gosched()
			}
			spins++

			// We test the lock on each iteration
			lock = atomic.Loaduintptr(&hdr.lock)
			// If the previous lock-holder released the lock, attempt to acquire again.
			if lock != holder {
				if spins > 20 {
					println("...g # ", g.goid, ": Spins:", spins, ", Backoff:", backoff)
				}
				break
			}
		}
	}

	g.releaseDepth++

	data := (*bucketData)(unsafe.Pointer(hdr))

	// Search the bucketData for the data needed
	for i, count := 0, hdr.count; i < MAX_SLOTS && count > 0; i++ {
		currHash := data.hash[i]

		// We skip any empty hashes, but keep note of how many non-empty we find to know when to stop early
		if currHash == EMPTY {
			continue
		}
		count--

		// Check if the hashes are equal
		if currHash == hash {
			otherKey := data.key(t, uintptr(i))

			// Perform indirection on otherKey if necessary
			if t.indirectkey {
				otherKey = *(*unsafe.Pointer)(otherKey)
			}

			// If the keys are equal
			if equal(key, otherKey) {
				return data.value(t, uintptr(i)), true
			}
		}
	}

	// Only get to this point if we have not found the value in the map
	return unsafe.Pointer(&DUMMY_RETVAL[0]), false
}

func cmapaccess2(t *maptype, h *hmap, key unsafe.Pointer) (unsafe.Pointer, bool) {
	// g := getg().m.curg
	// println("g #", getg().goid, ": cmapaccess2!")

	return cmapaccess(t, h, key, t.key.alg.equal)
}

func cmapaccess1(t *maptype, h *hmap, key unsafe.Pointer) unsafe.Pointer {
	// g := getg().m.curg
	// println("g #", getg().goid, ": cmapaccess1")
	retval, _ := cmapaccess2(t, h, key)

	// Only difference is that we discard the boolean
	return retval
}

/*
	sync.Interlocked region acquire. Currently it is a VERY lazy implementation, as it currently just acquires the
	bucket, but until instrumentation in the compiler, it will do. Note that the acquired bucket does not have it's lock released until after the call to maprelease
	explicitly at the end of the region due to how the compiler prevents implicit maprelease calls.
*/
func mapacquire(t *maptype, h *hmap, key unsafe.Pointer) unsafe.Pointer {
	println("Val: ", *(*int)(key))
	if h.chdr == nil {
		throw("sync.Interlocked invoked on a non-concurrent map!")
	}

	return cmapaccess1(t, h, key)
}

/*
	Removal from map.

	delete(map, key)
*/
func cmapdelete(t *maptype, h *hmap, key unsafe.Pointer) {
	// g := getg().m.curg
	// println("g #", getg().goid, ": cmapdelete")
	cmap := (*concurrentMap)(h.chdr)
	arr := &cmap.root
	var hash, idx, spins uintptr
	var backoff int64
	var hdr *bucketHdr
	g := getg()
	gptr := uintptr(unsafe.Pointer(g))

	// Finds the bucket associated with the key's hash; if it is recursive we jump back to here.
next:
	// Obtain the hash, index, and bucketHdr.
	hash = t.key.alg.hash(key, uintptr(arr.seed))
	idx = hash % uintptr(len(arr.buckets))
	hdr = (*bucketHdr)(atomic.Loadp(unsafe.Pointer(&arr.buckets[idx])))

	// Save time by looking ahead of time (testing) if the header or if the bucket is empty
	if hdr == nil || atomic.Loaduintptr(&hdr.count) == 0 {
		return
	}

	// Attempt to acquire lock
	for {
		// Reset backoff variables
		spins = 0
		backoff = DEFAULT_BACKOFF

		// Testing lock
		lock := atomic.Loaduintptr(&hdr.lock)

		// If the state of the bucket is INVALID, then either it's been deleted or been converted into an ARRAY; Reload and try again
		if lock == INVALID {
			// Reload hdr, since what it was pointed to has changed
			hdr = (*bucketHdr)(atomic.Loadp(unsafe.Pointer(&arr.buckets[idx])))
			// If the hdr was deleted, then the data we're trying to find isn't here anymore (if it was at all)
			// hdr.count == 0 if another Goroutine has created a new bucketData after the old became INVALID. In this
			// case, there's still nothing here for us.
			if hdr == nil || atomic.Loaduintptr(&hdr.count) == 0 {
				return
			}
			// Loop again.
			continue
		}

		// If it's recursive, try again on new bucket
		if lock == ARRAY {
			arr = (*bucketArray)(unsafe.Pointer(hdr))
			goto next
		}

		// If we hold the lock
		if lock == gptr {
			break
		}

		// If the lock is uncontested
		if lock == UNLOCKED {
			// Attempt to acquire
			if atomic.Casuintptr(&hdr.lock, UNLOCKED, gptr) {
				g.releaseBucket = unsafe.Pointer(hdr)
				break
			}
			continue
		}

		// Keep track of the current lock-holder
		holder := lock & LOCKED_MASK

		// Tight-spin until the current lock-holder releases lock
		for {
			if spins > SLEEP_AFTER_SPINS {
				timeSleep(backoff)

				// ≈33ms
				if backoff < MAX_BACKOFF {
					backoff *= 2
				}
			} else if spins > GOSCHED_AFTER_SPINS {
				Gosched()
			}
			spins++

			// We test the lock on each iteration
			lock = atomic.Loaduintptr(&hdr.lock)
			// If the previous lock-holder released the lock, attempt to acquire again.
			if lock != holder {
				if spins > 20 {
					println("...g # ", g.goid, ": Spins:", spins, ", Backoff:", backoff)
				}
				break
			}
		}
	}

	g.releaseDepth++

	data := (*bucketData)(unsafe.Pointer(hdr))

	// TODO: Document
	for i, count := 0, hdr.count; i < MAX_SLOTS && count > 0; i++ {
		currHash := data.hash[i]

		// TODO: Document
		if currHash == EMPTY {
			continue
		}
		count--

		// If the hash matches, we can compare
		if currHash == hash {
			otherKey := data.key(t, uintptr(i))

			// Perform indirection on otherKey if necessary
			if t.indirectkey {
				otherKey = *(*unsafe.Pointer)(otherKey)
			}

			// If they match, we are set to remove them from the bucket
			if t.key.alg.equal(key, otherKey) {
				memclr(data.key(t, uintptr(i)), uintptr(t.keysize))
				memclr(data.value(t, uintptr(i)), uintptr(t.valuesize))
				data.hash[i] = EMPTY
				atomic.Xadd((*uint32)(unsafe.Pointer(&h.count)), -1)
				atomic.Xadduintptr(&hdr.count, ^uintptr(0))
				break
			}
		}
	}

	// If this bucketData is empty, we release it, making it eaiser for the iterator to determine it is empty.
	// For emphasis: If we removed the last element, we delete the bucketData.
	if hdr.count == 0 {
		sync_atomic_StorePointer((*unsafe.Pointer)(unsafe.Pointer(&arr.buckets[idx])), nil)
		// arr.buckets[idx] = nil
		atomic.Storeuintptr(&hdr.lock, INVALID)
		g.releaseBucket = nil
		g.releaseDepth = 0

		// Also decrement number of buckets
		atomic.Xadduintptr(&arr.count, ^uintptr(0))
	}
}

/*
	sync.Interlocked variants of map functions. These are a more optimized variants of the normal map functions above, which
	take advantage of sync.Interlock's Interlocked-only access invariant. Each function will test if the attempted access key
	is the same as the interlocked (currently acquired) key. Any other keys violate the one-bucket-per-Goroutine invariant.

	The variants use a structure which keeps track of the information needed for fast access to the interlocked key-value pair,
	aptly interlockedInfo. If a key is 'deleted'
*/

/*
	Interlocked version of mapaccess do a fast key comparison, and then return the value associated with it.
*/
func cmapaccess_interlocked(t *maptype, h *hmap, info *interlockedInfo, key unsafe.Pointer) unsafe.Pointer {
	// Do not allow accesses with keys not currently owned.
	if !t.key.alg.equal(info.key, key) {
		throw("Key indexed must be same as interlocked key!")
	}

	// If the key had been 'deleted', return the zero'd portion
	if (info.flags & KEY_DELETED) != 0 {
		return unsafe.Pointer(&DUMMY_RETVAL[0])
	}

	// Otherwise return the value directly
	return info.value
}

/*
	Interlocked version of mapassign does a fast key comparison, and then writes to the value itself
*/
func cmapassign_interlocked(t *maptype, h *hmap, info *interlockedInfo, key, value unsafe.Pointer) {
	k := info.key
	v := info.value

	// Perform some indirection on key if necessary; necessary for test of equality
	if t.indirectkey {
		k = *(*unsafe.Pointer)(k)
	}

	// Do not allow accesses with keys not currently owned.
	if !t.key.alg.equal(k, key) {
		throw("Key indexed must be same as interlocked key!")
	}

	// Since we will be doing a memcpy regardless of if it is present or not, unset the KEY_DELETED bit, but keep track of old value
	present := (info.flags & KEY_DELETED) == 0
	info.flags &= ^KEY_DELETED

	// In the case that the key was present, we are updating, hence we need to check if we also need to update the key as well
	if present {
		// Perform some indirection if necessary
		if t.indirectvalue {
			v = *(*unsafe.Pointer)(v)
		}

		// If we are required to update key, do so
		if t.needkeyupdate {
			typedmemmove(t.key, k, key)
		}

		typedmemmove(t.elem, v, value)
	} else {
		// In the case it has been deleted and we assigning again, we do a direct assignment
		// Perform indirection needed
		if t.indirectvalue {
			// println(".........g #", getg().goid, ": Value is indirect")
			vmem := newobject(t.elem)
			*(*unsafe.Pointer)(v) = vmem
			v = vmem
		}

		typedmemmove(t.elem, v, value)

		// Increment since we decrement when we originally delete it.
		atomic.Xadduintptr(&info.hdr.count, 1)
		atomic.Xadd((*uint32)(unsafe.Pointer(&h.count)), 1)
	}
}

func cmapdelete_interlocked(t *maptype, h *hmap, info *interlockedInfo, key unsafe.Pointer) {
	// Do not allow accesses with keys not currently owned.
	if !t.key.alg.equal(info.key, key) {
		throw("Key indexed must be same as interlocked key!")
	}

	// If the key had already been 'deleted' we're done.
	if (info.flags & KEY_DELETED) != 0 {
		return
	}

	// Update flags and zero value
	info.flags &= KEY_DELETED
	memclr(info.value, uintptr(t.valuesize))
	atomic.Xadd((*uint32)(unsafe.Pointer(&h.count)), -1)
	atomic.Xadduintptr(&info.hdr.count, ^uintptr(0))
}

func maprelease_interlocked(t *maptype, h *hmap, info *interlockedInfo) {
	// Handle cases where the current key has been marked for deletion. At this point, the corresponding value has already been zero'd.
	// Note, for iteration it must handle deleting its own keys after each successful iteration.
	if (info.flags & KEY_DELETED) != 0 {
		memclr(unsafe.Pointer(info.key), uintptr(t.keysize))
		*info.hash = EMPTY
	}

	// Check for the case when we deleted all elements in this bucket, and if we did, invalidate and delete it
	if info.hdr.count == 0 {
		// Invalidate and release the bucket (as it is being deleted)
		sync_atomic_StorePointer((*unsafe.Pointer)(unsafe.Pointer(info.hdrPtr)), nil)
		atomic.Storeuintptr(&info.hdr.lock, INVALID)

		// Also decrement number of buckets
		atomic.Xadduintptr(&info.parentHdr.count, ^uintptr(0))
	} else {
		// Otherwise, just release the lock on the bucket
		atomic.Storeuintptr(&info.hdr.lock, UNLOCKED)
	}
}
