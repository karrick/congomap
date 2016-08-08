// +build race

package congomap

import (
	"math/rand"
	"sync"
	"testing"
	"time"
)

func testRace(t *testing.T, cgm Congomap) {
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
	cgm, _ := NewChannelMap(&Config{Lookup: randomFailOnLookup, TTL: time.Second})
	testRace(t, cgm)
	defer cgm.Close()
}

func TestRaceSyncAtomicMap(t *testing.T) {
	cgm, _ := NewSyncAtomicMap(&Config{Lookup: randomFailOnLookup, TTL: time.Second})
	testRace(t, cgm)
	defer cgm.Close()
}

func TestRaceSyncMutexMap(t *testing.T) {
	cgm, _ := NewSyncMutexMap(&Config{Lookup: randomFailOnLookup, TTL: time.Second})
	testRace(t, cgm)
	defer cgm.Close()
}

func TestRaceTwoLevelMap(t *testing.T) {
	cgm, _ := NewTwoLevelMap(&Config{Lookup: randomFailOnLookup, TTL: time.Second})
	testRace(t, cgm)
	defer cgm.Close()
}
