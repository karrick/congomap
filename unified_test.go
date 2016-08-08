package congomap

import (
	"errors"
	"math/rand"
	"sync"
	"testing"
	"time"
)

var errLookupFailed = errors.New("lookup failed")

func panicLookup(_ string) (interface{}, error) {
	panic("lookup panic")
}

func failingLookup(_ string) (interface{}, error) {
	return nil, errLookupFailed
}

func succeedingLookup(_ string) (interface{}, error) {
	return 42, nil
}

func randomFailOnLookup(_ string) (interface{}, error) {
	if rand.Float64() < 0.3 {
		return nil, errLookupFailed
	}
	return 42, nil
}

////////////////////////////////////////
// Load()

func loadNilFalse(t *testing.T, cgm Congomap, which, key string) {
	// t.Logf("Which: %q; Key: %q", which, key)
	value, ok := cgm.Load(key)
	if value != nil {
		t.Errorf("loadNilFalse: Which: %s; Key: %q; Actual: %#v; Expected: %#v", which, key, value, nil)
	}
	if ok != false {
		t.Errorf("loadNilFalse: Which: %s; Key: %q; Actual: %#v; Expected: %#v", which, key, ok, false)
	}
}

func loadValueTrue(t *testing.T, cgm Congomap, which, key string) {
	value, ok := cgm.Load(key)
	if value != 42 {
		t.Errorf("loadValueTrue: Which: %s; Key: %q; Actual: %#v; Expected: %#v", which, key, value, 42)
	}
	if ok != true {
		t.Errorf("loadValueTrue: Which: %s; Key: %q; Actual: %#v; Expected: %#v", which, key, ok, true)
	}
}

// LoadWithoutTTL

func loadNoTTL(t *testing.T, cgm Congomap, which string) {
	cgm.Store("hit", 42)
	loadNilFalse(t, cgm, which, "miss")
	loadValueTrue(t, cgm, which, "hit")
	_ = cgm.Close()
}

func TestLoadWithoutTTLChannelMap(t *testing.T) {
	cgm, _ := NewChannelMap(nil)
	loadNoTTL(t, cgm, "channel")
}

func TestLoadWithoutTTLSyncAtomicMap(t *testing.T) {
	cgm, _ := NewSyncAtomicMap(nil)
	loadNoTTL(t, cgm, "syncAtomic")
}

func TestLoadWithoutTTLSyncMutexMap(t *testing.T) {
	cgm, _ := NewSyncMutexMap(nil)
	loadNoTTL(t, cgm, "syncMutex")
}

func TestLoadWithoutTTLTwoLevelMap(t *testing.T) {
	cgm, _ := NewTwoLevelMap(nil)
	loadNoTTL(t, cgm, "twoLevel")
}

// LoadBeforeTTL

func loadBeforeTTL(t *testing.T, cgm Congomap, which string) {
	cgm.Store("hit", 42)
	loadNilFalse(t, cgm, which, "miss")
	loadValueTrue(t, cgm, which, "hit")
	_ = cgm.Close()
}

func TestLoadBeforeTTLChannelMap(t *testing.T) {
	cgm, _ := NewChannelMap(&Config{TTL: time.Minute})
	loadBeforeTTL(t, cgm, "channel")
}

func TestLoadBeforeTTLSyncAtomic(t *testing.T) {
	cgm, _ := NewSyncAtomicMap(&Config{TTL: time.Minute})
	loadBeforeTTL(t, cgm, "syncAtomic")
}

func TestLoadBeforeTTLSyncMutex(t *testing.T) {
	cgm, _ := NewSyncMutexMap(&Config{TTL: time.Minute})
	loadBeforeTTL(t, cgm, "syncMutex")
}

func TestLoadBeforeTTLTwoLevel(t *testing.T) {
	cgm, _ := NewTwoLevelMap(&Config{TTL: time.Minute})
	loadBeforeTTL(t, cgm, "twoLevel")
}

// LoadAfterTTL

func loadAfterTTL(t *testing.T, cgm Congomap, which string) {
	cgm.Store("hit", 42)
	time.Sleep(time.Millisecond)
	loadNilFalse(t, cgm, which, "miss")
	loadNilFalse(t, cgm, which, "hit")
	_ = cgm.Close()
}

func TestLoadAfterTTLChannelMap(t *testing.T) {
	cgm, _ := NewChannelMap(&Config{TTL: time.Nanosecond})
	loadAfterTTL(t, cgm, "channel")
}

