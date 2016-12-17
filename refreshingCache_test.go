package congomap_test

import (
	"sync/atomic"
	"testing"
	"time"

	"github.com/karrick/congomap"
)

func loadStoreNilErrNoLookupDefinedRefreshingCache(t *testing.T, cgm *congomap.RefreshingCache, which, key string) {
	value, err := cgm.LoadStore(key)
	if value != nil {
		t.Errorf("LoadStoreMiss: Which: %s; Key: %q; Actual: %#v; Expected: %#v", which, key, value, nil)
	}
	if _, ok := err.(congomap.ErrNoLookupDefined); err == nil || !ok {
		t.Errorf("LoadStoreMiss: Which: %s; Key: %q; Actual: %#v; Expected: %#v", which, key, err, congomap.ErrNoLookupDefined{})
	}
}

func loadStoreNilErrLookupFailedRefreshingCache(t *testing.T, cgm *congomap.RefreshingCache, which, key string) {
	value, err := cgm.LoadStore(key)
	if value != nil {
		t.Errorf("LoadStoreMiss: Which: %s; Key: %q; Actual: %#v; Expected: %#v", which, key, value, nil)
	}
	if err == nil || err.Error() != "lookup failed" {
		t.Errorf("LoadStoreMiss: Which: %s; Key: %q; Actual: %#v; Expected: %#v", which, key, err, errLookupFailed)
	}
}

func loadStoreValueNilRefreshingCache(t *testing.T, cgm *congomap.RefreshingCache, which, key string) {
	value, err := cgm.LoadStore(key)
	if value != 42 {
		t.Errorf("LoadStoreHitNoTTL: Which: %s; Key: %q; Actual: %#v; Expected: %#v", which, key, value, 42)
	}
	if err != nil {
		t.Errorf("LoadStoreHitNoTTL: Which: %s; Key: %q; Actual: %#v; Expected: %#v", which, key, err, nil)
	}
}

func loadStoreValueNilRefreshingCacheValue(t *testing.T, cgm *congomap.RefreshingCache, which, key string, expected int64) {
	value, err := cgm.LoadStore(key)
	if value.(int64) != expected {
		t.Errorf("LoadStoreHitNoTTL: Which: %s; Key: %q; Actual: %#v; Expected: %#v", which, key, value, expected)
	}
	if err != nil {
		t.Errorf("LoadStoreHitNoTTL: Which: %s; Key: %q; Actual: %#v; Expected: %#v", which, key, err, nil)
	}
}

// LoadStoreNoLookupNoTTL

func loadStoreNoLookupNoTTLRefreshingCache(t *testing.T, cgm *congomap.RefreshingCache, which string) {
	// defer cgm.Close()
	loadStoreNilErrNoLookupDefinedRefreshingCache(t, cgm, which, "miss")
	// cgm.Store("hit", 42)
	// loadStoreValueNilRefreshingCache(t, cgm, which, "hit")
}

func TestLoadStoreNoLookupNoTTLRefreshingCache(t *testing.T) {
	cgm, _ := congomap.NewRefreshingCache(nil)
	loadStoreNoLookupNoTTLRefreshingCache(t, cgm, "refreshingCache")
}

// LoadStoreFailingLookupNoTTL

func loadStoreFailingLookupNoTTLRefreshingCache(t *testing.T, cgm *congomap.RefreshingCache, which string) {
	// defer cgm.Close()
	cgm.Store("hit", 42)
	loadStoreNilErrLookupFailedRefreshingCache(t, cgm, which, "miss")
	loadStoreValueNilRefreshingCache(t, cgm, which, "hit")
}

func TestLoadStoreFailingLookupNoTTLRefreshingCache(t *testing.T) {
	cgm, _ := congomap.NewRefreshingCache(&congomap.RefreshingCacheConfig{Lookup: failingLookup})
	loadStoreFailingLookupNoTTLRefreshingCache(t, cgm, "refreshingCache")
}

// LoadStoreLookupNoTTL

// func ExampleLookupRefreshingCache() {
// 	someLengthlyComputation := func(key string) (interface{}, error) {
// 		time.Sleep(time.Duration(rand.Intn(25))*time.Millisecond + 50*time.Millisecond)
// 		return len(key), nil
// 	}

// 	// Create a Congomap, specifying what the lookup callback function is.
// 	cgm, err := congomap.NewRefreshingCache(&congomap.RefreshingCacheConfig{Lookup: someLengthlyComputation})
// 	if err != nil {
// 		log.Fatal(err)
// 	}
// 	// defer cgm.Close()

// 	// You can still use the regular Load and Store functions, which will not invoke the lookup
// 	// function.
// 	someKey := "flubber"
// 	cgm.Store(someKey, 42)
// 	value, ok := cgm.Load(someKey)
// 	if !ok {
// 		log.Fatal(fmt.Errorf("why isn't key in map? %q", someKey))
// 	}
// 	fmt.Println(value)

