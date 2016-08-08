package congomap

import (
	"sync"
	"time"
)

type SyncMutexMap struct {
	config *Config

	db     map[string]*ExpiringValue
	dbLock sync.RWMutex
	halt   chan struct{}
}

// NewSyncMutexMap returns a map that uses sync.RWMutex to serialize access. Keys must be strings.
func NewSyncMutexMap(config *Config) (Congomap, error) {
	if config == nil {
		config = &Config{}
	}
	cgm := &SyncMutexMap{
		config: config,
		db:     make(map[string]*ExpiringValue),
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
func (cgm *SyncMutexMap) Delete(key string) {
	cgm.dbLock.Lock()
	ev, ok := cgm.db[key]
	delete(cgm.db, key)
	cgm.dbLock.Unlock()

	if ok && cgm.config.Reaper != nil {
		cgm.config.Reaper(ev.Value)
	}
}

// GC forces elimination of keys in Congomap with values that have expired.
func (cgm *SyncMutexMap) GC() {
	var wg sync.WaitGroup

	cgm.dbLock.Lock()
	now := time.Now()

	for key, ev := range cgm.db {
		if !ev.Expiry.IsZero() && now.After(ev.Expiry) {
			delete(cgm.db, key)
			if cgm.config.Reaper != nil {
				wg.Add(1)
				go func(value interface{}) {
					cgm.config.Reaper(value)
					wg.Done()
				}(ev.Value)
			}
		}
	}

	cgm.dbLock.Unlock()
	wg.Wait()
}

// Load gets the value associated with the given key. When the key is in the map, it returns the
// value associated with the key and true. Otherwise it returns nil for the value and false.
func (cgm *SyncMutexMap) Load(key string) (interface{}, bool) {
	cgm.dbLock.RLock()
	ev, ok := cgm.db[key]
	cgm.dbLock.RUnlock()

	if ok && (ev.Expiry.IsZero() || ev.Expiry.After(time.Now())) {
		return ev.Value, true
	}

	return nil, false
}

// LoadStore gets the value associated with the given key if it's in the map. If it's not in the
// map, it calls the lookup function, and sets the value in the map to that returned by the lookup
// function.
func (cgm *SyncMutexMap) LoadStore(key string) (interface{}, error) {
	cgm.dbLock.Lock()
	defer cgm.dbLock.Unlock()

	ev, ok := cgm.db[key]
	if ok && (ev.Expiry.IsZero() || ev.Expiry.After(time.Now())) {
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
		delete(cgm.db, key)
		return nil, err
	}

	cgm.db[key] = newExpiringValue(value, cgm.config.TTL)
	return value, nil
}

// Store sets the value associated with the given key.
func (cgm *SyncMutexMap) Store(key string, value interface{}) {
	cgm.dbLock.Lock()

	ev, ok := cgm.db[key]

	var wg sync.WaitGroup
	if ok && cgm.config.Reaper != nil {
		wg.Add(1)
		go func(value interface{}) {
			cgm.config.Reaper(value)
			wg.Done()
		}(ev.Value)
	}

	cgm.db[key] = newExpiringValue(value, cgm.config.TTL)
	cgm.dbLock.Unlock()
	wg.Wait()
}

// Keys returns an array of key values stored in the map.
func (cgm *SyncMutexMap) Keys() (keys []string) {
	cgm.dbLock.RLock()
	defer cgm.dbLock.RUnlock()
	keys = make([]string, 0, len(cgm.db))
	for k := range cgm.db {
		keys = append(keys, k)
	}
	return
}

// Pairs returns a channel through which key value pairs are read. Pairs will lock the Congomap so
// that no other accessors can be used until the returned channel is closed.
func (cgm *SyncMutexMap) Pairs() <-chan Pair {
	keys := make([]string, 0, len(cgm.db))
	evs := make([]*ExpiringValue, 0, len(cgm.db))

	cgm.dbLock.RLock()
	for k, v := range cgm.db {
		keys = append(keys, k)
		evs = append(evs, v)
	}
	cgm.dbLock.RUnlock()

	pairs := make(chan Pair)

	go func(pairs chan<- Pair) {
		now := time.Now()

		var wg sync.WaitGroup
		wg.Add(len(keys))

		for i, key := range keys {
			go func(key string, ev *ExpiringValue) {
				if ev.Expiry.IsZero() || ev.Expiry.After(now) {
					pairs <- Pair{key, ev.Value}
				}
				wg.Done()
			}(key, evs[i])
		}

		wg.Wait()
		close(pairs)
	}(pairs)

	return pairs
}

// Close releases resources used by the Congomap.
func (cgm *SyncMutexMap) Close() error {
	close(cgm.halt)
	return nil
}

func (cgm *SyncMutexMap) run() {
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
		for key, ev := range cgm.db {
			delete(cgm.db, key)
			go func(ev *ExpiringValue) {
				cgm.config.Reaper(ev.Value)
				wg.Done()
			}(ev)
		}
		wg.Wait()
		cgm.dbLock.Unlock()
	}
}
