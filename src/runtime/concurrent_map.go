
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

    CHAINED = 0
    // bucketHdr's 'b' is a bucketArray. Note that once this is true, it always remains so.
    ARRAY = 1 << 0
    // If the bucketHdr's 'b' is locked, another goroutine is currently accessing it.
    LOCKED = 1 << 1

    // bucketData is unused
    UNUSED = 0
    // bucketData is in use
    USED = 1 << 0

    // See hashmap.go, this obtains a properly aligned offset to the data
    cdataOffset = unsafe.Offsetof(struct {
		b bucketData
		v int64
	}{}.v)
)

var DUMMY_RETVAL [MAXZERO]byte

type bucketHdr struct {
    // Is either a bucketArray or bucketData
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
    // What we are wrapping to
    startPos []iteratorPosition
    // Each index of pos represents a recursive array we are in
    stackPos []iteratorPosition
    // Offset we are inside of the bucketData
    offset uintptr
    // If we have traversed back to the root, but need to wrap around to completion
    wrapping bool
    // The bucketData we are iterating over
    data bucketData
}

type iteratorPosition struct {
    // Current index in the bucketArray we will process next
    idx uintptr
    // The index we started on, I.E the one we wrap up to until we consider a bucketArray exhausted
    startIdx uintptr
    // The bucketArray we are currently processing
    arr *bucketArray
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
        // println("......g #", g.goid, ": re-entered the lock")
        return &(arr.data[idx])
    } else if lockHolder != 0 {
        spins++
        Gosched()
        continue
    }
    ...
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
    spins := 0

    // println("...g #", g.goid, ": Obtained bucketHdr #", idx, " with addr:", hdr, "and seed:", uintptr(arr.seed), " and hash: ", uintptr(hash))

    // TODO: Use the holder 'g' to keep track of whether or not the holder is still running or not to determine if we should yield.
    for {
        // A simple CAS can be used to attempt to acquire the lock. If it is an ARRAY, this will always fail, as the flag will never be 0 again.
        if atomic.Casuintptr(&hdr.info, 0, LOCKED) {
            // println("......g #", g.goid, ":  acquired lock")
            g.releaseBucket = unsafe.Pointer(&arr.data[idx])
            // g.releaseM = acquirem()
            if spins > 0 {
                // println(".........g #", g.goid, ": wasted ", spins, " CPU cycles spinning")
            }
            return &(arr.data[idx]), hash, uintptr(arr.size)
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
                // println("...g #", g.goid, ": Recursing through nested array after spinning ", spins, "waiting for allocation")
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
    // println("......g #", getg().goid, ": (", toKeyType(k1).x, ", ", toKeyType(k1).y, ") vs. (", toKeyType(k2).x, ", ", toKeyType(k2).y, ")")
}

func printKey(k unsafe.Pointer) {
    // println("......g #", getg().goid, ": (", toKeyType(k).x, ", ", toKeyType(k).y, ")")
}

func (arr *bucketArray) init(t *maptype, size uintptr) {
    // g := getg().m.curg
    // println("......g #", g.goid, ": Initializing bucketArray with size ", size)
    arr.data = make([]bucketHdr, size)
    arr.size = uint32(size)
    arr.seed = fastrand1()
}

func (hdr *bucketHdr) dataToArray(t *maptype, size uintptr, equal func (k1, k2 unsafe.Pointer) bool) {
    // g := getg().m.curg
    // println("...g #", g.goid, ": Converting bucketData to bucketArray")
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

    // Atomically 'OR' ARRAY in place without releasing the spinlock
    for {
        oldInfo := atomic.Loaduintptr(&hdr.info)
        if atomic.Casuintptr(&hdr.info, oldInfo, oldInfo | ARRAY) {
            break;
        }
    }
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
    Obtains the next bucketData in this bucketArray. Each time we recurse, we must store
    the previous 
*/
func (it *concurrentIterator) nextBucket(t *maptype) bool {
    // The current index we are on is the top; idxPtr is kept to easily allow mutation.
    idxPtr := &it.idx[len(it.idx)-1]
    idx := *idxPtr
    
    // If the index is equal to array.size (out of bounds), then we exhausted all buckets here.
    if idx == uintptr(it.array.size) {
        // If it is not possible to go back, we have iterated over the entire map
        if it.array.backlink == nil {
            it.array = nil
            return false
        }

        // If we can, traverse through the backlink and pop off this idx from the stack
        it.array = it.array.backlink
        it.idx = it.idx[:len(it.idx)-1]

        // Recursively evaluate the next bucket (saves effort on our part)
        return it.nextBucket(t)
    }

    hdr := &it.array.data[idx]
    var data *bucketData
    spins := 0
    (*idxPtr)++

    // Find the next appropriate bucketData
    for {
        // A simple CAS can be used to attempt to acquire the lock. If it is an ARRAY, this will always fail, as the flag will never be 0 again.
        if atomic.Casuintptr(&hdr.info, 0, LOCKED) {
            // println("......g #", g.goid, ":  iterator acquired lock")
            if spins > 0 {
                // println(".........g #", g.goid, ": iterator wasted ", spins, " CPU cycles spinning")
            }
            
            // We have the bucket we want
            data = (*bucketData)(hdr.bucket)
            break
        }

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
                // println("...g #", g.goid, ": Iterator recursing through nested array after spinning ", spins, "waiting for allocation")
            }
            
            // Set the backlink in case it wasn't already, and setup safe recursive traversal
            oldArray := it.array
            it.array = (*bucketArray)(hdr.bucket)
            it.array.backlink = oldArray
            it.idx = append(it.idx, 0)
            
            // Now recurse to reduce needed work
            return it.nextBucket(t)
        }

        spins++
        Gosched()
    }

    // if the bucket is nil, then it's empty, move on to the next, recursively to save work
    if data == nil {
        // Atomically release lock
        for {
            oldFlags := atomic.Loaduintptr(&hdr.info)
            if atomic.Casuintptr(&hdr.info, oldFlags, uintptr(oldFlags &^ LOCKED)) {
                break
            }
        }

        // Now recurse to reduce needed work
        return it.nextBucket(t)
    }

    // If data is not nil, then we make and operate on a simple bucketCopy
    typedmemmove(t.bucketdata, unsafe.Pointer(&it.data), unsafe.Pointer(data))

    // Atomically release lock
    for {
        oldFlags := atomic.Loaduintptr(&hdr.info)
        if atomic.Casuintptr(&hdr.info, oldFlags, uintptr(oldFlags &^ LOCKED)) {
            break
        }
    }

    return true
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
    
    // Our current position 
    pos := iteratorPosition{0, 0, &cmap.root}
    // The pushed endIdx to wrap to for this recursive position
    stackEndIdx := make([]uintptr)
    // The pushed arrays and current indice for recursive positioning
    stackPos := make([]iteratorPosition)

    // Randomly decide the bucket we are starting at (to reduce convoying during concurrent iteration)
    seed := fastrand1()
    spins := 0
    
    setup:
        // idx is the current index, but endIndx is the one we quit at if we are reach idx again while being in the root
        idx := seed % pos.arr.size
        endIdx := idx
    
    next:
        hdr := &pos.arr.data[idx]
        idx = (idx + 1) % pos.arr.size

        for {
            // If we acquired the lock successfully
            if atomic.Casuintptr(&hdr.info, 0, LOCKED) {
                // If the bucket is nil, release the lock and try again
                if hdr.bucket == nil {
                    atomic.Storeuintptr(&hdr.info, 0)

                    // If we wrapped around
                    if idx == endIdx {
                        // If we wrapped all the way to the root, we found nothing
                        if pos.arr == &cmap.root {
                            goto empty
                        }

                        // Otherwise, we need to go back up one recursive bucketArray
                        goto pop
                    }

                    // If we have not wrapped all the way around, get the next
                    goto next
                }

                // If it has been found, we're golden
                goto found
            }

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
                
                // Push the current information
                goto push
            }

            spins++
            Gosched()
        }

    pop:
        // Pop information off stacks
        pos = stackPos[len(stackPos)-1]
        stackPos = stackPos[:len(stackPos)-1]
        idx = pos.startIdx
        endIdx = stackEndIdx[len(stackPos)-1]
        stackEndIdx = stackEndIdx[:len(stackPos)-1]

        // idx can be endIdx if it was on the last bucketHdr to wrap around
        if idx == endIdx {
            // if the bucketArray we are on is the root, we cannot pop anything else, and we did not find a suitable bucket
            if pos.arr == &cmap.root {
                goto empty
            }

            // Otherwise, we pop off the next one
            goto pop
        }
        // Start over from the upper-level bucketArray
        goto next
    push:
        // Push the current position and index; startIdx is the one we are currently on, idx is the one we are to process next
        pos.idx = idx
        pos.startIdx = idx - 1
        stackPos = append(stackPos, pos)

        // Push the index we are wrapping to
        stackEndIdx = append(stackEndIdx, endIdx)

        // The current position's bucketArray is the current header's bucket
        pos.arr = (*bucketArray)(hdr.bucket)

        // Setup on recurisve bucektArray
        goto setup
    // 'empty' label is called if and only if we have exhausted our search and came up with nothing
    empty:
        it.key = nil
        it.value = nil
        
        return
    // 'found' label is called if and only if we have found a non-nil bucketData (note we also need to release lock)
    found:
        // We need to also push the current pos on top of the stackPos
        stackPos = append(stackPos, pos)
        
        // Finally setup the concurrentIterator
        typedmemmove(t.bucketdata, unsafe.Pointer(&citer.data), hdr.bucket)

        // startPos and stackPos share the same elements at first so startPos can keep track of the wrapped bucketArray
        citer.startPos = stackPos
        citer.stackPos = stackPos

        return
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

    #14 => [R] ---------> #15 => [R] ------------> [D] <= #16
           [D] <= #7             [0] <= #5         [D] <= #17                                            
           [D] <= #8             [0] <= #6         [D] <= #18
     #9 => [R] -> [0] <= #10                 #4 => [R] -------> [0] <= #19
                  [0] <= #11                                    [0] <= #1
                  [0] <= #12                                    [0] <= #2          
                  [0] <= #13                                    [0] <= #3
                

    Note now the reason why we end up going back is because we cannot afford to hit the same bucket twice,
    and if we recurse to another array, we could accidentally hit it again on the way back. Hence, we have to wrap around to hit 
    everything.
