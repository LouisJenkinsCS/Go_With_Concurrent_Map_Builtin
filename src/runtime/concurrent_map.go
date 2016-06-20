
package runtime

import "unsafe"

const (
    MAXBUCKETS = 16
    MAXCHAIN = 8
)

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
    flags uintptr

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
    h.chdr = unsafe.Pointer(cmap);
    h.flags = 8
    
    return h
}