// 	// When you use the LoadStore function, and the key is not in the Congomap, the lookup funciton is
// 	// invoked, and the return value is stored in the Congomap and returned to the program.
// 	someKey = "blubber"
// 	value, err = cgm.LoadStore(someKey)
// 	if err != nil {
// 		log.Fatal(fmt.Errorf("why isn't key in map? %q", someKey))
// 	}

// 	fmt.Println(value)
// 	// Output:
// 	// 42
// 	// 7
// }

func loadStoreLookupNoTTLRefreshingCache(t *testing.T, cgm *congomap.RefreshingCache, which string) {
	// defer cgm.Close()
	cgm.Store("hit", 42)
	loadStoreValueNilRefreshingCache(t, cgm, which, "miss")
	loadStoreValueNilRefreshingCache(t, cgm, which, "hit")
}

func TestLoadStoreLookupNoTTLRefreshingCache(t *testing.T) {
	cgm, _ := congomap.NewRefreshingCache(&congomap.RefreshingCacheConfig{Lookup: succeedingLookup})
	loadStoreLookupNoTTLRefreshingCache(t, cgm, "refreshingCache")
}

// LoadStoreNoLookupBeforeTTL

func loadStoreNoLookupBeforeTTLRefreshingCache(t *testing.T, cgm *congomap.RefreshingCache, which string) {
	// defer cgm.Close()
	cgm.Store("hit", 42)
	loadStoreNilErrNoLookupDefinedRefreshingCache(t, cgm, which, "miss")
	loadStoreValueNilRefreshingCache(t, cgm, which, "hit")
}

func TestLoadStoreNoLookupBeforeTTLRefreshingCache(t *testing.T) {
	cgm, _ := congomap.NewRefreshingCache(&congomap.RefreshingCacheConfig{GoodExpiryDuration: time.Minute})
	loadStoreNoLookupBeforeTTLRefreshingCache(t, cgm, "refreshingCache")
}

// LoadStoreFailingLookupBeforeTTL

func loadStoreFailingLookupBeforeTTLRefreshingCache(t *testing.T, cgm *congomap.RefreshingCache, which string) {
	// defer cgm.Close()
	cgm.Store("hit", 42)
	loadStoreNilErrLookupFailedRefreshingCache(t, cgm, which, "miss")
	loadStoreValueNilRefreshingCache(t, cgm, which, "hit")
}

func TestLoadStoreFailingLookupBeforeTTLRefreshingCache(t *testing.T) {
	cgm, _ := congomap.NewRefreshingCache(&congomap.RefreshingCacheConfig{Lookup: failingLookup, GoodExpiryDuration: time.Minute})
	loadStoreFailingLookupBeforeTTLRefreshingCache(t, cgm, "refreshingCache")
}

// LoadStoreLookupBeforeTTL

func loadStoreLookupBeforeTTLRefreshingCache(t *testing.T, cgm *congomap.RefreshingCache, which string) {
	// defer cgm.Close()
	cgm.Store("hit", 42)
	loadStoreValueNilRefreshingCache(t, cgm, which, "miss")
	loadStoreValueNilRefreshingCache(t, cgm, which, "hit")
}

func TestLoadStoreLookupBeforeTTLRefreshingCache(t *testing.T) {
	cgm, _ := congomap.NewRefreshingCache(&congomap.RefreshingCacheConfig{Lookup: succeedingLookup, GoodExpiryDuration: time.Minute})
	loadStoreLookupBeforeTTLRefreshingCache(t, cgm, "refreshingCache")
}

// LoadStoreNoLookupAfterTTL

func loadStoreNoLookupAfterTTLRefreshingCache(t *testing.T, cgm *congomap.RefreshingCache, which string) {
	// defer cgm.Close()
	cgm.Store("hit", 42)
	time.Sleep(time.Millisecond)
	loadStoreNilErrNoLookupDefinedRefreshingCache(t, cgm, which, "miss")
	loadStoreNilErrNoLookupDefinedRefreshingCache(t, cgm, which, "hit")
}

func TestLoadStoreNoLookupAfterTTLRefreshingCache(t *testing.T) {
	cgm, _ := congomap.NewRefreshingCache(&congomap.RefreshingCacheConfig{GoodExpiryDuration: time.Nanosecond})
	loadStoreNoLookupAfterTTLRefreshingCache(t, cgm, "refreshingCache")
}

// LoadStoreFailingLookupAfterTTL

func loadStoreFailingLookupAfterTTLRefreshingCache(t *testing.T, cgm *congomap.RefreshingCache, which string) {
	// defer cgm.Close()
	cgm.Store("hit", 42)
	time.Sleep(time.Millisecond)
	loadStoreNilErrLookupFailedRefreshingCache(t, cgm, which, "miss")
	loadStoreNilErrLookupFailedRefreshingCache(t, cgm, which, "hit")
}

