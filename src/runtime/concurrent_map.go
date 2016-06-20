
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

    // bucketHdr's 'b' is unlocked (and is a bucketChain)
    UNLOCKED = 0
    // bucketHdr's 'b' is locked (and is a bucketChain)
    LOCKED = 1 << 0
    // bucketHdr's 'b' is a bucketArray. Note that once this is true, it always remains so.
    ARRAY = 1 << 1

)

var DUMMY_RETVAL [MAXZERO]byte

type bucketHdr struct {
    // Is either a bucketArray or bucketChain
    bucket unsafe.Pointer
    /*
        Flags:
            0 : Unlocked and Chained (b.(*bucketChain))
            1 : Unlocked (always) and Array (b.(*bucketArray))
            2 : Locked and Chained (always)
                - CAS Spin on the lock to obtain
                - Check if is chained, if so, proceed, else unlock and recurse
    */
    flags uint32

    /*
        Pointer to the 'g' (Goroutine) that currently acquired this lock. It can be noted that it's atomicstatus can
        determine if it is currently running or not, allowing for more fine-grained spinning.

        I.E:

        for {
            if isLocked(b) {
                g := (*g) atomic.Load(&b.g)
                
                if g == nil {
                    // The current g has changed, attempt to acquire lock again...
                }

                if atomic.Or8(g.atomicstatus, _Grunnable) != 0 {
                    // Back-off and continue spin
                } else {
                    // It is currently runnable, so just spin as it may finish within our time slice.
                }
            }
        }
    */
    g *g 
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
    // TODO: Find a way to compete with original hashmap's reduction in padding.
    key unsafe.Pointer
    val unsafe.Pointer
}

type concurrentMap struct {
    // Embedded to be contiguous in memory
    root bucketArray
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
        // Simple atomic load for flags
        flags := atomic.Load(&hdr.flags)

        // If it is a BucketArray, then this means we must traverse that bucket to find the right chain.
        if (flags & ARRAY) != 0 {
            return (*bucketArray)(hdr.bucket).findBucket(t, key)
        }

        // If we are locked, we need to spin again.
        if (flags & LOCKED) != 0 {
            // TODO: Backoff if heavy contention
            continue;
        }

        // Attempt to acquire the lock
        if atomic.Cas(&hdr.flags, flags, flags | LOCKED) {
            return &b.data[idx]
        }        
    }
}

func (hdr *bucketHdr) findBucket(t *maptype, key unsafe.Pointer) (*bucketChain, bool) {
    var lastEmpty *bucketChain
    for b := (*bucketChain)(hdr.bucket); b != nil; b = b.next {
        // If the key is nil, it is not in use.
        if b.key == nil {
            lastEmpty = b
        }

        // Whether we took the key by value or by reference, we resolve that here.
        otherKey := b.key
        if t.indirectkey {
            otherKey = *(*unsafe.Pointer)(otherKey)
        }

        if t.key.alg.equal(key, otherKey) {
            return b, true;
        }
    }

    return lastEmpty, false
}

func (hdr *bucketHdr)

func makecmap(t *maptype, hint int64, h *hmap, bucket unsafe.Pointer) *hmap {
    println("Inside of makecmap: Key: ", t.key.string(), "; Value: ", t.elem.string())
    println("Sizeof bucketHdr: ", unsafe.Sizeof(bucketHdr{}), "\nSizeof bucketArray: ", unsafe.Sizeof(bucketArray{}))
    
    // Initialize the hashmap if needed
    if h == nil {
		h = (*hmap)(newobject(t.hmap))
	}
    
    // Initialize and allocate our concurrentMap
    cmap := (*concurrentMap)(newobject(t.concurrentmap))
    for i := 0; i < MAXBUCKETS; i++ {
        cmap.root.data[i].bucket = newobject(t.bucketchain)
    }
    cmap.root.seed = fastrand1()

    h.chdr = unsafe.Pointer(cmap)
    h.flags = 8
    
    return h
}

func cmapassign1(t *maptype, h *hmap, key unsafe.Pointer, val unsafe.Pointer) {
    println("Inside of cmapassign1!")
    cmap := (*concurrentMap)(h.chdr)
    hdr := cmap.root.findBucket(t, key)
    b, pres := hdr.findBucket(t, key)

    // If the bucket is not present in the map, we must create a new one.
    if !pres {
        if !b {

        }
    }
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
        retval = b.val
        if t.indirectvalue {
            retval = *(*unsafe.Pointer)(retval)
        }
    } else {
        retval = unsafe.Pointer(&DUMMY_RETVAL[0])
    }

    hdr.unlock()

    return retval, pres
}

func cmapaccess1(t *maptype, h *hmap, key unsafe.Pointer) unsafe.Pointer {
    println("Inside of cmapassign1")
    retval, _ := cmapaccess2(t, h, key)

    // Only difference is that we discard the boolean
    return retval
}