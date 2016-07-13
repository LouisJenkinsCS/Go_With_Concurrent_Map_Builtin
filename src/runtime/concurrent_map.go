
package runtime

import (
    "runtime/internal/atomic"
    "unsafe"
)

/*
    TODO: While the concurrent map works for non fast types (string, uint32, and uint64) values,
    it will fail and produce undefined behavior in a way, making me believe the compiler
    does something special for fast types. Figure out what is causing this and fix it.

    TODO: Randomize iteration order to reduce convoying (really good idea)
*/

const (
    MAXZERO = 1024 // must match value in ../cmd/compile/internal/gc/walk.go
    
    // The maximum amount of buckets in a bucketArray
    MAXBUCKETS = 16
    // The maximum amount of buckets in a bucketData
    MAXCHAIN = 8

    // bucketHdr is unlocked and points to a bucketData
    UNLOCKED = 0
    // bucketHdr's 'b' is a bucketArray. Note that once this is true, it always remains so.
    RECURSIVE = 1 << 0
    // If the bucketHdr's bucket is a bucketData, and is held by an iterator
    ITERATOR = 1 << 1

    // If currHash is equal to this, it is not in use
    EMPTY = 0

    // Maximum spins until exponential backoff kicks in
    BACKOFF_AFTER_SPINS = 0

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
    hash [MAXCHAIN]uintptr
    /*
        key [MAXCHAIN]keyType
        val [MAXCHAIN]valType
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
}

func (data *bucketData) key(t *maptype, idx uintptr) unsafe.Pointer {
    // Cast data to unsafe.Pointer to bypass Go's type system
    rawData := unsafe.Pointer(data)
    // The array of keys are located at the beginning of cdataOffset, and is contiguous up to MAXCHAIN
    keyOffset := uintptr(rawData) + uintptr(cdataOffset)
    // Now the key at index 'idx' is located at idx * t.keysize
    ourKeyOffset := keyOffset + idx * uintptr(t.keysize)
    return unsafe.Pointer(ourKeyOffset)
}

func (data *bucketData) value(t *maptype, idx uintptr) unsafe.Pointer {
    // Cast data to unsafe.Pointer to bypass Go's type system
    rawData := unsafe.Pointer(data)
    // The array of keys are located at the beginning of cdataOffset, and is contiguous up to MAXCHAIN
    keyOffset := uintptr(rawData) + uintptr(cdataOffset)
    // The array of values are located at the end of the array of keys, located at MAXCHAIN * t.keysize
    valueOffset := keyOffset + MAXCHAIN * uintptr(t.keysize)
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

    // By setting offset to MAXCHAIN, it allows it to bypass the findKeyValue portion without modification
    citer.offset = MAXCHAIN
    
    cmapiternext(it)
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
        if offset < MAXCHAIN {
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

        // If the offset == MAXCHAIN, then we exhausted this bucketData, reset offset for next one 
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
        backoff = 1
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

            // If another iterator owns this lock, we can attempt to go through as a reader
            if lock == ITERATOR {
                readers := atomic.Loaduintptr(&hdr.readers)

                // If the reader counter is 0, then the last reader has already exited; loop again, no backoff
                if readers == 0 {
                    continue
                }
                
                
                // Attempt to get in as a reader.
                if atomic.Casuintptr(&hdr.readers, readers, readers + 1) {
                    break
                }
                continue
            }

            // Even as an iterator, we must acquire lock to safely mark as ITERATOR
            if lock == UNLOCKED {
                // Attempt to acquire
                if atomic.Casuintptr(&hdr.lock, 0, gptr) {
                    // As such we may mark the bucket as being owned by an ITERATOR and initialize reader count
                    atomic.Storeuintptr(&hdr.readers, 1)
                    atomic.Storeuintptr(&hdr.lock, ITERATOR)
                    break
                }
            }

            spins++
            if spins > BACKOFF_AFTER_SPINS {
                timeSleep(backoff)
                backoff *= 2
            }
        }

        // Atomic snapshot of the bucketData
        typedmemmove(t.bucketdata, unsafe.Pointer(&citer.data), hdr.bucket)
        
        // Decrement our count as a reader
        for {
            newCount := atomic.Loaduintptr(&hdr.readers)
            if newCount == 0 {
                throw("Bad reader counter (attempted to decrement at 0)")
            }
            if atomic.Casuintptr(&hdr.readers, newCount, newCount - 1) {
                // If we succeed, and newCount was 1, then we just decremented it to 0 and we are last ones out
                if newCount == 1 {
                    // It is our job to unmark this bucket.
                    atomic.Storeuintptr(&hdr.lock, 0)
                }
                break;
            }
        }

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
    cmap.root.data = make([]bucketHdr, MAXBUCKETS)
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
        backoff = 1

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
                    break
                }
            }

            if spins > BACKOFF_AFTER_SPINS {
                timeSleep(backoff)
                backoff *= 2
            }
            spins++
        }
        
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
        for i := 0; i < MAXCHAIN; i++ {
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
            for i := 0; i < MAXCHAIN; i++ {
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
                for j := 0; j < MAXCHAIN; j++ {
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

        // Atomically release lock
        lock := atomic.Loaduintptr(&hdr.lock)

        if lock == UNLOCKED || lock == RECURSIVE {
            throw("Attempt to release a lock that has status of UNLOCKED or RECURSIVE!!!")
        } else if lock == ITERATOR {
            // Decrement our count as a reader
            for {
                newCount := atomic.Loaduintptr(&hdr.readers)
                if newCount == 0 {
                    throw("Bad reader counter (attempted to decrement at 0)")
                }
                if atomic.Casuintptr(&hdr.readers, newCount, newCount - 1) {
                    // If we succeed, and newCount was 1, then we just decremented it to 0 and we are last ones out
                    if newCount == 1 {
                        // It is our job to unmark this bucket.
                        atomic.Storeuintptr(&hdr.lock, 0)
                    }
                    break;
                }
            }
        } else {
            gptr := uintptr(unsafe.Pointer(g))
            // We SHOULD own the lock, but a quick check to ensure validity
            if lock != gptr {
                throw("Somehow attempting to unlock a lock that doesn't belong to us!")
            }
            
            // Otherwise, just unlock it as normal
            atomic.Storeuintptr(&hdr.lock, 0)
        }

        g.releaseBucket = nil
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
        backoff = 1

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

            // If the lock is held by an iterator, we can attempt to go through as a reader
            if lock == ITERATOR {
                readers := atomic.Loaduintptr(&hdr.readers)
                
                // If the reader counter is 0, then the last reader has already exited; loop again, no backoff
                if readers == 0 {
                    continue
                }
                
                // Attempt to get in as a reader.
                if atomic.Casuintptr(&hdr.readers, readers, readers + 1) {
                    g.releaseBucket = unsafe.Pointer(hdr)
                    break
                }
                continue
            }

            // If the lock is uncontested
            if lock == UNLOCKED  {
                // Attempt to acquire
                if atomic.Casuintptr(&hdr.lock, 0, gptr) {
                    g.releaseBucket = unsafe.Pointer(hdr)
                    break
                }
            }

            if spins > BACKOFF_AFTER_SPINS {
                timeSleep(backoff)
                backoff *= 2
            }
            spins++
        }

        // If the bucket is nil, then it is empty, stop here
        if hdr.bucket == nil {
            return unsafe.Pointer(&DUMMY_RETVAL[0]), false
        }

        data := (*bucketData)(hdr.bucket)
        
        // Search the bucketData for the data needed
        for i := 0; i < MAXCHAIN; i++ {
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
        backoff = 1

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
            }

            if spins > BACKOFF_AFTER_SPINS {
                timeSleep(backoff)
                backoff *= 2
            }
            spins++
        }

        // If the bucket is nil, then the key is not present in the map
        if hdr.bucket == nil {
            return
        }

        data := (*bucketData)(hdr.bucket)

        // Number of buckets empty; used to signify whether or not we should delete this bucket when we finish.
        isEmpty := MAXCHAIN
        for i := 0; i < MAXCHAIN; i++ {
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
                    for j := i + 1; isEmpty == MAXCHAIN && j < MAXCHAIN; j++ {
                        // If it is not empty, we decrement the count of empty index
                        if data.hash[j] != 0 {
                            isEmpty--
                        }
                    }
                    break
                }
            }
        }

        // If isEmpty == MAXCHAIN, then there are no indice still in use, so we delete the bucket to allow the iterator an easier time
        if isEmpty == MAXCHAIN {
            hdr.bucket = nil
        }
}