func TestLoadStoreFailingLookupAfterTTLRefreshingCache(t *testing.T) {
	cgm, _ := congomap.NewRefreshingCache(&congomap.RefreshingCacheConfig{Lookup: failingLookup, GoodExpiryDuration: time.Nanosecond})
	loadStoreFailingLookupAfterTTLRefreshingCache(t, cgm, "refreshingCache")
}

// LoadStoreLookupAfterTTL

func loadStoreLookupAfterTTLRefreshingCache(t *testing.T, cgm *congomap.RefreshingCache, which string) {
	// defer cgm.Close()
	cgm.Store("hit", 42)
	time.Sleep(time.Millisecond)
	loadStoreValueNilRefreshingCache(t, cgm, which, "miss")
	loadStoreValueNilRefreshingCache(t, cgm, which, "hit")
}

func TestLoadStoreLookupAfterTTLRefreshingCache(t *testing.T) {
	cgm, _ := congomap.NewRefreshingCache(&congomap.RefreshingCacheConfig{Lookup: succeedingLookup, GoodExpiryDuration: time.Nanosecond})
	loadStoreLookupAfterTTLRefreshingCache(t, cgm, "refreshingCache")
}

////////////////////////////////////////
// NEW TESTS
////////////////////////////////////////

func TestRefreshingCacheNoStaleNoExpiry(t *testing.T) {
	which := "refreshingCache"
	cgm, err := congomap.NewRefreshingCache(&congomap.RefreshingCacheConfig{Lookup: succeedingLookup})
	if err != nil {
		t.Fatal(err)
	}
	cgm.Store("hit", 42)

	loadStoreValueNilRefreshingCache(t, cgm, which, "miss")
	loadStoreValueNilRefreshingCache(t, cgm, which, "hit")
}

func TestRefreshingCacheNoStaleExpiry(t *testing.T) {
	which := "refreshingCache"
	cgm, err := congomap.NewRefreshingCache(&congomap.RefreshingCacheConfig{
		Lookup:             succeedingLookup,
		GoodExpiryDuration: time.Millisecond,
	})
	if err != nil {
		t.Fatal(err)
	}
	cgm.Store("hit", 42)

	// hit, before expire
	loadStoreValueNilRefreshingCache(t, cgm, which, "hit")
	// hit, after expire
	time.Sleep(2 * time.Millisecond)
	loadStoreValueNilRefreshingCache(t, cgm, which, "hit")
	// miss
	loadStoreValueNilRefreshingCache(t, cgm, which, "miss")
}

func TestRefreshingCacheStaleNoExpiry(t *testing.T) {
	which := "refreshingCache"
	cgm, err := congomap.NewRefreshingCache(&congomap.RefreshingCacheConfig{
		Lookup:            succeedingLookup,
		GoodStaleDuration: time.Millisecond,
	})
	if err != nil {
		t.Fatal(err)
	}
	cgm.Store("hit", 42)

	// hit, before stale
	loadStoreValueNilRefreshingCache(t, cgm, which, "hit")
	// hit, after stale
	time.Sleep(2 * time.Millisecond)
	loadStoreValueNilRefreshingCache(t, cgm, which, "hit")
	// miss
	loadStoreValueNilRefreshingCache(t, cgm, which, "miss")
}

func TestRefreshingCacheStaleExpiry(t *testing.T) {
	var lookupCount int64
	succeedingLookup := func(_ string) (interface{}, error) {
		return atomic.AddInt64(&lookupCount, 1), nil
	}

	cgm, err := congomap.NewRefreshingCache(&congomap.RefreshingCacheConfig{
		Lookup:             succeedingLookup,
		GoodStaleDuration:  2 * time.Millisecond,
		GoodExpiryDuration: 3 * time.Millisecond,
	})
	if err != nil {
		t.Fatal(err)
	}
	cgm.Store("hit", int64(42))

	// miss
	loadStoreValueNilRefreshingCacheValue(t, cgm, "never stored", "miss", 1)
	// hit, before stale
	loadStoreValueNilRefreshingCacheValue(t, cgm, "before stale", "hit", 42)
	// hit, after stale & before expire
	time.Sleep(2 * time.Millisecond)
	loadStoreValueNilRefreshingCacheValue(t, cgm, "after stale, before expire", "hit", 42)

	// when wait a bit, the value should be filled in by lookup
	time.Sleep(1 * time.Millisecond)
	loadStoreValueNilRefreshingCacheValue(t, cgm, "after stale refresh", "hit", 2)

	// hit, after expire
	time.Sleep(4 * time.Millisecond)
	loadStoreValueNilRefreshingCacheValue(t, cgm, "after expire", "hit", 3)
}