func TestLoadAfterTTLSyncAtomicMap(t *testing.T) {
	cgm, _ := NewSyncAtomicMap(&Config{TTL: time.Nanosecond})
	loadAfterTTL(t, cgm, "syncAtomic")
}

func TestLoadAfterTTLSyncMutexMap(t *testing.T) {
	cgm, _ := NewSyncMutexMap(&Config{TTL: time.Nanosecond})
	loadAfterTTL(t, cgm, "syncMutex")
}

func TestLoadAfterTTLTwoLevelMap(t *testing.T) {
	cgm, _ := NewTwoLevelMap(&Config{TTL: time.Nanosecond})
	loadAfterTTL(t, cgm, "twoLevel")
}

////////////////////////////////////////
// LoadStore()

func loadStoreNilErrNoLookupDefined(t *testing.T, cgm Congomap, which, key string) {
	value, err := cgm.LoadStore(key)
	if value != nil {
		t.Errorf("LoadStoreMiss: Which: %s; Key: %q; Actual: %#v; Expected: %#v", which, key, value, nil)
	}
	if _, ok := err.(ErrNoLookupDefined); err == nil || !ok {
		t.Errorf("LoadStoreMiss: Which: %s; Key: %q; Actual: %#v; Expected: %#v", which, key, err, ErrNoLookupDefined{})
	}
}

func loadStoreNilErrLookupFailed(t *testing.T, cgm Congomap, which, key string) {
	value, err := cgm.LoadStore(key)
	if value != nil {
		t.Errorf("LoadStoreMiss: Which: %s; Key: %q; Actual: %#v; Expected: %#v", which, key, value, nil)
	}
	if err == nil || err.Error() != "lookup failed" {
		t.Errorf("LoadStoreMiss: Which: %s; Key: %q; Actual: %#v; Expected: %#v", which, key, err, errLookupFailed)
	}
}

func loadStoreValueNil(t *testing.T, cgm Congomap, which, key string) {
	value, err := cgm.LoadStore(key)
	if value != 42 {
		t.Errorf("LoadStoreHitNoTTL: Which: %s; Key: %q; Actual: %#v; Expected: %#v", which, key, value, 42)
	}
	if err != nil {
		t.Errorf("LoadStoreHitNoTTL: Which: %s; Key: %q; Actual: %#v; Expected: %#v", which, key, err, nil)
	}
}

// LoadStoreNoLookupNoTTL

func loadStoreNoLookupNoTTL(t *testing.T, cgm Congomap, which string) {
	cgm.Store("hit", 42)
	loadStoreNilErrNoLookupDefined(t, cgm, which, "miss")
	loadStoreValueNil(t, cgm, which, "hit")
	_ = cgm.Close()
}

func TestLoadStoreNoLookupNoTTLChannelMap(t *testing.T) {
	cgm, _ := NewChannelMap(nil)
	loadStoreNoLookupNoTTL(t, cgm, "channel")
}

func TestLoadStoreNoLookupNoTTLSyncAtomicMap(t *testing.T) {
	cgm, _ := NewSyncAtomicMap(nil)
	loadStoreNoLookupNoTTL(t, cgm, "syncAtomic")
}

func TestLoadStoreNoLookupNoTTLSyncMutexMap(t *testing.T) {
	cgm, _ := NewSyncMutexMap(nil)
	loadStoreNoLookupNoTTL(t, cgm, "syncMutex")
}

func TestLoadStoreNoLookupNoTTLTwoLevelMap(t *testing.T) {
	cgm, _ := NewTwoLevelMap(nil)
	loadStoreNoLookupNoTTL(t, cgm, "twoLevel")
}

// LoadStoreFailingLookupNoTTL

func loadStoreFailingLookupNoTTL(t *testing.T, cgm Congomap, which string) {
	cgm.Store("hit", 42)
	loadStoreNilErrLookupFailed(t, cgm, which, "miss")
	loadStoreValueNil(t, cgm, which, "hit")
	_ = cgm.Close()
}

func TestLoadStoreFailingLookupNoTTLChannelMap(t *testing.T) {
	cgm, _ := NewChannelMap(&Config{Lookup: failingLookup})
	loadStoreFailingLookupNoTTL(t, cgm, "channel")
}

func TestLoadStoreFailingLookupNoTTLSyncAtomicMap(t *testing.T) {
	cgm, _ := NewSyncAtomicMap(&Config{Lookup: failingLookup})
	loadStoreFailingLookupNoTTL(t, cgm, "syncAtomic")
}

