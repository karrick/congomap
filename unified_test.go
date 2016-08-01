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

////////////////////

func loadNoTTL(t *testing.T, cgm Congomap, which string) {
	cgm.Store("hit", 42)
	loadNilFalse(t, cgm, which, "miss")
	loadValueTrue(t, cgm, which, "hit")
}

func TestLoadNoTTL(t *testing.T) {
	var cgm Congomap

	cgm, _ = NewChannelMap()
	loadNoTTL(t, cgm, "sync-mutex")
	_ = cgm.Close()

	cgm, _ = NewSyncAtomicMap()
	loadNoTTL(t, cgm, "sync-atomic")
	_ = cgm.Close()

	cgm, _ = NewSyncMutexMap()
	loadNoTTL(t, cgm, "sync-mutex")
	_ = cgm.Close()

	cgm, _ = NewTwoLevelMap()
	loadNoTTL(t, cgm, "twoLevel")
	_ = cgm.Close()
}

func loadBeforeTTL(t *testing.T, cgm Congomap, which string) {
	cgm.Store("hit", 42)
	loadNilFalse(t, cgm, which, "miss")
	loadValueTrue(t, cgm, which, "hit")
}

func TestLoadBeforeTTL(t *testing.T) {
	var cgm Congomap

	cgm, _ = NewChannelMap(TTL(time.Minute))
	loadBeforeTTL(t, cgm, "sync-mutex")
	_ = cgm.Close()

	cgm, _ = NewSyncAtomicMap(TTL(time.Minute))
	loadBeforeTTL(t, cgm, "sync-atomic")
	_ = cgm.Close()

	cgm, _ = NewSyncMutexMap(TTL(time.Minute))
	loadBeforeTTL(t, cgm, "sync-mutex")
	_ = cgm.Close()

	cgm, _ = NewTwoLevelMap(TTL(time.Minute))
	loadBeforeTTL(t, cgm, "twoLevel")
	_ = cgm.Close()
}

func loadAfterTTL(t *testing.T, cgm Congomap, which string) {
	cgm.Store("hit", 42)
	time.Sleep(time.Millisecond)
	loadNilFalse(t, cgm, which, "miss")
	loadNilFalse(t, cgm, which, "hit")
}

