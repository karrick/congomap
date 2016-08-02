# Congomap

Concurrent Go Map

This repository serves as a set of examples for making maps that are accessible in concurrent Go
software. The types _can_ be used as a library, each with their own performance characteristics, but
I wrote it to determine which method produced the most readable code, and the most performant code.

[![GoDoc](https://godoc.org/github.com/karrick/congomap?status.svg)](https://godoc.org/github.com/karrick/congomap)

## Examples

This library exposes the `Congomap` interface, and three concrete types that adhere to that
interface. All three concrete types are available here because they have individual performance
characteristics, where one concrete type may be more appropriate for a desired used that one of the
other types.

### Creating a Congomap

Creating a `Congomap` is done by calling the instantiation function of the desired concrete type.

NOTE: To prevent resource leakage, always call the `Close` method on a `Congomap` after it is no
longer needed.

#### NewChannelMap

A channel map is modeled after the Go way of sharing memory: by communicating over channels. Reads
and writes are serialized by a Go routine processing anonymous functions. While not as fast as the
other methods for low-concurrency loads, this particular map outpaces the competition in
high-concurrency tests.

```Go
cgm, _ := cmap.NewChannelMap()
defer cgm.Close()
```

#### NewSyncAtomicMap

A sync atomic map uses the algorithm suggested in the documentation for `sync/atomic`. It is
designed for when a map is read many, many more times than it is written. Performance also depends
on the number of the keys in the map. The more keys in the map, the more expensive Store and
LoadStore will be.

```Go
cgm,_ := congomap.NewSyncAtomicMap()
defer cgm.Close()
```

#### NewSyncMutexMap

A sync mutex map uses simple read/write mutex primitives from the `sync` package. This results in a
highly performant way of synchronizing reads and writes to the map. This map is one of the fastest
for low-concurrency tests, but takes second or even third place for high-concurrency benchmarks.

```Go
cgm, _ := cmap.NewSyncMutexMap()
defer cgm.Close()
```

#### NewTwoLevelMap

A two-level map implements the map using a top-level lock that guarantees mutual exclusion on adding
or removing keys to the map, and individual locks for each key, guaranteeing mutual exclusion of
tasks attempting to mutate or read the value associated with a given key.

```Go
cgm, _ := cmap.NewTwoLevelMap()
defer cgm.Close()
```

### Storing values in a Congomap

Storing key value pairs in a `Congomap` is done with the Store method.

```Go
cgm.Store("someKey", 42)
cgm.Store("someOtherKey", struct{}{})
cgm.Store("yetAnotherKey", make(chan interface{}))
```

### Loading values from a Congomap

Loading values already stored in a `Congomap` is done with the Load method.

```Go
value, ok := cgm.Load("someKey")
if !ok {
    // key is not in the Congomap
}
```

### Lazy Lookups using LoadStore

Some maps are used as a lazy lookup device. When a key is not already found in the map, the callback
function is invoked with the specified key. If the callback function returns an error, then a nil
value and the error is returned from `LoadStore`. If the callback function returns no error, then
the returned value is stored in the `Congomap` and returned from `LoadStore`.

```Go
// Define a lookup function to be invoked when LoadStore is called for a key not stored in the
// Congomap.
lookup := func(key string) (interface{}, error) {
    return someLenghyComputation(key), nil
}

// Create a Congomap, specifying what the lookup callback function is.
cgm, err := congomap.NewSyncMutexMap(congomap.Lookup(lookup))
if err != nil {
    log.Fatal(err)
}

// You can still use the regular Load and Store functions, which will not invoke the lookup
// function.
cgm.Store("someKey", 42)
value, ok := cgm.Load("someKey")
if !ok {
    // key is not in the Congomap
}

// When you use the LoadStore function, and the key is not in the Congomap, the lookup funciton is
// invoked, and the return value is stored in the Congomap and returned to the program.
value, err := cgm.LoadStore("someKey")
if err != nil {
    // lookup functio returned an error
}
```

## Benchmarks

Part of the purpose of this library is to calculate the relative performance of these approaches to
access to a concurrent map. Here's a sample run on my Mac using Go 1.6.3:

```bash
go test -bench .
PASS
BenchmarkLoadChannelMap-8                        	 1000000	      1644 ns/op
BenchmarkLoadSyncAtomicMap-8                     	 5000000	       326 ns/op
BenchmarkLoadSyncMutexMap-8                      	 5000000	       314 ns/op
BenchmarkLoadTwoLevelMap-8                       	 5000000	       355 ns/op

BenchmarkLoadTTLChannelMap-8                     	 1000000	      1563 ns/op
BenchmarkLoadTTLSyncAtomicMap-8                  	 5000000	       348 ns/op
BenchmarkLoadTTLSyncMutexMap-8                   	 5000000	       322 ns/op
BenchmarkLoadTTLTwoLevelMap-8                    	 5000000	       357 ns/op

BenchmarkLoadStoreChannelMap-8                   	 1000000	      1643 ns/op
BenchmarkLoadStoreSyncAtomicMap-8                	 5000000	       323 ns/op
BenchmarkLoadStoreSyncMutexMap-8                 	 3000000	       351 ns/op
BenchmarkLoadStoreTwoLevelMap-8                  	 5000000	       363 ns/op

BenchmarkLoadStoreTTLChannelMap-8                	 1000000	      1452 ns/op
BenchmarkLoadStoreTTLSyncAtomicMap-8             	 5000000	       388 ns/op
BenchmarkLoadStoreTTLSyncMutexMap-8              	 5000000	       374 ns/op
BenchmarkLoadStoreTTLTwoLevelMap-8               	 5000000	       379 ns/op

BenchmarkHighConcurrencyFastLookupChannelMap-8   	    1000	   1696277 ns/op
BenchmarkHighConcurrencyFastLookupSyncAtomicMap-8	     100	  23849153 ns/op
BenchmarkHighConcurrencyFastLookupSyncMutexMap-8 	       1	1901949630 ns/op
BenchmarkHighConcurrencyFastLookupTwoLevelMap-8  	      10	 859849259 ns/op

BenchmarkHighConcurrencySlowLookupChannelMap-8   	    1000	   1623286 ns/op
BenchmarkHighConcurrencySlowLookupSyncAtomicMap-8	      20	  91569563 ns/op
BenchmarkHighConcurrencySlowLookupSyncMutexMap-8 	     100	  36404125 ns/op
BenchmarkHighConcurrencySlowLookupTwoLevelMap-8  	    5000	   3379887 ns/op

BenchmarkLowConcurrencyFastLookupChannelMap-8    	  100000	     16768 ns/op
BenchmarkLowConcurrencyFastLookupSyncAtomicMap-8 	    5000	    597813 ns/op
BenchmarkLowConcurrencyFastLookupSyncMutexMap-8  	  300000	     23472 ns/op
BenchmarkLowConcurrencyFastLookupTwoLevelMap-8   	  100000	     18847 ns/op

BenchmarkLowConcurrencySlowLookupChannelMap-8    	  100000	     16692 ns/op
BenchmarkLowConcurrencySlowLookupSyncAtomicMap-8 	    5000	    883376 ns/op
BenchmarkLowConcurrencySlowLookupSyncMutexMap-8  	  200000	     18199 ns/op
BenchmarkLowConcurrencySlowLookupTwoLevelMap-8   	  100000	     16219 ns/op
ok  	github.com/karrick/congomap	161.827s
```

Of note is the apparent loser for most of the tests is the `NewChannelMap`. However, those tests are
all low-concurrency tests. The HighConcurrency and SlowLookups benchmarks above are all using 1000
competing tasks all performing LoadStore operations on the same congomap. In these benchmarks, the
`NewChannelMap` out performs all the other types of maps.
