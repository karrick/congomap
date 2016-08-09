// +build race

package congomap_test

import (
	"math/rand"
	"sync"
	"testing"
	"time"

	"github.com/karrick/congomap"
)

func testRace(t *testing.T, cgm congomap.Congomap) {
	defer cgm.Close()

	const tasks = 1000
	keys := []string{"just", "a", "few", "keys", "to", "force", "lock", "contention"}

	var wg sync.WaitGroup
	wg.Add(tasks)

	for i := 0; i < tasks; i++ {
		go func() {
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
		}()
	}

	wg.Wait()
}

func TestRaceChannelMap(t *testing.T) {
	cgm, _ := congomap.NewChannelMap(congomap.Lookup(randomFailOnLookup), congomap.TTL(time.Second))
	testRace(t, cgm)
}

func TestRaceSyncAtomicMap(t *testing.T) {
	cgm, _ := congomap.NewSyncAtomicMap(congomap.Lookup(randomFailOnLookup), congomap.TTL(time.Second))
	testRace(t, cgm)
}

func TestRaceSyncMutexMap(t *testing.T) {
	cgm, _ := congomap.NewSyncMutexMap(congomap.Lookup(randomFailOnLookup), congomap.TTL(time.Second))
	testRace(t, cgm)
}

func TestRaceTwoLevelMap(t *testing.T) {
	cgm, _ := congomap.NewTwoLevelMap(congomap.Lookup(randomFailOnLookup), congomap.TTL(time.Second))
	testRace(t, cgm)
}
