#Concurrent Hash Map

##Why?

Go lacks a concurrent HashMap, and regardless of whether or not the developers think that "there is not enough demand for it", a concurrent language that does not support naturally concurrent maps outside of protecting a normal hashmap with a Reader-Writer Mutex is just... strange.

Even stranger is that Go does not provide any abstractions or tools to build your own concurrent maps or data structures. There is no (normal) way to hash an object outside of the runtime, and no way to make a truly concurrent HashMap outside of it either. There also is no way to implement the necessary primitives and semantics needed outside of modifying the compiler either.

This is an attempt at implementing and adding a truly concurrent map that allows both concurrent readers and writers.

##What?

The concurrent HashMap will allow atomic "loads" of a value obtained from a key, as well as atomic "stores" like assigning a value to a key. What does this mean? The Go's compiler normally turns an operation like `m[1] += m[2]`, into:

```go
autotmp_1 = 2
// What is returned is the address of the bucket which holds the value
autotmp_2 = *mapaccess1(maptype, &m, &autotmp_1)
autotmp_3 = 1
mapassign1(maptype, &m, &autotmp_3, &autotmp_2)
```

Note the race conditions of unprotected access. `m[2]` can potentially be changed before it is fully copied into `autotmp_2`, which is why normally concurrent access is not allowed. What the concurrent map does is, it will protect all access up to the end of the assignment, making it safe, no locks required.

##RWMutex vs Concurrent HashMap

A `RWMutex` can only allow one writer or multiple readers, but imagine if you have thousands of writers and readers, or even just a few writers and thousadns of readers. With a concurrent HashMap, the map is segmented, and locks are acquired only for that particular bucket. Meaning, so long as a reader or writer work on different buckets, they may work together concurrently.