*/
func cmapiternext(it *hiter) {
    citer := (*concurrentIterator)(it.citerHdr)
    root := &(*concurrentMap)(it.h.chdr).root
    data := &citer.data
    var key, value unsafe.Pointer

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
            key = it.data.key(t, it.offset)
            if t.indirectkey {
                key = *(*unsafe.Pointer)(key)
            }
            value := it.data.value(t, it.offset)
            if t.indirectvalue {
                value = *(*unsafe.Pointer)(value)
            }

            goto found
        }

        // If the offset == MAXCHAIN, then we exhausted this bucketData, reset offset for next one 
        citer.offset = 0
    
    findBucketData:
        // Pop the current position off the stack
        pos := &citer.stackPos[len(citer.stackPos)-1]

        // If we are currently wrapping back around
        if citer.wrapped {
            // End position is the top of the startPos stack, where we originally started iterating
            endPos := citer.startPos[len(citer.startPos)-1]

            // If we wrapped to the start
            if pos.idx == endPos.startIdx && pos.arr == endPos.arr {
                goto done
            }
        }

        // If we have hit the end of this bucketArray (and not wrapping)
        if !citer.wrapping && pos.idx == pos.arr.size {
            // Reset to 0
            pos.idx = 0

            // If we are currently on the root, we have iterated through everything but the nodes before startPos.idx, so do so now
            if pos.arr == root {
                // Setting wrapped flag to true triggers a specialized mode for the iterator
                citer.wrapping = true
                
                goto findBucketData
            }

            // Otherwise, we exhausted this bucket, go back one
            goto pop
        }
        hdr := &pos.arr.data[pos.idx]
        pos.idx++

    pop:

    push:

    done:
        it.key = nil
        it.value = nil
        return
    found:
        it.key = k
        it.value = v
        return
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
    // println("g #", g.goid, ": cmapassign1")
    // println("...g #", getg().goid, ": Adding Key: ", toKeyType(key).x, "," ,  toKeyType(key).y, " and Value: ", toKeyType(val).x, ",", toKeyType(val).y)
    cmap := (*concurrentMap)(h.chdr)
    hdr, hash, sz := cmap.root.findBucket(t, key)
    
    // If bucket is nil, then we allocate a new one.
    if hdr.bucket == nil {
        hdr.bucket = unsafe.Pointer(newobject(t.bucketdata))
        
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
        // When we convert dataToArray, it's size is double the previous
        hdr.dataToArray(t, sz * 2, t.key.alg.equal)

        // After converting it from to a RECURSIVE array, we can release the lock to allow further concurrency.
        maprelease()
        
        // Also we can save effort by just recursively calling this again.
        cmapassign1(t, h, key, val)  
        return
    }

    // At this point, firstEmpty is guaranteed to be non-zero, hence we can safely assign it
    data.assign(t, uintptr(firstEmpty), hash, key, val)
}

