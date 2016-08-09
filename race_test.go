// +build race

package congomap

import (
	"math/rand"
	"sync"
	"testing"
	"time"
)

func testRace(t *testing.T, cgm Congomap) {
	defer cgm.Close()

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

func TestRaceChannelMap(t *testing.T) {
	cgm, _ := NewChannelMap(Lookup(randomFailOnLookup), TTL(time.Second))
	testRace(t, cgm)
}

func TestRaceSyncAtomicMap(t *testing.T) {
	cgm, _ := NewSyncAtomicMap(Lookup(randomFailOnLookup), TTL(time.Second))
	testRace(t, cgm)
}

func TestRaceSyncMutexMap(t *testing.T) {
	cgm, _ := NewSyncMutexMap(Lookup(randomFailOnLookup), TTL(time.Second))
	testRace(t, cgm)
}

func TestRaceTwoLevelMap(t *testing.T) {
	cgm, _ := NewTwoLevelMap(Lookup(randomFailOnLookup), TTL(time.Second))
	testRace(t, cgm)
}
