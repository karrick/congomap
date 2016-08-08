package congomap

import (
	"sync"
	"sync/atomic"
	"time"
)

type SyncAtomicMap struct {
	config *Config

	db     atomic.Value
	dbLock sync.Mutex // used only by writers
	halt   chan struct{}
}

// NewSyncAtomicMap returns a map that uses sync/atomic.Value to serialize
// access.
func NewSyncAtomicMap(config *Config) (Congomap, error) {
	if config == nil {
		config = &Config{}
	}
	cgm := &SyncAtomicMap{
		config: config,
		halt:   make(chan struct{}),
	}
	cgm.db.Store(make(map[string]*ExpiringValue))
	if cgm.config.Lookup == nil {
		cgm.config.Lookup = func(_ string) (interface{}, error) {
			return nil, ErrNoLookupDefined{}
		}
	}
	go cgm.run()
	return cgm, nil
}

// Delete removes a key value pair from a Congomap.
func (cgm *SyncAtomicMap) Delete(key string) {
	cgm.dbLock.Lock() // synchronize with other potential writers
	m := cgm.copyNonExpiredData(nil)
	if cgm.config.Reaper != nil {
		if ev, ok := m[key]; ok {
			cgm.config.Reaper(ev.Value)
		}
	}
	delete(m, key)
	cgm.db.Store(m)
	cgm.dbLock.Unlock()
}

// GC forces elimination of keys in Congomap with values that have expired.
func (cgm *SyncAtomicMap) GC() {
	cgm.dbLock.Lock()
	m := cgm.copyNonExpiredData(nil)
	cgm.db.Store(m)
	cgm.dbLock.Unlock()
}

// Load gets the value associated with the given key. When the key is in the map, it returns the
// value associated with the key and true. Otherwise it returns nil for the value and false.
func (cgm *SyncAtomicMap) Load(key string) (interface{}, bool) {
	ev, ok := cgm.db.Load().(map[string]*ExpiringValue)[key]
	if ok && (ev.Expiry.IsZero() || ev.Expiry.After(time.Now())) {
		return ev.Value, true
	}
	return nil, false
}

// LoadStore gets the value associated with the given key if it's in the map. If it's not in the
// map, it calls the lookup function, and sets the value in the map to that returned by the lookup
// function.
func (cgm *SyncAtomicMap) LoadStore(key string) (interface{}, error) {
	cgm.dbLock.Lock()

	m1 := cgm.db.Load().(map[string]*ExpiringValue) // load current value of the data structure

	ev, ok := m1[key]
	if ok && (ev.Expiry.IsZero() || ev.Expiry.After(time.Now())) {
		cgm.dbLock.Unlock()
		return ev.Value, nil
	}

	var wg sync.WaitGroup
	defer wg.Wait()

	if ok && cgm.config.Reaper != nil {
		wg.Add(1)
		go func(value interface{}) {
			cgm.config.Reaper(value)
			wg.Done()
		}(ev.Value)
	}

	value, err := cgm.config.Lookup(key)
	if err != nil {
		cgm.dbLock.Unlock()
		return nil, err
	}

	m2 := cgm.copyNonExpiredData(m1)
	m2[key] = newExpiringValue(value, cgm.config.TTL) // do the update that we need
	cgm.db.Store(m2)
	cgm.dbLock.Unlock()

	return value, nil
}

// Store sets the value associated with the given key.
func (cgm *SyncAtomicMap) Store(key string, value interface{}) {
	cgm.dbLock.Lock()

	m := cgm.copyNonExpiredData(nil)

	ev, ok := m[key]

	var wg sync.WaitGroup
	if ok && cgm.config.Reaper != nil {
		wg.Add(1)
		go func(value interface{}) {
			cgm.config.Reaper(value)
			wg.Done()
		}(ev.Value)
	}

	m[key] = newExpiringValue(value, cgm.config.TTL) // do the update that we need
	cgm.db.Store(m)
	cgm.dbLock.Unlock()
	wg.Wait()
}

// Keys returns an array of key values stored in the map.
func (cgm *SyncAtomicMap) Keys() []string {
	var keys []string
	m1 := cgm.db.Load().(map[string]*ExpiringValue)
	for k := range m1 {
		keys = append(keys, k)
	}
	return keys
}

// Pairs returns a channel through which key value pairs are read. Pairs will lock the Congomap so
// that no other accessors can be used until the returned channel is closed.
func (cgm *SyncAtomicMap) Pairs() <-chan Pair {
	pairs := make(chan Pair)
	go func(pairs chan<- Pair) {
		cgm.dbLock.Lock()
		defer cgm.dbLock.Unlock()

		m1 := cgm.db.Load().(map[string]*ExpiringValue) // load current value of the data structure
		now := time.Now()
		for k, v := range m1 {
			if v.Expiry.IsZero() || v.Expiry.After(now) {
				pairs <- Pair{k, v.Value}
			}
		}
		close(pairs)
	}(pairs)
	return pairs
}

// Close releases resources used by the Congomap.
func (cgm *SyncAtomicMap) Close() error {
	close(cgm.halt)
	return nil
}

func (cgm *SyncAtomicMap) copyNonExpiredData(m1 map[string]*ExpiringValue) map[string]*ExpiringValue {
	now := time.Now()
	if m1 == nil {
		m1 = cgm.db.Load().(map[string]*ExpiringValue) // load current value of the data structure
	}
	m2 := make(map[string]*ExpiringValue) // create a new value

	var wg sync.WaitGroup

	for k, v := range m1 {
		if v.Expiry.IsZero() || v.Expiry.After(now) {
			m2[k] = v // copy non-expired data from the current object to the new one
		} else if cgm.config.Reaper != nil {
			wg.Add(1)
			go func(value interface{}) {
				cgm.config.Reaper(value)
				wg.Done()
			}(v.Value)
		}
	}

	wg.Wait()
	return m2
}

func (cgm *SyncAtomicMap) run() {
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
		m1 := cgm.db.Load().(map[string]*ExpiringValue) // load current value of the data structure
		var wg sync.WaitGroup
		wg.Add(len(m1))
		for _, ev := range m1 {
			go func(value interface{}) {
				cgm.config.Reaper(value)
				wg.Done()
			}(ev.Value)
		}
		wg.Wait()
	}
}
