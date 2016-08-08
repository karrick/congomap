package congomap

import (
	"sync"
	"time"
)

type TwoLevelMap struct {
	config *Config

	db     map[string]*lockingValue
	dbLock sync.RWMutex
	halt   chan struct{}
}

// lockingValue is a pointer to a value and the lock that protects it. All access to the
// ExpiringValue ought to be protected by use of the lock.
type lockingValue struct {
	l  sync.RWMutex
	ev *ExpiringValue // nil means not present
}

// NewTwoLevelMap returns a map that uses sync.RWMutex to serialize access. Keys must be strings.
//
//	package main
//
//	import (
//		"fmt"
//		"strconv"
//
//		congomap "gopkg.in/karrick/congomap.v3"
//	)
//
//	func main() {
//		var series congomap.Congomap
//
//		lookup := func(key string) (interface{}, error) {
//			value, err := strconv.Atoi(key)
//			if err != nil {
//				return nil, err
//			}
//			if value < 2 {
//				return 1, nil
//			}
//			first, err := series.LoadStore(strconv.Itoa(value - 1))
//			if err != nil {
//				return nil, err
//			}
//			second, err := series.LoadStore(strconv.Itoa(value - 2))
//			if err != nil {
//				return nil, err
//			}
//			return first.(int) + second.(int), nil
//		}
//
//		config := &congomap.Config{Lookup: lookup}
//
//		var err error
//		series, err = congomap.NewTwoLevelMap(config)
//		if err != nil {
//			panic(err)
//		}
//
//		_, err = series.LoadStore("10")
//		if err != nil {
//			panic(err)
//		}
//
//		for pair := range series.Pairs() {
//			fmt.Println(pair)
//		}
//	}
func NewTwoLevelMap(config *Config) (Congomap, error) {
	if config == nil {
		config = &Config{}
	}
	cgm := &TwoLevelMap{
		config: config,
		db:     make(map[string]*lockingValue),
		halt:   make(chan struct{}),
	}
	if cgm.config.Lookup == nil {
		cgm.config.Lookup = func(_ string) (interface{}, error) {
			return nil, ErrNoLookupDefined{}
		}
	}
	go cgm.run()
	return cgm, nil
}

// Delete removes a key value pair from a Congomap.
//
//	cgm, err := congomap.NewTwoLevelMap(nil)
//	if err != nil {
//	    panic(err)
//	}
//	defer cgm.Close()
//
//	cgm.Store("someKey", 42)
//	cgm.Delete("someKey")
func (cgm *TwoLevelMap) Delete(key string) {
	cgm.dbLock.Lock()
	lv, ok := cgm.db[key]
	delete(cgm.db, key)
	cgm.dbLock.Unlock()

	if ok && cgm.config.Reaper != nil {
		cgm.config.Reaper(lv.ev.Value)
	}
}

// GC forces elimination of keys in Congomap with values that have expired. This function is
// periodically called by the library, but may be called on demand.
//
//	cgm, err := congomap.NewTwoLevelMap(nil)
//	if err != nil {
//	    panic(err)
//	}
//	defer cgm.Close()
//
//	cgm.Store("someKey", 42)
//	cgm.Delete("someKey")
//	cgm.GC()
func (cgm *TwoLevelMap) GC() {
	// NOTE: should lock lv first, but then want to parallel so lock on a lv won't block
	// forever, but then would have race condition around deleting keys, hence, the key killer
	keys := make(chan string, len(cgm.db))

	cgm.dbLock.Lock()
	now := time.Now()

	var wg sync.WaitGroup
	wg.Add(len(cgm.db))
	for key, lv := range cgm.db {
		go func(key string, lv *lockingValue) {
			defer wg.Done()

			lv.l.Lock()
			defer lv.l.Unlock()

			if lv.ev != nil && !lv.ev.Expiry.IsZero() && now.After(lv.ev.Expiry) {
				keys <- key
				if cgm.config.Reaper != nil {
					cgm.config.Reaper(lv.ev.Value)
				}
			}
		}(key, lv)
	}
	wg.Wait()

	var keyKiller sync.WaitGroup
	keyKiller.Add(1)
	go func(keys <-chan string) {
		for key := range keys {
			delete(cgm.db, key)
		}
		keyKiller.Done()
	}(keys)

	close(keys)
	keyKiller.Wait()
	cgm.dbLock.Unlock()
}

// Load gets the value associated with the given key. When the key is in the map, it returns the
// value associated with the key and true. Otherwise it returns nil for the value and false.
//
//	cgm, err := congomap.NewTwoLevelMap(nil)
//	if err != nil {
//	    panic(err)
//	}
//	defer cgm.Close()
//
//	cgm.Store("someKey", 42)
//	value, ok := cgm.Load("someKey")
func (cgm *TwoLevelMap) Load(key string) (interface{}, bool) {
	cgm.dbLock.RLock()
	lv, ok := cgm.db[key]
	cgm.dbLock.RUnlock()

	if !ok {
		return nil, false
	}

	lv.l.RLock()
	defer lv.l.RUnlock()

	if lv.ev != nil && (lv.ev.Expiry.IsZero() || lv.ev.Expiry.After(time.Now())) {
		return lv.ev.Value, true
	}

	return nil, false
}

