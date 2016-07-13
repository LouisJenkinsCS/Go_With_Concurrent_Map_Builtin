
package runtime

import (
    "runtime/internal/atomic"
    "runtime/internal/sys"
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
    // Each index of pos represents a recursive array we are in
    stackPos []iteratorPosition
    // The 'size' of the stackPos. We explicitly need to keep track of this so we don't rely on len() function and can directly indice
    len uintptr
    // Offset we are inside of the bucketData
    offset uintptr
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
    gptr := uintptr(unsafe.Pointer(g))
    spins := 0
    var backoff int64 = 1

    // println("...g #", getg().goid, ": Obtained bucketHdr #", idx, " with addr:", hdr, "and seed:", uintptr(arr.seed), " and hash: ", uintptr(hash))

    for {
            flags := atomic.Loaduintptr(&hdr.info)
            lockHolder := flags &^ uintptr(0x7)

            // If is an RECURSIVE bucketArray
            if (flags & RECURSIVE) != 0 {
                // In process of conversion
                if lockHolder != 0 {
                    spins++

                    if spins > BACKOFF_AFTER_SPINS {
                        timeSleep(backoff)
                        backoff *= 2
                    }
                    continue
                }

                if spins > 0 {
                    // println("...g #", getg().goid, ": Recursing through nested array after spinning ", spins, "waiting for allocation")
                }
                return (*bucketArray)(hdr.bucket).findBucket(t, key)
            }

            // If an iterator holds this bucket, we spin
            if (flags & ITERATOR) != 0 {
                if spins > BACKOFF_AFTER_SPINS {
                    timeSleep(backoff)
                    backoff *= 2
                }
            }

            // Recursive locking; This cannot change (as its not possible for someone else to steal our lock)
            if lockHolder == gptr {
                return &(arr.data[idx]), hash, uintptr(arr.size)
            }

            // Test-And-Test-And-Set acquire lock
            if lockHolder == 0 {
                // If we acquired the lock successfully
                if atomic.Casuintptr(&hdr.info, 0, gptr) {
                    getg().releaseBucket = unsafe.Pointer(hdr)
                    if spins > 0 {
                        // println(".........g #", getg().goid, ": wasted ", spins, " CPU cycles spinning")
                    }
                    return &(arr.data[idx]), hash, uintptr(arr.size)
                }
            }

            spins++
            if spins > BACKOFF_AFTER_SPINS {
                timeSleep(backoff)
                backoff *= 2
            }
        }
}

type keyType struct {
    x, y int
} 

func toKeyType(key unsafe.Pointer) keyType {
    return *(*keyType)(key)
} 

func compare(k1, k2 unsafe.Pointer) {
    // println("......g #", getg().goid, ": (", toKeyType(k1).x, ", ", toKeyType(k1).y, ") vs. (", toKeyType(k2).x, ", ", toKeyType(k2).y, ")")
}

func printKey(k unsafe.Pointer) {
    // println("......g #", getg().goid, ": (", toKeyType(k).x, ", ", toKeyType(k).y, ")")
}

func (arr *bucketArray) init(t *maptype, size uintptr) {
    // g := getg().m.curg
    // println("......g #", getg().goid, ": Initializing bucketArray with size ", size)
    arr.data = make([]bucketHdr, size)
    arr.size = uint32(size)
    arr.seed = fastrand1()
}

func (hdr *bucketHdr) dataToArray(t *maptype, size uintptr, equal func (k1, k2 unsafe.Pointer) bool) {
    // g := getg().m.curg
    // println("...g #", getg().goid, ": Converting bucketData to bucketArray")
    array := (*bucketArray)(newobject(t.bucketarray))
    array.init(t, size)
    data := (*bucketData)(hdr.bucket)
    hdr.bucket = unsafe.Pointer(array)

    // Move each bucketData to their respective cell in the bucketArray
    for i := 0; i < MAXCHAIN; i++ {
        key := data.key(t, uintptr(i))
        val := data.value(t, uintptr(i))

        // Rehash the key with the new seed
        hash := t.key.alg.hash(key, uintptr(array.seed))
        idx := hash % uintptr(array.size)
        arrHdr := &array.data[idx]
        arrData := (*bucketData)(arrHdr.bucket)

        // This is the first insertion into this bucket, allocate
        if arrData == nil {
            arrHdr.bucket = newobject(t.bucketdata)
            arrData = (*bucketData)(arrHdr.bucket)

            // Directly insert into map
            arrData.assign(t, 0, hash, key, val)
            continue
        }

        // Otherwise, we know this is not the first, so we need to scan for the first empty (after the first)
        for j := 1; j < MAXCHAIN; j++ {
            if arrData.hash[j] == 0 {
                arrData.assign(t, uintptr(j), hash, key, val)
                break
            }
        }
    }

    // Set the RECURSIVE bit, release lock
    atomic.Storeuintptr(&hdr.info, RECURSIVE)
}

