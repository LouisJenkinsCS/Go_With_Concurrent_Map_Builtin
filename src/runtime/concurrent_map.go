
package runtime

import (
    "runtime/internal/atomic"
    "unsafe"
)

// Prime numbers pre-generated for the interlocked iterator to use when determining randomized start position
var primes = [...]int { 
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
    MAXZERO = 1024 // must match value in ../cmd/compile/internal/gc/walk.go
    
    // The maximum amount of buckets in a bucketArray
    DEFAULT_BUCKETS = 32
    // The maximum amount of buckets in a bucketData
    MAX_CHAINED_BUCKETS = 8

    // bucketHdr is unlocked and points to a bucketData
    UNLOCKED = 0
    // bucketHdr's 'b' is a bucketArray. Note that once this is true, it always remains so.
    RECURSIVE = 1 << 0
    // If the bucketHdr's bucket is a bucketData, and is held by an iterator
    ITERATOR = 1 << 1

    // If currHash is equal to this, it is not in use
    EMPTY = 0

    // Maximum spins until exponential backoff kicks in
    GOSCHED_AFTER_SPINS = 1
    SLEEP_AFTER_SPINS = 5

    // Default backoff
    DEFAULT_BACKOFF = 1024

    // See hashmap.go, this obtains a properly aligned offset to the data
    cdataOffset = unsafe.Offsetof(struct {
		b bucketData
		v int64
	}{}.v)
)

var DUMMY_RETVAL [MAXZERO]byte

/*
    bucketHdr is a header for each bucket, wherein it may point to either a bucketData or bucketArray depending on it's
    'lock' field. 'lock' can only be in one of the following states, all of which must be set/checked atomically...
    
    UNLOCKED:
        The bucket is uncontested (for now), and anyone may attempt to acquire it by setting the pointer to it's 'g'.
    OWNED:
        A Goroutine has set it's pointer to it's 'g', meaning it owns this header. This may ONLY happen when UNLOCKED.
        From here, it may mark the ITERATOR or RECURSIVE bit however it may please.
    ITERATOR:
        An iterator currently owns this bucket, and may allow other iterators through, or even readers, so long as they atomically
        check if 'readers' > 0, as 'readers' is only 0 if the original owner has already left (and unlocked and unmarked) the bucket.
        It is the job of the last 'reader' to unmark the ITERATOR bit when finished.
    RECURSIVE:
        The bucket has been converted, and the header now points to, a recursive bucektArray. This is a unique bit in that once it
        is set, it will no longer be unset.
*/
type bucketHdr struct {
    // The spinlock that each Goroutine will attempt to mutate in some way before making modifications on bucket
    lock uintptr
    // A count of readers currently occupying the below bucket (can only be non-zero if lock's ITERATOR bit is set)
    readers uintptr
    // The bucket (bucketData or bucketArray) this bucketHdr points to
    bucket unsafe.Pointer
}

/*
    bucketArray is a contiguous region of bucketHdr's, used to act as lookup for hashing keys to their
    corresponding values. Each bucketArray has it's own 'seed', which is used when hashing the key to their
    next index, as this greatly reduces overall collision. Each bucketArray, as they can be RECURSIVE, keep
    a backlink to their previous bucketArray and the backIdx of the indice that pointed to this bucketArray.
    The backIdx and backLink must be set by the Goroutine that setup this data.
*/
type bucketArray struct {
    // A contiguous region of bucketHdr's
    data []bucketHdr
    // Seed is different for each bucketArray to ensure that the re-hashing resolves to different indice
    seed uint32
    // Associated with the backlink (see below), used to signify what index the bucketHdr that pointed to this is
    backIdx uint32
    // Pointer to the previous bucketArray that pointed to this, if there is one.
    backLink *bucketArray
}

/*
    bucketData is a contiguous region of memory that contains all hashes (which can be used to identify if a certain cell is occupied),
    and as well, a contiguous array of keys and then values, optimized to reduce padding and increase performance.
*/
type bucketData struct {
    // Hash of the key-value corresponding to this index. If it is 0, it is empty. Aligned to cache line (64 bytes)
    hash [MAX_CHAINED_BUCKETS]uintptr
    /*
        key [MAX_CHAINED_BUCKETS]keyType
        val [MAX_CHAINED_BUCKETS]valType
    */
}

