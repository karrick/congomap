package congomap_test

import (
	"math/rand"
	"sync"
	"testing"
	"time"

	"github.com/karrick/congomap"
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

func preloadCongomap(cgm congomap.Congomap) {
	for _, k1 := range states {
		for _, k2 := range states {
			cgm.Store(k1+"-"+k2, randomState())
		}
	}
}

func parallelLoaders(b *testing.B, cgm congomap.Congomap) {
	defer cgm.Close()
	preloadCongomap(cgm)
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			preventCompilerOptimizingOutBenchmarks, _ = cgm.Load(randomKey())
		}
	})
}

func parallelLoadStorers(b *testing.B, cgm congomap.Congomap) {
	defer cgm.Close()
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
	cgm, _ := congomap.NewChannelMap()
	parallelLoaders(b, cgm)
}

func BenchmarkLoadSyncAtomicMap(b *testing.B) {
	cgm, _ := congomap.NewSyncAtomicMap()
	parallelLoaders(b, cgm)
}

func BenchmarkLoadSyncMutexMap(b *testing.B) {
	cgm, _ := congomap.NewSyncMutexMap()
	parallelLoaders(b, cgm)
}

func BenchmarkLoadTwoLevelMap(b *testing.B) {
	cgm, _ := congomap.NewTwoLevelMap()
	parallelLoaders(b, cgm)
}

// LoadTTL

func BenchmarkLoadTTLChannelMap(b *testing.B) {
	cgm, _ := congomap.NewChannelMap(congomap.TTL(time.Second))
	parallelLoaders(b, cgm)
}

func BenchmarkLoadTTLSyncAtomicMap(b *testing.B) {
	cgm, _ := congomap.NewSyncAtomicMap(congomap.TTL(time.Second))
	parallelLoaders(b, cgm)
}

func BenchmarkLoadTTLSyncMutexMap(b *testing.B) {
	cgm, _ := congomap.NewSyncMutexMap(congomap.TTL(time.Second))
	parallelLoaders(b, cgm)
}

func BenchmarkLoadTTLTwoLevelMap(b *testing.B) {
	cgm, _ := congomap.NewTwoLevelMap(congomap.TTL(time.Second))
	parallelLoaders(b, cgm)
}

// LoadStore

func BenchmarkLoadStoreChannelMap(b *testing.B) {
	cgm, _ := congomap.NewChannelMap()
	parallelLoadStorers(b, cgm)
}

func BenchmarkLoadStoreSyncAtomicMap(b *testing.B) {
	cgm, _ := congomap.NewSyncAtomicMap()
	parallelLoadStorers(b, cgm)
}

func BenchmarkLoadStoreSyncMutexMap(b *testing.B) {
	cgm, _ := congomap.NewSyncMutexMap()
	parallelLoadStorers(b, cgm)
}

func BenchmarkLoadStoreTwoLevelMap(b *testing.B) {
	cgm, _ := congomap.NewTwoLevelMap()
	parallelLoadStorers(b, cgm)
}

// LoadStoreTTL

func BenchmarkLoadStoreTTLChannelMap(b *testing.B) {
	cgm, _ := congomap.NewChannelMap(congomap.TTL(time.Second))
	parallelLoadStorers(b, cgm)
}

func BenchmarkLoadStoreTTLSyncAtomicMap(b *testing.B) {
	cgm, _ := congomap.NewSyncAtomicMap(congomap.TTL(time.Second))
	parallelLoadStorers(b, cgm)
}

func BenchmarkLoadStoreTTLSyncMutexMap(b *testing.B) {
	cgm, _ := congomap.NewSyncMutexMap(congomap.TTL(time.Second))
	parallelLoadStorers(b, cgm)
}

func BenchmarkLoadStoreTTLTwoLevelMap(b *testing.B) {
	cgm, _ := congomap.NewTwoLevelMap(congomap.TTL(time.Second))
	parallelLoadStorers(b, cgm)
}

// benchmarks

