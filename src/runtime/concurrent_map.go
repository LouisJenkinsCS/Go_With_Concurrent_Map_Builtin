
package runtime

import (
    "runtime/internal/atomic"
    "unsafe"
)

const (
    MAXZERO = 1024 // must match value in ../cmd/compile/internal/gc/walk.go
    
    // The maximum amount of buckets in a bucketArray
    MAXBUCKETS = 16
    // The maximum amount of buckets in a bucketChain
    MAXCHAIN = 8

    CHAINED = 0
    // bucketHdr's 'b' is a bucketArray. Note that once this is true, it always remains so.
    ARRAY = 1 << 0

    // bucketChain is unused
    UNUSED = 0
    // bucketChain is in use
    USED = 1 << 0
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
    data [MAXBUCKETS]bucketHdr
    // Seed is different for each bucketArray to ensure that the re-hashing resolves to different indice
    seed uint32
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
    keyOffset := uintptr(rawChain) + unsafe.Sizeof(chain)
    return unsafe.Pointer(keyOffset)
}

func (chain *bucketChain) value(t *maptype) unsafe.Pointer {
    // Cast chain to an unsafe.Pointer to bypass Go's type system
    rawChain := unsafe.Pointer(chain)
    // The key offset is right at the end of the bucketChain
    keyOffset := uintptr(rawChain) + unsafe.Sizeof(chain)
    // The value offset is right after the end of the key (The size of the key is kept in maptype)
    valOffset := keyOffset + uintptr(t.keysize)

    return unsafe.Pointer(valOffset)
}

/*
    findBucket will find the requested bucket by it's key and return the associated
    bucketHdr, allowing the caller to make further changes safely. It is also up to the
    caller to relinquish the lock or update flags when they are finished.
*/
func (b *bucketArray) findBucket(t *maptype, key unsafe.Pointer) *bucketHdr {
    // Hash the key using bucketArray-specific seed to obtain index of header
    idx := t.key.alg.hash(key, uintptr(b.seed)) % MAXBUCKETS
    hdr := b.data[idx]

    // TODO: Use the holder 'g' to keep track of whether or not the holder is still running or not to determine if we should yield.
    for {
        // Simple atomic load for flags.
        flags := atomic.Loaduintptr(&hdr.info)

        // Obtain the lower three marked bits to be OR'd back into flags if we succeed the CAS
        lowBits := uintptr(flags & 0x7)

        // If it is a BucketArray, then this means we must traverse that bucket to find the right chain.
        if (lowBits & ARRAY) != 0 {
            return (*bucketArray)(hdr.bucket).findBucket(t, key)
        }

        // Check if we are the current holders of the lock, spin if someone else holds the lock
        gptr := uintptr(unsafe.Pointer(getg()))
        lockHolder := uintptr(flags &^ 0x7)
        if lockHolder == gptr {
            return &b.data[idx]
        } else if lockHolder != 0 {
            continue
        }
        
        // Attempt to acquire the lock ourselves atomically
        gptr = gptr | lowBits
        if atomic.Casuintptr(&hdr.info, flags, gptr) {
            return &b.data[idx]
        }
    }
}

func (hdr *bucketHdr) unlock() {
    // TODO: Implement unlocking mechanism (atomic)
}

func (hdr *bucketHdr) findBucket(t *maptype, key unsafe.Pointer) (*bucketChain, bool) {
    var lastEmpty *bucketChain
    for b := (*bucketChain)(hdr.bucket); b != nil; b = b.next {
        // If the bucketChain is not in use
        if b.flags == UNUSED {
            lastEmpty = b
            continue
        }

        // Whether we took the key by value or by reference, we resolve that here.
        otherKey := b.key(t)
        if t.indirectkey {
            otherKey = *(*unsafe.Pointer)(otherKey)
        }

        if t.key.alg.equal(key, otherKey) {
            return b, true;
        }
    }

    return lastEmpty, false
}

