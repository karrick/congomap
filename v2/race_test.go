// +build race

package congomap_test

import (
	"math/rand"
	"sync"
	"testing"
	"time"

	congomap "github.com/karrick/congomap/v2"
)

func testRace(t *testing.T, cgm congomap.Congomap) {
	defer func() { _ = cgm.Close() }()

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
	cgm, err := congomap.NewChannelMap(congomap.Lookup(randomFailOnLookup), congomap.TTL(time.Second))
	if err != nil {
		t.Fatal(err)
	}
	testRace(t, cgm)
}

func TestRaceSyncAtomicMap(t *testing.T) {
	cgm, err := congomap.NewSyncAtomicMap(congomap.Lookup(randomFailOnLookup), congomap.TTL(time.Second))
	if err != nil {
		t.Fatal(err)
	}
	testRace(t, cgm)
}

func TestRaceSyncMutexMap(t *testing.T) {
	cgm, err := congomap.NewSyncMutexMap(congomap.Lookup(randomFailOnLookup), congomap.TTL(time.Second))
	if err != nil {
		t.Fatal(err)
	}
	testRace(t, cgm)
}

func TestRaceTwoLevelMap(t *testing.T) {
	cgm, err := congomap.NewTwoLevelMap(congomap.Lookup(randomFailOnLookup), congomap.TTL(time.Second))
	if err != nil {
		t.Fatal(err)
	}
	testRace(t, cgm)
}
