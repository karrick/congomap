package congomap_test

import (
	"errors"
	"fmt"
	"log"
	"math/rand"
	"sort"
	"sync"
	"testing"
	"time"

	"github.com/karrick/congomap"
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

func ExampleTwoLevelMap_Load() {
	cgm, err := congomap.NewTwoLevelMap()
	if err != nil {
		panic(err)
	}
	defer func() { _ = cgm.Close() }()

	// you can store any Go type in a Congomap
	cgm.Store("someKeyString", 42)
	cgm.Store("anotherKey", struct{}{})
	cgm.Store("yetAnotherKey", make(chan interface{}))

	// but when you retrieve it, you are responsible to perform type assertions
	key := "someKeyString"
	value, ok := cgm.Load(key)
	if !ok {
		panic(fmt.Errorf("cannot find %q", key))
	}

	fmt.Println(value)
	// Output: 42
}

func loadNilFalse(t *testing.T, cgm congomap.Congomap, which, key string) {
	// t.Logf("Which: %q; Key: %q", which, key)
	value, ok := cgm.Load(key)
	if value != nil {
		t.Errorf("loadNilFalse: Which: %s; Key: %q; Actual: %#v; Expected: %#v", which, key, value, nil)
	}
	if ok != false {
		t.Errorf("loadNilFalse: Which: %s; Key: %q; Actual: %#v; Expected: %#v", which, key, ok, false)
	}
}

func loadValueTrue(t *testing.T, cgm congomap.Congomap, which, key string) {
	value, ok := cgm.Load(key)
	if value != 42 {
		t.Errorf("loadValueTrue: Which: %s; Key: %q; Actual: %#v; Expected: %#v", which, key, value, 42)
	}
	if ok != true {
		t.Errorf("loadValueTrue: Which: %s; Key: %q; Actual: %#v; Expected: %#v", which, key, ok, true)
	}
}

// LoadWithoutTTL

func loadNoTTL(t *testing.T, cgm congomap.Congomap, which string) {
	defer func() { _ = cgm.Close() }()
	cgm.Store("hit", 42)
	loadNilFalse(t, cgm, which, "miss")
	loadValueTrue(t, cgm, which, "hit")
}

func TestLoadWithoutTTLChannelMap(t *testing.T) {
	cgm, _ := congomap.NewChannelMap()
	loadNoTTL(t, cgm, "channel")
}

func TestLoadWithoutTTLSyncAtomicMap(t *testing.T) {
	cgm, _ := congomap.NewSyncAtomicMap()
	loadNoTTL(t, cgm, "syncAtomic")
}

func TestLoadWithoutTTLSyncMutexMap(t *testing.T) {
	cgm, _ := congomap.NewSyncMutexMap()
	loadNoTTL(t, cgm, "syncMutex")
}

func TestLoadWithoutTTLTwoLevelMap(t *testing.T) {
	cgm, _ := congomap.NewTwoLevelMap()
	loadNoTTL(t, cgm, "twoLevel")
}

// LoadBeforeTTL

func ExampleTTL_1() {
	// Create a Congomap, specifying what the default TTL is.
	cgm, err := congomap.NewTwoLevelMap(congomap.TTL(5 * time.Minute))
	if err != nil {
		log.Fatal(err)
	}
	defer func() { _ = cgm.Close() }()

	// While the below key-value pair will expire 5 minutes from now, there is no guarantee when
	// the Reaper, if declared, would be called.
	cgm.Store("someKey", 42)
}

func ExampleTTL_2() {
	// In this example, no default TTL is defined, so values will never expire by default. A
	// particular value can still have an expiry and will be invalidated after its particular
	// expiry passes.
	cgm, err := congomap.NewTwoLevelMap()
	if err != nil {
		log.Fatal(err)
	}
	defer func() { _ = cgm.Close() }()

	key := "someKey"

	cgm.Store(key, &congomap.ExpiringValue{Value: 42, Expiry: time.Now().Add(time.Millisecond)})
	time.Sleep(2 * time.Millisecond)

	_, ok := cgm.Load(key)
	if !ok {
		fmt.Printf("cannot find key %q", key)
	}

	// At this point, the Load would have triggered the Reaper function, if it was declared when
	// the Congomap was created. Furthermore, value will be nil, and ok will be false.

	// Output:
	// cannot find key "someKey"
}

func loadBeforeTTL(t *testing.T, cgm congomap.Congomap, which string) {
	defer func() { _ = cgm.Close() }()
	cgm.Store("hit", 42)
	loadNilFalse(t, cgm, which, "miss")
	loadValueTrue(t, cgm, which, "hit")
}

func TestLoadBeforeTTLChannelMap(t *testing.T) {
	cgm, _ := congomap.NewChannelMap(congomap.TTL(time.Minute))
	loadBeforeTTL(t, cgm, "channel")
}

func TestLoadBeforeTTLSyncAtomic(t *testing.T) {
	cgm, _ := congomap.NewSyncAtomicMap(congomap.TTL(time.Minute))
	loadBeforeTTL(t, cgm, "syncAtomic")
}

func TestLoadBeforeTTLSyncMutex(t *testing.T) {
	cgm, _ := congomap.NewSyncMutexMap(congomap.TTL(time.Minute))
	loadBeforeTTL(t, cgm, "syncMutex")
}

func TestLoadBeforeTTLTwoLevel(t *testing.T) {
	cgm, _ := congomap.NewTwoLevelMap(congomap.TTL(time.Minute))
	loadBeforeTTL(t, cgm, "twoLevel")
}

// LoadAfterTTL

func loadAfterTTL(t *testing.T, cgm congomap.Congomap, which string) {
	defer func() { _ = cgm.Close() }()
	cgm.Store("hit", 42)
	time.Sleep(time.Millisecond)
	loadNilFalse(t, cgm, which, "miss")
	loadNilFalse(t, cgm, which, "hit")
}

func TestLoadAfterTTLChannelMap(t *testing.T) {
	cgm, _ := congomap.NewChannelMap(congomap.TTL(time.Nanosecond))
	loadAfterTTL(t, cgm, "channel")
}

func TestLoadAfterTTLSyncAtomicMap(t *testing.T) {
	cgm, _ := congomap.NewSyncAtomicMap(congomap.TTL(time.Nanosecond))
	loadAfterTTL(t, cgm, "syncAtomic")
}

func TestLoadAfterTTLSyncMutexMap(t *testing.T) {
	cgm, _ := congomap.NewSyncMutexMap(congomap.TTL(time.Nanosecond))
	loadAfterTTL(t, cgm, "syncMutex")
}

func TestLoadAfterTTLTwoLevelMap(t *testing.T) {
	cgm, _ := congomap.NewTwoLevelMap(congomap.TTL(time.Nanosecond))
	loadAfterTTL(t, cgm, "twoLevel")
}

////////////////////////////////////////
// LoadStore()

func ExampleTwoLevelMap_LoadStore() {
	someLengthlyComputation := func(key string) (interface{}, error) {
		time.Sleep(time.Duration(rand.Intn(25))*time.Millisecond + 50*time.Millisecond)
		return len(key), nil
	}

	// Create a Congomap, specifying what the lookup callback function is.
	cgm, err := congomap.NewTwoLevelMap(congomap.Lookup(someLengthlyComputation))
	if err != nil {
		log.Fatal(err)
	}
	defer func() { _ = cgm.Close() }()

	// You can still use the regular Load and Store functions, which will not invoke the lookup
	// function.
	someKey := "flubber"
	cgm.Store(someKey, 42)
	value, ok := cgm.Load(someKey)
	if !ok {
		log.Fatal(fmt.Errorf("why isn't key in map? %q", someKey))
	}
	fmt.Println(value)

	// When you use the LoadStore function, and the key is not in the Congomap, the lookup funciton is
	// invoked, and the return value is stored in the Congomap and returned to the program.
	someKey = "blubber"
	value, err = cgm.LoadStore(someKey)
	if err != nil {
		log.Fatal(fmt.Errorf("why isn't key in map? %q", someKey))
	}

	fmt.Println(value)
	// Output:
	// 42
	// 7
}

func loadStoreNilErrNoLookupDefined(t *testing.T, cgm congomap.Congomap, which, key string) {
	value, err := cgm.LoadStore(key)
	if value != nil {
		t.Errorf("LoadStoreMiss: Which: %s; Key: %q; Actual: %#v; Expected: %#v", which, key, value, nil)
	}
	if _, ok := err.(congomap.ErrNoLookupDefined); err == nil || !ok {
		t.Errorf("LoadStoreMiss: Which: %s; Key: %q; Actual: %#v; Expected: %#v", which, key, err, congomap.ErrNoLookupDefined{})
	}
}

func loadStoreNilErrLookupFailed(t *testing.T, cgm congomap.Congomap, which, key string) {
	value, err := cgm.LoadStore(key)
	if value != nil {
		t.Errorf("LoadStoreMiss: Which: %s; Key: %q; Actual: %#v; Expected: %#v", which, key, value, nil)
	}
	if err == nil || err.Error() != "lookup failed" {
		t.Errorf("LoadStoreMiss: Which: %s; Key: %q; Actual: %#v; Expected: %#v", which, key, err, errLookupFailed)
	}
}

func loadStoreValueNil(t *testing.T, cgm congomap.Congomap, which, key string) {
	value, err := cgm.LoadStore(key)
	if value != 42 {
		t.Errorf("LoadStoreHitNoTTL: Which: %s; Key: %q; Actual: %#v; Expected: %#v", which, key, value, 42)
	}
	if err != nil {
		t.Errorf("LoadStoreHitNoTTL: Which: %s; Key: %q; Actual: %#v; Expected: %#v", which, key, err, nil)
	}
}

// LoadStoreNoLookupNoTTL

func loadStoreNoLookupNoTTL(t *testing.T, cgm congomap.Congomap, which string) {
	defer func() { _ = cgm.Close() }()
	cgm.Store("hit", 42)
	loadStoreNilErrNoLookupDefined(t, cgm, which, "miss")
	loadStoreValueNil(t, cgm, which, "hit")
}

func TestLoadStoreNoLookupNoTTLChannelMap(t *testing.T) {
	cgm, _ := congomap.NewChannelMap()
	loadStoreNoLookupNoTTL(t, cgm, "channel")
}

func TestLoadStoreNoLookupNoTTLSyncAtomicMap(t *testing.T) {
	cgm, _ := congomap.NewSyncAtomicMap()
	loadStoreNoLookupNoTTL(t, cgm, "syncAtomic")
}

func TestLoadStoreNoLookupNoTTLSyncMutexMap(t *testing.T) {
	cgm, _ := congomap.NewSyncMutexMap()
	loadStoreNoLookupNoTTL(t, cgm, "syncMutex")
}

func TestLoadStoreNoLookupNoTTLTwoLevelMap(t *testing.T) {
	cgm, _ := congomap.NewTwoLevelMap()
	loadStoreNoLookupNoTTL(t, cgm, "twoLevel")
}

// LoadStoreFailingLookupNoTTL

func loadStoreFailingLookupNoTTL(t *testing.T, cgm congomap.Congomap, which string) {
	defer func() { _ = cgm.Close() }()
	cgm.Store("hit", 42)
	loadStoreNilErrLookupFailed(t, cgm, which, "miss")
	loadStoreValueNil(t, cgm, which, "hit")
}

func TestLoadStoreFailingLookupNoTTLChannelMap(t *testing.T) {
	cgm, _ := congomap.NewChannelMap(congomap.Lookup(failingLookup))
	loadStoreFailingLookupNoTTL(t, cgm, "channel")
}

func TestLoadStoreFailingLookupNoTTLSyncAtomicMap(t *testing.T) {
	cgm, _ := congomap.NewSyncAtomicMap(congomap.Lookup(failingLookup))
	loadStoreFailingLookupNoTTL(t, cgm, "syncAtomic")
}

func TestLoadStoreFailingLookupNoTTLSyncMutexMap(t *testing.T) {
	cgm, _ := congomap.NewSyncMutexMap(congomap.Lookup(failingLookup))
	loadStoreFailingLookupNoTTL(t, cgm, "syncMutex")
}

func TestLoadStoreFailingLookupNoTTLTwoLevelMap(t *testing.T) {
	cgm, _ := congomap.NewTwoLevelMap(congomap.Lookup(failingLookup))
	loadStoreFailingLookupNoTTL(t, cgm, "twoLevel")
}

// LoadStoreLookupNoTTL

func ExampleLookup() {
	someLengthlyComputation := func(key string) (interface{}, error) {
		time.Sleep(time.Duration(rand.Intn(25))*time.Millisecond + 50*time.Millisecond)
		return len(key), nil
	}

	// Create a Congomap, specifying what the lookup callback function is.
	cgm, err := congomap.NewTwoLevelMap(congomap.Lookup(someLengthlyComputation))
	if err != nil {
		log.Fatal(err)
	}
	defer func() { _ = cgm.Close() }()

	// You can still use the regular Load and Store functions, which will not invoke the lookup
	// function.
	someKey := "flubber"
	cgm.Store(someKey, 42)
	value, ok := cgm.Load(someKey)
	if !ok {
		log.Fatal(fmt.Errorf("why isn't key in map? %q", someKey))
	}
	fmt.Println(value)

	// When you use the LoadStore function, and the key is not in the Congomap, the lookup funciton is
	// invoked, and the return value is stored in the Congomap and returned to the program.
	someKey = "blubber"
	value, err = cgm.LoadStore(someKey)
	if err != nil {
		log.Fatal(fmt.Errorf("why isn't key in map? %q", someKey))
	}

	fmt.Println(value)
	// Output:
	// 42
	// 7
}

func loadStoreLookupNoTTL(t *testing.T, cgm congomap.Congomap, which string) {
	defer func() { _ = cgm.Close() }()
	cgm.Store("hit", 42)
	loadStoreValueNil(t, cgm, which, "miss")
	loadStoreValueNil(t, cgm, which, "hit")
}

func TestLoadStoreLookupNoTTLChannelMap(t *testing.T) {
	cgm, _ := congomap.NewChannelMap(congomap.Lookup(succeedingLookup))
	loadStoreLookupNoTTL(t, cgm, "channel")
}

func TestLoadStoreLookupNoTTLSyncAtomicMap(t *testing.T) {
	cgm, _ := congomap.NewSyncAtomicMap(congomap.Lookup(succeedingLookup))
	loadStoreLookupNoTTL(t, cgm, "syncAtomic")
}

func TestLoadStoreLookupNoTTLSyncMutexMap(t *testing.T) {
	cgm, _ := congomap.NewSyncMutexMap(congomap.Lookup(succeedingLookup))
	loadStoreLookupNoTTL(t, cgm, "syncMutex")
}

func TestLoadStoreLookupNoTTLTwoLevelMap(t *testing.T) {
	cgm, _ := congomap.NewTwoLevelMap(congomap.Lookup(succeedingLookup))
	loadStoreLookupNoTTL(t, cgm, "twoLevel")
}

// LoadStoreNoLookupBeforeTTL

func loadStoreNoLookupBeforeTTL(t *testing.T, cgm congomap.Congomap, which string) {
	defer func() { _ = cgm.Close() }()
	cgm.Store("hit", 42)
	loadStoreNilErrNoLookupDefined(t, cgm, which, "miss")
	loadStoreValueNil(t, cgm, which, "hit")
}

func TestLoadStoreNoLookupBeforeTTLChannelMap(t *testing.T) {
	cgm, _ := congomap.NewChannelMap(congomap.TTL(time.Minute))
	loadStoreNoLookupBeforeTTL(t, cgm, "channel")
}

func TestLoadStoreNoLookupBeforeTTLSyncAtomicMap(t *testing.T) {
	cgm, _ := congomap.NewSyncAtomicMap(congomap.TTL(time.Minute))
	loadStoreNoLookupBeforeTTL(t, cgm, "syncAtomic")
}

func TestLoadStoreNoLookupBeforeTTLSyncMutexMap(t *testing.T) {
	cgm, _ := congomap.NewSyncMutexMap(congomap.TTL(time.Minute))
	loadStoreNoLookupBeforeTTL(t, cgm, "syncMutex")
}

func TestLoadStoreNoLookupBeforeTTLTwoLevelMap(t *testing.T) {
	cgm, _ := congomap.NewTwoLevelMap(congomap.TTL(time.Minute))
	loadStoreNoLookupBeforeTTL(t, cgm, "twoLevel")
}

// LoadStoreFailingLookupBeforeTTL

func loadStoreFailingLookupBeforeTTL(t *testing.T, cgm congomap.Congomap, which string) {
	defer func() { _ = cgm.Close() }()
	cgm.Store("hit", 42)
	loadStoreNilErrLookupFailed(t, cgm, which, "miss")
	loadStoreValueNil(t, cgm, which, "hit")
}

func TestLoadStoreFailingLookupBeforeTTLChannelMap(t *testing.T) {
	cgm, _ := congomap.NewChannelMap(congomap.Lookup(failingLookup), congomap.TTL(time.Minute))
	loadStoreFailingLookupBeforeTTL(t, cgm, "channel")
}

func TestLoadStoreFailingLookupBeforeTTLSyncAtomicMap(t *testing.T) {
	cgm, _ := congomap.NewSyncAtomicMap(congomap.Lookup(failingLookup), congomap.TTL(time.Minute))
	loadStoreFailingLookupBeforeTTL(t, cgm, "syncAtomic")
}

func TestLoadStoreFailingLookupBeforeTTLSyncMutexMap(t *testing.T) {
	cgm, _ := congomap.NewSyncMutexMap(congomap.Lookup(failingLookup), congomap.TTL(time.Minute))
	loadStoreFailingLookupBeforeTTL(t, cgm, "syncMutex")
}

func TestLoadStoreFailingLookupBeforeTTLTwoLevelMap(t *testing.T) {
	cgm, _ := congomap.NewTwoLevelMap(congomap.Lookup(failingLookup), congomap.TTL(time.Minute))
	loadStoreFailingLookupBeforeTTL(t, cgm, "twoLevel")
}

// LoadStoreLookupBeforeTTL

func loadStoreLookupBeforeTTL(t *testing.T, cgm congomap.Congomap, which string) {
	defer func() { _ = cgm.Close() }()
	cgm.Store("hit", 42)
	loadStoreValueNil(t, cgm, which, "miss")
	loadStoreValueNil(t, cgm, which, "hit")
}

func TestLoadStoreLookupBeforeTTLChannelMap(t *testing.T) {
	cgm, _ := congomap.NewChannelMap(congomap.Lookup(succeedingLookup), congomap.TTL(time.Minute))
	loadStoreLookupBeforeTTL(t, cgm, "channel")
}

func TestLoadStoreLookupBeforeTTLSyncAtomicMap(t *testing.T) {
	cgm, _ := congomap.NewSyncAtomicMap(congomap.Lookup(succeedingLookup), congomap.TTL(time.Minute))
	loadStoreLookupBeforeTTL(t, cgm, "syncAtomic")
}

func TestLoadStoreLookupBeforeTTLSyncMutexMap(t *testing.T) {
	cgm, _ := congomap.NewSyncMutexMap(congomap.Lookup(succeedingLookup), congomap.TTL(time.Minute))
	loadStoreLookupBeforeTTL(t, cgm, "syncMutex")
}

func TestLoadStoreLookupBeforeTTLTwoLevelMap(t *testing.T) {
	cgm, _ := congomap.NewTwoLevelMap(congomap.Lookup(succeedingLookup), congomap.TTL(time.Minute))
	loadStoreLookupBeforeTTL(t, cgm, "twoLevel")
}

// LoadStoreNoLookupAfterTTL

func loadStoreNoLookupAfterTTL(t *testing.T, cgm congomap.Congomap, which string) {
	defer func() { _ = cgm.Close() }()
	cgm.Store("hit", 42)
	time.Sleep(time.Millisecond)
	loadStoreNilErrNoLookupDefined(t, cgm, which, "miss")
	loadStoreNilErrNoLookupDefined(t, cgm, which, "hit")
}

func TestLoadStoreNoLookupAfterTTLChannelMap(t *testing.T) {
	cgm, _ := congomap.NewChannelMap(congomap.TTL(time.Nanosecond))
	loadStoreNoLookupAfterTTL(t, cgm, "channel")
}

func TestLoadStoreNoLookupAfterTTLSyncAtomicMap(t *testing.T) {
	cgm, _ := congomap.NewSyncAtomicMap(congomap.TTL(time.Nanosecond))
	loadStoreNoLookupAfterTTL(t, cgm, "syncAtomic")
}

func TestLoadStoreNoLookupAfterTTLSyncMutexMap(t *testing.T) {
	cgm, _ := congomap.NewSyncMutexMap(congomap.TTL(time.Nanosecond))
	loadStoreNoLookupAfterTTL(t, cgm, "syncMutex")
}

func TestLoadStoreNoLookupAfterTTLTwoLevelMap(t *testing.T) {
	cgm, _ := congomap.NewTwoLevelMap(congomap.TTL(time.Nanosecond))
	loadStoreNoLookupAfterTTL(t, cgm, "twoLevel")
}

// LoadStoreFailingLookupAfterTTL

func loadStoreFailingLookupAfterTTL(t *testing.T, cgm congomap.Congomap, which string) {
	defer func() { _ = cgm.Close() }()
	cgm.Store("hit", 42)
	time.Sleep(time.Millisecond)
	loadStoreNilErrLookupFailed(t, cgm, which, "miss")
	loadStoreNilErrLookupFailed(t, cgm, which, "hit")
}

func TestLoadStoreFailingLookupAfterTTLChannelMap(t *testing.T) {
	cgm, _ := congomap.NewChannelMap(congomap.Lookup(failingLookup), congomap.TTL(time.Nanosecond))
	loadStoreFailingLookupAfterTTL(t, cgm, "channel")
}

func TestLoadStoreFailingLookupAfterTTLSyncAtomicMap(t *testing.T) {
	cgm, _ := congomap.NewSyncAtomicMap(congomap.Lookup(failingLookup), congomap.TTL(time.Nanosecond))
	loadStoreFailingLookupAfterTTL(t, cgm, "syncAtomic")
}

func TestLoadStoreFailingLookupAfterTTLSyncMutexMap(t *testing.T) {
	cgm, _ := congomap.NewSyncMutexMap(congomap.Lookup(failingLookup), congomap.TTL(time.Nanosecond))
	loadStoreFailingLookupAfterTTL(t, cgm, "syncMutex")
}

func TestLoadStoreFailingLookupAfterTTLTwoLevelMap(t *testing.T) {
	cgm, _ := congomap.NewTwoLevelMap(congomap.Lookup(failingLookup), congomap.TTL(time.Nanosecond))
	loadStoreFailingLookupAfterTTL(t, cgm, "twoLevel")
}

// LoadStoreLookupAfterTTL

func loadStoreLookupAfterTTL(t *testing.T, cgm congomap.Congomap, which string) {
	defer func() { _ = cgm.Close() }()
	cgm.Store("hit", 42)
	time.Sleep(time.Millisecond)
	loadStoreValueNil(t, cgm, which, "miss")
	loadStoreValueNil(t, cgm, which, "hit")
}

func TestLoadStoreLookupAfterTTLChannelMap(t *testing.T) {
	cgm, _ := congomap.NewChannelMap(congomap.Lookup(succeedingLookup), congomap.TTL(time.Nanosecond))
	loadStoreLookupAfterTTL(t, cgm, "channel")
}

func TestLoadStoreLookupAfterTTLSyncAtomicMap(t *testing.T) {
	cgm, _ := congomap.NewSyncAtomicMap(congomap.Lookup(succeedingLookup), congomap.TTL(time.Nanosecond))
	loadStoreLookupAfterTTL(t, cgm, "syncAtomic")
}

func TestLoadStoreLookupAfterTTLSyncMutexMap(t *testing.T) {
	cgm, _ := congomap.NewSyncMutexMap(congomap.Lookup(succeedingLookup), congomap.TTL(time.Nanosecond))
	loadStoreLookupAfterTTL(t, cgm, "syncMutex")
}

func TestLoadStoreLookupAfterTTLTwoLevelMap(t *testing.T) {
	cgm, _ := congomap.NewTwoLevelMap(congomap.Lookup(succeedingLookup), congomap.TTL(time.Nanosecond))
	loadStoreLookupAfterTTL(t, cgm, "twoLevel")
}

////////////////////////////////////////
// Pairs()

func ExampleTwoLevelMap_Pairs() {
	cgm, err := congomap.NewTwoLevelMap()
	if err != nil {
		log.Fatal(err)
	}
	defer func() { _ = cgm.Close() }()

	cgm.Store("abc", 123)

	for pair := range cgm.Pairs() {
		fmt.Println(pair.Key, pair.Value)
	}
	// Output: abc 123
}

func testPairs(t *testing.T, cgm congomap.Congomap, which string) {
	defer func() { _ = cgm.Close() }()
	cgm.Store("first", "Clark")
	cgm.Store("last", "Kent")
	for pair := range cgm.Pairs() {
		if _, ok := pair.Value.(string); !ok {
			t.Errorf("Actual: %#v; Expected: %#v", ok, true)
		}
	}
}

func TestPairsChannelMap(t *testing.T) {
	cgm, _ := congomap.NewChannelMap()
	testPairs(t, cgm, "channel")
}

func TestPairsSyncAtomicMap(t *testing.T) {
	cgm, _ := congomap.NewSyncAtomicMap()
	testPairs(t, cgm, "syncAtomic")
}

func TestPairsSyncMutexMap(t *testing.T) {
	cgm, _ := congomap.NewSyncMutexMap()
	testPairs(t, cgm, "syncMutex")
}

func TestPairsTwoLevelMap(t *testing.T) {
	cgm, _ := congomap.NewTwoLevelMap()
	testPairs(t, cgm, "twoLevel")
}

// ReaperInvokedDuringDelete

func ExampleReaper() {
	// Create a Congomap, specifying what the reaper callback function is.
	cgm, err := congomap.NewTwoLevelMap(congomap.Reaper(func(value interface{}) {
		fmt.Println("value", value, "expired")
	}))
	if err != nil {
		log.Fatal(err)
	}
	defer func() { _ = cgm.Close() }()

	cgm.Store("someKey", 42) // reaper is not called because nothing was replaced
	cgm.Delete("someKey")    // if declared, reaper is called during this delete.

	// Output: value 42 expired
}

func ExampleTwoLevelMap_Delete() {
	// Create a Congomap, specifying what the reaper callback function is.
	cgm, err := congomap.NewTwoLevelMap(congomap.Reaper(func(value interface{}) {
		fmt.Println("value", value, "expired")
	}))
	if err != nil {
		log.Fatal(err)
	}
	defer func() { _ = cgm.Close() }()

	cgm.Store("someKey", 42) // reaper is not called because nothing was replaced
	cgm.Delete("someKey")    // if declared, reaper is called during this delete.

	// Output: value 42 expired
}

func createReaper(t *testing.T, wg *sync.WaitGroup, which string) func(interface{}) {
	expected := 42
	return func(value interface{}) {
		if v, ok := value.(int); !ok || v != expected {
			t.Errorf("reaper receives value during delete; Which: %s; Key: %q; Actual: %#v; Expected: %#v", which, value, expected)
		}
		wg.Done()
	}
}

func createReaperTesterInvokeDuringDelete(t *testing.T, wg *sync.WaitGroup) func(congomap.Congomap) {
	expected := 42
	return func(cgm congomap.Congomap) {
		defer func() { _ = cgm.Close() }()
		cgm.Store("hit", expected)
		wg.Add(1)
		cgm.Delete("hit")
		wg.Wait()
	}
}

func TestReaperInvokedDuringDeleteChannelMap(t *testing.T) {
	var wg sync.WaitGroup
	cgm, _ := congomap.NewChannelMap(congomap.Reaper(createReaper(t, &wg, "channel")))
	createReaperTesterInvokeDuringDelete(t, &wg)(cgm)
}

func TestReaperInvokedDuringDeleteSyncAtomicMap(t *testing.T) {
	var wg sync.WaitGroup
	cgm, _ := congomap.NewSyncAtomicMap(congomap.Reaper(createReaper(t, &wg, "syncAtomic")))
	createReaperTesterInvokeDuringDelete(t, &wg)(cgm)
}

func TestReaperInvokedDuringDeleteSyncMutexMap(t *testing.T) {
	var wg sync.WaitGroup
	cgm, _ := congomap.NewSyncMutexMap(congomap.Reaper(createReaper(t, &wg, "syncMutex")))
	createReaperTesterInvokeDuringDelete(t, &wg)(cgm)
}

func TestReaperInvokedDuringDeleteTwoLevelMap(t *testing.T) {
	var wg sync.WaitGroup
	cgm, _ := congomap.NewTwoLevelMap(congomap.Reaper(createReaper(t, &wg, "twoLevel")))
	createReaperTesterInvokeDuringDelete(t, &wg)(cgm)
}

// ReaperInvokedDuringGC

func ExampleTwoLevelMap_GC() {
	// Note no default TTL is defined, so values will never expire by default.
	cgm, err := congomap.NewTwoLevelMap(congomap.Reaper(func(value interface{}) {
		fmt.Println("value", value, "expired")
	}))
	if err != nil {
		log.Fatal(err)
	}
	defer func() { _ = cgm.Close() }()

	cgm.Store("someKey", &congomap.ExpiringValue{Value: 42, Expiry: time.Now().Add(time.Millisecond)})
	time.Sleep(2 * time.Millisecond)
	cgm.GC() // if declared, the reaper would have been called with 42 as its arguement
	// Output: value 42 expired
}

func createReaperTesterInvokeDuringGC(t *testing.T, wg *sync.WaitGroup) func(congomap.Congomap) {
	expected := 42
	return func(cgm congomap.Congomap) {
		defer func() { _ = cgm.Close() }()
		cgm.Store("hit", &congomap.ExpiringValue{Value: expected, Expiry: time.Now().Add(time.Nanosecond)})
		time.Sleep(time.Millisecond)
		wg.Add(1)
		cgm.GC()
		wg.Wait()
	}
}

func TestReaperInvokedDuringGCChannelMap(t *testing.T) {
	var wg sync.WaitGroup
	cgm, _ := congomap.NewChannelMap(congomap.Reaper(createReaper(t, &wg, "channel")))
	createReaperTesterInvokeDuringGC(t, &wg)(cgm)
}

func TestReaperInvokedDuringGCSyncAtomicMap(t *testing.T) {
	var wg sync.WaitGroup
	cgm, _ := congomap.NewSyncAtomicMap(congomap.Reaper(createReaper(t, &wg, "syncAtomic")))
	createReaperTesterInvokeDuringGC(t, &wg)(cgm)
}

func TestReaperInvokedDuringGCSyncMutexMap(t *testing.T) {
	var wg sync.WaitGroup
	cgm, _ := congomap.NewSyncMutexMap(congomap.Reaper(createReaper(t, &wg, "syncMutex")))
	createReaperTesterInvokeDuringGC(t, &wg)(cgm)
}

func TestReaperInvokedDuringGCTwoLevelMap(t *testing.T) {
	var wg sync.WaitGroup
	cgm, _ := congomap.NewTwoLevelMap(congomap.Reaper(createReaper(t, &wg, "twoLevel")))
	createReaperTesterInvokeDuringGC(t, &wg)(cgm)
}

// ReaperInvokedDuringClose

func createReaperTesterInvokeDuringClose(t *testing.T, wg *sync.WaitGroup) func(congomap.Congomap) {
	expected := 42
	return func(cgm congomap.Congomap) {
		cgm.Store("hit", &congomap.ExpiringValue{Value: expected, Expiry: time.Now().Add(time.Nanosecond)})
		time.Sleep(time.Millisecond)
		wg.Add(1)
		_ = cgm.Close()
		wg.Wait()
	}
}

func TestReaperInvokedDuringCloseChannelMap(t *testing.T) {
	var wg sync.WaitGroup
	cgm, _ := congomap.NewChannelMap(congomap.Reaper(createReaper(t, &wg, "channel")))
	createReaperTesterInvokeDuringClose(t, &wg)(cgm)
}

func TestReaperInvokedDuringCloseSyncAtomicMap(t *testing.T) {
	var wg sync.WaitGroup
	cgm, _ := congomap.NewSyncAtomicMap(congomap.Reaper(createReaper(t, &wg, "syncAtomic")))
	createReaperTesterInvokeDuringClose(t, &wg)(cgm)
}

func TestReaperInvokedDuringCloseSyncMutexMap(t *testing.T) {
	var wg sync.WaitGroup
	cgm, _ := congomap.NewSyncMutexMap(congomap.Reaper(createReaper(t, &wg, "syncMutex")))
	createReaperTesterInvokeDuringClose(t, &wg)(cgm)
}

func TestReaperInvokedDuringCloseTwoLevelMap(t *testing.T) {
	var wg sync.WaitGroup
	cgm, _ := congomap.NewTwoLevelMap(congomap.Reaper(createReaper(t, &wg, "twoLevel")))
	createReaperTesterInvokeDuringClose(t, &wg)(cgm)
}

// Keys

func ExampleTwoLevelMap_Keys() {
	cgm, err := congomap.NewTwoLevelMap()
	if err != nil {
		log.Fatal(err)
	}
	defer func() { _ = cgm.Close() }()

	cgm.Store("abc", 123)
	cgm.Store("def", 456)
	keys := cgm.Keys()
	sort.Strings(keys)
	fmt.Println(keys)
	// Output: [abc def]
}
