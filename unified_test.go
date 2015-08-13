package congomap

import (
	"errors"
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

////////////////////////////////////////
// Load()

func loadNilFalse(t *testing.T, cgm Congomap, which, key string) {
	value, ok := cgm.Load(key)
	if value != nil {
		t.Errorf("loadNilFalse: Which: %s; Actual: %#v; Expected: %#v", which, value, nil)
	}
	if ok != false {
		t.Errorf("loadNilFalse: Which: %s; Actual: %#v; Expected: %#v", which, ok, false)
	}
}

func loadValueTrue(t *testing.T, cgm Congomap, which, key string) {
	value, ok := cgm.Load(key)
	if value != 42 {
		t.Errorf("loadValueTrue: Which: %s; Actual: %#v; Expected: %#v", which, value, 42)
	}
	if ok != true {
		t.Errorf("loadValueTrue: Which: %s; Actual: %#v; Expected: %#v", which, ok, true)
	}
}

////////////////////

func loadNoTTL(t *testing.T, cgm Congomap, which string) {
	cgm.Store("hit", 42)
	loadNilFalse(t, cgm, which, "miss")
	loadValueTrue(t, cgm, which, "hit")
}

func TestLoadNoTTL(t *testing.T) {
	cgm, _ := NewSyncMutexMap()
	loadNoTTL(t, cgm, "sync-mutex")
	cgm.Halt()

	cgm, _ = NewSyncAtomicMap()
	loadNoTTL(t, cgm, "sync-atomic")
	cgm.Halt()

	cgm, _ = NewChannelMap()
	loadNoTTL(t, cgm, "channel")
	cgm.Halt()
}

func loadBeforeTTL(t *testing.T, cgm Congomap, which string) {
	cgm.Store("hit", 42)
	loadNilFalse(t, cgm, which, "miss")
	loadValueTrue(t, cgm, which, "hit")
}

func TestLoadBeforeTTL(t *testing.T) {
	cgm, _ := NewSyncMutexMap(TTL(time.Minute))
	loadBeforeTTL(t, cgm, "sync-mutex")
	cgm.Halt()

	cgm, _ = NewSyncAtomicMap(TTL(time.Minute))
	loadBeforeTTL(t, cgm, "sync-atomic")
	cgm.Halt()

	cgm, _ = NewChannelMap(TTL(time.Minute))
	loadBeforeTTL(t, cgm, "channel")
	cgm.Halt()
}

func loadAfterTTL(t *testing.T, cgm Congomap, which string) {
	cgm.Store("hit", 42)
	time.Sleep(time.Millisecond)
	loadNilFalse(t, cgm, which, "miss")
	loadNilFalse(t, cgm, which, "hit")
}

func TestLoadAfterTTL(t *testing.T) {
	cgm, _ := NewSyncMutexMap(TTL(time.Nanosecond))
	loadAfterTTL(t, cgm, "sync-mutex")
	cgm.Halt()

	cgm, _ = NewSyncAtomicMap(TTL(time.Nanosecond))
	loadAfterTTL(t, cgm, "sync-atomic")
	cgm.Halt()

	cgm, _ = NewChannelMap(TTL(time.Nanosecond))
	loadAfterTTL(t, cgm, "channel")
	cgm.Halt()
}

////////////////////////////////////////
// LoadStore()

func loadStoreNilErrNoLookupDefined(t *testing.T, cgm Congomap, which, key string) {
	value, err := cgm.LoadStore(key)
	if value != nil {
		t.Errorf("LoadStoreMiss: Which: %s; Actual: %#v; Expected: %#v", which, value, nil)
	}
	if _, ok := err.(ErrNoLookupDefined); err == nil || !ok {
		t.Errorf("LoadStoreMiss: Which: %s; Actual: %#v; Expected: %#v", which, err, ErrNoLookupDefined{})
	}
}

func loadStoreNilErrLookupFailed(t *testing.T, cgm Congomap, which, key string) {
	value, err := cgm.LoadStore(key)
	if value != nil {
		t.Errorf("LoadStoreMiss: Which: %s; Actual: %#v; Expected: %#v", which, value, nil)
	}
	if err == nil || err.Error() != "lookup failed" {
		t.Errorf("LoadStoreMiss: Which: %s; Actual: %#v; Expected: %#v", which, err, errLookupFailed)
	}
}

func loadStoreValueNil(t *testing.T, cgm Congomap, which, key string) {
	value, err := cgm.LoadStore(key)
	if value != 42 {
		t.Errorf("LoadStoreHitNoTTL: Which: %s; Actual: %#v; Expected: %#v", which, value, 42)
	}
	if err != nil {
		t.Errorf("LoadStoreHitNoTTL: Which: %s; Actual: %#v; Expected: %#v", which, err, nil)
	}
}

////////////////////

func TestLoadStoreNoLookupNoTTL(t *testing.T) {
	cgm, _ := NewSyncMutexMap()
	loadStoreNoLookupNoTTL(t, cgm, "sync-mutex")
	cgm.Halt()

	cgm, _ = NewSyncAtomicMap()
	loadStoreNoLookupNoTTL(t, cgm, "sync-atomic")
	cgm.Halt()

	cgm, _ = NewChannelMap()
	loadStoreNoLookupNoTTL(t, cgm, "channel")
	cgm.Halt()
}

func loadStoreNoLookupNoTTL(t *testing.T, cgm Congomap, which string) {
	cgm.Store("hit", 42)
	loadStoreNilErrNoLookupDefined(t, cgm, which, "miss")
	loadStoreValueNil(t, cgm, which, "hit")
}

////

func TestLoadStoreFailingLookupNoTTL(t *testing.T) {
	cgm, _ := NewSyncMutexMap(Lookup(failingLookup))
	loadStoreFailingLookupNoTTL(t, cgm, "sync-mutex")
	cgm.Halt()

	cgm, _ = NewSyncAtomicMap(Lookup(failingLookup))
	loadStoreFailingLookupNoTTL(t, cgm, "sync-atomic")
	cgm.Halt()

	cgm, _ = NewChannelMap(Lookup(failingLookup))
	loadStoreFailingLookupNoTTL(t, cgm, "channel")
	cgm.Halt()
}

func loadStoreFailingLookupNoTTL(t *testing.T, cgm Congomap, which string) {
	cgm.Store("hit", 42)
	loadStoreNilErrLookupFailed(t, cgm, which, "miss")
	loadStoreValueNil(t, cgm, which, "hit")
}

////

func TestLoadStoreLookupNoTTL(t *testing.T) {
	cgm, _ := NewSyncMutexMap(Lookup(succeedingLookup))
	loadStoreLookupNoTTL(t, cgm, "sync-mutex")
	cgm.Halt()

	cgm, _ = NewSyncAtomicMap(Lookup(succeedingLookup))
	loadStoreLookupNoTTL(t, cgm, "sync-atomic")
	cgm.Halt()

	cgm, _ = NewChannelMap(Lookup(succeedingLookup))
	loadStoreLookupNoTTL(t, cgm, "channel")
	cgm.Halt()
}

func loadStoreLookupNoTTL(t *testing.T, cgm Congomap, which string) {
	cgm.Store("hit", 42)
	loadStoreValueNil(t, cgm, which, "miss")
	loadStoreValueNil(t, cgm, which, "hit")
}

////////////////////

func TestLoadStoreNoLookupBeforeTTL(t *testing.T) {
	cgm, _ := NewSyncMutexMap(TTL(time.Minute))
	loadStoreNoLookupBeforeTTL(t, cgm, "sync-mutex")
	cgm.Halt()

	cgm, _ = NewSyncAtomicMap(TTL(time.Minute))
	loadStoreNoLookupBeforeTTL(t, cgm, "sync-atomic")
	cgm.Halt()

	cgm, _ = NewChannelMap(TTL(time.Minute))
	loadStoreNoLookupBeforeTTL(t, cgm, "channel")
	cgm.Halt()
}

func loadStoreNoLookupBeforeTTL(t *testing.T, cgm Congomap, which string) {
	cgm.Store("hit", 42)
	loadStoreNilErrNoLookupDefined(t, cgm, which, "miss")
	loadStoreValueNil(t, cgm, which, "hit")
}

////

func TestLoadStoreFailingLookupBeforeTTL(t *testing.T) {
	cgm, _ := NewSyncMutexMap(Lookup(failingLookup), TTL(time.Minute))
	loadStoreFailingLookupBeforeTTL(t, cgm, "sync-mutex")
	cgm.Halt()

	cgm, _ = NewSyncAtomicMap(Lookup(failingLookup), TTL(time.Minute))
	loadStoreFailingLookupBeforeTTL(t, cgm, "sync-atomic")
	cgm.Halt()

	cgm, _ = NewChannelMap(Lookup(failingLookup), TTL(time.Minute))
	loadStoreFailingLookupBeforeTTL(t, cgm, "channel")
	cgm.Halt()
}

func loadStoreFailingLookupBeforeTTL(t *testing.T, cgm Congomap, which string) {
	cgm.Store("hit", 42)
	loadStoreNilErrLookupFailed(t, cgm, which, "miss")
	loadStoreValueNil(t, cgm, which, "hit")
}

////

func TestLoadStoreLookupBeforeTTL(t *testing.T) {
	cgm, _ := NewSyncMutexMap(Lookup(succeedingLookup), TTL(time.Minute))
	loadStoreLookupBeforeTTL(t, cgm, "sync-mutex")
	cgm.Halt()

	cgm, _ = NewSyncAtomicMap(Lookup(succeedingLookup), TTL(time.Minute))
	loadStoreLookupBeforeTTL(t, cgm, "sync-atomic")
	cgm.Halt()

	cgm, _ = NewChannelMap(Lookup(succeedingLookup), TTL(time.Minute))
	loadStoreLookupBeforeTTL(t, cgm, "channel")
	cgm.Halt()
}

func loadStoreLookupBeforeTTL(t *testing.T, cgm Congomap, which string) {
	cgm.Store("hit", 42)
	loadStoreValueNil(t, cgm, which, "miss")
	loadStoreValueNil(t, cgm, which, "hit")
}

////////////////////

func TestLoadStoreNoLookupAfterTTL(t *testing.T) {
	cgm, _ := NewSyncMutexMap(TTL(time.Nanosecond))
	loadStoreNoLookupAfterTTL(t, cgm, "sync-mutex")
	cgm.Halt()

	cgm, _ = NewSyncAtomicMap(TTL(time.Nanosecond))
	loadStoreNoLookupAfterTTL(t, cgm, "sync-atomic")
	cgm.Halt()

	cgm, _ = NewChannelMap(TTL(time.Nanosecond))
	loadStoreNoLookupAfterTTL(t, cgm, "channel")
	cgm.Halt()
}

func loadStoreNoLookupAfterTTL(t *testing.T, cgm Congomap, which string) {
	cgm.Store("hit", 42)
	time.Sleep(time.Millisecond)
	loadStoreNilErrNoLookupDefined(t, cgm, which, "miss")
	loadStoreNilErrNoLookupDefined(t, cgm, which, "hit")
}

////

func TestLoadStoreFailingLookupAfterTTL(t *testing.T) {
	cgm, _ := NewSyncMutexMap(Lookup(failingLookup), TTL(time.Nanosecond))
	loadStoreFailingLookupAfterTTL(t, cgm, "sync-mutex")
	cgm.Halt()

	cgm, _ = NewSyncAtomicMap(Lookup(failingLookup), TTL(time.Nanosecond))
	loadStoreFailingLookupAfterTTL(t, cgm, "sync-atomic")
	cgm.Halt()

	cgm, _ = NewChannelMap(Lookup(failingLookup), TTL(time.Nanosecond))
	loadStoreFailingLookupAfterTTL(t, cgm, "channel")
	cgm.Halt()
}

func loadStoreFailingLookupAfterTTL(t *testing.T, cgm Congomap, which string) {
	cgm.Store("hit", 42)
	time.Sleep(time.Millisecond)
	loadStoreNilErrLookupFailed(t, cgm, which, "miss")
	loadStoreNilErrLookupFailed(t, cgm, which, "hit")
}

////

func TestLoadStoreLookupAfterTTL(t *testing.T) {
	cgm, _ := NewSyncMutexMap(Lookup(succeedingLookup), TTL(time.Nanosecond))
	loadStoreLookupAfterTTL(t, cgm, "sync-mutex")
	cgm.Halt()

	cgm, _ = NewSyncAtomicMap(Lookup(succeedingLookup), TTL(time.Nanosecond))
	loadStoreLookupAfterTTL(t, cgm, "sync-atomic")
	cgm.Halt()

	cgm, _ = NewChannelMap(Lookup(succeedingLookup), TTL(time.Nanosecond))
	loadStoreLookupAfterTTL(t, cgm, "channel")
	cgm.Halt()
}

func loadStoreLookupAfterTTL(t *testing.T, cgm Congomap, which string) {
	cgm.Store("hit", 42)
	time.Sleep(time.Millisecond)
	loadStoreValueNil(t, cgm, which, "miss")
	loadStoreValueNil(t, cgm, which, "hit")
}
