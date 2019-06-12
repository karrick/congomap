package congomap

import (
	"sync"
	"time"
)

type syncMutexMap struct {
	db       map[string]expiringValue
	duration time.Duration
	halt     chan struct{}
	lock     sync.RWMutex
	lookup   func(string) (interface{}, error)
	reaper   func(interface{})
	ttl      bool

	loading      map[string]*sync.WaitGroup
	loading_lock sync.Mutex
}

// NewSyncMutexMap returns a map that uses sync.RWMutex to serialize
// access. Keys must be strings.
func NewSyncMutexMap(setters ...Setter) (Congomap, error) {
	cgm := &syncMutexMap{
		db:   make(map[string]expiringValue),
		halt: make(chan struct{}),

		loading: make(map[string]*sync.WaitGroup),
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
	cgm.duration = duration
	cgm.ttl = true
	return nil
}

// Delete removes a key value pair from a Congomap.
func (cgm *syncMutexMap) Delete(key string) {
	cgm.lock.Lock()
	if cgm.reaper != nil {
		if ev, ok := cgm.db[key]; ok {
			cgm.reaper(ev.value)
		}
	}
	delete(cgm.db, key)
	cgm.lock.Unlock()
}

// GC forces elimination of keys in Congomap with values that have
// expired.
func (cgm *syncMutexMap) GC() {
	if cgm.ttl {
		cgm.lock.Lock()
		now := time.Now().UnixNano()
		var keysToRemove []string
		for key, ev := range cgm.db {
			if ev.expiry < now {
				keysToRemove = append(keysToRemove, key)
			}
		}
		for _, key := range keysToRemove {
			if cgm.reaper != nil {
				cgm.reaper(cgm.db[key].value)
			}
			delete(cgm.db, key)
		}
		cgm.lock.Unlock()
	}
}

// Load gets the value associated with the given key. When the key is
// in the map, it returns the value associated with the key and
// true. Otherwise it returns nil for the value and false.
func (cgm *syncMutexMap) Load(key string) (interface{}, bool) {
	cgm.lock.RLock()
	defer cgm.lock.RUnlock()
	ev, ok := cgm.db[key]
	if ok && (!cgm.ttl || ev.expiry > time.Now().UnixNano()) {
		return ev.value, true
	}
	return nil, false
}

// Store sets the value associated with the given key.
func (cgm *syncMutexMap) Store(key string, value interface{}) {
	cgm.lock.Lock()
	ev := expiringValue{value: value}
	if cgm.ttl {
		ev.expiry = time.Now().UnixNano() + int64(cgm.duration)
	}
	cgm.db[key] = ev
	cgm.lock.Unlock()
}

// LoadStore gets the value associated with the given key if it's in
// the map. If it's not in the map, it calls the lookup function, and
// sets the value in the map to that returned by the lookup function.
func (cgm *syncMutexMap) LoadStore(key string) (interface{}, error) {
	cgm.lock.Lock()
	ev, ok := cgm.db[key]
	if ok && (!cgm.ttl || ev.expiry > time.Now().UnixNano()) {
		return ev.value, nil
	}
	cgm.lock.Unlock() // Unlock whole map, since we are just loading

	// Lock the loading map
	cgm.loading_lock.Lock()
	wg, ok := cgm.loading[key]

	// If someone else is already loading, lets just wait on them
	if ok {
		cgm.loading_lock.Unlock()
		wg.Wait()
		return cgm.LoadStore(key) // TODO: don't recurse?
	} else {
		// No one else is loading

		// Lets create a wait group, and unlock the loading map
		var wg sync.WaitGroup
		wg.Add(1)
		cgm.loading[key] = &wg
		cgm.loading_lock.Unlock()

		// Do the actual load
		// key was expired or not in db
		value, err := cgm.lookup(key)
		if err != nil {
			return nil, err
		}
		ev = expiringValue{value: value}
		if cgm.ttl {
			ev.expiry = time.Now().UnixNano() + int64(cgm.duration)
		}

		// We have the value, lets set it and remove the loading entry
		cgm.lock.Lock()
		cgm.db[key] = ev
		cgm.lock.Unlock()

		// Remove our entry of loading
		cgm.loading_lock.Lock()
		delete(cgm.loading, key)
		cgm.loading_lock.Unlock()

		// mark the thing as loaded
		wg.Done()
		return value, nil
	}
}

// Keys returns an array of key values stored in the map.
func (cgm *syncMutexMap) Keys() (keys []string) {
	cgm.lock.RLock()
	defer cgm.lock.RUnlock()
	keys = make([]string, 0, len(cgm.db))
	for k := range cgm.db {
		keys = append(keys, k)
	}
	return
}

// Pairs returns a channel through which key value pairs are
// read. Pairs will lock the Congomap so that no other accessors can
// be used until the returned channel is closed.
func (cgm *syncMutexMap) Pairs() <-chan *Pair {
	cgm.lock.RLock()

	pairs := make(chan *Pair)
	go func(pairs chan<- *Pair) {
		now := time.Now().UnixNano()
		for k, v := range cgm.db {
			if !cgm.ttl || (v.expiry > now) {
				pairs <- &Pair{k, v.value}
			}
		}
		close(pairs)
		cgm.lock.RUnlock()
	}(pairs)
	return pairs
}

// Close releases resources used by the Congomap.
func (cgm *syncMutexMap) Close() error {
	cgm.halt <- struct{}{}
	return nil
}

// Halt releases resources used by the Congomap.
func (cgm *syncMutexMap) Halt() {
	cgm.halt <- struct{}{}
}
func (cgm *syncMutexMap) run() {
	duration := 5 * cgm.duration
	if !cgm.ttl {
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
		for _, ev := range cgm.db {
			cgm.reaper(ev.value)
		}
	}
}
