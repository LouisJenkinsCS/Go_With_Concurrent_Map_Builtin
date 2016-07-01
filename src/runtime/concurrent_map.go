
package runtime

import (
    "runtime/internal/atomic"
    "unsafe"
)

/*
    TODO: While the concurrent map works for non fast types (string, uint32, and uint64) values,
    it will fail and produce undefined behavior in a way, making me believe the compiler
    does something special for fast types. Figure out what is causing this and fix it.
*/

const (
    MAXZERO = 1024 // must match value in ../cmd/compile/internal/gc/walk.go
    
    // The maximum amount of buckets in a bucketArray
    MAXBUCKETS = 16
    // The maximum amount of buckets in a bucketChain
    MAXCHAIN = 8

    CHAINED = 0
    // bucketHdr's 'b' is a bucketArray. Note that once this is true, it always remains so.
    ARRAY = 1 << 0
    // If the bucketHdr's 'b' is locked, another goroutine is currently accessing it.
    LOCKED = 1 << 1

    // bucketChain is unused
    UNUSED = 0
    // bucketChain is in use
    USED = 1 << 0

    // The bucketChain is full and should be resized
    BUCKET_STATUS_FULL = 1
    // The bucketChain is empty (and head is nil) and caller should handle allocation of bucket.
    BUCKET_STATUS_EMPTY = 2
    // The bucketChain has been walked and successfully found a key that matched the request
    BUCKET_STATUS_FOUND = 3
    // The bucketChain has been walked and has not successfully found a key that matched the request
    BUCKET_STATUS_NOT_FOUND = 4
    // The bucketChain has stopped at the first empty node (as per request)
    BUCKET_STATUS_FIRST_EMPTY = 5
    // The bucketChain has been walked and ended prematurely, but returned the last used node (as per request)
    BUCKET_STATUS_LAST_FOUND = 6

    // Search will default to finding the bucket with the matching key, returning nil if it does not find one
    BUCKET_SEARCH_DEFAULT = 0
    // Search for the first empty bucket. Prevents the needless search of buckets beyond the first if that is all we need.
    BUCKET_SEARCH_FIRST_EMPTY = 1
    // If we walk the bucketChain and find nil, return the last bucket found.
    BUCKET_SEARCH_LAST_FOUND = 2

    // See hashmap.go, this obtains a properly aligned offset to the data
    cdataOffset = unsafe.Offsetof(struct {
		b bucketChain
		v int64
	}{}.v)
)

type bucketStatus int
type bucketSearch int

var DUMMY_RETVAL [MAXZERO]byte

type bucketHdr struct {
    // Is either a bucketArray or bucketChain
    bucket unsafe.Pointer
    /*
    	Recursive Lock Variant:
	        The least significant 3 bits are used as flags and the rest
	        are a pointer to the current 'g' that holds the lock. The 'g' is
	        used to allow it to become reentrant.
	
	        Checking the lock can be as easy as masking out the first three bits.
	        An overall CAS can also make it easier to tell when it is no longer locked,
	        as as soon as the 'g' holding the lock finishes, it will zero the portion
	        used to store the address. This can cause the CAS to fail during a spin
	        and be used to detect if manages to acquire the lock.
	        
    	Non-Recursive Lock Variant:
    		The least significant 3 bits are used as flags, while the rest remain unused.
    		
    		Checking the lock is more efficient in that it only needs to attempt to CAS
    		this flag from 0 to LOCKED. Since this flag shares ARRAY bit, if ARRAY bit is ever
    		set, it becomes impossible to CAS from 0 to LOCKED, as 0 != ARRAY, and hence will
    		always fail, meaning a simple CAS can check both if it is LOCKED or if it is an
    		ARRAY, and even attempt to acquire the lock all in the same instruction.
    */
    info uintptr
}

type bucketArray struct {
    data []bucketHdr
    // Seed is different for each bucketArray to ensure that the re-hashing resolves to different indice
    seed uint32
    // Size of the current slice of bucketHdr's
    size uint32
}

/*
    bucketData is a contiguous region of memory that contains all 
*/
type bucketData struct {
    // Hash of the key-value corresponding to this index. If it is 0, it is empty. Aligned to cache line (64 bytes)
    hash [MAXCHAIN]uintptr
}

