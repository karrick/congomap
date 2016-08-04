package congomap

import (
	"sync"
	"time"
)

type syncMutexMap struct {
	db     map[string]*ExpiringValue
	dbLock sync.RWMutex

	halt   chan struct{}
	lookup func(string) (interface{}, error)
	reaper func(interface{})

	ttlEnabled  bool
	ttlDuration time.Duration
}

// NewSyncMutexMap returns a map that uses sync.RWMutex to serialize
// access. Keys must be strings.
func NewSyncMutexMap(setters ...Setter) (Congomap, error) {
	cgm := &syncMutexMap{
		db:   make(map[string]*ExpiringValue),
		halt: make(chan struct{}),
	}
	for _, setter := range setters {
		if err := setter(cgm); err != nil {
			return nil, err
		}
	}
	if cgm.lookup == nil {
		cgm.lookup = func(_ string) (interface{}, error) {
			return nil, ErrNoLookupDefined{}
		}
	}
	go cgm.run()
	return cgm, nil
}

// Lookup sets the lookup callback function for this Congomap for use
// when `LoadStore` is called and a requested key is not in the map.
func (cgm *syncMutexMap) Lookup(lookup func(string) (interface{}, error)) error {
	cgm.lookup = lookup
	return nil
}

// Reaper is used to specify what function is to be called when
// garbage collecting item from the Congomap.
func (cgm *syncMutexMap) Reaper(reaper func(interface{})) error {
	cgm.reaper = reaper
	return nil
}

// TTL sets the time-to-live for values stored in the Congomap.
func (cgm *syncMutexMap) TTL(duration time.Duration) error {
	if duration <= 0 {
		return ErrInvalidDuration(duration)
	}
	cgm.ttlDuration = duration
	cgm.ttlEnabled = true
	return nil
}

// Delete removes a key value pair from a Congomap.
func (cgm *syncMutexMap) Delete(key string) {
	cgm.dbLock.Lock()
	ev, ok := cgm.db[key]
	delete(cgm.db, key)
	cgm.dbLock.Unlock()

	if ok && cgm.reaper != nil {
		cgm.reaper(ev.Value)
	}
}

// GC forces elimination of keys in Congomap with values that have
// expired.
func (cgm *syncMutexMap) GC() {
	var wg sync.WaitGroup

	cgm.dbLock.Lock()
	now := time.Now()

	for key, ev := range cgm.db {
		if !ev.Expiry.IsZero() && now.After(ev.Expiry) {
			delete(cgm.db, key)
			if cgm.reaper != nil {
				wg.Add(1)
				go func(value interface{}) {
					cgm.reaper(value)
					wg.Done()
				}(ev.Value)
			}
		}
	}

	cgm.dbLock.Unlock()
	wg.Wait()
}

// Load gets the value associated with the given key. When the key is
// in the map, it returns the value associated with the key and
// true. Otherwise it returns nil for the value and false.
func (cgm *syncMutexMap) Load(key string) (interface{}, bool) {
	cgm.dbLock.RLock()
	ev, ok := cgm.db[key]
	cgm.dbLock.RUnlock()

	if ok && (ev.Expiry.IsZero() || ev.Expiry.After(time.Now())) {
		return ev.Value, true
	}

	return nil, false
}

// LoadStore gets the value associated with the given key if it's in
// the map. If it's not in the map, it calls the lookup function, and
// sets the value in the map to that returned by the lookup function.
func (cgm *syncMutexMap) LoadStore(key string) (interface{}, error) {
	cgm.dbLock.Lock()
	defer cgm.dbLock.Unlock()

	ev, ok := cgm.db[key]
	if ok && (ev.Expiry.IsZero() || ev.Expiry.After(time.Now())) {
		return ev.Value, nil
	}

	var wg sync.WaitGroup
	defer wg.Wait()
	if ok && cgm.reaper != nil {
		wg.Add(1)
		go func(value interface{}) {
			cgm.reaper(value)
			wg.Done()
		}(ev.Value)
	}

	value, err := cgm.lookup(key)
	if err != nil {
		delete(cgm.db, key)
		return nil, err
	}

	cgm.db[key] = cgm.ensureExpiringValue(value)
	return value, nil
}

// Store sets the value associated with the given key.
func (cgm *syncMutexMap) Store(key string, value interface{}) {
	cgm.dbLock.Lock()

	ev, ok := cgm.db[key]

	var wg sync.WaitGroup
	if ok && cgm.reaper != nil {
		wg.Add(1)
		go func(value interface{}) {
			cgm.reaper(value)
			wg.Done()
		}(ev.Value)
	}

	cgm.db[key] = cgm.ensureExpiringValue(value)
	cgm.dbLock.Unlock()
	wg.Wait()
}

// Keys returns an array of key values stored in the map.
func (cgm *syncMutexMap) Keys() (keys []string) {
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
func (cgm *syncMutexMap) Pairs() <-chan *Pair {
	keys := make([]string, 0, len(cgm.db))
	evs := make([]*ExpiringValue, 0, len(cgm.db))

	cgm.dbLock.RLock()
	for k, v := range cgm.db {
		keys = append(keys, k)
		evs = append(evs, v)
	}
	cgm.dbLock.RUnlock()

	pairs := make(chan *Pair)

	go func(pairs chan<- *Pair) {
		now := time.Now()

		var wg sync.WaitGroup
		wg.Add(len(keys))

		for i, key := range keys {
			go func(key string, ev *ExpiringValue) {
				if ev.Expiry.IsZero() || ev.Expiry.After(now) {
					pairs <- &Pair{key, ev.Value}
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
func (cgm *syncMutexMap) Close() error {
	close(cgm.halt)
	return nil
}

func (cgm *syncMutexMap) ensureExpiringValue(value interface{}) *ExpiringValue {
	switch val := value.(type) {
	case *ExpiringValue:
		return val
	default:
		if cgm.ttlEnabled {
			return &ExpiringValue{Value: value, Expiry: time.Now().Add(cgm.ttlDuration)}
		}
		return &ExpiringValue{Value: value}
	}
}

func (cgm *syncMutexMap) run() {
	duration := 5 * cgm.ttlDuration
	if !cgm.ttlEnabled {
		duration = time.Hour
	} else if duration < time.Second {
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

	if cgm.reaper != nil {
		cgm.dbLock.Lock()
		var wg sync.WaitGroup
		wg.Add(len(cgm.db))
		for key, ev := range cgm.db {
			delete(cgm.db, key)
			go func(ev *ExpiringValue) {
				cgm.reaper(ev.Value)
				wg.Done()
			}(ev)
		}
		wg.Wait()
		cgm.dbLock.Unlock()
	}
}
