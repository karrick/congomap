package congomap_test

import (
	"math/rand"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	congomap "github.com/karrick/congomap/v2"
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
	defer func() { _ = cgm.Close() }()
	preloadCongomap(cgm)
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, _ = cgm.Load(randomKey())
		}
	})
}

func parallelLoadStorers(b *testing.B, cgm congomap.Congomap) {
	defer func() { _ = cgm.Close() }()
	preloadCongomap(cgm)
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, _ = cgm.LoadStore(randomKey())
		}
	})
}

// Load

func BenchmarkLoadChannelMap(b *testing.B) {
	cgm, err := congomap.NewChannelMap()
	if err != nil {
		b.Fatal(err)
	}
	parallelLoaders(b, cgm)
}

func BenchmarkLoadSyncAtomicMap(b *testing.B) {
	cgm, err := congomap.NewSyncAtomicMap()
	if err != nil {
		b.Fatal(err)
	}
	parallelLoaders(b, cgm)
}

func BenchmarkLoadSyncMutexMap(b *testing.B) {
	cgm, err := congomap.NewSyncMutexMap()
	if err != nil {
		b.Fatal(err)
	}
	parallelLoaders(b, cgm)
}

func BenchmarkLoadTwoLevelMap(b *testing.B) {
	cgm, err := congomap.NewTwoLevelMap()
	if err != nil {
		b.Fatal(err)
	}
	parallelLoaders(b, cgm)
}

// LoadTTL

func BenchmarkLoadTTLChannelMap(b *testing.B) {
	cgm, err := congomap.NewChannelMap(congomap.TTL(time.Second))
	if err != nil {
		b.Fatal(err)
	}
	parallelLoaders(b, cgm)
}

func BenchmarkLoadTTLSyncAtomicMap(b *testing.B) {
	cgm, err := congomap.NewSyncAtomicMap(congomap.TTL(time.Second))
	if err != nil {
		b.Fatal(err)
	}
	parallelLoaders(b, cgm)
}

func BenchmarkLoadTTLSyncMutexMap(b *testing.B) {
	cgm, err := congomap.NewSyncMutexMap(congomap.TTL(time.Second))
	if err != nil {
		b.Fatal(err)
	}
	parallelLoaders(b, cgm)
}

func BenchmarkLoadTTLTwoLevelMap(b *testing.B) {
	cgm, err := congomap.NewTwoLevelMap(congomap.TTL(time.Second))
	if err != nil {
		b.Fatal(err)
	}
	parallelLoaders(b, cgm)
}

// LoadStore

func BenchmarkLoadStoreChannelMap(b *testing.B) {
	cgm, err := congomap.NewChannelMap()
	if err != nil {
		b.Fatal(err)
	}
	parallelLoadStorers(b, cgm)
}

func BenchmarkLoadStoreSyncAtomicMap(b *testing.B) {
	cgm, err := congomap.NewSyncAtomicMap()
	if err != nil {
		b.Fatal(err)
	}
	parallelLoadStorers(b, cgm)
}

func BenchmarkLoadStoreSyncMutexMap(b *testing.B) {
	cgm, err := congomap.NewSyncMutexMap()
	if err != nil {
		b.Fatal(err)
	}
	parallelLoadStorers(b, cgm)
}

func BenchmarkLoadStoreTwoLevelMap(b *testing.B) {
	cgm, err := congomap.NewTwoLevelMap()
	if err != nil {
		b.Fatal(err)
	}
	parallelLoadStorers(b, cgm)
}

// LoadStoreTTL

func BenchmarkLoadStoreTTLChannelMap(b *testing.B) {
	cgm, err := congomap.NewChannelMap(congomap.TTL(time.Second))
	if err != nil {
		b.Fatal(err)
	}
	parallelLoadStorers(b, cgm)
}

func BenchmarkLoadStoreTTLSyncAtomicMap(b *testing.B) {
	cgm, err := congomap.NewSyncAtomicMap(congomap.TTL(time.Second))
	if err != nil {
		b.Fatal(err)
	}
	parallelLoadStorers(b, cgm)
}

func BenchmarkLoadStoreTTLSyncMutexMap(b *testing.B) {
	cgm, err := congomap.NewSyncMutexMap(congomap.TTL(time.Second))
	if err != nil {
		b.Fatal(err)
	}
	parallelLoadStorers(b, cgm)
}

func BenchmarkLoadStoreTTLTwoLevelMap(b *testing.B) {
	cgm, err := congomap.NewTwoLevelMap(congomap.TTL(time.Second))
	if err != nil {
		b.Fatal(err)
	}
	parallelLoadStorers(b, cgm)
}