/*
    bucketChain is a chain of buckets, wherein it acts as a simple single-linked list 
    of nodes of buckets. A bucketChain with a nil key is not in-use and is safe to be
    recycled, as access should be protected by a lock in the bucketHdr this belongs to.
*/
type bucketChain struct {
    next *bucketChain
    flags uintptr
    /*
        Two extra fields can be accessed through the use of unsafe.Pointer, needed because we take
        each key and value by value, (if it is a indirectKey, then we just store the pointer).
        Hence, immediately following next, is each key and value
    */
}

type concurrentMap struct {
    // Embedded to be contiguous in memory
    root bucketArray
}


func (chain *bucketChain) key(t *maptype) unsafe.Pointer {
    // Cast chain to an unsafe.Pointer to bypass Go's type system
    rawChain := unsafe.Pointer(chain)
    // The key offset is right at the end of the bucketChain
    keyOffset := uintptr(rawChain) + uintptr(cdataOffset)
    return unsafe.Pointer(keyOffset)
}

func (data *bucketData) key(t *maptype, idx uintptr) {
    // Cast data to unsafe.Pointer to bypass Go's type system
    rawData := unsafe.Pointer(data)
    // The array of keys are located at the beginning of cdataOffset, and is contiguous up to MAXCHAIN
    keyOffset := uintptr(rawData) + uintptr(cdataOffset)
    // Now the key at index 'idx' is located at idx * t.keysize
    ourKeyOffset := keyOffset + idx * t.keysize
    return unsafe.Pointer(ourKeyOffset)
}

func (data *bucketData) value(t *maptype, idx uintptr) {
    // Cast data to unsafe.Pointer to bypass Go's type system
    rawData := unsafe.Pointer(data)
    // The array of keys are located at the beginning of cdataOffset, and is contiguous up to MAXCHAIN
    keyOffset := uintptr(rawData) + uintptr(cdataOffset)
    // The array of values are located at the end of the array of keys, located at MAXCHAIN * t.keysize
    valueOffset := keyOffset + MAXCHAIN * t.keysize
    // Now the value at index 'idx' is located at idx * t.valuesize
    ourValueOffset := valueOffset + idx * t.valuesize
    return unsafe.Pointer(ourValueOffset)
}

func (chain *bucketChain) value(t *maptype) unsafe.Pointer {
    // Cast chain to an unsafe.Pointer to bypass Go's type system
    rawChain := unsafe.Pointer(chain)
    // The key offset is right at the end of the bucketChain
    keyOffset := uintptr(rawChain) + uintptr(cdataOffset)
    // The value offset is right after the end of the key (The size of the key is kept in maptype)
    valOffset := keyOffset + uintptr(t.keysize)

    return unsafe.Pointer(valOffset)
}

/*
    Code below is used in the recursive-lock variant...
    gptr := uintptr(unsafe.Pointer(getg().m.curg))
    ...
    // Check if we are the current holders of the lock, spin if someone else holds the lock
    lockHolder := uintptr(flags &^ 0x7)
    ...
    if lockHolder == gptr {
        println("......g #", g.goid, ": re-entered the lock")
        return &(arr.data[idx])
    } else if lockHolder != 0 {
        spins++
        Gosched()
        continue
    }
    ...
    // Attempt to acquire the lock ourselves atomically
    if atomic.Casuintptr(&(arr.data[idx].info), flags, gptr | lowBits) {
        println("......g #", g.goid, ":  acquired lock")
        g.releaseBucket = unsafe.Pointer(&arr.data[idx])
        // g.releaseM = acquirem()
        if spins > 0 {
            println(".........g #", g.goid, ": wasted ", spins, " CPU cycles spinning")
        }
        return &(arr.data[idx])
    }
*/

