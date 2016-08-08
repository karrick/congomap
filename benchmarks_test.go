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
	cgm, _ := NewChannelMap(nil)
	defer cgm.Close()
	parallelLoaders(b, cgm)
}

func BenchmarkLoadSyncAtomicMap(b *testing.B) {
	cgm, _ := NewSyncAtomicMap(nil)
	defer cgm.Close()
	parallelLoaders(b, cgm)
}

func BenchmarkLoadSyncMutexMap(b *testing.B) {
	cgm, _ := NewSyncMutexMap(nil)
	defer cgm.Close()
	parallelLoaders(b, cgm)
}

func BenchmarkLoadTwoLevelMap(b *testing.B) {
	cgm, _ := NewTwoLevelMap(nil)
	defer cgm.Close()
	parallelLoaders(b, cgm)
}

// LoadTTL

func BenchmarkLoadTTLChannelMap(b *testing.B) {
	cgm, _ := NewChannelMap(&Config{TTL: time.Second})
	defer cgm.Close()
	parallelLoaders(b, cgm)
}

func BenchmarkLoadTTLSyncAtomicMap(b *testing.B) {
	cgm, _ := NewSyncAtomicMap(&Config{TTL: time.Second})
	defer cgm.Close()
	parallelLoaders(b, cgm)
}

func BenchmarkLoadTTLSyncMutexMap(b *testing.B) {
	cgm, _ := NewSyncMutexMap(&Config{TTL: time.Second})
	defer cgm.Close()
	parallelLoaders(b, cgm)
}

func BenchmarkLoadTTLTwoLevelMap(b *testing.B) {
	cgm, _ := NewTwoLevelMap(&Config{TTL: time.Second})
	defer cgm.Close()
	parallelLoaders(b, cgm)
}

// LoadStore

func BenchmarkLoadStoreChannelMap(b *testing.B) {
	cgm, _ := NewChannelMap(nil)
	defer cgm.Close()
	parallelLoadStorers(b, cgm)
}

func BenchmarkLoadStoreSyncAtomicMap(b *testing.B) {
	cgm, _ := NewSyncAtomicMap(nil)
	defer cgm.Close()
	parallelLoadStorers(b, cgm)
}

func BenchmarkLoadStoreSyncMutexMap(b *testing.B) {
	cgm, _ := NewSyncMutexMap(nil)
	defer cgm.Close()
	parallelLoadStorers(b, cgm)
}

func BenchmarkLoadStoreTwoLevelMap(b *testing.B) {
	cgm, _ := NewTwoLevelMap(nil)
	defer cgm.Close()
	parallelLoadStorers(b, cgm)
}

// LoadStoreTTL

func BenchmarkLoadStoreTTLChannelMap(b *testing.B) {
	cgm, _ := NewChannelMap(&Config{TTL: time.Second})
	defer cgm.Close()
	parallelLoadStorers(b, cgm)
}

func BenchmarkLoadStoreTTLSyncAtomicMap(b *testing.B) {
	cgm, _ := NewSyncAtomicMap(&Config{TTL: time.Second})
	defer cgm.Close()
	parallelLoadStorers(b, cgm)
}

func BenchmarkLoadStoreTTLSyncMutexMap(b *testing.B) {
	cgm, _ := NewSyncMutexMap(&Config{TTL: time.Second})
	defer cgm.Close()
	parallelLoadStorers(b, cgm)
}

func BenchmarkLoadStoreTTLTwoLevelMap(b *testing.B) {
	cgm, _ := NewTwoLevelMap(&Config{TTL: time.Second})
	defer cgm.Close()
	parallelLoadStorers(b, cgm)
}

// benchmarks

func benchmark(b *testing.B, cgm Congomap, loaderCount, storerCount, loadStorerCount int) {
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
	cgm, _ := NewChannelMap(&Config{TTL: time.Minute})
	defer cgm.Close()
	benchmark(b, cgm, 1, 1, 1000)
}

func BenchmarkHighConcurrencyFastLookupSyncAtomicMap(b *testing.B) {
	cgm, _ := NewSyncAtomicMap(&Config{TTL: time.Minute})
	defer cgm.Close()
	benchmark(b, cgm, 1, 1, 1000)
}

func BenchmarkHighConcurrencyFastLookupSyncMutexMap(b *testing.B) {
	cgm, _ := NewSyncMutexMap(&Config{TTL: time.Minute})
	defer cgm.Close()
	benchmark(b, cgm, 1, 1, 1000)
}

func BenchmarkHighConcurrencyFastLookupTwoLevelMap(b *testing.B) {
	cgm, _ := NewTwoLevelMap(&Config{TTL: time.Minute})
	defer cgm.Close()
	benchmark(b, cgm, 1, 1, 1000)
}

// lookup takes random time

func BenchmarkHighConcurrencySlowLookupChannelMap(b *testing.B) {
	cgm, _ := NewChannelMap(&Config{Lookup: randomSlowLookup})
	defer cgm.Close()
	benchmark(b, cgm, 1, 1, 1000)
}