func TestLoadStoreFailingLookupNoTTLSyncMutexMap(t *testing.T) {
	cgm, _ := NewSyncMutexMap(&Config{Lookup: failingLookup})
	loadStoreFailingLookupNoTTL(t, cgm, "syncMutex")
}

func TestLoadStoreFailingLookupNoTTLTwoLevelMap(t *testing.T) {
	cgm, _ := NewTwoLevelMap(&Config{Lookup: failingLookup})
	loadStoreFailingLookupNoTTL(t, cgm, "twoLevel")
}

// LoadStoreLookupNoTTL

func loadStoreLookupNoTTL(t *testing.T, cgm Congomap, which string) {
	cgm.Store("hit", 42)
	loadStoreValueNil(t, cgm, which, "miss")
	loadStoreValueNil(t, cgm, which, "hit")
	_ = cgm.Close()
}

func TestLoadStoreLookupNoTTLChannelMap(t *testing.T) {
	cgm, _ := NewChannelMap(&Config{Lookup: succeedingLookup})
	loadStoreLookupNoTTL(t, cgm, "channel")
}

func TestLoadStoreLookupNoTTLSyncAtomicMap(t *testing.T) {
	cgm, _ := NewSyncAtomicMap(&Config{Lookup: succeedingLookup})
	loadStoreLookupNoTTL(t, cgm, "syncAtomic")
}

func TestLoadStoreLookupNoTTLSyncMutexMap(t *testing.T) {
	cgm, _ := NewSyncMutexMap(&Config{Lookup: succeedingLookup})
	loadStoreLookupNoTTL(t, cgm, "syncMutex")
}

func TestLoadStoreLookupNoTTLTwoLevelMap(t *testing.T) {
	cgm, _ := NewTwoLevelMap(&Config{Lookup: succeedingLookup})
	loadStoreLookupNoTTL(t, cgm, "twoLevel")
}

// LoadStoreNoLookupBeforeTTL

func loadStoreNoLookupBeforeTTL(t *testing.T, cgm Congomap, which string) {
	cgm.Store("hit", 42)
	loadStoreNilErrNoLookupDefined(t, cgm, which, "miss")
	loadStoreValueNil(t, cgm, which, "hit")
	_ = cgm.Close()
}

func TestLoadStoreNoLookupBeforeTTLChannelMap(t *testing.T) {
	cgm, _ := NewChannelMap(&Config{TTL: time.Minute})
	loadStoreNoLookupBeforeTTL(t, cgm, "channel")
}

func TestLoadStoreNoLookupBeforeTTLSyncAtomicMap(t *testing.T) {
	cgm, _ := NewSyncAtomicMap(&Config{TTL: time.Minute})
	loadStoreNoLookupBeforeTTL(t, cgm, "syncAtomic")
}

func TestLoadStoreNoLookupBeforeTTLSyncMutexMap(t *testing.T) {
	cgm, _ := NewSyncMutexMap(&Config{TTL: time.Minute})
	loadStoreNoLookupBeforeTTL(t, cgm, "syncMutex")
}

func TestLoadStoreNoLookupBeforeTTLTwoLevelMap(t *testing.T) {
	cgm, _ := NewTwoLevelMap(&Config{TTL: time.Minute})
	loadStoreNoLookupBeforeTTL(t, cgm, "twoLevel")
}

// LoadStoreFailingLookupBeforeTTL

func loadStoreFailingLookupBeforeTTL(t *testing.T, cgm Congomap, which string) {
	cgm.Store("hit", 42)
	loadStoreNilErrLookupFailed(t, cgm, which, "miss")
	loadStoreValueNil(t, cgm, which, "hit")
	_ = cgm.Close()
}

func TestLoadStoreFailingLookupBeforeTTLChannelMap(t *testing.T) {
	cgm, _ := NewChannelMap(&Config{Lookup: failingLookup, TTL: time.Minute})
	loadStoreFailingLookupBeforeTTL(t, cgm, "channel")
}

func TestLoadStoreFailingLookupBeforeTTLSyncAtomicMap(t *testing.T) {
	cgm, _ := NewSyncAtomicMap(&Config{Lookup: failingLookup, TTL: time.Minute})
	loadStoreFailingLookupBeforeTTL(t, cgm, "syncAtomic")
}

