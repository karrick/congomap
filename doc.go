// The MIT License (MIT)
//
// Copyright (c) 2015 Karrick McDermott
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

/*
Package congomap provides a concurrency-safe Go Map.

This repository serves as a set of examples for making maps that are accessible in concurrent Go
software. The types can be used as a library, each with their own performance characteristics, but
I wrote it to determine which method produced the most readable code, and the most performant code.

Basics

This library exposes the Congomap interface, and a few concrete types that adhere to that
interface. All provided concrete types are available here because they have individual performance
characteristics, where one concrete type may be more appropriate for a desired use case than one of
the other types.

WARNING: To prevent resource leakage, always call the Congomap's Close method after it is no longer
needed.

    cgm, err := congomap.NewTwoLevelMap()
    if err != nil {
        panic(err)
    }
    defer func() { _ = cgm.Close() }()

    // you can store any Go type in a Congomap
    cgm.Store("someKeyString", 42)
    cgm.Store("anotherKey", struct{}{})
    cgm.Store("yetAnotherKey", make(chan interface{}))

    // but when you retrieve it, you are responsible to perform type assertions
    key := "yetAnotherKey"
    value, ok := cgm.Load(key)
    if !ok {
        panic(fmt.Errorf("cannot find %q", key))
    }
    value = value.(chan interface{})

Customizable Features

- Lazy Loading with Lookup callback

All Congomaps support providing a custom Lookup callback function that the Congomap invokes to
lookup the value of a key not yet present in the data store when the LoadStore method is
invoked. This is useful when you want to load a value for a key from the Congomap, but perhaps the
value has yet to be stored. Congomap then invokes the Lookup function with the key string as its
argument, then stores the return value of the Lookup function in the Congomap for future
requests. If the Lookup instead returns an error, no value is stored in the Congomap.

See the example provided in godoc for more information on taking advantage of this feature.

- Expiration Notification with Reaper callback

All Congomaps support providing a custom Reaper callback function that the Congomap invokes when a
value is expired from the data store, either by exceeding its TTL or by being replaced with another
value during a Store operation. This is useful when your program needs to perform some sort of
cleanup on the feature that was in the Congomap.

Note that when the Congomap is closed, if a Reaper callback function is provided, it will be called
repeatedly with each value that was stored in the Congomap.

See the example provided in godoc for more information on taking advantage of this feature.

- Default entry Time-to-Live (TTL)

All Congomaps support providing a default time-to-live for values stored in the Congomap. If *not*
provided, items stored in the Congomap will remain there until expired by being superceded by the
Store operation. If a default TTL *is* provided, then items will expire and must be refetched.

Note that whether or not a custom TTL is provided when creating a Congomap, if the Store method or
customized Lookup callback function ever return a pointer to an ExpringValue object, the default TTL
is ignored and the item will expire when the ExpiringValue's Expiry passes. If the ExpiringValue's
Expiry is the zero time, then this data item will not auto-expire from the data store.

See the example provided in godoc for more information on taking advantage of this feature.

Provided Concrete Congomap Types

- NewChannelMap

A channel map is modeled after the Go way of sharing memory: by communicating over channels. Reads
and writes are serialized by a Go routine processing anonymous functions. While not as fast as the
other methods for low-concurrency loads, this particular map outpaces the competition in
high-concurrency tests.

- NewSyncAtomicMap

A sync atomic map uses the algorithm suggested in the documentation for `sync/atomic`. It is
designed for when a map is read many, many more times than it is written. Performance also depends
on the number of the keys in the map. The more keys in the map, the more expensive Store and
LoadStore will be.

- NewSyncMutexMap

A sync mutex map uses simple read/write mutex primitives from the `sync` package. This results in a
highly performant way of synchronizing reads and writes to the map. This map is one of the fastest
for low-concurrency tests, but takes second or even third place for high-concurrency benchmarks.

- NewTwoLevelMap

A two-level map implements the map using a top-level lock that guarantees mutual exclusion on adding
or removing keys to the map, and individual locks for each key, guaranteeing mutual exclusion of
tasks attempting to mutate or read the value associated with a given key.

Benchmarks

The initial motivation of creating this library was to calculate the relative performance of these
approaches to access to a concurrent map. Here's a sample run on my Mac using Go 1.6.3.

For these benchmarks, each Congomap is pre-loaded with 2500 key-value pairs, and each competing go
routine must make 1000 mutations to the data store.

High concurrency benchmarks just over 1000 competing go routines all making changes to a single
Congomap object, whereas low concurrency refers to just over 10 go routines all making 1000 changes
to a single Congomap object.

Fast lookups means the Lookup function immediately responds. Slow lookups means the Lookup function
slept 100 ± 50 ms before returning.

    go test -bench=Concurrency
    PASS

    BenchmarkHighConcurrencyFastLookupChannelMap-8 1000 1719902 ns/op
    BenchmarkHighConcurrencyFastLookupSyncAtomicMap-8 100 22276241 ns/op
    BenchmarkHighConcurrencyFastLookupSyncMutexMap-8 1 1632581613 ns/op
    BenchmarkHighConcurrencyFastLookupTwoLevelMap-8 3000 507488 ns/op

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


*/
package congomap