// benchmarks

func benchmark(b *testing.B, cgm congomap.Congomap, loaderCount, storerCount, loadStorerCount int) {
	defer func() { _ = cgm.Close() }()

	preloadCongomap(cgm)

	var stop atomic.Value
	var wg sync.WaitGroup

	wg.Add(loaderCount)
	for i := 0; i < loaderCount; i++ {
		go func() {
			for stop.Load() == nil {
				_, _ = cgm.Load(randomKey())
			}
			wg.Done()
		}()
	}

	wg.Add(storerCount)
	for i := 0; i < storerCount; i++ {
		go func() {
			for stop.Load() == nil {
				cgm.Store(randomKey(), randomState())
			}
			wg.Done()
		}()
	}

	wg.Add(loadStorerCount)
	for i := 0; i < loadStorerCount; i++ {
		go func() {
			for stop.Load() == nil {
				_, _ = cgm.LoadStore(randomKey())
			}
			wg.Done()
		}()
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, _ = cgm.LoadStore(randomKey())
	}

	stop.Store(struct{}{})
	wg.Wait()
}

func randomSlowLookup(_ string) (interface{}, error) {
	delay := 50*time.Millisecond + time.Duration(rand.Intn(100))*time.Millisecond
	time.Sleep(delay)
	return 42, nil
}

// High Concurrency

func BenchmarkHighConcurrencyFastLookupChannelMap(b *testing.B) {
	cgm, err := congomap.NewChannelMap(congomap.TTL(time.Minute))
	if err != nil {
		b.Fatal(err)
	}
	benchmark(b, cgm, 1, 1, 1000)
}

func BenchmarkHighConcurrencyFastLookupSyncAtomicMap(b *testing.B) {
	cgm, err := congomap.NewSyncAtomicMap(congomap.TTL(time.Minute))
	if err != nil {
		b.Fatal(err)
	}
	benchmark(b, cgm, 1, 1, 1000)
}

func BenchmarkHighConcurrencyFastLookupSyncMutexMap(b *testing.B) {
	cgm, err := congomap.NewSyncMutexMap(congomap.TTL(time.Minute))
	if err != nil {
		b.Fatal(err)
	}
	benchmark(b, cgm, 1, 1, 1000)
}

func BenchmarkHighConcurrencyFastLookupTwoLevelMap(b *testing.B) {
	cgm, err := congomap.NewTwoLevelMap(congomap.TTL(time.Minute))
	if err != nil {
		b.Fatal(err)
	}
	benchmark(b, cgm, 1, 1, 1000)
}

// High Read Concurrency

func BenchmarkHighReadConcurrencyFastLookupChannelMap(b *testing.B) {
	cgm, err := congomap.NewChannelMap(congomap.TTL(time.Minute))
	if err != nil {
		b.Fatal(err)
	}
	benchmark(b, cgm, 1000, 0, 0)
}

func BenchmarkHighReadConcurrencyFastLookupSyncAtomicMap(b *testing.B) {
	cgm, err := congomap.NewSyncAtomicMap(congomap.TTL(time.Minute))
	if err != nil {
		b.Fatal(err)
	}
	benchmark(b, cgm, 1000, 0, 0)
}

func BenchmarkHighReadConcurrencyFastLookupSyncMutexMap(b *testing.B) {
	cgm, err := congomap.NewSyncMutexMap(congomap.TTL(time.Minute))
	if err != nil {
		b.Fatal(err)
	}
	benchmark(b, cgm, 1000, 0, 0)
}

func BenchmarkHighReadConcurrencyFastLookupTwoLevelMap(b *testing.B) {
	cgm, err := congomap.NewTwoLevelMap(congomap.TTL(time.Minute))
	if err != nil {
		b.Fatal(err)
	}
	benchmark(b, cgm, 1000, 0, 0)
}

// lookup takes random time

func BenchmarkHighConcurrencySlowLookupChannelMap(b *testing.B) {
	cgm, err := congomap.NewChannelMap(congomap.Lookup(randomSlowLookup))
	if err != nil {
		b.Fatal(err)
	}
	benchmark(b, cgm, 1, 1, 1000)
}

func BenchmarkHighConcurrencySlowLookupSyncAtomicMap(b *testing.B) {
	cgm, err := congomap.NewSyncAtomicMap(congomap.Lookup(randomSlowLookup))
	if err != nil {
		b.Fatal(err)
	}
	benchmark(b, cgm, 1, 1, 1000)
}

