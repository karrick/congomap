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
    cgm, err := cmap.NewChannelMap()
    if err != nil {
        panic(err)
    }
    defer cgm.Close()
```

#### NewSyncAtomicMap

A sync atomic map uses the algorithm suggested in the documentation for `sync/atomic`. It is
designed for when a map is read many, many more times than it is written. Performance also depends
on the number of the keys in the map. The more keys in the map, the more expensive Store and
LoadStore will be.

```Go
    cgm,_ := congomap.NewSyncAtomicMap()
    if err != nil {
        panic(err)
    }
    defer cgm.Close()
```

#### NewSyncMutexMap

A sync mutex map uses simple read/write mutex primitives from the `sync` package. This results in a
highly performant way of synchronizing reads and writes to the map. This map is one of the fastest
for low-concurrency tests, but takes second or even third place for high-concurrency benchmarks.

```Go
    cgm, err := cmap.NewSyncMutexMap()
    if err != nil {
        panic(err)
    }
    defer cgm.Close()
```

#### NewTwoLevelMap

A two-level map implements the map using a top-level lock that guarantees mutual exclusion on adding
or removing keys to the map, and individual locks for each key, guaranteeing mutual exclusion of
tasks attempting to mutate or read the value associated with a given key.

```Go
    cgm, err := cmap.NewTwoLevelMap()
    if err != nil {
        panic(err)
    }
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
    cgm, err := congomap.NewTwoLevelMap(congomap.Lookup(lookup))
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
        // lookup function returned an error
    }
```

## Benchmarks

Part of the purpose of this library is to calculate the relative performance of these approaches to
access to a concurrent map. Here's a sample run on my Mac using Go 1.6.3:

```bash
go test -bench .
PASS
BenchmarkLoadChannelMap-8                                1000000          1698 ns/op
BenchmarkLoadSyncAtomicMap-8                             5000000           331 ns/op
BenchmarkLoadSyncMutexMap-8                              5000000           313 ns/op
BenchmarkLoadTwoLevelMap-8                               5000000           357 ns/op
BenchmarkLoadTwoLevelPrimeMap-8                          5000000           355 ns/op

BenchmarkLoadTTLChannelMap-8                             1000000          1588 ns/op
BenchmarkLoadTTLSyncAtomicMap-8                          5000000           352 ns/op
BenchmarkLoadTTLSyncMutexMap-8                           5000000           316 ns/op
BenchmarkLoadTTLTwoLevelMap-8                            5000000           354 ns/op
BenchmarkLoadTTLTwoLevelPrimeMap-8                       5000000           357 ns/op

BenchmarkLoadStoreChannelMap-8                           1000000          1632 ns/op
BenchmarkLoadStoreSyncAtomicMap-8                        5000000           330 ns/op
BenchmarkLoadStoreSyncMutexMap-8                         5000000           352 ns/op
BenchmarkLoadStoreTwoLevelMap-8                          5000000           359 ns/op
BenchmarkLoadStoreTwoLevelPrimeMap-8                     5000000           357 ns/op

BenchmarkLoadStoreTTLChannelMap-8                        1000000          1492 ns/op
BenchmarkLoadStoreTTLSyncAtomicMap-8                     5000000           379 ns/op
BenchmarkLoadStoreTTLSyncMutexMap-8                      5000000           376 ns/op
BenchmarkLoadStoreTTLTwoLevelMap-8                       5000000           359 ns/op
BenchmarkLoadStoreTTLTwoLevelPrimeMap-8                  5000000           378 ns/op

BenchmarkHighConcurrencyFastLookupChannelMap-8              1000       1719902 ns/op
BenchmarkHighConcurrencyFastLookupSyncAtomicMap-8            100      22276241 ns/op
BenchmarkHighConcurrencyFastLookupSyncMutexMap-8               1	1632581613 ns/op
BenchmarkHighConcurrencyFastLookupTwoLevelMap-8             3000        507488 ns/op

BenchmarkHighConcurrencySlowLookupChannelMap-8              1000       1625607 ns/op
BenchmarkHighConcurrencySlowLookupSyncAtomicMap-8             30      60743763 ns/op
BenchmarkHighConcurrencySlowLookupSyncMutexMap-8             100     202947478 ns/op
BenchmarkHighConcurrencySlowLookupTwoLevelMap-8             2000        541066 ns/op

BenchmarkLowConcurrencyFastLookupChannelMap-8             100000         16790 ns/op
BenchmarkLowConcurrencyFastLookupSyncAtomicMap-8            3000        383506 ns/op
BenchmarkLowConcurrencyFastLookupSyncMutexMap-8            30000         35809 ns/op
BenchmarkLowConcurrencyFastLookupTwoLevelMap-8            300000          5335 ns/op

BenchmarkLowConcurrencySlowLookupChannelMap-8             100000         17335 ns/op
BenchmarkLowConcurrencySlowLookupSyncAtomicMap-8            3000        819874 ns/op
BenchmarkLowConcurrencySlowLookupSyncMutexMap-8            30000         33580 ns/op
BenchmarkLowConcurrencySlowLookupTwoLevelMap-8            300000          5229 ns/op

ok      github.com/karrick/congomap	187.273s
```
