
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
    MAXBUCKETS = 32
    // The maximum amount of buckets in a bucketChain
    MAXCHAIN = 16

    CHAINED = 0
    // bucketHdr's 'b' is a bucketArray. Note that once this is true, it always remains so.
    ARRAY = 1 << 0

    // bucketChain is unused
    UNUSED = 0
    // bucketChain is in use
    USED = 1 << 0

    // Debug flag
    DEBUG = true

    // See hashmap.go, this obtains a properly aligned offset to the data
    cdataOffset = unsafe.Offsetof(struct {
		b bucketChain
		v int64
	}{}.v)
)

var DUMMY_RETVAL [MAXZERO]byte

type bucketHdr struct {
    // Is either a bucketArray or bucketChain
    bucket unsafe.Pointer
    /*
        The least significant 3 bits are used as flags and the rest
        are a pointer to the current 'g' that holds the lock. The 'g' is
        used to allow it to become reentrant.

        Checking the lock can be as easy as masking out the first three bits.
        An overall CAS can also make it easier to tell when it is no longer locked,
        as as soon as the 'g' holding the lock finishes, it will zero the portion
        used to store the address. This can cause the CAS to fail during a spin
        and be used to detect if manages to acquire the lock.
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
    findBucket will find the requested bucket by it's key and return the associated
    bucketHdr, allowing the caller to make further changes safely. It is also up to the
    caller to relinquish the lock or update flags when they are finished.
*/
func (arr *bucketArray) findBucket(t *maptype, key unsafe.Pointer) *bucketHdr {
    // Hash the key using bucketArray-specific seed to obtain index of header
    hash := t.key.alg.hash(key, uintptr(arr.seed))
    idx := hash % uintptr(arr.size)
    hdr := &(arr.data[idx])
    g := getg().m.curg
    gptr := uintptr(unsafe.Pointer(getg().m.curg))
    spins := 0

    // println("...g #", g.goid, ": Obtained bucketHdr #", idx, " with seed: ", arr.seed, " and hash: ", hash)

    // TODO: Use the holder 'g' to keep track of whether or not the holder is still running or not to determine if we should yield.
    for {
        // Simple atomic load for flags.
        flags := atomic.Loaduintptr(&hdr.info)

        // Check if we are the current holders of the lock, spin if someone else holds the lock
        lockHolder := uintptr(flags &^ 0x7)

        // println(".........g #", g.goid, ": gptr: ", gptr, ";lockHolder: ", lockHolder, ";flags: ", flags)

        if lockHolder == gptr {
            // println("......g #", g.goid, ": re-entered the lock")
            return &(arr.data[idx])
        } else if lockHolder != 0 {
            spins++
            Gosched()
            continue
        }

        // Obtain the lower three marked bits to be OR'd back into flags if we succeed the CAS
        lowBits := uintptr(flags & 0x7)

        // If it is a BucketArray, then this means we must traverse that bucket to find the right chain.
        if (lowBits & ARRAY) != 0 {
            // println("...g #", g.goid, ": Recursing through nested array after spinning ", spins, "waiting for allocation")
            return (*bucketArray)(hdr.bucket).findBucket(t, key)
        }

        // Attempt to acquire the lock ourselves atomically
        if atomic.Casuintptr(&(arr.data[idx].info), flags, gptr | lowBits) {
            // println("......g #", g.goid, ":  acquired lock")
            g.releaseBucket = unsafe.Pointer(&arr.data[idx])
            // g.releaseM = acquirem()
            if spins > 0 {
                // println(".........g #", g.goid, ": wasted ", spins, " CPU cycles spinning")
            }
            return &(arr.data[idx])
        }

        spins++
        Gosched()
    }
}

func (hdr *bucketHdr) findBucket(t *maptype, key unsafe.Pointer, equal func (k1, k2 unsafe.Pointer) bool) (*bucketChain, bool) {
    var firstEmpty *bucketChain
    // g := getg().m.curg
    b := (*bucketChain)(hdr.bucket)
    // println("...g #", g.goid, ": Searching from root bucketChain", b);
    for i := 0; i < MAXCHAIN; i++ {
        // println("......g #", g.goid, ": Searching bucket #", i, "with address", b, "pointing to", b.next)
        // If the bucketChain is not in use
        if b.flags == UNUSED {
            // We will be directly returning the first empty bucket if not found.
            if firstEmpty == nil {
                firstEmpty = b
            }
            b = b.next
            continue
        }

        // println(".........g #", g.goid, ": had b.flags != UNUSED")

        // Whether we took the key by value or by reference, we resolve that here.
        otherKey := b.key(t)
        if t.indirectkey {
            otherKey = *(*unsafe.Pointer)(otherKey)
        }

        if equal(key, otherKey) {
            // println("......g #", g.goid, ": Keys match")
            return b, true
        }

        b = b.next
    }

    return firstEmpty, false
}

func (arr *bucketArray) init(t *maptype, size uint32) {
    // g := getg().m.curg
    // println("......g #", g.goid, ": Initializing bucketArray with size ", size)
    arr.data = make([]bucketHdr, size)
    arr.size = size
    // First thing we need to do is allocate all bucketChain's in each bucketHdr
    for i := 0; uint32(i) < size; i++ {
        chain := (*bucketChain)(mallocgc(cdataOffset + uintptr(t.keysize) + uintptr(t.valuesize), t.bucketchain, true))
        // println(".........g #", g.goid, ": Allocated root bucketChain #", i + 1, " with Address ", chain)
        tmpChain := chain
        hdr := &arr.data[i]

        // Populate up to MAXCHAIN bucketChain's. Avoids having to create call this repeatedly in the future.
        for j := 1; j < MAXCHAIN; j++ {
            tmpChain.next = (*bucketChain)(mallocgc(cdataOffset + uintptr(t.keysize) + uintptr(t.valuesize), t.bucketchain, true))
            tmpChain.flags = UNUSED
            // println("............g #", g.goid, ": Allocated nested bucketChain #", j, " with Address ", tmpChain, " pointing to ", tmpChain.next)
            tmpChain = tmpChain.next

        }

        hdr.bucket = unsafe.Pointer(chain)
        hdr.info = 0
    }

    arr.seed = fastrand1()
}

// Assumes hdr holds a bucketChain and is not currently locked
func (hdr *bucketHdr) add(t *maptype, key unsafe.Pointer, value unsafe.Pointer, size uint32, equal func (k1, k2 unsafe.Pointer) bool) {
    // g := getg().m.curg
    // println("...g #", g.goid, ": Adding key-value pair to bucketHdr")
    b, pres := hdr.findBucket(t, key, equal)

    // If the bucket is not present in the map, we must create a new one.
    if !pres {
        // If a bucket was not returned, then the bucket is actually full.
        if b == nil {
            // println("......g #", g.goid, ": A bucket is full, resizing")
            // Unchain the bucket pointed to by hdr. 'bucket' now points to a bucketArray
            hdr.chainToArray(t, key, size * 2, equal)
            array := (*bucketArray)(hdr.bucket)

            // Rehash the key and recursively add it to the new bucketArray
            idx := t.key.alg.hash(key, uintptr(array.seed)) % uintptr(array.size)
            array.data[idx].add(t, key, value, array.size, equal)
        } else {
            // println("......g #", g.goid, ": Key is not present in the map, but a bucket is empty, reusing")
            // if b != nil, then it returns the last bucket not in-use so we can recycle
            // These are actually the memory locations where the key and value reside
            k := b.key(t)
            v := b.value(t)

            // If either key or value is indirect, we must allocate a new object and copy into it.
            if t.indirectkey {
                // println(".........g #", g.goid, ": Key is indirect")
                kmem := newobject(t.key)
                *(*unsafe.Pointer)(k) = kmem
                k = kmem
            }
            if t.indirectvalue {
                // println(".........g #", g.goid, ": Value is indirect")
                vmem := newobject(t.elem)
                *(*unsafe.Pointer)(v) = vmem
                v = vmem
            }

            // Copies memory in a way that it updates the GC to know objects pointed to by this copy should not be collected
            typedmemmove(t.key, k, key)
            typedmemmove(t.elem, v, value)

            b.flags = USED
        }
    } else {
        // println("......g #", g.goid, ": Key is present in map, updating bucket")
        // If we are in this block, then the bucket exists and we just update it.
        k := b.key(t)
        v := b.value(t)

        // Indirect Key and Values need some indirection
        if t.indirectkey {
            k = *(*unsafe.Pointer)(k)
        }
        if t.indirectvalue {
            v = *(*unsafe.Pointer)(v)
        }

        // If we are required to update key, do so
        if t.needkeyupdate {
            typedmemmove(t.key, k, key)
        }

        typedmemmove(t.elem, v, value)
    }
}

func (hdr *bucketHdr) chainToArray(t *maptype, key unsafe.Pointer, size uint32, equal func(k1, k2 unsafe.Pointer) bool) {
    // g := getg().m.curg
    // println("...g #", g.goid, ": Converting bucketChain to bucketArray")
    array := (*bucketArray)(newobject(t.bucketarray))
    array.init(t, size)
    for b := (*bucketChain)(hdr.bucket); b != nil; b = b.next {
        idx := t.key.alg.hash(b.key(t), uintptr(array.seed)) % uintptr(array.size)
        array.data[idx].add(t, b.key(t), b.value(t), size, equal)
        
        // println("......g #", g.goid, ": Moved bucket ", b, " to index #", idx)

        // Now we must call memclr to let GC know the old copies no longer contains a reference to data.
        // memclr works by sweeping from [offset, offset + n) 
        memclr(b.key(t), uintptr(t.keysize) + uintptr(t.valuesize))
        b.flags = UNUSED
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

func makecmap(t *maptype, hint int64, h *hmap, bucket unsafe.Pointer) *hmap {
    // println("Inside of makecmap: Key: ", t.key.string(), "; Value: ", t.elem.string())
    // println("Sizeof bucketHdr: ", unsafe.Sizeof(bucketHdr{}), "\nSizeof bucketArray: ", unsafe.Sizeof(bucketArray{}))
    // println("Sizeof bucketChain: ", t.bucketchain.size, "\nSizeof cdataOffset: ", cdataOffset, "\nSizeof bmap", t.bucket.size)
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

func cmapassign1(t *maptype, h *hmap, key unsafe.Pointer, val unsafe.Pointer) {
    // g := getg().m.curg
    // println("g #", g.goid, ": cmapassign1")
    cmap := (*concurrentMap)(h.chdr)
    hdr := cmap.root.findBucket(t, key)
    hdr.add(t, key, val, cmap.root.size, t.key.alg.equal)
    h.count++
}

func maprelease() {
    g := getg()
    if g.releaseBucket != nil {
        // println("g #", g.goid, ": released lock")
        hdr := (*bucketHdr)(g.releaseBucket)
        
        // Atomically release lock
        for {
            oldFlags := atomic.Loaduintptr(&hdr.info)
            if atomic.Casuintptr(&hdr.info, oldFlags, uintptr(oldFlags & 0x7)) {
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
    // g := getg().m.curg
    // println("g #", g.goid, ": cmapaccess1_fast32!")
    retval, _ := cmapaccess2_fast32(t, h, key)
    return retval
}

func cmapaccess2_fast32(t *maptype, h *hmap, key uint32) (unsafe.Pointer, bool) {
    // g := getg().m.curg
    // println("g #", g.goid, ": cmapaccess2_fast32!")
    keyPtr := noescape(unsafe.Pointer(&key))
    // TODO: Set chdr to be of type concurrentMap, then no need to cast
    cmap := (*concurrentMap)(h.chdr)
    hdr := cmap.root.findBucket(t, keyPtr)
    b, pres := hdr.findBucket(t, keyPtr, 
        func (k1, k2 unsafe.Pointer) bool { 
            return *(*uint32)(k1) == *(*uint32)(k2) 
        })
    var retval unsafe.Pointer

    // If the bucket is present in the map...
    if pres {
        retval = b.value(t)
        if t.indirectvalue {
            retval = *(*unsafe.Pointer)(retval)
        }
    } else {
        retval = unsafe.Pointer(&DUMMY_RETVAL[0])
    }

    // hdr.unlock()

    return retval, pres
}


func cmapaccess1_fast64(t *maptype, h *hmap, key uint64) unsafe.Pointer {
    // g := getg().m.curg
    // println("g #", g.goid, ": cmapaccess1_fast64")
    retval, _ := cmapaccess2_fast64(t, h, key)
    return retval
}

func cmapaccess2_fast64(t *maptype, h *hmap, key uint64) (unsafe.Pointer, bool) {
    // g := getg().m.curg
    // println("g #", g.goid, ": cmapaccess2_fast64!")
    keyPtr := noescape(unsafe.Pointer(&key))
    // TODO: Set chdr to be of type concurrentMap, then no need to cast
    cmap := (*concurrentMap)(h.chdr)
    hdr := cmap.root.findBucket(t, keyPtr)
    b, pres := hdr.findBucket(t, keyPtr, 
        func (k1, k2 unsafe.Pointer) bool { 
            return *(*uint64)(k1) == *(*uint64)(k2) 
        })
    var retval unsafe.Pointer

    // If the bucket is present in the map...
    if pres {
        retval = b.value(t)
        if t.indirectvalue {
            retval = *(*unsafe.Pointer)(retval)
        }
    } else {
        retval = unsafe.Pointer(&DUMMY_RETVAL[0])
    }

    // hdr.unlock()

    return retval, pres
}

func cmapaccess1_faststr(t *maptype, h *hmap, key string) unsafe.Pointer {
    // g := getg().m.curg
    // println("g #", g.goid, ": cmapaccess1_faststr")
    retval, _ := cmapaccess2_faststr(t, h, key)
    return retval
}

func cmapaccess2_faststr(t *maptype, h *hmap, key string) (unsafe.Pointer, bool) {
    // g := getg().m.curg
    // println("g #", g.goid, ": cmapaccess2_faststr")
    keyPtr := noescape(unsafe.Pointer(&key))
    // TODO: Set chdr to be of type concurrentMap, then no need to cast
    cmap := (*concurrentMap)(h.chdr)
    hdr := cmap.root.findBucket(t, keyPtr)
    b, pres := hdr.findBucket(t, keyPtr, 
        func (k1, k2 unsafe.Pointer) bool { 
            sk1 := (*stringStruct)(k1)
            sk2 := (*stringStruct)(k2)
            return sk1.len == sk2.len && 
            (sk1.str == sk2.str || memequal(sk1.str, sk2.str, uintptr(sk1.len))) 
        })
    var retval unsafe.Pointer

    // If the bucket is present in the map...
    if pres {
        retval = b.value(t)
        if t.indirectvalue {
            retval = *(*unsafe.Pointer)(retval)
        }
    } else {
        retval = unsafe.Pointer(&DUMMY_RETVAL[0])
    }

    // hdr.unlock()

    return retval, pres
}


func cmapaccess2(t *maptype, h *hmap, key unsafe.Pointer) (unsafe.Pointer, bool) {
    // g := getg().m.curg
    // println("g #", g.goid, ": cmapaccess2!")
    // TODO: Set chdr to be of type concurrentMap, then no need to cast
    cmap := (*concurrentMap)(h.chdr)
    hdr := cmap.root.findBucket(t, key)
    b, pres := hdr.findBucket(t, key, t.key.alg.equal)
    var retval unsafe.Pointer

    // If the bucket is present in the map...
    if pres {
        retval = b.value(t)
        if t.indirectvalue {
            retval = *(*unsafe.Pointer)(retval)
        }
    } else {
        retval = unsafe.Pointer(&DUMMY_RETVAL[0])
    }

    // hdr.unlock()

    return retval, pres
}

func cmapaccess1(t *maptype, h *hmap, key unsafe.Pointer) unsafe.Pointer {
    // g := getg().m.curg
    // println("g #", g.goid, ": cmapaccess1")
    retval, _ := cmapaccess2(t, h, key)

    // Only difference is that we discard the boolean
    return retval
}