type concurrentMap struct {
    // Embedded to be contiguous in memory
    root bucketArray
}

type concurrentIterator struct {
    // The current index we are on
    idx uint32
    // Offset we are inside of the bucketData
    offset uint32
    // The bucketArray we currently are working on
    arr *bucketArray
    // The bucketData we are iterating over
    data bucketData
    // For the interlocked iterator, keeps track of randomized root we started at (and will wrap to)
    rootStartIdx uintptr
}

func (data *bucketData) key(t *maptype, idx uintptr) unsafe.Pointer {
    // Cast data to unsafe.Pointer to bypass Go's type system
    rawData := unsafe.Pointer(data)
    // The array of keys are located at the beginning of cdataOffset, and is contiguous up to MAX_CHAINED_BUCKETS
    keyOffset := uintptr(rawData) + uintptr(cdataOffset)
    // Now the key at index 'idx' is located at idx * t.keysize
    ourKeyOffset := keyOffset + idx * uintptr(t.keysize)
    return unsafe.Pointer(ourKeyOffset)
}

func (data *bucketData) value(t *maptype, idx uintptr) unsafe.Pointer {
    // Cast data to unsafe.Pointer to bypass Go's type system
    rawData := unsafe.Pointer(data)
    // The array of keys are located at the beginning of cdataOffset, and is contiguous up to MAX_CHAINED_BUCKETS
    keyOffset := uintptr(rawData) + uintptr(cdataOffset)
    // The array of values are located at the end of the array of keys, located at MAX_CHAINED_BUCKETS * t.keysize
    valueOffset := keyOffset + MAX_CHAINED_BUCKETS * uintptr(t.keysize)
    // Now the value at index 'idx' is located at idx * t.valuesize
    ourValueOffset := valueOffset + idx * uintptr(t.valuesize)
    return unsafe.Pointer(ourValueOffset)
}

/*
    Code below is used in the recursive-lock variant...
    gptr := uintptr(unsafe.Pointer(getg().m.curg))
    ...
    // Check if we are the current holders of the lock, spin if someone else holds the lock
    lockHolder := uintptr(flags &^ 0x7)
    ...
    if lockHolder == gptr {
        // println("......g #", getg().goid, ": re-entered the lock")
        return &(arr.data[idx])
    } else if lockHolder != 0 {
        spins++
        Gosched()
        continue
    }
    ...
    // Attempt to acquire the lock ourselves atomically
    if atomic.Casuintptr(&(arr.data[idx].info), flags, gptr | lowBits) {
        // println("......g #", getg().goid, ":  acquired lock")
        getg().releaseBucket = unsafe.Pointer(&arr.data[idx])
        // getg().releaseM = acquirem()
        if spins > 0 {
            // println(".........g #", getg().goid, ": wasted ", spins, " CPU cycles spinning")
        }
        return &(arr.data[idx])
    }
*/

func (data *bucketData) assign(t *maptype, idx, hash uintptr, key, value unsafe.Pointer) {
    k := data.key(t, idx)
    v := data.value(t, idx)

    // println("...g #", getg().goid, ": Assigning to index", idx, "of bucket", data, "=>Key: ", toKeyType(key).x, "," ,  toKeyType(key).y, " and Value: ", toKeyType(value).x, ",", toKeyType(value).y)

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

// Assumes the index 'idx' key already matches the passed key
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

    // By setting offset to MAX_CHAINED_BUCKETS, it allows it to bypass the findKeyValue portion without modification
    citer.offset = MAX_CHAINED_BUCKETS
    
    cmapiternext(it)
}

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
    
    // By setting offset to MAX_CHAINED_BUCKETS, it allows it to bypass the findKeyValue portion without modification
    citer.offset = MAX_CHAINED_BUCKETS
    
    // Randomized root start index is a random prime, modulo the number of root buckets
    citer.rootStartIdx = uintptr(primes[fastrand1() % uint32(len(primes))] % DEFAULT_BUCKETS)
    citer.idx = uint32((citer.rootStartIdx + 1) % DEFAULT_BUCKETS)

    cmapiternext_interlocked(it)
}