/*
    findBucket will find the requested bucket by it's key and return the associated
    bucketHdr, as well as the hash used to obtain the bucket, allowing the caller to make further changes safely. 
    It is also up to the caller to relinquish the lock and/or update flags when they are finished.
*/
func (arr *bucketArray) findBucket(t *maptype, key unsafe.Pointer) (*bucketHdr, uintptr, uintptr) {
    // Hash the key using bucketArray-specific seed to obtain index of header
    hash := t.key.alg.hash(key, uintptr(arr.seed))
    idx := hash % uintptr(arr.size)
    hdr := &(arr.data[idx])
    g := getg().m.curg
    spins := 0

    println("...g #", g.goid, ": Obtained bucketHdr #", idx, " with seed: ", uintptr(arr.seed), " and hash: ", uintptr(hash))

    // TODO: Use the holder 'g' to keep track of whether or not the holder is still running or not to determine if we should yield.
    for {
        // A simple CAS can be used to attempt to acquire the lock. If it is an ARRAY, this will always fail, as the flag will never be 0 again.
        if atomic.Casuintptr(&hdr.info, 0, LOCKED) {
            println("......g #", g.goid, ":  acquired lock")
            g.releaseBucket = unsafe.Pointer(&arr.data[idx])
            // g.releaseM = acquirem()
            if spins > 0 {
                println(".........g #", g.goid, ": wasted ", spins, " CPU cycles spinning")
            }
            return &(arr.data[idx]), hash, arr.size
        }

        /*
            If we fail the above, then one of two things happens and have certain implications...

            1) Another goroutine currently has the lock. In this case, we simply back off until it no longer
               has that lock. Note that we must explicitly check if it is LOCKED again, as it is possible for it
               to be locked and have the ARRAY flag set, as the thread is currently converting it.

            2) The bucket is already an ARRAY and we only need to recurse through it to find what we need. Note now that
               this invariant means that no other 'g' may acquire this lock, as they will all fail. Hence, by checking if
               it is LOCKED before checking if it is an ARRAY and recursing we can prevent race conditions. Note that only
               the array that converted the bucket to an array and set the ARRAY flag may hold the lock.
        */

        flags := atomic.Loaduintptr(&hdr.info)

        // This can never be locked again after converted to array, so completely safe.
        if (flags & LOCKED) != 0 {
            spins++
            Gosched()
            continue
        }

        // At this point, we know that it isn't LOCKED, and if it is ARRAY, then it will remain that way.
        if (flags & ARRAY) != 0 {
            // Sanity check for if a thread locked it after we loaded flags
            if !atomic.Casuintptr(&hdr.info, flags, flags) {
                spins++
                Gosched()
                continue
            }

            if spins > 0 {
                println("...g #", g.goid, ": Recursing through nested array after spinning ", spins, "waiting for allocation")
            }
            return (*bucketArray)(hdr.bucket).findBucket(t, key)
        }

        spins++
        Gosched()
    }
}

type keyType struct {
    x, y int
} 

func toKeyType(key unsafe.Pointer) keyType {
    return *(*keyType)(key)
} 

func compare(k1, k2 unsafe.Pointer) {
    println("......g #", getg().goid, ": (", toKeyType(k1).x, ", ", toKeyType(k1).y, ") vs. (", toKeyType(k2).x, ", ", toKeyType(k2).y, ")")
}

func printKey(k unsafe.Pointer) {
    println("......g #", getg().goid, ": (", toKeyType(k).x, ", ", toKeyType(k).y, ")")
}

func (b *bucketChain) print(t *maptype) {
    k := toKeyType(b.key(t))
    v := toKeyType(b.value(t))
    println("......g #", getg().goid, ": Key{", k.x, ",", k.y, "};Value{", v.x, ",", v.y, "}")
}

