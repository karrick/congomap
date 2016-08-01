package congomap

import (
	"math/rand"
	"sync"
	"testing"
	"time"
)

var states = []string{
	"Alabama",
	"Alaska",
	"Arizona",
	"Arkansas",
	"California",
	"Colorado",
	"Connecticut",
	"Delaware",
	"Florida",
	"Georgia",
	"Hawaii",
	"Idaho",
	"Illinois Indiana",
	"Iowa",
	"Kansas",
	"Kentucky",
	"Louisiana",
	"Maine",
	"Maryland",
	"Massachusetts",
	"Michigan",
	"Minnesota",
	"Mississippi",
	"Missouri",
	"Montana Nebraska",
	"Nevada",
	"New Hampshire",
	"New Jersey",
	"New Mexico",
	"New York",
	"North Carolina",
	"North Dakota",
	"Ohio",
	"Oklahoma",
	"Oregon",
	"Pennsylvania Rhode Island",
	"South Carolina",
	"South Dakota",
	"Tennessee",
	"Texas",
	"Utah",
	"Vermont",
	"Virginia",
	"Washington",
	"West Virginia",
	"Wisconsin",
	"Wyoming",
}

var preventCompilerOptimizingOutBenchmarks interface{}

func randomState() string {
	return states[rand.Intn(len(states))]
}

func preloadCongomap(cgm Congomap) {
	const preloadKeys = 10000

	for i := 0; i < preloadKeys; i++ {
		key := randomState() + randomState() + randomState() + randomState()
		cgm.Store(key, randomState())
	}
}

func parallelLoaders(b *testing.B, cgm Congomap) {
	preloadCongomap(cgm)
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, _ = cgm.Load(randomState())
		}
	})
}

func parallelLoadStorers(b *testing.B, cgm Congomap) {
	preloadCongomap(cgm)
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, _ = cgm.LoadStore(randomState())
		}
	})
}

// Load

func BenchmarkLoadChannelMap(b *testing.B) {
	cgm, _ := NewChannelMap()
	defer cgm.Close()
	parallelLoaders(b, cgm)
}

func BenchmarkLoadSyncAtomicMap(b *testing.B) {
	cgm, _ := NewSyncAtomicMap()
	defer cgm.Close()
	parallelLoaders(b, cgm)
}

func BenchmarkLoadSyncMutexMap(b *testing.B) {
	cgm, _ := NewSyncMutexMap()
	defer cgm.Close()
	parallelLoaders(b, cgm)
}

func BenchmarkLoadTwoLevelMap(b *testing.B) {
	cgm, _ := NewTwoLevelMap()
	defer cgm.Close()
	parallelLoaders(b, cgm)
}

// LoadTTL

func BenchmarkLoadTTLChannelMap(b *testing.B) {
	cgm, _ := NewChannelMap(TTL(time.Second))
	defer cgm.Close()
	parallelLoaders(b, cgm)
}

func BenchmarkLoadTTLSyncAtomicMap(b *testing.B) {
	cgm, _ := NewSyncAtomicMap(TTL(time.Second))
	defer cgm.Close()
	parallelLoaders(b, cgm)
}

func BenchmarkLoadTTLSyncMutexMap(b *testing.B) {
	cgm, _ := NewSyncMutexMap(TTL(time.Second))
	defer cgm.Close()
	parallelLoaders(b, cgm)
}

func BenchmarkLoadTTLTwoLevelMap(b *testing.B) {
	cgm, _ := NewTwoLevelMap(TTL(time.Second))
	defer cgm.Close()
	parallelLoaders(b, cgm)
}

// LoadStore

func BenchmarkLoadStoreChannelMap(b *testing.B) {
	cgm, _ := NewChannelMap()
	defer cgm.Close()
	parallelLoadStorers(b, cgm)
}

func BenchmarkLoadStoreSyncAtomicMap(b *testing.B) {
	cgm, _ := NewSyncAtomicMap()
	defer cgm.Close()
	parallelLoadStorers(b, cgm)
}