func TestLoadStoreFailingLookupBeforeTTLSyncMutexMap(t *testing.T) {
	cgm, _ := NewSyncMutexMap(&Config{Lookup: failingLookup, TTL: time.Minute})
	loadStoreFailingLookupBeforeTTL(t, cgm, "syncMutex")
}

func TestLoadStoreFailingLookupBeforeTTLTwoLevelMap(t *testing.T) {
	cgm, _ := NewTwoLevelMap(&Config{Lookup: failingLookup, TTL: time.Minute})
	loadStoreFailingLookupBeforeTTL(t, cgm, "twoLevel")
}

// LoadStoreLookupBeforeTTL

func loadStoreLookupBeforeTTL(t *testing.T, cgm Congomap, which string) {
	cgm.Store("hit", 42)
	loadStoreValueNil(t, cgm, which, "miss")
	loadStoreValueNil(t, cgm, which, "hit")
	_ = cgm.Close()
}

func TestLoadStoreLookupBeforeTTLChannelMap(t *testing.T) {
	cgm, _ := NewChannelMap(&Config{Lookup: succeedingLookup, TTL: time.Minute})
	loadStoreLookupBeforeTTL(t, cgm, "channel")
}

func TestLoadStoreLookupBeforeTTLSyncAtomicMap(t *testing.T) {
	cgm, _ := NewSyncAtomicMap(&Config{Lookup: succeedingLookup, TTL: time.Minute})
	loadStoreLookupBeforeTTL(t, cgm, "syncAtomic")
}

func TestLoadStoreLookupBeforeTTLSyncMutexMap(t *testing.T) {
	cgm, _ := NewSyncMutexMap(&Config{Lookup: succeedingLookup, TTL: time.Minute})
	loadStoreLookupBeforeTTL(t, cgm, "syncMutex")
}

func TestLoadStoreLookupBeforeTTLTwoLevelMap(t *testing.T) {
	cgm, _ := NewTwoLevelMap(&Config{Lookup: succeedingLookup, TTL: time.Minute})
	loadStoreLookupBeforeTTL(t, cgm, "twoLevel")
}

// LoadStoreNoLookupAfterTTL

func loadStoreNoLookupAfterTTL(t *testing.T, cgm Congomap, which string) {
	cgm.Store("hit", 42)
	time.Sleep(time.Millisecond)
	loadStoreNilErrNoLookupDefined(t, cgm, which, "miss")
	loadStoreNilErrNoLookupDefined(t, cgm, which, "hit")
	_ = cgm.Close()
}
func TestLoadStoreNoLookupAfterTTLChannelMap(t *testing.T) {
	cgm, _ := NewChannelMap(&Config{TTL: time.Nanosecond})
	loadStoreNoLookupAfterTTL(t, cgm, "channel")
}

func TestLoadStoreNoLookupAfterTTLSyncAtomicMap(t *testing.T) {
	cgm, _ := NewSyncAtomicMap(&Config{TTL: time.Nanosecond})
	loadStoreNoLookupAfterTTL(t, cgm, "syncAtomic")
}

func TestLoadStoreNoLookupAfterTTLSyncMutexMap(t *testing.T) {
	cgm, _ := NewSyncMutexMap(&Config{TTL: time.Nanosecond})
	loadStoreNoLookupAfterTTL(t, cgm, "syncMutex")
}

func TestLoadStoreNoLookupAfterTTLTwoLevelMap(t *testing.T) {
	cgm, _ := NewTwoLevelMap(&Config{TTL: time.Nanosecond})
	loadStoreNoLookupAfterTTL(t, cgm, "twoLevel")
}

// LoadStoreFailingLookupAfterTTL

func loadStoreFailingLookupAfterTTL(t *testing.T, cgm Congomap, which string) {
	cgm.Store("hit", 42)
	time.Sleep(time.Millisecond)
	loadStoreNilErrLookupFailed(t, cgm, which, "miss")
	loadStoreNilErrLookupFailed(t, cgm, which, "hit")
	_ = cgm.Close()
}

func TestLoadStoreFailingLookupAfterTTLChannelMap(t *testing.T) {
	cgm, _ := NewChannelMap(&Config{Lookup: failingLookup, TTL: time.Nanosecond})
	loadStoreFailingLookupAfterTTL(t, cgm, "channel")
}

func TestLoadStoreFailingLookupAfterTTLSyncAtomicMap(t *testing.T) {
	cgm, _ := NewSyncAtomicMap(&Config{Lookup: failingLookup, TTL: time.Nanosecond})
	loadStoreFailingLookupAfterTTL(t, cgm, "syncAtomic")
}