func TestLoadAfterTTL(t *testing.T) {
	var cgm Congomap

	cgm, _ = NewChannelMap(TTL(time.Nanosecond))
	loadAfterTTL(t, cgm, "sync-mutex")
	_ = cgm.Close()

	cgm, _ = NewSyncAtomicMap(TTL(time.Nanosecond))
	loadAfterTTL(t, cgm, "sync-atomic")
	_ = cgm.Close()

	cgm, _ = NewSyncMutexMap(TTL(time.Nanosecond))
	loadAfterTTL(t, cgm, "sync-mutex")
	_ = cgm.Close()

	cgm, _ = NewTwoLevelMap(TTL(time.Nanosecond))
	loadAfterTTL(t, cgm, "twoLevel")
	_ = cgm.Close()
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

////////////////////

func TestLoadStoreNoLookupNoTTL(t *testing.T) {
	var cgm Congomap

	cgm, _ = NewChannelMap()
	loadStoreNoLookupNoTTL(t, cgm, "sync-mutex")
	_ = cgm.Close()

	cgm, _ = NewSyncAtomicMap()
	loadStoreNoLookupNoTTL(t, cgm, "sync-atomic")
	_ = cgm.Close()

	cgm, _ = NewSyncMutexMap()
	loadStoreNoLookupNoTTL(t, cgm, "sync-mutex")
	_ = cgm.Close()

	cgm, _ = NewTwoLevelMap()
	loadStoreNoLookupNoTTL(t, cgm, "twoLevel")
	_ = cgm.Close()
}

func loadStoreNoLookupNoTTL(t *testing.T, cgm Congomap, which string) {
	cgm.Store("hit", 42)
	loadStoreNilErrNoLookupDefined(t, cgm, which, "miss")
	loadStoreValueNil(t, cgm, which, "hit")
}

////

func TestLoadStoreFailingLookupNoTTL(t *testing.T) {
	var cgm Congomap

	cgm, _ = NewChannelMap(Lookup(failingLookup))
	loadStoreFailingLookupNoTTL(t, cgm, "sync-mutex")
	_ = cgm.Close()

	cgm, _ = NewSyncAtomicMap(Lookup(failingLookup))
	loadStoreFailingLookupNoTTL(t, cgm, "sync-atomic")
	_ = cgm.Close()

	cgm, _ = NewSyncMutexMap(Lookup(failingLookup))
	loadStoreFailingLookupNoTTL(t, cgm, "sync-mutex")
	_ = cgm.Close()

	cgm, _ = NewTwoLevelMap(Lookup(failingLookup))
	loadStoreFailingLookupNoTTL(t, cgm, "twoLevel")
	_ = cgm.Close()
}

func loadStoreFailingLookupNoTTL(t *testing.T, cgm Congomap, which string) {
	cgm.Store("hit", 42)
	loadStoreNilErrLookupFailed(t, cgm, which, "miss")
	loadStoreValueNil(t, cgm, which, "hit")
}

////

func TestLoadStoreLookupNoTTL(t *testing.T) {
	var cgm Congomap

	cgm, _ = NewChannelMap(Lookup(succeedingLookup))
	loadStoreLookupNoTTL(t, cgm, "sync-mutex")
	_ = cgm.Close()

	cgm, _ = NewSyncAtomicMap(Lookup(succeedingLookup))
	loadStoreLookupNoTTL(t, cgm, "sync-atomic")
	_ = cgm.Close()

	cgm, _ = NewSyncMutexMap(Lookup(succeedingLookup))
	loadStoreLookupNoTTL(t, cgm, "sync-mutex")
	_ = cgm.Close()

	cgm, _ = NewTwoLevelMap(Lookup(succeedingLookup))
	loadStoreLookupNoTTL(t, cgm, "twoLevel")
	_ = cgm.Close()
}

func loadStoreLookupNoTTL(t *testing.T, cgm Congomap, which string) {
	cgm.Store("hit", 42)
	loadStoreValueNil(t, cgm, which, "miss")
	loadStoreValueNil(t, cgm, which, "hit")
}

////////////////////

func TestLoadStoreNoLookupBeforeTTL(t *testing.T) {
	var cgm Congomap

	cgm, _ = NewChannelMap(TTL(time.Minute))
	loadStoreNoLookupBeforeTTL(t, cgm, "sync-mutex")
	_ = cgm.Close()

	cgm, _ = NewSyncAtomicMap(TTL(time.Minute))
	loadStoreNoLookupBeforeTTL(t, cgm, "sync-atomic")
	_ = cgm.Close()

	cgm, _ = NewSyncMutexMap(TTL(time.Minute))
	loadStoreNoLookupBeforeTTL(t, cgm, "sync-mutex")
	_ = cgm.Close()

	cgm, _ = NewTwoLevelMap(TTL(time.Minute))
	loadStoreNoLookupBeforeTTL(t, cgm, "twoLevel")
	_ = cgm.Close()
}

func loadStoreNoLookupBeforeTTL(t *testing.T, cgm Congomap, which string) {
	cgm.Store("hit", 42)
	loadStoreNilErrNoLookupDefined(t, cgm, which, "miss")
	loadStoreValueNil(t, cgm, which, "hit")
}

////

func TestLoadStoreFailingLookupBeforeTTL(t *testing.T) {
	var cgm Congomap

	cgm, _ = NewChannelMap(Lookup(failingLookup), TTL(time.Minute))
	loadStoreFailingLookupBeforeTTL(t, cgm, "sync-mutex")
	_ = cgm.Close()

	cgm, _ = NewSyncAtomicMap(Lookup(failingLookup), TTL(time.Minute))
	loadStoreFailingLookupBeforeTTL(t, cgm, "sync-atomic")
	_ = cgm.Close()

	cgm, _ = NewSyncMutexMap(Lookup(failingLookup), TTL(time.Minute))
	loadStoreFailingLookupBeforeTTL(t, cgm, "sync-mutex")
	_ = cgm.Close()

	cgm, _ = NewTwoLevelMap(Lookup(failingLookup), TTL(time.Minute))
	loadStoreFailingLookupBeforeTTL(t, cgm, "twoLevel")
	_ = cgm.Close()
}

func loadStoreFailingLookupBeforeTTL(t *testing.T, cgm Congomap, which string) {
	cgm.Store("hit", 42)
	loadStoreNilErrLookupFailed(t, cgm, which, "miss")
	loadStoreValueNil(t, cgm, which, "hit")
}

////

func TestLoadStoreLookupBeforeTTL(t *testing.T) {
	var cgm Congomap

	cgm, _ = NewChannelMap(Lookup(succeedingLookup), TTL(time.Minute))
	loadStoreLookupBeforeTTL(t, cgm, "sync-mutex")
	_ = cgm.Close()

	cgm, _ = NewSyncAtomicMap(Lookup(succeedingLookup), TTL(time.Minute))
	loadStoreLookupBeforeTTL(t, cgm, "sync-atomic")
	_ = cgm.Close()

	cgm, _ = NewSyncMutexMap(Lookup(succeedingLookup), TTL(time.Minute))
	loadStoreLookupBeforeTTL(t, cgm, "sync-mutex")
	_ = cgm.Close()

	cgm, _ = NewTwoLevelMap(Lookup(succeedingLookup), TTL(time.Minute))
	loadStoreLookupBeforeTTL(t, cgm, "twoLevel")
	_ = cgm.Close()
}

func loadStoreLookupBeforeTTL(t *testing.T, cgm Congomap, which string) {
	cgm.Store("hit", 42)
	loadStoreValueNil(t, cgm, which, "miss")
	loadStoreValueNil(t, cgm, which, "hit")
}

////////////////////

func TestLoadStoreNoLookupAfterTTL(t *testing.T) {
	var cgm Congomap

	cgm, _ = NewChannelMap(TTL(time.Nanosecond))
	loadStoreNoLookupAfterTTL(t, cgm, "sync-mutex")
	_ = cgm.Close()

	cgm, _ = NewSyncAtomicMap(TTL(time.Nanosecond))
	loadStoreNoLookupAfterTTL(t, cgm, "sync-atomic")
	_ = cgm.Close()

	cgm, _ = NewSyncMutexMap(TTL(time.Nanosecond))
	loadStoreNoLookupAfterTTL(t, cgm, "sync-mutex")
	_ = cgm.Close()

	cgm, _ = NewTwoLevelMap(TTL(time.Nanosecond))
	loadStoreNoLookupAfterTTL(t, cgm, "twoLevel")
	_ = cgm.Close()
}

func loadStoreNoLookupAfterTTL(t *testing.T, cgm Congomap, which string) {
	cgm.Store("hit", 42)
	time.Sleep(time.Millisecond)
	loadStoreNilErrNoLookupDefined(t, cgm, which, "miss")
	loadStoreNilErrNoLookupDefined(t, cgm, which, "hit")
}

////

func TestLoadStoreFailingLookupAfterTTL(t *testing.T) {
	var cgm Congomap

	cgm, _ = NewChannelMap(Lookup(failingLookup), TTL(time.Nanosecond))
	loadStoreFailingLookupAfterTTL(t, cgm, "sync-mutex")
	_ = cgm.Close()

	cgm, _ = NewSyncAtomicMap(Lookup(failingLookup), TTL(time.Nanosecond))
	loadStoreFailingLookupAfterTTL(t, cgm, "sync-atomic")
	_ = cgm.Close()

	cgm, _ = NewSyncMutexMap(Lookup(failingLookup), TTL(time.Nanosecond))
	loadStoreFailingLookupAfterTTL(t, cgm, "sync-mutex")
	_ = cgm.Close()

	cgm, _ = NewTwoLevelMap(Lookup(failingLookup), TTL(time.Nanosecond))
	loadStoreFailingLookupAfterTTL(t, cgm, "twoLevel")
	_ = cgm.Close()
}

func loadStoreFailingLookupAfterTTL(t *testing.T, cgm Congomap, which string) {
	cgm.Store("hit", 42)
	time.Sleep(time.Millisecond)
	loadStoreNilErrLookupFailed(t, cgm, which, "miss")
	loadStoreNilErrLookupFailed(t, cgm, which, "hit")
}

////

func TestLoadStoreLookupAfterTTL(t *testing.T) {
	var cgm Congomap

	cgm, _ = NewChannelMap(Lookup(succeedingLookup), TTL(time.Nanosecond))
	loadStoreLookupAfterTTL(t, cgm, "sync-mutex")
	_ = cgm.Close()

	cgm, _ = NewSyncAtomicMap(Lookup(succeedingLookup), TTL(time.Nanosecond))
	loadStoreLookupAfterTTL(t, cgm, "sync-atomic")
	_ = cgm.Close()

	cgm, _ = NewSyncMutexMap(Lookup(succeedingLookup), TTL(time.Nanosecond))
	loadStoreLookupAfterTTL(t, cgm, "sync-mutex")
	_ = cgm.Close()

	cgm, _ = NewTwoLevelMap(Lookup(succeedingLookup), TTL(time.Nanosecond))
	loadStoreLookupAfterTTL(t, cgm, "twoLevel")
	_ = cgm.Close()
}

func loadStoreLookupAfterTTL(t *testing.T, cgm Congomap, which string) {
	cgm.Store("hit", 42)
	time.Sleep(time.Millisecond)
	loadStoreValueNil(t, cgm, which, "miss")
	loadStoreValueNil(t, cgm, which, "hit")
}

////////////////////////////////////////
// Pairs()

func TestPairs(t *testing.T) {
	test := func(t *testing.T, cgm Congomap, which string) {
		cgm.Store("first", "Clark")
		cgm.Store("last", "Kent")
		for pair := range cgm.Pairs() {
			if _, ok := pair.Value.(string); !ok {
				t.Errorf("Actual: %#v; Expected: %#v", ok, true)
			}
		}
	}

	var cgm Congomap

	cgm, _ = NewChannelMap()
	test(t, cgm, "sync-atomic")
	_ = cgm.Close()

	cgm, _ = NewSyncMutexMap()
	test(t, cgm, "sync-mutex")
	_ = cgm.Close()

	cgm, _ = NewSyncAtomicMap()
	test(t, cgm, "sync-atomic")
	_ = cgm.Close()

	cgm, _ = NewTwoLevelMap()
	test(t, cgm, "twoLevel")
	_ = cgm.Close()
}

////////////////////////////////////////
// Reaper()

func TestReaperInvokedDuringDelete(t *testing.T) {
	expected := 42
	var which string
	var wg sync.WaitGroup

	reaper := func(value interface{}) {
		if v, ok := value.(int); !ok || v != expected {
			t.Errorf("reaper receives value during delete; Which: %s; Key: %q; Actual: %#v; Expected: %#v", which, value, expected)
		}
		wg.Done()
	}

	test := func(cgm Congomap, w string) {
		defer cgm.Close()

		which = w
		cgm.Store("hit", 42)

		wg.Add(1)
		cgm.Delete("hit")
		wg.Wait()
	}

	var cgm Congomap

	cgm, _ = NewChannelMap(Reaper(reaper))
	test(cgm, "channel")

	cgm, _ = NewSyncAtomicMap(Reaper(reaper))
	test(cgm, "sync-atomic")

	cgm, _ = NewSyncMutexMap(Reaper(reaper))
	test(cgm, "sync-mutex")

	cgm, _ = NewTwoLevelMap(Reaper(reaper))
	test(cgm, "two-level")
}

func TestReaperInvokedDuringGC(t *testing.T) {
	expected := 42
	var which string
	var wg sync.WaitGroup

	reaper := func(value interface{}) {
		if v, ok := value.(int); !ok || v != expected {
			t.Errorf("reaper receives value during delete; Which: %s; Actual: %#v; Expected: %#v", which, value, expected)
		}
		wg.Done()
	}

	test := func(cgm Congomap, w string) {
		defer cgm.Close()

		which = w
		cgm.Store("hit", 42)
		time.Sleep(time.Millisecond)

		wg.Add(1)
		cgm.GC()
		wg.Wait()
	}

	var cgm Congomap

	cgm, _ = NewChannelMap(TTL(time.Nanosecond), Reaper(reaper))
	test(cgm, "channel")

	cgm, _ = NewSyncAtomicMap(TTL(time.Nanosecond), Reaper(reaper))
	test(cgm, "sync-atomic")

	cgm, _ = NewSyncMutexMap(TTL(time.Nanosecond), Reaper(reaper))
	test(cgm, "sync-mutex")

	cgm, _ = NewTwoLevelMap(TTL(time.Nanosecond), Reaper(reaper))
	test(cgm, "two-level")
}

func TestReaperInvokedDuringClose(t *testing.T) {
	expected := 42
	var wg sync.WaitGroup
	var which string

	reaper := func(value interface{}) {
		if v, ok := value.(int); !ok || v != expected {
			t.Errorf("reaper receives value during delete; Which: %s; Actual: %#v; Expected: %#v", which, value, expected)
		}
		wg.Done()
	}

	test := func(cgm Congomap, w string) {
		which = w
		cgm.Store("hit", 42)
		time.Sleep(time.Millisecond)

		wg.Add(1)
		_ = cgm.Close()
		wg.Wait()
	}

	var cgm Congomap

	cgm, _ = NewChannelMap(Reaper(reaper))
	test(cgm, "channel")

	cgm, _ = NewSyncAtomicMap(Reaper(reaper))
	test(cgm, "sync-atomic")

	cgm, _ = NewSyncMutexMap(Reaper(reaper))
	test(cgm, "sync-mutex")

	cgm, _ = NewTwoLevelMap(Reaper(reaper))
	test(cgm, "two-level")
}