/*
    bucketHdr's findBucket is used to walk the list of bucketChain's pointed to by bucketHdr's 'bucket' field.
    Note that this assumes explicitly that you already hold the lock, and that the bucketHdr is chained.

    This function is customizable to not violate the D.R.Y rule. The 'key' is passed to compare with the current
    bucketChain's key, using the 'equal' callback function (needed due to how Fast Types are checked different than
    normal types).
*/
func (hdr *bucketHdr) findBucket(t *maptype, key unsafe.Pointer, opt bucketSearch, equal func (k1, k2 unsafe.Pointer) bool) (*bucketChain, bucketStatus)  {
    g := getg().m.curg
    b := (*bucketChain)(hdr.bucket)
    if b == nil {
        return nil, BUCKET_STATUS_EMPTY
    }
    println("...g #", g.goid, ": Searching from root bucketChain", b);
    for i := 0; i < MAXCHAIN; i++ {
        // Explicit case: If we are trying to find the first empty, we can skip the rest of this
        if opt == BUCKET_SEARCH_FIRST_EMPTY {
            if b.flags == UNUSED {
                return b, BUCKET_STATUS_FIRST_EMPTY
            }
            b = b.next
            continue
        }

        // Explicit case: If we are trying to find the last bucket, but t is nil, then we are not doing key evaluations, just keep going
        if opt == BUCKET_SEARCH_LAST_FOUND && t == nil {
            if b.next == nil {
                return b, BUCKET_SEARCH_LAST_FOUND
            }
            b = b.next
            continue
        }

        // Note that this can only be true if BUCKET_SEARCH_LAST_FOUND was not specified, as the bucket is guaranteed to not be empty at this point, nor at it's peak capacity.
        if b == nil {
            return nil, BUCKET_STATUS_NOT_FOUND
        }
        println("......g #", g.goid, ": Searching bucket #", i, "with address", b, "pointing to", b.next)
        // If the bucketChain is not in use
        if b.flags == UNUSED {
            b = b.next
            continue
        }

        println(".........g #", g.goid, ": had b.flags != UNUSED")

        // Whether we took the key by value or by reference, we resolve that here.
        otherKey := b.key(t)
        if t.indirectkey {
            otherKey = *(*unsafe.Pointer)(otherKey)
        }

        compare(key, otherKey)
        b.print(t)

        if equal(key, otherKey) {
            println("......g #", g.goid, ": Keys match")
            return b, BUCKET_STATUS_FOUND
        }

        // If this is not the last possible bucketChain, and the caller wants to return the last bucket found, do so.
        if (i + 1) < MAXCHAIN && b.next == nil && (opt & BUCKET_SEARCH_LAST_FOUND) != 0 {
            return b, BUCKET_STATUS_LAST_FOUND
        }

        b = b.next
    }

    return nil, BUCKET_STATUS_FULL
}

func newBucketChain(t *maptype, key unsafe.Pointer, value unsafe.Pointer) (chain *bucketChain) {
    chain = (*bucketChain)(newobject(t.bucketchain))
    k := chain.key(t)
    v := chain.value(t)

    // If either key or value is indirect, we must allocate a new object and copy into it.
    if t.indirectkey {
        println(".........g #", getg().goid, ": Key is indirect")
        kmem := newobject(t.key)
        *(*unsafe.Pointer)(k) = kmem
        k = kmem
    }
    if t.indirectvalue {
        println(".........g #", getg().goid, ": Value is indirect")
        vmem := newobject(t.elem)
        *(*unsafe.Pointer)(v) = vmem
        v = vmem
    }

    // Copies memory in a way that it updates the GC to know objects pointed to by this copy should not be collected
    typedmemmove(t.key, k, key)
    typedmemmove(t.elem, v, value)

    chain.flags = USED
    return
}

func (bucket *bucketChain) update(t *maptype, key unsafe.Pointer, value unsafe.Pointer) {
    v := bucket.value(t)

    // Indirect Key and Values need some indirection
    if t.indirectvalue {
        v = *(*unsafe.Pointer)(v)
    }

    // If we are required to update key, do so
    if t.needkeyupdate {
        k := bucket.key(t)
        if t.indirectkey {
            k = *(*unsafe.Pointer)(k)
        }
        typedmemmove(t.key, k, key)
    }

    typedmemmove(t.elem, v, value)
}


func (arr *bucketArray) init(t *maptype, size uint32) {
    g := getg().m.curg
    println("......g #", g.goid, ": Initializing bucketArray with size ", size)
    arr.data = make([]bucketHdr, size)
    arr.size = size
    arr.seed = fastrand1()
}