func BenchmarkHighConcurrencySlowLookupSyncMutexMap(b *testing.B) {
	cgm, err := congomap.NewSyncMutexMap(congomap.Lookup(randomSlowLookup))
	if err != nil {
		b.Fatal(err)
	}
	benchmark(b, cgm, 1, 1, 1000)
}

func BenchmarkHighConcurrencySlowLookupTwoLevelMap(b *testing.B) {
	cgm, err := congomap.NewTwoLevelMap(congomap.Lookup(randomSlowLookup))
	if err != nil {
		b.Fatal(err)
	}
	benchmark(b, cgm, 1, 1, 1000)
}

// Low Concurrency

func BenchmarkLowConcurrencyFastLookupChannelMap(b *testing.B) {
	cgm, err := congomap.NewChannelMap(congomap.TTL(time.Minute))
	if err != nil {
		b.Fatal(err)
	}
	benchmark(b, cgm, 1, 1, 10)
}

func BenchmarkLowConcurrencyFastLookupSyncAtomicMap(b *testing.B) {
	cgm, err := congomap.NewSyncAtomicMap(congomap.TTL(time.Minute))
	if err != nil {
		b.Fatal(err)
	}
	benchmark(b, cgm, 1, 1, 10)
}

func BenchmarkLowConcurrencyFastLookupSyncMutexMap(b *testing.B) {
	cgm, err := congomap.NewSyncMutexMap(congomap.TTL(time.Minute))
	if err != nil {
		b.Fatal(err)
	}
	benchmark(b, cgm, 1, 1, 10)
}

func BenchmarkLowConcurrencyFastLookupTwoLevelMap(b *testing.B) {
	cgm, err := congomap.NewTwoLevelMap(congomap.TTL(time.Minute))
	if err != nil {
		b.Fatal(err)
	}
	benchmark(b, cgm, 1, 1, 10)
}

// lookup takes random time

func BenchmarkLowConcurrencySlowLookupChannelMap(b *testing.B) {
	cgm, err := congomap.NewChannelMap(congomap.Lookup(randomSlowLookup))
	if err != nil {
		b.Fatal(err)
	}
	benchmark(b, cgm, 1, 1, 10)
}

func BenchmarkLowConcurrencySlowLookupSyncAtomicMap(b *testing.B) {
	cgm, err := congomap.NewSyncAtomicMap(congomap.Lookup(randomSlowLookup))
	if err != nil {
		b.Fatal(err)
	}
	benchmark(b, cgm, 1, 1, 10)
}

func BenchmarkLowConcurrencySlowLookupSyncMutexMap(b *testing.B) {
	cgm, err := congomap.NewSyncMutexMap(congomap.Lookup(randomSlowLookup))
	if err != nil {
		b.Fatal(err)
	}
	benchmark(b, cgm, 1, 1, 10)
}

func BenchmarkLowConcurrencySlowLookupTwoLevelMap(b *testing.B) {
	cgm, err := congomap.NewTwoLevelMap(congomap.Lookup(randomSlowLookup))
	if err != nil {
		b.Fatal(err)
	}
	benchmark(b, cgm, 1, 1, 10)
}

// High Contention

func benchmarkHighContention(cgm congomap.Congomap) {
	defer func() { _ = cgm.Close() }()

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
	cgm, err := congomap.NewChannelMap(congomap.Lookup(randomFailOnLookup), congomap.TTL(time.Second))
	if err != nil {
		b.Fatal(err)
	}
	benchmarkHighContention(cgm)
}

func BenchmarkHighContentionSyncAtomicMap(b *testing.B) {
	cgm, err := congomap.NewSyncAtomicMap(congomap.Lookup(randomFailOnLookup), congomap.TTL(time.Second))
	if err != nil {
		b.Fatal(err)
	}
	benchmarkHighContention(cgm)
}

func BenchmarkHighContentionSyncMutexMap(b *testing.B) {
	cgm, err := congomap.NewSyncMutexMap(congomap.Lookup(randomFailOnLookup), congomap.TTL(time.Second))
	if err != nil {
		b.Fatal(err)
	}
	benchmarkHighContention(cgm)
}

func BenchmarkHighContentionTwoLevelMap(b *testing.B) {
	cgm, err := congomap.NewTwoLevelMap(congomap.Lookup(randomFailOnLookup), congomap.TTL(time.Second))
	if err != nil {
		b.Fatal(err)
	}
	benchmarkHighContention(cgm)
}
