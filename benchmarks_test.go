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

func randomKey() string {
	return randomState() + "-" + randomState()
}

func preloadCongomap(cgm Congomap) {
	for _, k1 := range states {
		for _, k2 := range states {
			cgm.Store(k1+"-"+k2, randomState())
		}
	}
}

func parallelLoaders(b *testing.B, cgm Congomap) {
	preloadCongomap(cgm)
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			preventCompilerOptimizingOutBenchmarks, _ = cgm.Load(randomKey())
		}
	})
}

func parallelLoadStorers(b *testing.B, cgm Congomap) {
	preloadCongomap(cgm)
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			preventCompilerOptimizingOutBenchmarks, _ = cgm.LoadStore(randomKey())
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

// ManyLoadStorers

const (
	loaderCount     = 1
	storerCount     = 1
	loadStorerCount = 1000
)

func benchmarkHighConcurrency(b *testing.B, cgm Congomap) {
	preloadCongomap(cgm)

	var stop bool
	var wg sync.WaitGroup

	wg.Add(loaderCount)
	for i := 0; i < loaderCount; i++ {
		go func() {
			var r interface{}
			_ = r
			for !stop {
				r, _ = cgm.Load(randomKey())
			}
			wg.Done()
		}()
	}

	wg.Add(storerCount)
	for i := 0; i < storerCount; i++ {
		go func() {
			for !stop {
				cgm.Store(randomKey(), randomState())
			}
			wg.Done()
		}()
	}

	wg.Add(loadStorerCount)
	for i := 0; i < loadStorerCount; i++ {
		go func() {
			var r interface{}
			_ = r
			for !stop {
				r, _ = cgm.LoadStore(randomKey())
			}
			wg.Done()
		}()
	}

	b.ResetTimer()

	var r interface{}
	for i := 0; i < b.N; i++ {
		r, _ = cgm.LoadStore(randomKey())
	}

	stop = true
	wg.Wait()

	preventCompilerOptimizingOutBenchmarks = r
}

func BenchmarkHighConcurrencyChannelMap(b *testing.B) {
	cgm, _ := NewChannelMap(TTL(time.Minute))
	defer cgm.Close()
	benchmarkHighConcurrency(b, cgm)
}

func BenchmarkHighConcurrencySyncAtomicMap(b *testing.B) {
	cgm, _ := NewSyncAtomicMap(TTL(time.Minute))
	defer cgm.Close()
	benchmarkHighConcurrency(b, cgm)
}

func BenchmarkHighConcurrencySyncMutexMap(b *testing.B) {
	cgm, _ := NewSyncMutexMap(TTL(time.Minute))
	defer cgm.Close()
	benchmarkHighConcurrency(b, cgm)
}

func BenchmarkHighConcurrencyTwoLevelMap(b *testing.B) {
	cgm, _ := NewTwoLevelMap(TTL(time.Minute))
	defer cgm.Close()
	benchmarkHighConcurrency(b, cgm)
}

// lookup takes random time

func randomSlowLookup(_ string) (interface{}, error) {
	delay := 25*time.Millisecond + time.Duration(rand.Intn(50))*time.Millisecond
	time.Sleep(delay)
	return 42, nil
}

func BenchmarkSlowLookupsChannelMap(b *testing.B) {
	cgm, _ := NewChannelMap(Lookup(randomSlowLookup))
	defer cgm.Close()
	benchmarkHighConcurrency(b, cgm)
}

func BenchmarkSlowLookupsSyncAtomicMap(b *testing.B) {
	cgm, _ := NewSyncAtomicMap(Lookup(randomSlowLookup))
	defer cgm.Close()
	benchmarkHighConcurrency(b, cgm)
}

func BenchmarkSlowLookupsSyncMutexMap(b *testing.B) {
	cgm, _ := NewSyncMutexMap(Lookup(randomSlowLookup))
	defer cgm.Close()
	benchmarkHighConcurrency(b, cgm)
}

func BenchmarkSlowLookupsTwoLevelMap(b *testing.B) {
	cgm, _ := NewTwoLevelMap(Lookup(randomSlowLookup))
	defer cgm.Close()
	benchmarkHighConcurrency(b, cgm)
}