// Assumes hdr holds a bucketChain and is not currently locked
func (hdr *bucketHdr) add(t *maptype, h *hmap, key unsafe.Pointer, value unsafe.Pointer, size uint32, equal func (k1, k2 unsafe.Pointer) bool) {
    g := getg().m.curg
    println("...g #", g.goid, ": Adding key-value pair to bucketHdr")
    bucket, status := hdr.findBucket(t, key, BUCKET_SEARCH_LAST_FOUND, equal)

    switch status {
        // If it is empty, we create our own root bucketChain
        case BUCKET_STATUS_EMPTY:
            println("......g #", g.goid, ": Bucket is empty, attaching a new root bucket")
            hdr.bucket = unsafe.Pointer(newBucketChain(t, key, value))
            h.count++
            (*bucketChain)(hdr.bucket).print(t)
        // If the key was found in the map, we update the value and key if needed
        case BUCKET_STATUS_FOUND:
            println("......g #", g.goid, ": Key is present in map, updating bucket")
            bucket.update(t, key, value)
            bucket.print(t)
        // If the bucketChain has no spare buckets to recycle, append to the the bucket.
        case BUCKET_STATUS_LAST_FOUND:
            println("......g #", g.goid, ": Key is not present in the map, but there is room for more, allocating")
            bucket.next = newBucketChain(t, key, value)
            h.count++
            bucket.next.print(t)
        // If the bucketChain is full, we must explicitly convert it to another bucket and recursively evaluate pass control.
        case BUCKET_STATUS_FULL:
            println("......g #", g.goid, ": A bucket is full, resizing")
            hdr.chainToArray(t, key, size * 2, equal)
            array := (*bucketArray)(hdr.bucket)
            hash :=  t.key.alg.hash(key, uintptr(array.seed))
            idx := hash % uintptr(array.size)
            newHdr := &(array.data[idx])
            newHdr.add(t, h, key, value, array.size, equal)
            println(".........g #", g.goid, ": Moved a key with hash", uintptr(hash), "to idx", idx)
        default:
            println("Bad status: ", status)
            throw("Bad status inside of add(...)")
    }
}

// Claims ownership of the bucketChain, used during chainToArray conversion
func (hdr *bucketHdr) claim(t *maptype, claimBucket *bucketChain) {
    // For findBucket, if BUCKET_SEARCH_FIRST_EMPTY is specified, it will not use the rest of these parameters, hence left nil
    bucket, status := hdr.findBucket(nil, nil, BUCKET_SEARCH_LAST_FOUND, nil)

    switch status {
        // If it is empty, we simply make it our first.
        case BUCKET_STATUS_EMPTY:
            hdr.bucket = unsafe.Pointer(claimBucket)
            println(".........g #", getg().goid, ": Bucket is new root...")
        // If instead we find the first empty, we append to that.
        case BUCKET_STATUS_LAST_FOUND:
            bucket.next = claimBucket
            println(".........g #", getg().goid, ": Bucket is chained...")
    }

    claimBucket.print(t)
}

func (hdr *bucketHdr) chainToArray(t *maptype, key unsafe.Pointer, size uint32, equal func(k1, k2 unsafe.Pointer) bool) {
    g := getg().m.curg
    println("...g #", g.goid, ": Converting bucketChain to bucketArray")
    array := (*bucketArray)(newobject(t.bucketarray))
    array.init(t, size)

    // Move each bucketChain to their respective cell in the bucketArray
    for b := (*bucketChain)(hdr.bucket); b != nil; {
        hash := t.key.alg.hash(b.key(t), uintptr(array.seed))
        idx := hash % uintptr(array.size)
        next := b.next
        b.next = nil
        println("......g #", g.goid, ": Moved bucket ", b, " to index #", idx, "with hash: ", uintptr(hash))
        newHdr := &array.data[idx]
        newHdr.claim(t, b)
        b = next
    }

    hdr.bucket = unsafe.Pointer(array)
    
    // Atomically 'OR' ARRAY in place without releasing the spinlock
    for {
        oldInfo := atomic.Loaduintptr(&hdr.info)
        if atomic.Casuintptr(&hdr.info, oldInfo, oldInfo | ARRAY) {
            break;
        }
    }
}