func TestLoadStoreFailingLookupAfterTTLSyncMutexMap(t *testing.T) {
	cgm, _ := NewSyncMutexMap(&Config{Lookup: failingLookup, TTL: time.Nanosecond})
	loadStoreFailingLookupAfterTTL(t, cgm, "syncMutex")
}

func TestLoadStoreFailingLookupAfterTTLTwoLevelMap(t *testing.T) {
	cgm, _ := NewTwoLevelMap(&Config{Lookup: failingLookup, TTL: time.Nanosecond})
	loadStoreFailingLookupAfterTTL(t, cgm, "twoLevel")
}

// LoadStoreLookupAfterTTL

func loadStoreLookupAfterTTL(t *testing.T, cgm Congomap, which string) {
	cgm.Store("hit", 42)
	time.Sleep(time.Millisecond)
	loadStoreValueNil(t, cgm, which, "miss")
	loadStoreValueNil(t, cgm, which, "hit")
	_ = cgm.Close()
}

func TestLoadStoreLookupAfterTTLChannelMap(t *testing.T) {
	cgm, _ := NewChannelMap(&Config{Lookup: succeedingLookup, TTL: time.Nanosecond})
	loadStoreLookupAfterTTL(t, cgm, "channel")
}

func TestLoadStoreLookupAfterTTLSyncAtomicMap(t *testing.T) {
	cgm, _ := NewSyncAtomicMap(&Config{Lookup: succeedingLookup, TTL: time.Nanosecond})
	loadStoreLookupAfterTTL(t, cgm, "syncAtomic")
}

func TestLoadStoreLookupAfterTTLSyncMutexMap(t *testing.T) {
	cgm, _ := NewSyncMutexMap(&Config{Lookup: succeedingLookup, TTL: time.Nanosecond})
	loadStoreLookupAfterTTL(t, cgm, "syncMutex")
}

func TestLoadStoreLookupAfterTTLTwoLevelMap(t *testing.T) {
	cgm, _ := NewTwoLevelMap(&Config{Lookup: succeedingLookup, TTL: time.Nanosecond})
	loadStoreLookupAfterTTL(t, cgm, "twoLevel")
}

////////////////////////////////////////
// Pairs()

func testPairs(t *testing.T, cgm Congomap, which string) {
	cgm.Store("first", "Clark")
	cgm.Store("last", "Kent")
	for pair := range cgm.Pairs() {
		if _, ok := pair.Value.(string); !ok {
			t.Errorf("Actual: %#v; Expected: %#v", ok, true)
		}
	}
	_ = cgm.Close()
}

func TestPairsChannelMap(t *testing.T) {
	cgm, _ := NewChannelMap(nil)
	testPairs(t, cgm, "channel")
}

func TestPairsSyncAtomicMap(t *testing.T) {
	cgm, _ := NewSyncAtomicMap(nil)
	testPairs(t, cgm, "syncAtomic")
}

func TestPairsSyncMutexMap(t *testing.T) {
	cgm, _ := NewSyncMutexMap(nil)
	testPairs(t, cgm, "syncMutex")
}

func TestPairsTwoLevelMap(t *testing.T) {
	cgm, _ := NewTwoLevelMap(nil)
	testPairs(t, cgm, "twoLevel")
}

// ReaperInvokedDuringDelete

func createReaper(t *testing.T, wg *sync.WaitGroup, which string) func(interface{}) {
	expected := 42
	return func(value interface{}) {
		if v, ok := value.(int); !ok || v != expected {
			t.Errorf("reaper receives value during delete; Which: %s; Key: %q; Actual: %#v; Expected: %#v", which, value, expected)
		}
		wg.Done()
	}
}

func createReaperTesterInvokeDuringDelete(t *testing.T, wg *sync.WaitGroup) func(Congomap) {
	expected := 42
	return func(cgm Congomap) {
		defer cgm.Close()
		cgm.Store("hit", expected)
		wg.Add(1)
		cgm.Delete("hit")
		wg.Wait()
	}
}

func TestReaperInvokedDuringDeleteChannelMap(t *testing.T) {
	var wg sync.WaitGroup
	cgm, _ := NewChannelMap(&Config{Reaper: createReaper(t, &wg, "channel")})
	createReaperTesterInvokeDuringDelete(t, &wg)(cgm)
}