func BenchmarkHighConcurrencySlowLookupSyncAtomicMap(b *testing.B) {
	cgm, _ := NewSyncAtomicMap(&Config{Lookup: randomSlowLookup})
	defer cgm.Close()
	benchmark(b, cgm, 1, 1, 1000)
}

func BenchmarkHighConcurrencySlowLookupSyncMutexMap(b *testing.B) {
	cgm, _ := NewSyncMutexMap(&Config{Lookup: randomSlowLookup})
	defer cgm.Close()
	benchmark(b, cgm, 1, 1, 1000)
}

func BenchmarkHighConcurrencySlowLookupTwoLevelMap(b *testing.B) {
	cgm, _ := NewTwoLevelMap(&Config{Lookup: randomSlowLookup})
	defer cgm.Close()
	benchmark(b, cgm, 1, 1, 1000)
}

// Low Concurrency

func BenchmarkLowConcurrencyFastLookupChannelMap(b *testing.B) {
	cgm, _ := NewChannelMap(&Config{TTL: time.Minute})
	defer cgm.Close()
	benchmark(b, cgm, 1, 1, 10)
}

func BenchmarkLowConcurrencyFastLookupSyncAtomicMap(b *testing.B) {
	cgm, _ := NewSyncAtomicMap(&Config{TTL: time.Minute})
	defer cgm.Close()
	benchmark(b, cgm, 1, 1, 10)
}

func BenchmarkLowConcurrencyFastLookupSyncMutexMap(b *testing.B) {
	cgm, _ := NewSyncMutexMap(&Config{TTL: time.Minute})
	defer cgm.Close()
	benchmark(b, cgm, 1, 1, 10)
}

func BenchmarkLowConcurrencyFastLookupTwoLevelMap(b *testing.B) {
	cgm, _ := NewTwoLevelMap(&Config{TTL: time.Minute})
	defer cgm.Close()
	benchmark(b, cgm, 1, 1, 10)
}

// lookup takes random time

func BenchmarkLowConcurrencySlowLookupChannelMap(b *testing.B) {
	cgm, _ := NewChannelMap(&Config{Lookup: randomSlowLookup})
	defer cgm.Close()
	benchmark(b, cgm, 1, 1, 10)
}

func BenchmarkLowConcurrencySlowLookupSyncAtomicMap(b *testing.B) {
	cgm, _ := NewSyncAtomicMap(&Config{Lookup: randomSlowLookup})
	defer cgm.Close()
	benchmark(b, cgm, 1, 1, 10)
}

func BenchmarkLowConcurrencySlowLookupSyncMutexMap(b *testing.B) {
	cgm, _ := NewSyncMutexMap(&Config{Lookup: randomSlowLookup})
	defer cgm.Close()
	benchmark(b, cgm, 1, 1, 10)
}

func BenchmarkLowConcurrencySlowLookupTwoLevelMap(b *testing.B) {
	cgm, _ := NewTwoLevelMap(&Config{Lookup: randomSlowLookup})
	defer cgm.Close()
	benchmark(b, cgm, 1, 1, 10)
}

//

func benchmarkHighContention(cgm Congomap) {
	const tasks = 1000
	keys := []string{"just", "a", "few", "keys", "to", "force", "lock", "contention"}

	var wg sync.WaitGroup
	wg.Add(tasks)

	for i := 0; i < tasks; i++ {
		go func(cgm Congomap, wg *sync.WaitGroup, keys []string) {
			const iterations = 1000

			for j := 0; j < iterations; j++ {
				key := keys[rand.Intn(len(keys))]
				if j%4 == 0 {
					cgm.Delete(key)
				} else {
					_, _ = cgm.LoadStore(key)
				}
			}
			wg.Done()
		}(cgm, &wg, keys)
	}

	wg.Wait()
}

func BenchmarkHighContentionChannelMap(b *testing.B) {
	cgm, _ := NewChannelMap(&Config{Lookup: randomFailOnLookup, TTL: time.Second})
	benchmarkHighContention(cgm)
	defer cgm.Close()
}

func BenchmarkHighContentionSyncAtomicMap(b *testing.B) {
	cgm, _ := NewSyncAtomicMap(&Config{Lookup: randomFailOnLookup, TTL: time.Second})
	benchmarkHighContention(cgm)
	defer cgm.Close()
}

func BenchmarkHighContentionSyncMutexMap(b *testing.B) {
	cgm, _ := NewSyncMutexMap(&Config{Lookup: randomFailOnLookup, TTL: time.Second})
	benchmarkHighContention(cgm)
	defer cgm.Close()
}

func BenchmarkHighContentionTwoLevelMap(b *testing.B) {
	cgm, _ := NewTwoLevelMap(&Config{Lookup: randomFailOnLookup, TTL: time.Second})
	benchmarkHighContention(cgm)
	defer cgm.Close()
}