func (hdr *bucketHdr) dataToArray(t *maptype, size uintptr, equal func (k1, k2 unsafe.Pointer) bool) {
    g := getg().m.curg
    println("...g #", g.goid, ": Converting bucketData to bucketArray")
    array := (*bucketArray)(newobject(t.bucketarray))
    array.init(t, size)
    data := (*bucketData)(hdr.bucket)

    // Move each bucketData to their respective cell in the bucketArray
    for i := 0; i < MAXCHAIN; i++ {
        key := data.key(t, i)
        val := data.value(t, i)

        // Rehash the key with the new seed
        hash := t.key.alg.hash(key, uintptr(array.seed))
        idx := hash % uintptr(array.size)
        arrHdr := arr.data[idx]
        arrData := (*bucketData)(arrHdr.bucket)

        // This is the first insertion into this bucket, allocate
        if arrData == nil {
            arrHdr.bucket = (*bucketData)(newobject(t.BucketData))
            arrData = (*bucketData)(arrHdr.bucket)

            // Directly insert into map
            arrData.assign(t, 0, hash, key, value)
            continue
        }

        // Otherwise, we know this is not the first, so we need to scan for the first empty (after the first)
        for j := 1; j < MAXCHAIN; j++ {
            if arrData.hash[j] == 0 {
                arrData.assign(t, j, hash, key, value)
                break
            }
        }
    }
}

func makecmap(t *maptype, hint int64, h *hmap, bucket unsafe.Pointer) *hmap {
    println("Inside of makecmap: Key: ", t.key.string(), "; Value: ", t.elem.string())
    println("Sizeof bucketHdr: ", unsafe.Sizeof(bucketHdr{}), "\nSizeof bucketArray: ", unsafe.Sizeof(bucketArray{}))
    println("Sizeof bucketChain: ", t.bucketchain.size, "\nSizeof cdataOffset: ", cdataOffset, "\nSizeof bmap", t.bucket.size, "\nName of bmap: ", t.bucket.string())
    println("Name of bucketChain: ", t.bucketchain.string(), "\nName of bucketArray: ", t.bucketarray.string(), "\nName of bucketHdr: ", t.buckethdr.string())
    // Initialize the hashmap if needed
    if h == nil {
		h = (*hmap)(newobject(t.hmap))
	}
    
    // Initialize and allocate our concurrentMap
    cmap := (*concurrentMap)(newobject(t.concurrentmap))
    cmap.root.init(t, MAXBUCKETS)

    h.chdr = unsafe.Pointer(cmap)
    h.flags = 8
    
    return h
}