func benchmark(b *testing.B, cgm congomap.Congomap, loaderCount, storerCount, loadStorerCount int) {
	defer cgm.Close()

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

func randomSlowLookup(_ string) (interface{}, error) {
	delay := 25*time.Millisecond + time.Duration(rand.Intn(50))*time.Millisecond
	time.Sleep(delay)
	return 42, nil
}

// High Concurrency

func BenchmarkHighConcurrencyFastLookupChannelMap(b *testing.B) {
	cgm, _ := congomap.NewChannelMap(congomap.TTL(time.Minute))
	benchmark(b, cgm, 1, 1, 1000)
}

func BenchmarkHighConcurrencyFastLookupSyncAtomicMap(b *testing.B) {
	cgm, _ := congomap.NewSyncAtomicMap(congomap.TTL(time.Minute))
	benchmark(b, cgm, 1, 1, 1000)
}

func BenchmarkHighConcurrencyFastLookupSyncMutexMap(b *testing.B) {
	cgm, _ := congomap.NewSyncMutexMap(congomap.TTL(time.Minute))
	benchmark(b, cgm, 1, 1, 1000)
}

func BenchmarkHighConcurrencyFastLookupTwoLevelMap(b *testing.B) {
	cgm, _ := congomap.NewTwoLevelMap(congomap.TTL(time.Minute))
	benchmark(b, cgm, 1, 1, 1000)
}

// High Read Concurrency

func BenchmarkHighReadConcurrencyFastLookupChannelMap(b *testing.B) {
	cgm, _ := congomap.NewChannelMap(congomap.TTL(time.Minute))
	benchmark(b, cgm, 1000, 0, 0)
}

func BenchmarkHighReadConcurrencyFastLookupSyncAtomicMap(b *testing.B) {
	cgm, _ := congomap.NewSyncAtomicMap(congomap.TTL(time.Minute))
	benchmark(b, cgm, 1000, 0, 0)
}

func BenchmarkHighReadConcurrencyFastLookupSyncMutexMap(b *testing.B) {
	cgm, _ := congomap.NewSyncMutexMap(congomap.TTL(time.Minute))
	benchmark(b, cgm, 1000, 0, 0)
}

func BenchmarkHighReadConcurrencyFastLookupTwoLevelMap(b *testing.B) {
	cgm, _ := congomap.NewTwoLevelMap(congomap.TTL(time.Minute))
	benchmark(b, cgm, 1000, 0, 0)
}

// lookup takes random time

func BenchmarkHighConcurrencySlowLookupChannelMap(b *testing.B) {
	cgm, _ := congomap.NewChannelMap(congomap.Lookup(randomSlowLookup))
	benchmark(b, cgm, 1, 1, 1000)
}

func BenchmarkHighConcurrencySlowLookupSyncAtomicMap(b *testing.B) {
	cgm, _ := congomap.NewSyncAtomicMap(congomap.Lookup(randomSlowLookup))
	benchmark(b, cgm, 1, 1, 1000)
}

func BenchmarkHighConcurrencySlowLookupSyncMutexMap(b *testing.B) {
	cgm, _ := congomap.NewSyncMutexMap(congomap.Lookup(randomSlowLookup))
	benchmark(b, cgm, 1, 1, 1000)
}

func BenchmarkHighConcurrencySlowLookupTwoLevelMap(b *testing.B) {
	cgm, _ := congomap.NewTwoLevelMap(congomap.Lookup(randomSlowLookup))
	benchmark(b, cgm, 1, 1, 1000)
}

// Low Concurrency

func BenchmarkLowConcurrencyFastLookupChannelMap(b *testing.B) {
	cgm, _ := congomap.NewChannelMap(congomap.TTL(time.Minute))
	benchmark(b, cgm, 1, 1, 10)
}

func BenchmarkLowConcurrencyFastLookupSyncAtomicMap(b *testing.B) {
	cgm, _ := congomap.NewSyncAtomicMap(congomap.TTL(time.Minute))
	benchmark(b, cgm, 1, 1, 10)
}

func BenchmarkLowConcurrencyFastLookupSyncMutexMap(b *testing.B) {
	cgm, _ := congomap.NewSyncMutexMap(congomap.TTL(time.Minute))
	benchmark(b, cgm, 1, 1, 10)
}

func BenchmarkLowConcurrencyFastLookupTwoLevelMap(b *testing.B) {
	cgm, _ := congomap.NewTwoLevelMap(congomap.TTL(time.Minute))
	benchmark(b, cgm, 1, 1, 10)
}

// lookup takes random time

func BenchmarkLowConcurrencySlowLookupChannelMap(b *testing.B) {
	cgm, _ := congomap.NewChannelMap(congomap.Lookup(randomSlowLookup))
	benchmark(b, cgm, 1, 1, 10)
}

func BenchmarkLowConcurrencySlowLookupSyncAtomicMap(b *testing.B) {
	cgm, _ := congomap.NewSyncAtomicMap(congomap.Lookup(randomSlowLookup))
	benchmark(b, cgm, 1, 1, 10)
}

func BenchmarkLowConcurrencySlowLookupSyncMutexMap(b *testing.B) {
	cgm, _ := congomap.NewSyncMutexMap(congomap.Lookup(randomSlowLookup))
	benchmark(b, cgm, 1, 1, 10)
}

func BenchmarkLowConcurrencySlowLookupTwoLevelMap(b *testing.B) {
	cgm, _ := congomap.NewTwoLevelMap(congomap.Lookup(randomSlowLookup))
	benchmark(b, cgm, 1, 1, 10)
}

// High Contention

func benchmarkHighContention(cgm congomap.Congomap) {
	defer cgm.Close()

	const tasks = 1000
	keys := []string{"just", "a", "few", "keys", "to", "force", "lock", "contention"}

	var wg sync.WaitGroup
	wg.Add(tasks)

	for i := 0; i < tasks; i++ {
		go func() {
			const iterations = 1000

			for j := 0; j < iterations; j++ {
				if j%4 == 0 {
					cgm.Delete(keys[rand.Intn(len(keys))])
				} else {
					_, _ = cgm.LoadStore(keys[rand.Intn(len(keys))])
				}
			}

			wg.Done()
		}()
	}

	wg.Wait()
}

func BenchmarkHighContentionChannelMap(b *testing.B) {
	cgm, _ := congomap.NewChannelMap(congomap.Lookup(randomFailOnLookup), congomap.TTL(time.Second))
	benchmarkHighContention(cgm)
}

func BenchmarkHighContentionSyncAtomicMap(b *testing.B) {
	cgm, _ := congomap.NewSyncAtomicMap(congomap.Lookup(randomFailOnLookup), congomap.TTL(time.Second))
	benchmarkHighContention(cgm)
}

func BenchmarkHighContentionSyncMutexMap(b *testing.B) {
	cgm, _ := congomap.NewSyncMutexMap(congomap.Lookup(randomFailOnLookup), congomap.TTL(time.Second))
	benchmarkHighContention(cgm)
}

func BenchmarkHighContentionTwoLevelMap(b *testing.B) {
	cgm, _ := congomap.NewTwoLevelMap(congomap.Lookup(randomFailOnLookup), congomap.TTL(time.Second))
	benchmarkHighContention(cgm)
}