func (it *concurrentIterator) nextKeyValue(t *maptype) (unsafe.Pointer, unsafe.Pointer) {
    for ; it.offset < MAXCHAIN; it.offset++ {
        // If the hash is 0, then it is not in use.
        if it.data.hash[it.offset] == 0 {
            continue
        }

        // TODO: Potentially scan the map again to see if the key has been deleted

        // Perform necessary indirection on key and value if needed
        k := it.data.key(t, it.offset)
        if t.indirectkey {
            k = *(*unsafe.Pointer)(k)
        }

        v := it.data.value(t, it.offset)
        if t.indirectvalue {
            v = *(*unsafe.Pointer)(v)
        }

        // Increment the offset so next call is updated to retrieve next
        it.offset++

        return k, v
    }

    // If we find nothing, we return nothing
    return nil, nil
}

/*
        We must find a random bucket, but to do so, we must also recurse through any RECURSIVE bucketArray's we
        come across. Each time we obtain the lock successfully, we check to see if the cell is empty, and if it
        is we simply move on to the next one, making doubly sure we do not loop around more than once. 

        All of this is accomplished by saving and restoring position information in two slices used as stacks. Goto
        statements are used for readability and to reduce complexity of the overall control flow.

        To help imagine the complexity of picking a randomized bucket when each bucketArray can be recursive, see the
        below diagram.

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

        Imagine now that we randomly start at the 4th cell. We must note we need to get back to 

#6 =>   [D]
        [D]
        [0]
#1 =>   [R] -> [0] <= #2
               [0] <= #3
               [0] <= #4
               [0] <= #5

        As can be seen, the iterator will need to be able to wrap all the way around. This is a very
        complicated operation.
*/
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
    citer := (*concurrentIterator)(newobject(t.concurrentiterator))
    it.citerHdr = unsafe.Pointer(citer)
    var len uintptr = 0
    
    // Our current position 
    pos := iteratorPosition{0, 0, &cmap.root}
    // The pushed arrays and current indice for recursive positioning
    stackPos := make([]iteratorPosition, 16)
    // Current header
    var hdr *bucketHdr

    // Randomly decide the bucket we are starting at (to reduce convoying during concurrent iteration).
    seed := (uintptr(fastrand1()) | uintptr(1 << ((sys.PtrSize * 8) - 1)) | uintptr(1))
    // println("Random Generated Seed: ", seed)
    spins := 0
    var backoff int64 = 1
    
    setup:
        // idx is the current index, but startIdx is the one we quit at if we are reach idx again
        pos.idx = seed % uintptr(pos.arr.size)
        pos.startIdx = pos.idx

    next:
        hdr = &pos.arr.data[pos.idx]
        pos.idx = (pos.idx + 1) % uintptr(pos.arr.size)

         // If we wrapped around
        if pos.idx == pos.startIdx {
            // If we wrapped all the way to the root, we found nothing
            if pos.arr == &cmap.root {
                it.key = nil
                it.value = nil     
                return
            }

            // Otherwise, we need to go back up one recursive bucketArray. Pop to restore iterator position
            len--
            pos = stackPos[len]

            // Start over from the upper-level bucketArray
            goto next
        }

        // Read ahead of time if we should skip to the next or attempt to lock and acquire (Test-And-Test-And-Set)
        if atomic.Loadp(unsafe.Pointer(&hdr.bucket)) == nil {
            goto next
        }

        backoff = 1
        spins = 0
        g := getg()
        gptr := uintptr(unsafe.Pointer(g))

        for {
            flags := atomic.Loaduintptr(&hdr.info)
            lockHolder := flags &^ 0x7

            // If is an RECURSIVE
            if (flags & RECURSIVE) != 0 {
                // In process of conversion
                if lockHolder != 0 {
                    spins++
                    if spins > BACKOFF_AFTER_SPINS {
                        timeSleep(backoff)
                        backoff *= 2
                    }
                    continue
                }

                // Once RECURSIVE is set and it is not locked, it can never change. Push the current position on stack
                if len == uintptr(cap(stackPos)) {
                    stackPos = append(stackPos, pos)
                } else {
                    stackPos[len] = pos
                }
                len++

                // The current position's bucketArray is the current header's bucket
                pos.arr = (*bucketArray)(hdr.bucket)

                // Setup on recurisve bucektArray
                goto setup
            }

            // If another iterator owns this bucket, attempt to get in
            if (flags & ITERATOR) != 0 {
                // We know from the fact that if ITERATOR bit is set, then it points to bucketData
                data := (*bucketData)(hdr.bucket)

                // If data == nil, the bucket could have been released and another thread could potentially have deleted it
                if data == nil {
                    return
                }
                
                // Fetch readers; If it is 0, then the holder who originally owned it no longer does, and so we try again
                readers := atomic.Loaduintptr(&data.readers)
                if readers == 0 {
                    continue
                }
                
                // If in this snapshot, readers != 0, attempt to increment count
                if atomic.Casuintptr(&data.readers, readers, readers + 1) {
                    // If we succeed, we may safely read the data. Boilerplate copy...
                    typedmemmove(t.bucketdata, unsafe.Pointer(&citer.data), unsafe.Pointer(data))
                    
                    // Decrement our count over the lock
                    for {
                        newCount := atomic.Loaduintptr(&data.readers)
                        if newCount == 0 {
                            throw("Bad reader counter (attempted to decrement at 0)")
                        }
                        if atomic.Casuintptr(&data.readers, newCount, newCount - 1) {
                            // If we succeed, and newCount was 1, then we just decremented it to 0 and we are last ones out
                            if newCount == 1 {
                                // It is our job to unmark this bucket.
                                atomic.Storeuintptr(&hdr.info, 0)
                            }
                            break;
                        }
                    }
                }
                continue
            }

            // Test-And-Test-And-Set acquire lock
            if lockHolder == 0 {
                // We must first check if the bucket pointed to by hdr is nil, which requires the lock
                if atomic.Casuintptr(&hdr.info, 0, gptr) {
                    // If the bucket is nil, release the lock and try again
                    if hdr.bucket == nil {
                        atomic.Storeuintptr(&hdr.info, 0)
                        goto next
                    }

                    // Otherwise, since we already own this bucket, update it's readers and mark the bucket
                    data := (*bucketData)(hdr.bucket)
                    data.readers++
                    atomic.Storeuintptr(&hdr.info, ITERATOR)

                    // If it has been found, we're golden. We need to also push the current iterator position for cmapiternext to use
                    if len == uintptr(cap(stackPos)) {
                        stackPos = append(stackPos, pos)
                    } else {
                        stackPos[len] = pos
                    }

                    // Finally setup the concurrentIterator and release lock
                    typedmemmove(t.bucketdata, unsafe.Pointer(&citer.data), unsafe.Pointer(data))
                    
                    // Decrement our count over the lock
                    for {
                        newCount := atomic.Loaduintptr(&data.readers)
                        if atomic.Casuintptr(&data.readers, newCount, newCount - 1) {
                            // If we succeed, and newCount was 1, then we just decremented it to 0 and we are last ones out
                            if newCount == 1 {
                                // It is our job to unmark this bucket.
                                atomic.Storeuintptr(&hdr.info, 0)
                            }
                            break;
                        }
                    }


                    // Set the stack of iterator positions for cmapiternext to use
                    citer.stackPos = stackPos
                    citer.len = len

                    // Set the key and value
                    for i := 0; i < MAXCHAIN; i++ {
                        if citer.data.hash[i] != 0 {
                            it.key = citer.data.key(t, uintptr(i))
                            it.value = citer.data.value(t, uintptr(i))
                            citer.offset = uintptr(i)
                            return
                        }
                    }
                    throw("Iterated over bucketData, but no key-value found")
                }
            }

            spins++
            if spins > BACKOFF_AFTER_SPINS {
                timeSleep(backoff)
                backoff *= 2
            }
        }
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
    root := &(*concurrentMap)(it.h.chdr).root
    data := &citer.data
    t := it.t
    var hdr *bucketHdr
    var key, value unsafe.Pointer
    spins := 0
    var backoff int64 = 1

    findKeyValue:
        offset := citer.offset
        citer.offset++

        // If there is more to find, do so
        if offset < MAXCHAIN {
            // If the hash is 0, that index is empty
            if data.hash[offset] == 0 {
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
    
    // 'setup' goto label is called to setup the iterator position and performs wrapping checks. This may be skipped if done manually.
    setup:
        // Pop the current position off the stack
        // println("g #", getg().goid, ": citer.len: ", citer.len, ";len: ", len(citer.stackPos), ";cap: ", cap(citer.stackPos))
        pos := &citer.stackPos[citer.len]

        // If we already wrapped all the way around
        if pos.idx == pos.startIdx {
            // If this is the root, we can't pop anymore
            if pos.arr == root {
                it.key = nil
                it.value = nil
                return
            } else {
                // Discard the current position (pop)
                citer.len--

                goto setup
            }
        }
    
    // 'next' goto label is called to advance the iterator to the next bucketArray.
    next:
        hdr = &pos.arr.data[pos.idx]
        pos.idx = (pos.idx + 1) % uintptr(pos.arr.size)

        // Read ahead of time if we should skip to the next or attempt to lock and acquire (Test-And-Test-And-Set)
        if atomic.Loadp(unsafe.Pointer(&hdr.bucket)) == nil {
            goto setup
        } 

        spins = 0
        backoff = 1
        g := getg()
        gptr := uintptr(unsafe.Pointer(g))

        for {
            flags := atomic.Loaduintptr(&hdr.info)
            lockHolder := flags &^ 0x7

            // If is an RECURSIVE
            if (flags & RECURSIVE) != 0 {
                // In process of conversion
                if lockHolder != 0 {
                    spins++
                    if spins > BACKOFF_AFTER_SPINS {
                        timeSleep(backoff)
                        backoff *= 2
                    }
                    continue
                }

                // Once RECURSIVE is set and it is not locked, it can never change. Setup new iterator position and push.
                citer.len++
                if citer.len == uintptr(cap(citer.stackPos)) {
                    citer.stackPos = append(citer.stackPos, iteratorPosition{0, 0, (*bucketArray)(hdr.bucket)})
                } else {
                    citer.stackPos[citer.len] = iteratorPosition{0, 0, (*bucketArray)(hdr.bucket)}
                }

                // Note above that idx == startIdx, hence we explicitly skip the 'setup' portion and set pos ourselves
                pos = &citer.stackPos[citer.len]

                goto next
            }

            // If another iterator owns this bucket, attempt to get in
            if (flags & ITERATOR) != 0 {
                // We know from the fact that if ITERATOR bit is set, then it points to bucketData
                data := (*bucketData)(hdr.bucket)
                
                // However, if data == nil, then the bucket was released, and another thread could potentially have deleted it
                if data == nil {
                    continue
                }

                // Fetch readers; If it is 0, then the holder who originally owned it no longer does, and so we try again
                readers := atomic.Loaduintptr(&data.readers)
                if readers == 0 {
                    continue
                }
                
                // If in this snapshot, readers != 0, attempt to increment count
                if atomic.Casuintptr(&data.readers, readers, readers + 1) {
                    // If we succeed, we may safely read the data. Boilerplate copy...
                    typedmemmove(t.bucketdata, unsafe.Pointer(&citer.data), unsafe.Pointer(data))
                    
                    // Decrement our count over the lock
                    for {
                        newCount := atomic.Loaduintptr(&data.readers)
                        if newCount == 0 {
                            throw("Bad reader counter (attempted to decrement at 0)")
                        }
                        if atomic.Casuintptr(&data.readers, newCount, newCount - 1) {
                            // If we succeed, and newCount was 1, then we just decremented it to 0 and we are last ones out
                            if newCount == 1 {
                                // It is our job to unmark this bucket.
                                atomic.Storeuintptr(&hdr.info, 0)
                            }
                            break;
                        }
                    }
                }
                continue
            }

            // Test-And-Test-And-Set acquire lock
            if lockHolder == 0 {
                // We must first check if the bucket pointed to by hdr is nil, which requires the lock
                if atomic.Casuintptr(&hdr.info, 0, gptr) {
                    // If the bucket is nil, release the lock and try again
                    if hdr.bucket == nil {
                        atomic.Storeuintptr(&hdr.info, 0)
                        goto next
                    }

                    // Otherwise, since we already own this bucket, update it's readers and mark the bucket
                    data := (*bucketData)(hdr.bucket)
                    data.readers++
                    atomic.Storeuintptr(&hdr.info, ITERATOR)

                    // Make a copy of the bucketData
                    typedmemmove(t.bucketdata, unsafe.Pointer(&citer.data), unsafe.Pointer(data))
                    
                    // Decrement our count over the lock
                    for {
                        newCount := atomic.Loaduintptr(&data.readers)
                        if atomic.Casuintptr(&data.readers, newCount, newCount - 1) {
                            // If we succeed, and newCount was 1, then we just decremented it to 0 and we are last ones out
                            if newCount == 1 {
                                // It is our job to unmark this bucket.
                                atomic.Storeuintptr(&hdr.info, 0)
                            }
                            break;
                        }
                    }

                    goto findKeyValue
                }
            }

            spins++
            if spins > BACKOFF_AFTER_SPINS {
                timeSleep(backoff)
                backoff *= 2
            }
        }
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
    cmap.root.init(t, MAXBUCKETS)

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
        hash = t.key.alg.hash(key, arr.seed)
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
            newArr := (*bucketArray)(newobject(t.bucketArray))
            newArr.data = make([]bucketHdr, arr.size * 2)
            newArr.seed = fastrand1()
            newArr.backLink = arr
            newArr.backIdx = uint32(idx)

            // Rehash and move all key-value pairs
            for i := 0; i < MAXCHAIN; i++ {
                k := data.key(t, uintptr(i))
                v := data.value(t, uintptr(i))

                // Rehash the key to the new seed
                newHash := t.key.alg.hash(key, newArr.seed)
                newIdx := newHash % uintptr(len(newArr.data))
                newHdr := &newArr.data[newIdx]
                newData := (*bucketData)(newHdr.bucket)
                
                // Check if the bucket is nil, meaning we haven't allocated to it yet.
                if newData == nil {
                    newData = (*bucketData)(newobject(t.bucketdata))
                    newData.assign(t, 0, newHash, k, v)
                    continue
                }

                // If it is not nil, then we must scan for the first non-empty slot
                for j := 0; j < MAXCHAIN; j++ {
                    currHash := newData.hash[j]
                    if currHash != EMPTY {
                        newData.assign(t, uintptr(j), newHash, k, v)
                        break
                    }
                }
            }

            // Now dispose of old data and update the header's bucket
            memclr(data, t.bucketdata.size)
            hdr.bucket = newArr
            
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
        flags := atomic.Loaduintptr(&hdr.info)
        atomic.Storeuintptr(&hdr.info, uintptr(flags & 0x7))

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
        hash = t.key.alg.hash(key, arr.seed)
        idx = hash % uintptr(len(arr.data))
        hdr = &arr.data[idx]
        spins = 0
        backoff = 1

        // Save time by looking ahead of time (testing) if the header is nil
        if atomic.Loadp(&hdr.bucket) == nil {
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
    hdr, hash, _ := cmap.root.findBucket(t, key)
    data := (*bucketData)(hdr.bucket)

    if data == nil {
        return
    }

    // Number of buckets empty; used to signify whether or not we should delete this bucket when we finish.
    isEmpty := MAXCHAIN
    for i := 0; i < MAXCHAIN; i++ {
        currHash := data.hash[i]

        if currHash == 0 {
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
                // memclr clears the memory in such a way that we no longer store a reference to it's pointer data
                memclr(data.key(t, uintptr(i)), uintptr(t.keysize))
                memclr(data.value(t, uintptr(i)), uintptr(t.valuesize))
                // Hahs of 0 marks bucket at empty and reusable
                data.hash[i] = 0
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

