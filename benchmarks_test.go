package congomap

import (
	"math/rand"
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

func withLoadedCongomap(cm Congomap, loaders, storers, loadstorers int, fn func()) {
	var stop bool
	exited := make(chan struct{})
	for i := 0; i < loaders; i++ {
		go func() {
			for !stop {
				_, _ = cm.Load(randomState())
			}
			exited <- struct{}{}
		}()
	}
	for i := 0; i < storers; i++ {
		go func() {
			for !stop {
				cm.Store(randomState(), randomState())
			}
			exited <- struct{}{}
		}()
	}
	for i := 0; i < loadstorers; i++ {
		go func() {
			for !stop {
				_, _ = cm.LoadStore(randomState())
			}
			exited <- struct{}{}
		}()
	}

	fn()

	stop = true
	for i := 0; i < loaders+storers+loadstorers; i++ {
		<-exited
	}
}

func parallelBench(b *testing.B, cm Congomap, fn func(*testing.PB)) {
	// doesn't necessarily fill the entire map
	for i := 0; i < len(states); i++ {
		cm.Store(randomState(), randomState())
	}
	b.RunParallel(fn)
}

func parallelLoaders(b *testing.B, cm Congomap) {
	parallelBench(b, cm, func(pb *testing.PB) {
		for pb.Next() {
			cm.Load(randomState())
		}
	})
}

func parallelLoadStorers(b *testing.B, cm Congomap) {
	parallelBench(b, cm, func(pb *testing.PB) {
		for pb.Next() {
			cm.LoadStore(randomState())
		}
	})
}

func BenchmarkChannelMapLoad(b *testing.B) {
	cm, _ := NewChannelMap()
	defer cm.Halt()
	parallelLoaders(b, cm)
}

func BenchmarkChannelMapLoadStore(b *testing.B) {
	cm, _ := NewChannelMap()
	defer cm.Halt()
	parallelLoadStorers(b, cm)
}

func BenchmarkSyncAtomicMapLoad(b *testing.B) {
	cm, _ := NewSyncAtomicMap()
	defer cm.Halt()
	parallelLoaders(b, cm)
}

func BenchmarkSyncAtomicMapLoadTTL(b *testing.B) {
	cm, _ := NewSyncAtomicMap(TTL(time.Second))
	defer cm.Halt()
	parallelLoaders(b, cm)
}

func BenchmarkSyncAtomicMapLoadStore(b *testing.B) {
	cm, _ := NewSyncAtomicMap()
	defer cm.Halt()
	parallelLoadStorers(b, cm)
}

func BenchmarkSyncMutexMapLoad(b *testing.B) {
	cm, _ := NewSyncMutexMap()
	defer cm.Halt()
	parallelLoaders(b, cm)
}

func BenchmarkSyncMutexMapLoadTTL(b *testing.B) {
	cm, _ := NewSyncMutexMap(TTL(time.Second))
	defer cm.Halt()
	parallelLoaders(b, cm)
}

func BenchmarkSyncMutexMapLoadStore(b *testing.B) {
	cm, _ := NewSyncMutexMap()
	defer cm.Halt()
	parallelLoadStorers(b, cm)
}

func BenchmarkSyncMutexShardedMapLoad(b *testing.B) {
	cm, _ := NewSyncMutexShardedMap()
	defer cm.Halt()
	parallelLoaders(b, cm)
}

func BenchmarkSyncMutexShardedMapLoadTTL(b *testing.B) {
	cm, _ := NewSyncMutexShardedMap(TTL(time.Second))
	defer cm.Halt()
	parallelLoaders(b, cm)
}

func BenchmarkSyncMutexShardedMapLoadStore(b *testing.B) {
	cm, _ := NewSyncMutexShardedMap()
	defer cm.Halt()
	parallelLoadStorers(b, cm)
}

func _BenchmarkSyncMutexMapManyLoadersLoaderPerspective(b *testing.B) {
	lookup := func(_ string) (interface{}, error) { return randomState(), nil }
	cm, _ := NewSyncMutexMap(Lookup(lookup))
	defer cm.Halt()

	var r interface{}
	withLoadedCongomap(cm, 20, 1, 1, func() {
		for i := 0; i < b.N; i++ {
			r, _ = cm.Load(randomState())
		}
	})

	preventCompilerOptimizingOutBenchmarks = r
}

func _BenchmarkChannelMapManyLoadersLoaderPerspective(b *testing.B) {
	lookup := func(_ string) (interface{}, error) { return randomState(), nil }
	cm, _ := NewChannelMap(Lookup(lookup))
	defer cm.Halt()

	var r interface{}
	withLoadedCongomap(cm, 20, 1, 1, func() {
		for i := 0; i < b.N; i++ {
			r, _ = cm.Load(randomState())
		}
	})

	preventCompilerOptimizingOutBenchmarks = r
}

func _BenchmarkSyncMutexMapManyLoadersLoadStorerPerspective(b *testing.B) {
	var r interface{}
	cm, _ := NewSyncMutexMap()
	defer cm.Halt()

	withLoadedCongomap(cm, 20, 1, 1, func() {
		for i := 0; i < b.N; i++ {
			r, _ = cm.LoadStore(randomState())
		}
	})

	preventCompilerOptimizingOutBenchmarks = r
}

// consider adding loaded benchmarks for other types
// consider loading up the map with 10k entries