func (arr *bucketArray) init(t *maptype) {
    // First thing we need to do is allocate all bucketChain's in each bucketHdr
    for i := 0; i < MAXBUCKETS; i++ {
        chain := (*bucketChain)(newobject(t.bucketchain))
        
        // Populate up to MAXCHAIN bucketChain's. Avoids having to create call this repeatedly in the future.
        for j := 1; j < MAXCHAIN; j++ {
            chain.next = (*bucketChain) (newobject(t.bucketchain))
            chain.flags = UNUSED
            chain = chain.next
            chain.next = nil
        }
    }

    arr.seed = fastrand1()
}

// Assumes hdr holds a bucketChain and is not currently locked
func (hdr *bucketHdr) add(t *maptype, key unsafe.Pointer, value unsafe.Pointer) {
    b, pres := hdr.findBucket(t, key)

    // If the bucket is not present in the map, we must create a new one.
    if !pres {
        // If a bucket was not returned, then the bucket is actually full.
        if b == nil {
            // Unchain the bucket pointed to by hdr. 'bucket' now points to a bucketArray
            hdr.chainToArray(t, key)
            array := (*bucketArray)(hdr.bucket)

            // Rehash the key and recursively add it to the new bucketArray
            idx := t.key.alg.hash(key, uintptr(array.seed)) % MAXBUCKETS
            array.data[idx].add(t, key, value)
        } else {
            // if b != nil, then it returns the last bucket not in-use so we can recycle
            // These are actually the memory locations where the key and value reside
            k := b.key(t)
            v := b.value(t)

            // If either key or value is indirect, we must allocate a new object and copy into it.
            if t.indirectkey {
                kmem := newobject(t.key)
                *(*unsafe.Pointer)(k) = kmem
                k = kmem
            }
            if t.indirectvalue {
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

func (hdr *bucketHdr) chainToArray(t *maptype, key unsafe.Pointer) {
    array := (*bucketArray)(newobject(t.bucketarray))
    array.init(t)
    for b := (*bucketChain)(hdr.bucket); b.flags != 0; b = b.next {
        idx := t.key.alg.hash(b.key(t), uintptr(array.seed)) % MAXBUCKETS
        array.data[idx].add(t, b.key(t), b.value(t))

        // Now we must call memclr to let GC know the old copies no longer contains a reference to data.
        // memclr works by sweeping from [offset, offset + n) 
        memclr(b.key(t), uintptr(t.keysize) + uintptr(t.valuesize))
        b.flags = UNUSED
    }

    hdr.bucket = unsafe.Pointer(array)
    // TODO: Set flags to ARRAY
}

func makecmap(t *maptype, hint int64, h *hmap, bucket unsafe.Pointer) *hmap {
    println("Inside of makecmap: Key: ", t.key.string(), "; Value: ", t.elem.string())
    println("Sizeof bucketHdr: ", unsafe.Sizeof(bucketHdr{}), "\nSizeof bucketArray: ", unsafe.Sizeof(bucketArray{}))
    
    // Initialize the hashmap if needed
    if h == nil {
		h = (*hmap)(newobject(t.hmap))
	}
    
    // Initialize and allocate our concurrentMap
    cmap := (*concurrentMap)(newobject(t.concurrentmap))
    cmap.root.init(t)

    h.chdr = unsafe.Pointer(cmap)
    h.flags = 8
    
    return h
}

func cmapassign1(t *maptype, h *hmap, key unsafe.Pointer, val unsafe.Pointer) {
    println("Inside of cmapassign1!")
    cmap := (*concurrentMap)(h.chdr)
    hdr := cmap.root.findBucket(t, key)
    hdr.add(t, key, val)
}

func cmapaccess2(t *maptype, h *hmap, key unsafe.Pointer) (unsafe.Pointer, bool) {
    println("Inside of cmapaccess2!")
    // TODO: Set chdr to be of type concurrentMap, then no need to cast
    cmap := (*concurrentMap)(h.chdr)
    hdr := cmap.root.findBucket(t, key)
    b, pres := hdr.findBucket(t, key)
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
    println("Inside of cmapassign1")
    retval, _ := cmapaccess2(t, h, key)

    // Only difference is that we discard the boolean
    return retval
}