func (data *bucketData) assign(t *mapkey, idx, hash uintptr, key, value unsafe.Pointer) {
    k := data.key(idx)
    v := data.value(idx)

    if t.indirectkey {
        println(".........g #", getg().goid, ": Key is indirect")
        kmem := newobject(t.key)
        *(*unsafe.Pointer)(k) = kmem
        k = kmem
    }
    if t.indirectvalue {
        println(".........g #", getg().goid, ": Value is indirect")
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
func (data *bucketData) update(t *mapkey, idx uintptr, key, value unsafe.Pointer) {
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

func cmapassign1(t *maptype, h *hmap, key unsafe.Pointer, val unsafe.Pointer) {
    g := getg().m.curg
    println("g #", g.goid, ": cmapassign1")
    cmap := (*concurrentMap)(h.chdr)
    hdr, hash, sz := cmap.root.findBucket(t, key)
    
    // If bucket is nil, then we allocate a new one.
    if hdr.bucket == nil {
        hdr.bucket = unsafe.Pointer(newobject(t.BucketData))
        
        // Since we just created a new bucket, directly add the key and value to the map
        (*bucketData)(hdr.bucket).assign(t, hash, 0, key, val)
        atomic.Xadd(&h.count, 1)
        return
    }

    data := (*bucketData)(hdr.bucket)
    firstEmpty := -1
    // Otherwise, we must scan all hashes to find a matching hash; if they match, check if they are equal
    for i := 0; i < MAXCHAIN {
        currHash := data.hash[i]
        // The hash is 0 if it is not in use
        if currHash == 0 {
            // Keep track of the first empty so we know what to assign into if we do not find a match
            if firstEmpty == -1 {
                firstEmpty = i
            }
            continue
        }

        // If the hash matches, check to see if keys are equal
        if hash == data.hash[i] {
            otherKey := data.key(t, i)

            // If they are equal, update...
            if t.alg.key.equal(key, otherKey) {
                data.update(t, i, key, value)
                return
            }
        }
    }

    // If firstEmpty is still -1, that means we did not find any empty slots, and should convert immediate
    if firstEmpty == -1 {
        // When we convert dataToArray, it's size is double the previous
        hdr.dataToArray(t, sz * 2, t.key.alg.equal)

        // After converting it from to a RECURSIVE array, we can release the lock to allow further concurrency.
        maprelease()
        
        // Also we can save effort by just recursively calling this again.
        cmapassign1(t, h, key, val)
    }

    // At this point, firstEmpty is guaranteed to be non-zero, hence we can safely assign it
    data.assign(t, firstEmpty, hash, key, value)
}

func maprelease() {
    g := getg()
    if g.releaseBucket != nil {
        println("g #", g.goid, ": released lock")
        hdr := (*bucketHdr)(g.releaseBucket)
        
        // Atomically release lock
        for {
            oldFlags := atomic.Loaduintptr(&hdr.info)
            if atomic.Casuintptr(&hdr.info, oldFlags, uintptr(oldFlags &^ LOCKED)) {
                break
            }
        }

        g.releaseBucket = nil
        // releasem(g.releaseM)
        // g.releaseM = nil
    }
}

/*
    Concurrent hashmap_fast.go function implementations
*/
func cmapaccess1_fast32(t *maptype, h *hmap, key uint32) unsafe.Pointer {
    g := getg().m.curg
    println("g #", g.goid, ": cmapaccess1_fast32!")
    retval, _ := cmapaccess2_fast32(t, h, key)
    return retval
}

func cmapaccess2_fast32(t *maptype, h *hmap, key uint32) (unsafe.Pointer, bool) {
    g := getg().m.curg
    println("g #", g.goid, ": cmapaccess2_fast32!")
    return cmapaccess(t, h, noescape(unsafe.Pointer(&key)), 
        func (k1, k2 unsafe.Pointer) bool { 
            return *(*uint32)(k1) == *(*uint32)(k2) 
        })
}


func cmapaccess1_fast64(t *maptype, h *hmap, key uint64) unsafe.Pointer {
    g := getg().m.curg
    println("g #", g.goid, ": cmapaccess1_fast64")
    retval, _ := cmapaccess2_fast64(t, h, key)
    return retval
}

func cmapaccess2_fast64(t *maptype, h *hmap, key uint64) (unsafe.Pointer, bool) {
    g := getg().m.curg
    println("g #", g.goid, ": cmapaccess2_fast64!")
    return cmapaccess(t, h, noescape(unsafe.Pointer(&key)), 
        func (k1, k2 unsafe.Pointer) bool { 
            return *(*uint64)(k1) == *(*uint64)(k2) 
        })
}

func cmapaccess1_faststr(t *maptype, h *hmap, key string) unsafe.Pointer {
    g := getg().m.curg
    println("g #", g.goid, ": cmapaccess1_faststr")
    retval, _ := cmapaccess2_faststr(t, h, key)
    return retval
}

func cmapaccess2_faststr(t *maptype, h *hmap, key string) (unsafe.Pointer, bool) {
    g := getg().m.curg
    println("g #", g.goid, ": cmapaccess2_faststr")
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
    cmap := (*concurrentMap)(h.chdr)
    hdr, hash, _ := cmap.root.findBucket(t, key)

    // If the bucket is nil, then it is empty, stop here
    if hdr.bucket == nil {
        return unsafe.Pointer(&DUMMY_RETVAL[0]), false
    }

    data := (*bucketData)(hdr.bucket)
    for i := 0; i < MAXCHAIN; i++ {
        currHash := data.hash[i]
        
        // Check if the hashes are equal
        if currHash == hash {
            otherKey := data.key(t, i)

            // If the keys are equal
            if equal(key, otherKey) {
                return data.value(t, i), true
            }
        }
    }

    // Only get to this point if we have not found the value in the map
    return unsafe.Pointer(&DUMMY_RETVAL[0]), false
}


func cmapaccess2(t *maptype, h *hmap, key unsafe.Pointer) (unsafe.Pointer, bool) {
    g := getg().m.curg
    println("g #", g.goid, ": cmapaccess2!")

    return cmapaccess(t, h, key, t.key.alg.equal)
}

func cmapaccess1(t *maptype, h *hmap, key unsafe.Pointer) unsafe.Pointer {
    g := getg().m.curg
    println("g #", g.goid, ": cmapaccess1")
    retval, _ := cmapaccess2(t, h, key)

    // Only difference is that we discard the boolean
    return retval
}