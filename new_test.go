package congomap

import (
	"fmt"
	"sync"
	"testing"
	"time"
)

func slowSucceedingLookup(_ string) (interface{}, error) {
	time.Sleep(time.Second)
	return 42, nil
}

func asyncAccess(cgm Congomap, key string, wg *sync.WaitGroup) (interface{}, error) {
	defer wg.Done()
	val, ok := cgm.LoadStore(key)
	return val, ok
}

func testConcurrentLoad(t *testing.T, cgm Congomap) {
	// How many times to access at once
	totalTimes := 100

	var wg sync.WaitGroup

	start := time.Now()

	for i := 0; i < totalTimes; i++ {
		wg.Add(1)
		go asyncAccess(cgm, fmt.Sprintf("test%d", i), &wg)
	}

	wg.Wait()

	elapsed := time.Since(start)

	if elapsed > time.Second*2 {
		t.Fatalf("Load took too long")
	}

	var k string
	for i := 0; i < totalTimes; i++ {
		k = fmt.Sprintf("test%d", i)
		val, ok := cgm.Load(k)
		if !ok {
			t.Fatalf("Key %v missing from congomap %v", k, cgm)
		}
		if val != 42 {
			t.Fatalf("Value for %v incorrect in congomap, expected 42 got %v", k, val)
		}
	}

}

func TestConcurrentChannel(t *testing.T) {
	cgm, _ := NewChannelMap(Lookup(slowSucceedingLookup))
	testConcurrentLoad(t, cgm)
	cgm.Close()
}

func TestConcurrentMutex(t *testing.T) {
	cgm, _ := NewSyncMutexMap(Lookup(slowSucceedingLookup))
	testConcurrentLoad(t, cgm)
	cgm.Close()
}

func TestConcurrentAtomic(t *testing.T) {
	cgm, _ := NewSyncAtomicMap(Lookup(slowSucceedingLookup))
	testConcurrentLoad(t, cgm)
	cgm.Close()
}