func maprelease() {
    g := getg()
    if g.releaseBucket != nil {
        // println("g #", g.goid, ": released lock")
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
    // g := getg().m.curg
    // println("g #", g.goid, ": cmapaccess1_fast32!")
    retval, _ := cmapaccess2_fast32(t, h, key)
    return retval
}

func cmapaccess2_fast32(t *maptype, h *hmap, key uint32) (unsafe.Pointer, bool) {
    // g := getg().m.curg
    // println("g #", g.goid, ": cmapaccess2_fast32!")
    return cmapaccess(t, h, noescape(unsafe.Pointer(&key)), 
        func (k1, k2 unsafe.Pointer) bool { 
            return *(*uint32)(k1) == *(*uint32)(k2) 
        })
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
    return cmapaccess(t, h, noescape(unsafe.Pointer(&key)), 
        func (k1, k2 unsafe.Pointer) bool { 
            return *(*uint64)(k1) == *(*uint64)(k2) 
        })
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
    // println("g #", g.goid, ": cmapaccess2!")

    return cmapaccess(t, h, key, t.key.alg.equal)
}

func cmapaccess1(t *maptype, h *hmap, key unsafe.Pointer) unsafe.Pointer {
    // g := getg().m.curg
    // println("g #", g.goid, ": cmapaccess1")
    retval, _ := cmapaccess2(t, h, key)

    // Only difference is that we discard the boolean
    return retval
}

func cmapdelete(t *maptype, h *hmap, key unsafe.Pointer) {
    // g := getg().m.curg
    // println("g #", g.goid, ": cmapdelete")

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

