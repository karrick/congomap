# Congomap

Concurrent Go Map

This repository serves as a set of examples for making maps that are
accessible in concurrent Go software. The types _can_ be used as a
library, each with their own performance characteristics, but I wrote
it to determine which method produced the most readable code, and the
most performant code.

## Examples

This library exposes the `Congomap` interface, and three concrete
types that adhere to that interface. All three concrete types are
available here because they have individual performance
characteristics, where one concrete type may be more appropriate for a
desired used that one of the other types.

### Creating a Congomap

Creating a `Congomap` is done by calling the instantiation function of
the desired concrete type.

NOTE: To prevent resource leakage, always call the `Halt` method on a
`Congomap` after it is no longer needed.

#### NewSyncMutexMap

A sync mutex map uses simple read/write mutex primitives from the
`sync` package. This results in a highly performant way of
synchronizing reads and writes to the map. This is about seven times
the performance of the `NewChannelMap` method and is the fastest of
those tested in this library.

```Go
cgm, _ := cmap.NewSyncMutexMap()
defer cgm.Halt()
```

#### NewSyncAtomicMap

A sync atomic map uses the algorithm suggested in the documentation
for `sync/atomic`. It is designed for when a map is read many, many
more times than it is written. Performance also depends on the number
of the keys in the map.

```Go
cgm,_ := congomap.NewSyncAtomicMap()
defer cgm.Halt()
```

#### NewChannelMap

A channel map is modeled after the Go way of sharing memory: by
communicating over channels. Reads and writes are serialized by a Go
routine processing anonymous functions. Not as fast as the other
methods.

```Go
cgm, _ := cmap.NewChannelMap()
defer cgm.Halt()
```

### Storing values in a Congomap

Storing key value pairs in a `Congomap` is done with the Store method.

```Go
cgm.Store("someKey", 42)
cgm.Store("someOtherKey", struct{}{})
cgm.Store("yetAnotherKey", make(chan interface{}))
```

### Loading values from a Congomap

Loading values already stored in a `Congomap` is done with the Load
method.

```Go
value, ok := cgm.Load("someKey")
if !ok {
    // key is not in the Congomap
}
```

### Lazy Lookups using LoadStore

Some maps are used as a lazy lookup device. When a key is not already
found in the map, the callback function is invoked with the specified
key. If the callback function returns an error, then a nil value and
the error is returned from `LoadStore`. If the callback function
returns no error, then the returned value is stored in the `Congomap`
and returned from `LoadStore`.

```Go
// Define a lookup function to be invoked when LoadStore is called
// for a key not stored in the Congomap.
lookup := func(key string) (interface{}, error) {
    return someLenghyComputation(key), nil
}

// Create a Congomap, specifying what the lookup callback function
// is.
cgm, err := congomap.NewSyncMutexMap(congomap.Lookup(lookup))
if err != nil {
    log.Fatal(err)
}

// You can still use the regular Load and Store functions, which will
// not involve the lookup function.
cgm.Store("someKey", 42)
value, ok := cgm.Load("someKey")
if !ok {
    // key is not in the Congomap
}

// When you use the LoadStore function, and the key is not in the
// Congomap, the lookup funciton is invoked, and the return value is
// stored in the Congomap and returned to the program.
value, err := cgm.LoadStore("someKey")
if err != nil {
    // lookup functio returned an error
}
```

## Benchmarks

Part of the purpose of this library is to calculate the relative
performance of these approaches to access to a concurrent map. Here's
a sample run on my Mac using Go 1.4:

```bash
go test -bench .
PASS
BenchmarkChannelMapLoad  1000000          1446 ns/op
BenchmarkChannelMapLoadStore     1000000          1610 ns/op
BenchmarkSyncAtomicMapLoad	10000000           231 ns/op
BenchmarkSyncAtomicMapLoadStore  5000000           227 ns/op
BenchmarkSyncMutexMapLoad	10000000           199 ns/op
BenchmarkSyncMutexMapLoadStore   5000000           219 ns/op
BenchmarkSyncMutexMapManyLoadersLoaderPerspective    2000000           690 ns/op
BenchmarkSyncMutexMapManyLoadersLoadStorerPerspective    1000000          9182 ns/op
ok      gitli.corp.linkedin.com/sre-infra/cmap.git	22.076s
```