func TestReaperInvokedDuringDeleteSyncAtomicMap(t *testing.T) {
	var wg sync.WaitGroup
	cgm, _ := NewSyncAtomicMap(&Config{Reaper: createReaper(t, &wg, "syncAtomic")})
	createReaperTesterInvokeDuringDelete(t, &wg)(cgm)
}

func TestReaperInvokedDuringDeleteSyncMutexMap(t *testing.T) {
	var wg sync.WaitGroup
	cgm, _ := NewSyncMutexMap(&Config{Reaper: createReaper(t, &wg, "syncMutex")})
	createReaperTesterInvokeDuringDelete(t, &wg)(cgm)
}

func TestReaperInvokedDuringDeleteTwoLevelMap(t *testing.T) {
	var wg sync.WaitGroup
	cgm, _ := NewTwoLevelMap(&Config{Reaper: createReaper(t, &wg, "twoLevel")})
	createReaperTesterInvokeDuringDelete(t, &wg)(cgm)
}

// ReaperInvokedDuringGC

func createReaperTesterInvokeDuringGC(t *testing.T, wg *sync.WaitGroup) func(Congomap) {
	expected := 42
	return func(cgm Congomap) {
		defer cgm.Close()
		cgm.Store("hit", &ExpiringValue{Value: expected, Expiry: time.Now().Add(time.Nanosecond)})
		time.Sleep(time.Millisecond)
		wg.Add(1)
		cgm.GC()
		wg.Wait()
	}
}

func TestReaperInvokedDuringGCChannelMap(t *testing.T) {
	var wg sync.WaitGroup
	cgm, _ := NewChannelMap(&Config{Reaper: createReaper(t, &wg, "channel")})
	createReaperTesterInvokeDuringGC(t, &wg)(cgm)
}

func TestReaperInvokedDuringGCSyncAtomicMap(t *testing.T) {
	var wg sync.WaitGroup
	cgm, _ := NewSyncAtomicMap(&Config{Reaper: createReaper(t, &wg, "syncAtomic")})
	createReaperTesterInvokeDuringGC(t, &wg)(cgm)
}

func TestReaperInvokedDuringGCSyncMutexMap(t *testing.T) {
	var wg sync.WaitGroup
	cgm, _ := NewSyncMutexMap(&Config{Reaper: createReaper(t, &wg, "syncMutex")})
	createReaperTesterInvokeDuringGC(t, &wg)(cgm)
}

func TestReaperInvokedDuringGCTwoLevelMap(t *testing.T) {
	var wg sync.WaitGroup
	cgm, _ := NewTwoLevelMap(&Config{Reaper: createReaper(t, &wg, "twoLevel")})
	createReaperTesterInvokeDuringGC(t, &wg)(cgm)
}

// ReaperInvokedDuringClose

func createReaperTesterInvokeDuringClose(t *testing.T, wg *sync.WaitGroup) func(Congomap) {
	expected := 42
	return func(cgm Congomap) {
		cgm.Store("hit", &ExpiringValue{Value: expected, Expiry: time.Now().Add(time.Nanosecond)})
		time.Sleep(time.Millisecond)
		wg.Add(1)
		_ = cgm.Close()
		wg.Wait()
	}
}

func TestReaperInvokedDuringCloseChannelMap(t *testing.T) {
	var wg sync.WaitGroup
	cgm, _ := NewChannelMap(&Config{Reaper: createReaper(t, &wg, "channel")})
	createReaperTesterInvokeDuringClose(t, &wg)(cgm)
}

func TestReaperInvokedDuringCloseSyncAtomicMap(t *testing.T) {
	var wg sync.WaitGroup
	cgm, _ := NewSyncAtomicMap(&Config{Reaper: createReaper(t, &wg, "syncAtomic")})
	createReaperTesterInvokeDuringClose(t, &wg)(cgm)
}

func TestReaperInvokedDuringCloseSyncMutexMap(t *testing.T) {
	var wg sync.WaitGroup
	cgm, _ := NewSyncMutexMap(&Config{Reaper: createReaper(t, &wg, "syncMutex")})
	createReaperTesterInvokeDuringClose(t, &wg)(cgm)
}

func TestReaperInvokedDuringCloseTwoLevelMap(t *testing.T) {
	var wg sync.WaitGroup
	cgm, _ := NewTwoLevelMap(&Config{Reaper: createReaper(t, &wg, "twoLevel")})
	createReaperTesterInvokeDuringClose(t, &wg)(cgm)
}