/*
    To help imagine the complexity of picking a randomized bucket when each bucketArray can be recursive, see the below diagram.

        [0] = Empty bucketData cell
        [D] = Non-Empty bucketData cell
        [R] = Recursive bucketArray cell
        ->  = Points to (used by recursive bucketArray cells)
        =>  = Iterator pointer (Where the iterator currently is)
        #N  = The N'th step in the diagram

        [D]
        [D]
        [0]
        [R] -> [0]
               [0]
               [0]
               [0]


    Imagine now that we randomly start at the 3rd cell...


    #7 =>   [D]
            [D]
    #1 =>   [0]
    #2 =>   [R] -> [0] <= #3
                   [0] <= #4
                   [0] <= #5
                   [0] <= #6
                
    This is the simple case. This assumes, of course, we started on a not already-nested cell. Now imagine a more realistic case

            [R] ----------------> [R] ------------> [D]
            [D]                   [0]               [D]                                             
            [D]                   [0]               [D]
            [R] -> [0]                              [R] -------> [0]
                   [0]                                           [0] // Start here
                   [0]                                           [0]                
                   [0]                                           [0]
                
    I may be overcomplicating things here...

    #14 => [R] -------------> => [R] ------------> [D] <= #6
           [D] <= #11            [0] <= #9         [D] <= #7                                            
           [D] <= #12            [0] <= #10        [D] <= #8
    #13 => [R] -> [0] <= #14                 #5 => [R] -------> [0] <= #4
                  [0] <= #15                                    [0] <= #1
                  [0] <= #16                                    [0] <= #2          
                  [0] <= #17                                    [0] <= #3
                

    Note now the reason why we end up going back is because we cannot afford to hit the same bucket twice,
    and if we recurse to another array, we could accidentally hit it again on the way back. Hence, we have to wrap around to hit 
    everything.
*/
func cmapiternext(it *hiter) {
    citer := (*concurrentIterator)(it.citerHdr)
    data := &citer.data
    t := it.t
    var hdr *bucketHdr
    var key, value unsafe.Pointer
    spins := 0
    var backoff int64 = 1

    findKeyValue:
        offset := uintptr(citer.offset)
        citer.offset++

        // If there is more to find, do so
        if offset < MAX_CHAINED_BUCKETS {
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

        // If the offset == MAX_CHAINED_BUCKETS, then we exhausted this bucketData, reset offset for next one 
        citer.offset = 0
    
    next:
        // If we have hit the last cell (as in, finished processing this bucketArray)
        if citer.idx == uint32(len(citer.arr.data)) {
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
        hdr = &citer.arr.data[citer.idx]
        citer.idx++

        // Read ahead of time if we should skip to the next or attempt to lock and acquire (Test-And-Test-And-Set)
        if atomic.Loadp(unsafe.Pointer(&hdr.bucket)) == nil {
            goto next
        }

        spins = 0
        backoff = DEFAULT_BACKOFF
        g := getg()
        gptr := uintptr(unsafe.Pointer(g))

        for {
            lock := atomic.Loaduintptr(&hdr.lock)

            // If it's recursive, recurse through and start over
            if lock == RECURSIVE {
                citer.arr = (*bucketArray)(hdr.bucket)
                citer.idx = 0
                
                goto next
            }

            // Acquire lock on bucket
            if lock == UNLOCKED {
                // Attempt to acquire
                if atomic.Casuintptr(&hdr.lock, 0, gptr) {
                    // If the bucket is nil, we can't iterate through it. Go to next one
                    if hdr.bucket == nil {
                        atomic.Storeuintptr(&hdr.lock, UNLOCKED)
                        goto next
                    }
                    break
                }
                continue
            }

            // If it is not RECURSIVE or UNLOCKED, another thread holds this bucket; Tight-spin until no one holds the lock
            for {
                // Backoff
                if spins > SLEEP_AFTER_SPINS {
                    timeSleep(backoff)
                    
                    // ≈33ms
                    if backoff < 33000000 {
                        backoff *= 2
                    }
                } else if spins > GOSCHED_AFTER_SPINS {
                    Gosched()
                }
                spins++

                // We test the lock on each iteration
                lock = atomic.Loaduintptr(&hdr.lock)
                // If no one currently holds the lock it is either RECURSIVE or UNLOCKED, so re-enter outer loop
                if lock == UNLOCKED || lock == RECURSIVE {
                    if spins > 20 {
                        println("...g # ", g.goid, ": Spins:", spins, ", Backoff:", backoff)
                    }
                    break
                }
            }
            
            // Reset backoff variables
            spins = 0
            backoff = DEFAULT_BACKOFF
        }

        // Atomic snapshot of the bucketData
        typedmemmove(t.bucketdata, unsafe.Pointer(&citer.data), hdr.bucket)
        
        // Release lock
        atomic.Storeuintptr(&hdr.lock, UNLOCKED)

        goto findKeyValue
}

func cmapiternext_interlocked(it *hiter) {
    g := getg()
    gptr := uintptr(unsafe.Pointer(g))
    citer := (*concurrentIterator)(it.citerHdr)
    var data *bucketData
    t := it.t
    var hdr *bucketHdr
    var key, value unsafe.Pointer
    spins := 0
    var backoff int64 = 1

    findKeyValue:
        offset := uintptr(citer.offset)
        citer.offset++

        if g.releaseBucket != nil {
            data = (*bucketData)((*bucketHdr)(g.releaseBucket).bucket)
        }

        // If there is more to find, do so
        if offset < MAX_CHAINED_BUCKETS {
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

        // If the offset == MAX_CHAINED_BUCKETS, then we exhausted this bucketData, reset offset for next one 
        citer.offset = 0

        // Since for interlocked iteration, we hold on to the lock until we no longer have more to iterate over, we must release it before acquiring a new one
        maprelease()
    
    next:
        // If this is the root, we need to make sure we wrap to rootStartIdx
        if citer.arr.backLink == nil {
            // If we wrapped around, we are done
            if uintptr(citer.idx) == citer.rootStartIdx {
                it.key = nil
                it.value = nil
                return
            } else if citer.idx == uint32(len(citer.arr.data)) {
                // Wrap around
                citer.idx = 0
            }
        } else if citer.idx == uint32(len(citer.arr.data)) {
            // In this case, we have finished iterating through this nested bucketArray, so go back one
            citer.idx = citer.arr.backIdx
            citer.arr = citer.arr.backLink

            // Increment idx by one to move on to next bucketHdr
            citer.idx++

            goto next
        }

        // Obtain header (and forward index by one for next iteration)
        hdr = &citer.arr.data[citer.idx]
        citer.idx++

        // Read ahead of time if we should skip to the next or attempt to lock and acquire (Test-And-Test-And-Set)
        if atomic.Loadp(unsafe.Pointer(&hdr.bucket)) == nil {
            goto next
        }

        spins = 0
        backoff = DEFAULT_BACKOFF

        for {
            lock := atomic.Loaduintptr(&hdr.lock)

            // If it's recursive, recurse through and start over
            if lock == RECURSIVE {
                citer.arr = (*bucketArray)(hdr.bucket)
                citer.idx = 0
                
                goto next
            }

            // Acquire lock on bucket
            if lock == UNLOCKED {
                // Attempt to acquire
                if atomic.Casuintptr(&hdr.lock, 0, gptr) {
                    // If the bucket is nil, we can't iterate through it. Go to next one
                    if hdr.bucket == nil {
                        atomic.Storeuintptr(&hdr.lock, UNLOCKED)
                        goto next
                    }
                    g.releaseBucket = unsafe.Pointer(hdr)
                    break
                }
                continue
            }

            // If it is not RECURSIVE or UNLOCKED, another thread holds this bucket; Tight-spin until no one holds the lock
            for {
                // Backoff
                if spins > SLEEP_AFTER_SPINS {
                    timeSleep(backoff)

                    // ≈33ms
                    if backoff < 33000000 {
                        backoff *= 2
                    }
                } else if spins > GOSCHED_AFTER_SPINS {
                    Gosched()
                }
                spins++

                // We test the lock on each iteration
                lock = atomic.Loaduintptr(&hdr.lock)
                // If no one currently holds the lock it is either RECURSIVE or UNLOCKED, so re-enter outer loop
                if lock == UNLOCKED || lock == RECURSIVE {
                    if spins > 20 {
                        println("...g # ", g.goid, ": Spins:", spins, ", Backoff:", backoff)
                    }
                    break
                }
            }
            
            // Reset backoff variables
            spins = 0
            backoff = DEFAULT_BACKOFF
        }
        g.releaseDepth++

        goto findKeyValue
}

func makecmap(t *maptype, hint int64, h *hmap, bucket unsafe.Pointer) *hmap {
    // println("Inside of makecmap: Key: ", t.key.string(), "; Value: ", t.elem.string())
    // println("Sizeof bucketHdr: ", unsafe.Sizeof(bucketHdr{}), "\nSizeof bucketArray: ", unsafe.Sizeof(bucketArray{}))
    // println("Sizeof bucketData: ", t.bucketdata.size, "\nSizeof cdataOffset: ", cdataOffset, "\nSizeof bmap", t.bucket.size, "\nName of bmap: ", t.bucket.string())
    // println("Name of bucketData: ", t.bucketdata.string(), "\nName of bucketArray: ", t.bucketarray.string(), "\nName of bucketHdr: ", t.buckethdr.string())
    // Initialize the hashmap if needed
    if h == nil {
		h = (*hmap)(newobject(t.hmap))
	}
    
    // Initialize and allocate our concurrentMap
    cmap := (*concurrentMap)(newobject(t.concurrentmap))
    cmap.root.data = make([]bucketHdr, DEFAULT_BUCKETS)
    cmap.root.seed = fastrand1()


    h.chdr = unsafe.Pointer(cmap)
    h.flags = 8
    
    return h
}

func cmapassign1(t *maptype, h *hmap, key unsafe.Pointer, val unsafe.Pointer) {
    // g := getg().m.curg
    // println("g #", getg().goid, ": cmapassign1")
    // println("...g #", getg().goid, ": Adding Key: ", toKeyType(key).x, "," ,  toKeyType(key).y, " and Value: ", toKeyType(val).x, ",", toKeyType(val).y)
    cmap := (*concurrentMap)(h.chdr)
    arr := &cmap.root
    var hash, idx, spins uintptr
    var backoff int64
    var hdr *bucketHdr
    g := getg()
    gptr := uintptr(unsafe.Pointer(g))

    next :
        // Obtain the hash, index, and bucketHdr.
        hash = t.key.alg.hash(key, uintptr(arr.seed))
        idx = hash % uintptr(len(arr.data))
        hdr = &arr.data[idx]
        spins = 0
        backoff = DEFAULT_BACKOFF

        // Attempt to acquire lock
        for {
            lock := atomic.Loaduintptr(&hdr.lock)

            // If it's recursive, try again on new bucket
            if lock == RECURSIVE {
                arr = (*bucketArray)(hdr.bucket)
                goto next
            }
            
            // If we hold the lock
            if lock == gptr {
                break
            }

            // If the lock is uncontested
            if lock == UNLOCKED  {
                // Attempt to acquire
                if atomic.Casuintptr(&hdr.lock, UNLOCKED, gptr) {
                    g.releaseBucket = unsafe.Pointer(hdr)
                    // println("...g # ", g.goid, ": Acquired lock")
                    break
                }
                continue
            }

            // If it is not RECURSIVE or UNLOCKED, another thread holds this bucket; Tight-spin until no one holds the lock
            for {
                if spins > SLEEP_AFTER_SPINS {
                    timeSleep(backoff)
                    
                    // ≈33ms
                    if backoff < 33000000 {
                        backoff *= 2
                    }
                } else if spins > GOSCHED_AFTER_SPINS {
                    Gosched()
                }
                spins++

                // We test the lock on each iteration
                lock = atomic.Loaduintptr(&hdr.lock)
                // If no one currently holds the lock it is either RECURSIVE or UNLOCKED, so re-enter outer loop
                if lock == UNLOCKED || lock == RECURSIVE {
                    if spins > 20 {
                        println("...g # ", g.goid, ": Spins:", spins, ", Backoff:", backoff)
                    }
                    break
                }
            }
            
            // Reset backoff variables
            spins = 0
            backoff = DEFAULT_BACKOFF
        }

        g.releaseDepth++
        
        // If bucket is nil, then we allocate a new one.
        if hdr.bucket == nil {
            hdr.bucket = newobject(t.bucketdata)
            
            // Since we just created a new bucket, directly add the key and value to the map
            (*bucketData)(hdr.bucket).assign(t, 0, hash, key, val)
            atomic.Xadd((*uint32)(unsafe.Pointer(&h.count)), 1)
            return
        }

        data := (*bucketData)(hdr.bucket)
        firstEmpty := -1

        // Otherwise, we must scan all hashes to find a matching hash; if they match, check if they are equal
        for i := 0; i < MAX_CHAINED_BUCKETS; i++ {
            currHash := data.hash[i]
            if currHash == EMPTY {
                // Keep track of the first empty so we know what to assign into if we do not find a match
                if firstEmpty == -1 {
                    firstEmpty = i
                }
                continue
            }

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

        // If firstEmpty is still -1, that means we did not find any empty slots, and should convert immediate
        if firstEmpty == -1 {
            // Allocate and initialize
            newArr := (*bucketArray)(newobject(t.bucketarray))
            newArr.data = make([]bucketHdr, len(arr.data) * 2)
            newArr.seed = fastrand1()
            newArr.backLink = arr
            newArr.backIdx = uint32(idx)

            // Rehash and move all key-value pairs
            for i := 0; i < MAX_CHAINED_BUCKETS; i++ {
                k := data.key(t, uintptr(i))
                v := data.value(t, uintptr(i))

                // Rehash the key to the new seed
                newHash := t.key.alg.hash(k, uintptr(newArr.seed))
                newIdx := newHash % uintptr(len(newArr.data))
                newHdr := &newArr.data[newIdx]
                newData := (*bucketData)(newHdr.bucket)
                
                // Check if the bucket is nil, meaning we haven't allocated to it yet.
                if newData == nil {
                    newHdr.bucket = newobject(t.bucketdata)
                    (*bucketData)(newHdr.bucket).assign(t, 0, newHash, k, v)
                    continue
                }

                // If it is not nil, then we must scan for the first non-empty slot
                for j := 0; j < MAX_CHAINED_BUCKETS; j++ {
                    currHash := newData.hash[j]
                    if currHash == EMPTY {
                        newData.assign(t, uintptr(j), newHash, k, v)
                        break
                    }
                }
            }

            // Now dispose of old data and update the header's bucket
            memclr(unsafe.Pointer(data), t.bucketdata.size)
            hdr.bucket = unsafe.Pointer(newArr)
            
            // Now that we have converted the bucket successfully, we still haven't assigned nor found a spot for the key-value pairs.
            // In this case, simply mark the bucket as RECURSIVE and try again, to reduce contention and increase concurrency over the lock
            arr = newArr
            g.releaseDepth--
            atomic.Storeuintptr(&hdr.lock, RECURSIVE)
            goto next
        }

    // At this point, firstEmpty is guaranteed to be non-zero and within bounds, hence we can safely assign to it
    data.assign(t, uintptr(firstEmpty), hash, key, val)
}

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
        func (k1, k2 unsafe.Pointer) bool { 
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
        func (k1, k2 unsafe.Pointer) bool { 
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
        func (k1, k2 unsafe.Pointer) bool { 
            sk1 := (*stringStruct)(k1)
            sk2 := (*stringStruct)(k2)
            return sk1.len == sk2.len && 
            (sk1.str == sk2.str || memequal(sk1.str, sk2.str, uintptr(sk1.len))) 
        })
}

// The primary cmapaccess function
func cmapaccess(t *maptype, h *hmap, key unsafe.Pointer, equal func(k1, k2 unsafe.Pointer) bool) (unsafe.Pointer, bool) {
    // println("...g #", getg().goid, ": Searching for Key: ", toKeyType(key).x, "," ,  toKeyType(key).y)
    cmap := (*concurrentMap)(h.chdr)
    arr := &cmap.root
    var hash, idx, spins uintptr
    var backoff int64
    var hdr *bucketHdr
    g := getg()
    gptr := uintptr(unsafe.Pointer(g))

    next:
        // Obtain the hash, index, and bucketHdr
        hash = t.key.alg.hash(key, uintptr(arr.seed))
        idx = hash % uintptr(len(arr.data))
        hdr = &arr.data[idx]
        spins = 0
        backoff = DEFAULT_BACKOFF

        // Save time by looking ahead of time (testing) if the header is nil
        if atomic.Loadp(unsafe.Pointer(&hdr.bucket)) == nil {
            return unsafe.Pointer(&DUMMY_RETVAL[0]), false
        }
        
        // Attempt to acquire lock
        for {
            lock := atomic.Loaduintptr(&hdr.lock)
            
            // If it's recursive, try again on new bucket
            if lock == RECURSIVE {
                arr = (*bucketArray)(hdr.bucket)
                goto next
            }

            // If we hold the lock
            if lock == gptr {
                break
            }

            // If the lock is uncontested
            if lock == UNLOCKED  {
                // Attempt to acquire
                if atomic.Casuintptr(&hdr.lock, 0, gptr) {
                    g.releaseBucket = unsafe.Pointer(hdr)
                    break
                }
                continue
            }

            // If it is not RECURSIVE or UNLOCKED, another thread holds this bucket; Tight-spin until no one holds the lock
            for {
                // Backoff
                if spins > SLEEP_AFTER_SPINS {
                    timeSleep(backoff)
                    
                    // ≈33ms
                    if backoff < 33000000 {
                        backoff *= 2
                    }
                } else if spins > GOSCHED_AFTER_SPINS {
                    Gosched()
                }
                spins++

                // We test the lock on each iteration
                lock = atomic.Loaduintptr(&hdr.lock)
                // If no one currently holds the lock it is either RECURSIVE or UNLOCKED, so re-enter outer loop
                if lock == UNLOCKED || lock == RECURSIVE {
                    if spins > 20 {
                        println("...g # ", g.goid, ": Spins:", spins, ", Backoff:", backoff)
                    }
                    break
                }
            }
            
            // Reset backoff variables
            spins = 0
            backoff = DEFAULT_BACKOFF
        }

        g.releaseDepth++

        // If the bucket is nil, then it is empty, stop here
        if hdr.bucket == nil {
            return unsafe.Pointer(&DUMMY_RETVAL[0]), false
        }

        data := (*bucketData)(hdr.bucket)
        
        // Search the bucketData for the data needed
        for i := 0; i < MAX_CHAINED_BUCKETS; i++ {
            currHash := data.hash[i]
            
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
    Priority/Locking Order:
        A key's priority is the order in which it appears in the map, from top down. What this means
        is this... imagine the following, the priorities are denoted by the number adjacent to it.

        [] <= #1
        [] <= #2
        [] <= #3
        [] <= #4

        What this mean is that, keys are acquired in an order from top down. Now imagine a nested map

        [] -----------------------> [] #1
        [] -----------------> [] #5 [] #2
        [] ----------> [] #9  [] #6 [] #3
        [] --> [] #13  [] #10 [] #7 [] #4
               [] #14  [] #11 [] #8
               [] #15  [] #12  
               [] #16
        
        Wherever a key happens to hash is the order it will be acquired in. 

        Imagine the following:

        [ ] -------------------------> [B] #1
        [ ] ------------------> [ ] #5 [ ] #2
        [ ] ----------> [E] #9  [ ] #6 [C] #3
        [ ] -> [G] #13  [D] #10 [F] #7 [ ] #4
               [ ] #14  [ ] #11 [ ] #8
               [A] #15  [ ] #12  
               [ ] #16
        
        The order of lock acquisition is:
            {B, C, F, E, D, G, A}

        Now what about if there is some mutations in between locating the appropriate bucketHdr's and acquiring their lock...

        Deleted: B

        [ ] -------------------------> [ ] #1
        [ ] ------------------> [ ] #5 [ ] #2
        [ ] ----------> [E] #9  [ ] #6 [C] #3
        [ ] -> [G] #13  [D] #10 [F] #7 [ ] #4
               [ ] #14  [ ] #11 [ ] #8
               [A] #15  [ ] #12  
               [ ] #16
        
        This is easy to handle, as we can just check if the key exists after acquring the lock. If it isn't present, depending on
        what the user specified, we either allocate another object for the user, or return a failure (and let compiler generate code
        for handling errors)

        Now, lets try a more interesting problem...

        Resized: B

        [ ] --------------------------> [ ] -----> [ ] #1
        [ ] ------------------> [ ] #8  [ ] #5     [ ] #2
        [ ] ----------> [E] #12 [ ] #9  [C] #6     [B] #3
        [ ] -> [G] #16  [D] #13 [F] #10 [ ] #7     [ ] #4
               [ ] #17  [ ] #14 [ ] #11
               [A] #18  [ ] #15  
               [ ] #19

        Note: The only change is that the priorities shifted by a set amount. This shift is mostly irrelevant, because
        the lock acquisition order does not change:
            {B, C, F, E, D, G, A}
        
        As keys cannot be moved, unless hashed to a recursive bucket (which may be a reason why we shouldn't limit nesting depth),
        modifications are only an issue when you have more than one key assigned to a single bucket (upon which you handle those
        respectively).

        Sorry if I'm being unclear, a bit tired.
*/
func mapacquire(t *maptype, h *hmap, key unsafe.Pointer) unsafe.Pointer {
    println("Val: ", *(*int)(key))
    if h.chdr == nil {
        throw("sync.Interlocked invoked on a non-concurrent map!")
    }
    // // Dummy slice of keys (until we acquire multiple keys and pass them here)
    // var keys []unsafe.Pointer
    // cmap := (*concurrentMap)(h.chdr)

    // var keyBuckets [16][]*bucketHdr
    // for _, k := range keys {
    //     arr := &cmap.root
    //     hash := t.key.alg.hash(key, arr.seed)
    //     idx := hash % uintptr(len(arr.data))
    //     keyBuckets[idx] = append(keyBuckets[idx], &arr.data[idx])
    // }


    return cmapaccess1(t, h, key)
}

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

    next :
        // Obtain the hash, index, and bucketHdr.
        hash = t.key.alg.hash(key, uintptr(arr.seed))
        idx = hash % uintptr(len(arr.data))
        hdr = &arr.data[idx]
        spins = 0
        backoff = DEFAULT_BACKOFF

        // Attempt to acquire lock
        for {
            lock := atomic.Loaduintptr(&hdr.lock)

            // If it's recursive, try again on new bucket
            if lock == RECURSIVE {
                arr = (*bucketArray)(hdr.bucket)
                goto next
            }
            
            // If we hold the lock
            if lock == gptr {
                break
            }

            // If the lock is uncontested
            if lock == UNLOCKED  {
                // Attempt to acquire
                if atomic.Casuintptr(&hdr.lock, 0, gptr) {
                    g.releaseBucket = unsafe.Pointer(hdr)
                    break
                }
                continue
            }

            // If it is not RECURSIVE or UNLOCKED, another thread holds this bucket; Tight-spin until no one holds the lock
            for {
                if spins > SLEEP_AFTER_SPINS {
                    timeSleep(backoff)
                    
                    // ≈33ms
                    if backoff < 33000000 {
                        backoff *= 2
                    }
                } else if spins > GOSCHED_AFTER_SPINS {
                    Gosched()
                }
                spins++

                // We test the lock on each iteration
                lock = atomic.Loaduintptr(&hdr.lock)
                // If no one currently holds the lock it is either RECURSIVE or UNLOCKED, so re-enter outer loop
                if lock == UNLOCKED || lock == RECURSIVE {
                    if spins > 20 {
                        println("...g # ", g.goid, ": Spins:", spins, ", Backoff:", backoff)
                    }
                    break
                }
            }
            
            // Reset backoff variables
            spins = 0
            backoff = DEFAULT_BACKOFF
        }

        g.releaseDepth++

        // If the bucket is nil, then the key is not present in the map
        if hdr.bucket == nil {
            return
        }

        data := (*bucketData)(hdr.bucket)

        // Number of buckets empty; used to signify whether or not we should delete this bucket when we finish.
        isEmpty := MAX_CHAINED_BUCKETS
        for i := 0; i < MAX_CHAINED_BUCKETS; i++ {
            currHash := data.hash[i]

            if currHash == EMPTY {
                continue
            }

            // If there is a hash that is not 0, then the bucket is not empty
            isEmpty--

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
                    isEmpty++

                    // We save time by directly looping through the rest of the hashes to determine if there are other non-empty indice, besides this one, if we haven't found one already
                    for j := i + 1; isEmpty == MAX_CHAINED_BUCKETS && j < MAX_CHAINED_BUCKETS; j++ {
                        // If it is not empty, we decrement the count of empty index
                        if data.hash[j] != 0 {
                            isEmpty--
                        }
                    }
                    break
                }
            }
        }

        // If isEmpty == MAX_CHAINED_BUCKETS, then there are no indice still in use, so we delete the bucket to allow the iterator an easier time
        if isEmpty == MAX_CHAINED_BUCKETS {
            hdr.bucket = nil
        }
}