func BenchmarkLoadStoreSyncMutexMap(b *testing.B) {
	cgm, _ := NewSyncMutexMap()
	defer cgm.Close()
	parallelLoadStorers(b, cgm)
}

func BenchmarkLoadStoreTwoLevelMap(b *testing.B) {
	cgm, _ := NewTwoLevelMap()
	defer cgm.Close()
	parallelLoadStorers(b, cgm)
}

// LoadStoreTTL

func BenchmarkLoadStoreTTLChannelMap(b *testing.B) {
	cgm, _ := NewChannelMap(TTL(time.Second))
	defer cgm.Close()
	parallelLoadStorers(b, cgm)
}

func BenchmarkLoadStoreTTLSyncAtomicMap(b *testing.B) {
	cgm, _ := NewSyncAtomicMap(TTL(time.Second))
	defer cgm.Close()
	parallelLoadStorers(b, cgm)
}

func BenchmarkLoadStoreTTLSyncMutexMap(b *testing.B) {
	cgm, _ := NewSyncMutexMap(TTL(time.Second))
	defer cgm.Close()
	parallelLoadStorers(b, cgm)
}

func BenchmarkLoadStoreTTLTwoLevelMap(b *testing.B) {
	cgm, _ := NewTwoLevelMap(TTL(time.Second))
	defer cgm.Close()
	parallelLoadStorers(b, cgm)
}

//

func withLoadedCongomap(cgm Congomap, loaders, storers, loadstorers int, callback func()) {
	var stop bool
	var wg sync.WaitGroup

	preloadCongomap(cgm)

	wg.Add(loaders)
	for i := 0; i < loaders; i++ {
		go func() {
			for !stop {
				_, _ = cgm.Load(randomState())
			}
			wg.Done()
		}()
	}

	wg.Add(storers)
	for i := 0; i < storers; i++ {
		go func() {
			for !stop {
				cgm.Store(randomState(), randomState())
			}
			wg.Done()
		}()
	}

	wg.Add(loadstorers)
	for i := 0; i < loadstorers; i++ {
		go func() {
			for !stop {
				_, _ = cgm.LoadStore(randomState())
			}
			wg.Done()
		}()
	}

	callback()
	stop = true
	wg.Wait()
}

// Many LoadStorers

const manyLoadStorers = 1000

func BenchmarkManyLoadStorersChannelMap(b *testing.B) {
	var r interface{}
	cgm, _ := NewChannelMap()
	defer cgm.Close()

	withLoadedCongomap(cgm, 0, 0, manyLoadStorers, func() {
		for i := 0; i < b.N; i++ {
			r, _ = cgm.LoadStore(randomState())
		}
	})

	preventCompilerOptimizingOutBenchmarks = r
}

func BenchmarkManyLoadStorersSyncAtomicMap(b *testing.B) {
	var r interface{}
	cgm, _ := NewSyncAtomicMap()
	defer cgm.Close()

	withLoadedCongomap(cgm, 0, 0, manyLoadStorers, func() {
		for i := 0; i < b.N; i++ {
			r, _ = cgm.LoadStore(randomState())
		}
	})

	preventCompilerOptimizingOutBenchmarks = r
}

func BenchmarkManyLoadStorersSyncMutexMap(b *testing.B) {
	var r interface{}
	cgm, _ := NewSyncMutexMap()
	defer cgm.Close()

	withLoadedCongomap(cgm, 0, 0, manyLoadStorers, func() {
		for i := 0; i < b.N; i++ {
			r, _ = cgm.LoadStore(randomState())
		}
	})

	preventCompilerOptimizingOutBenchmarks = r
}

func BenchmarkManyLoadStorersTwoLevelMap(b *testing.B) {
	var r interface{}
	cgm, _ := NewTwoLevelMap()
	defer cgm.Close()

	withLoadedCongomap(cgm, 0, 0, manyLoadStorers, func() {
		for i := 0; i < b.N; i++ {
			r, _ = cgm.LoadStore(randomState())
		}
	})

	preventCompilerOptimizingOutBenchmarks = r
}