// LoadStore gets the value associated with the given key if it's in the map. If it's not in the
// map, it calls the lookup function, and sets the value in the map to that returned by the lookup
// function.
func (cgm *TwoLevelMap) LoadStore(key string) (interface{}, error) {
	cgm.dbLock.RLock()
	lv, ok := cgm.db[key]
	cgm.dbLock.RUnlock()
	if !ok {
		cgm.dbLock.Lock()
		lv, ok = cgm.db[key]
		if !ok {
			lv = &lockingValue{}
			cgm.db[key] = lv
		}
		cgm.dbLock.Unlock()
	}

	lv.l.Lock()
	defer lv.l.Unlock()

	// value might have been filled by another go-routine
	if lv.ev != nil && (lv.ev.Expiry.IsZero() || lv.ev.Expiry.After(time.Now())) {
		return lv.ev.Value, nil
	}

	var wg sync.WaitGroup
	if ok && cgm.config.Reaper != nil {
		wg.Add(1)
		go func(value interface{}) {
			defer wg.Done()
			cgm.config.Reaper(value)
		}(lv.ev.Value)
	}

	value, err := cgm.config.Lookup(key)
	if err != nil {
		lv.ev = nil
		return nil, err
	}

	lv.ev = newExpiringValue(value, cgm.config.TTL)
	wg.Wait()
	return value, nil
}

// Store sets the value associated with the given key.
func (cgm *TwoLevelMap) Store(key string, value interface{}) {
	cgm.dbLock.RLock()
	lv, ok := cgm.db[key]
	cgm.dbLock.RUnlock()
	if !ok {
		cgm.dbLock.Lock()
		lv, ok = cgm.db[key]
		if !ok {
			lv = &lockingValue{}
			cgm.db[key] = lv
		}
		cgm.dbLock.Unlock()
	}

	lv.l.Lock()
	defer lv.l.Unlock()

	var wg sync.WaitGroup
	if ok && cgm.config.Reaper != nil {
		wg.Add(1)
		go func(value interface{}) {
			defer wg.Done()
			cgm.config.Reaper(value)
		}(lv.ev.Value)
	}

	lv.ev = newExpiringValue(value, cgm.config.TTL)
	wg.Wait()
}

// Keys returns an array of key values stored in the map.
func (cgm *TwoLevelMap) Keys() []string {
	cgm.dbLock.RLock()
	keys := make([]string, 0, len(cgm.db))
	for k := range cgm.db {
		keys = append(keys, k)
	}
	cgm.dbLock.RUnlock()
	return keys
}

// Pairs returns a channel through which key value pairs are read. Pairs will lock the Congomap so
// that no other accessors can be used until the returned channel is closed.
func (cgm *TwoLevelMap) Pairs() <-chan Pair {
	keys := make([]string, 0, len(cgm.db))
	lockedValues := make([]*lockingValue, 0, len(cgm.db))

	cgm.dbLock.RLock()
	for key, lv := range cgm.db {
		keys = append(keys, key)
		lockedValues = append(lockedValues, lv)
	}
	cgm.dbLock.RUnlock()

	pairs := make(chan Pair, len(keys))

	go func(pairs chan<- Pair) {
		now := time.Now()

		var wg sync.WaitGroup
		wg.Add(len(keys))

		for i, key := range keys {
			go func(key string, lv *lockingValue) {
				lv.l.Lock()
				if lv.ev != nil && (lv.ev.Expiry.IsZero() || lv.ev.Expiry.After(now)) {
					pairs <- Pair{key, lv.ev.Value}
				}
				lv.l.Unlock()
				wg.Done()
			}(key, lockedValues[i])
		}

		wg.Wait()
		close(pairs)
	}(pairs)

	return pairs
}

// Close releases resources used by the Congomap.
func (cgm *TwoLevelMap) Close() error {
	close(cgm.halt)
	return nil
}

func (cgm *TwoLevelMap) run() {
	duration := 15 * time.Minute
	if cgm.config.TTL > 0 && cgm.config.TTL <= time.Second {
		duration = time.Minute
	}

	active := true
	for active {
		select {
		case <-time.After(duration):
			cgm.GC()
		case <-cgm.halt:
			active = false
		}
	}

	if cgm.config.Reaper != nil {
		cgm.dbLock.Lock()
		var wg sync.WaitGroup
		wg.Add(len(cgm.db))
		for key, lv := range cgm.db {
			delete(cgm.db, key)
			go func(value interface{}) {
				defer wg.Done()
				cgm.config.Reaper(value)
			}(lv.ev.Value)
		}
		cgm.dbLock.Unlock()
		wg.Wait()
	}
}
