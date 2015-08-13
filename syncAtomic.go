package congomap

import (
	"sync"
	"sync/atomic"
	"time"
)

type syncAtomicMap struct {
	db       atomic.Value
	duration time.Duration
	halt     chan struct{}
	lock     sync.Mutex // used only by writers
	lookup   func(string) (interface{}, error)
	ttl      bool
}

// NewSyncAtomicMap returns a map that uses sync/atomic.Value to serialize
// access.
func NewSyncAtomicMap(setters ...Setter) (Congomap, error) {
	cgm := &syncAtomicMap{halt: make(chan struct{})}
	cgm.db.Store(make(map[string]expiringValue))
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
func (cgm *syncAtomicMap) Lookup(lookup func(string) (interface{}, error)) error {
	cgm.lookup = lookup
	return nil
}

// TTL sets the time-to-live for values stored in the Congomap.
func (cgm *syncAtomicMap) TTL(duration time.Duration) error {
	if duration <= 0 {
		return ErrInvalidDuration(duration)
	}
	cgm.duration = duration
	cgm.ttl = true
	return nil
}

// Delete removes a key value pair from a Congomap.
func (cgm *syncAtomicMap) Delete(key string) {
	cgm.lock.Lock() // synchronize with other potential writers
	m := cgm.copyNonExpiredData(nil)
	delete(m, key)  // remove the specified item
	cgm.db.Store(m) // atomically replace the current object with the new one
	// At this point all new readers start working with the new version.
	// The old version will be garbage collected once the existing readers
	// (if any) are done with it.
	cgm.lock.Unlock()
}

// GC forces elimination of keys in Congomap with values that have
// expired.
func (cgm *syncAtomicMap) GC() {
	if cgm.ttl {
		cgm.lock.Lock()
		m := cgm.copyNonExpiredData(nil)
		cgm.db.Store(m) // atomically replace the current object with the new one
		cgm.lock.Unlock()
	}
}

// Load gets the value associated with the given key. When the key is
// in the map, it returns the value associated with the key and
// true. Otherwise it returns nil for the value and false.
func (cgm *syncAtomicMap) Load(key string) (interface{}, bool) {
	ev, ok := cgm.db.Load().(map[string]expiringValue)[key]
	if ok && (!cgm.ttl || ev.expiry > time.Now().UnixNano()) {
		return ev.value, true
	}
	return nil, false
}

// Store sets the value associated with the given key.
func (cgm *syncAtomicMap) Store(key string, value interface{}) {
	cgm.lock.Lock() // synchronize with other potential writers
	m := cgm.copyNonExpiredData(nil)
	ev := expiringValue{value: value}
	if cgm.ttl {
		ev.expiry = time.Now().UnixNano() + int64(cgm.duration)
	}
	m[key] = ev     // do the update that we need
	cgm.db.Store(m) // atomically replace the current object with the new one
	// At this point all new readers start working with the new version.
	// The old version will be garbage collected once the existing readers
	// (if any) are done with it.
	cgm.lock.Unlock()
}

// LoadStore gets the value associated with the given key if it's in
// the map. If it's not in the map, it calls the lookup function, and
// sets the value in the map to that returned by the lookup function.
func (cgm *syncAtomicMap) LoadStore(key string) (interface{}, error) {
	cgm.lock.Lock() // synchronize with other potential writers
	defer cgm.lock.Unlock()

	m1 := cgm.db.Load().(map[string]expiringValue) // load current value of the data structure

	// load
	ev, ok := m1[key]
	if ok && (!cgm.ttl || ev.expiry > time.Now().UnixNano()) {
		return ev.value, nil
	}
	// key was expired or not in db
	value, err := cgm.lookup(key)
	if err != nil {
		return nil, err
	}

	m2 := cgm.copyNonExpiredData(m1)
	ev = expiringValue{value: value}
	if cgm.ttl {
		ev.expiry = time.Now().UnixNano() + int64(cgm.duration)
	}
	m2[key] = ev     // do the update that we need
	cgm.db.Store(m2) // atomically replace the current object with the new one
	return value, nil
}

// Keys returns an array of key values stored in the map.
func (cgm *syncAtomicMap) Keys() []string {
	keys := make([]string, 0)
	m1 := cgm.db.Load().(map[string]expiringValue) // load current value of the data structure
	for k := range m1 {
		keys = append(keys, k)
	}
	return keys
}

// Pairs returns a channel through which key value pairs are
// read. Pairs will lock the Congomap so that no other accessors can
// be used until the returned channel is closed.
func (cgm *syncAtomicMap) Pairs() <-chan *Pair {
	cgm.lock.Lock()
	defer cgm.lock.Unlock()

	pairs := make(chan *Pair)
	go func(pairs chan<- *Pair) {
		m1 := cgm.db.Load().(map[string]expiringValue) // load current value of the data structure
		now := time.Now().UnixNano()
		for k, v := range m1 {
			if !cgm.ttl || (v.expiry > now) {
				pairs <- &Pair{k, v}
			}
		}
		close(pairs)
	}(pairs)
	return pairs
}

// Halt releases resources used by the Congomap.
func (cgm *syncAtomicMap) Halt() {
	cgm.halt <- struct{}{}
}

func (cgm *syncAtomicMap) run() {
	duration := 5 * cgm.duration
	if !cgm.ttl {
		duration = time.Hour
	} else if duration < time.Second {
		duration = time.Minute
	}
	for {
		select {
		case <-time.After(duration):
			cgm.GC()
		case <-cgm.halt:
			break
		}
	}
}

func (cgm *syncAtomicMap) copyNonExpiredData(m1 map[string]expiringValue) map[string]expiringValue {
	var now int64
	if cgm.ttl {
		now = time.Now().UnixNano()
	}
	if m1 == nil {
		m1 = cgm.db.Load().(map[string]expiringValue) // load current value of the data structure
	}
	m2 := make(map[string]expiringValue) // create a new value
	for k, v := range m1 {
		if !cgm.ttl || v.expiry > now {
			m2[k] = v // copy non-expired data from the current object to the new one
		}
	}
	return m2
}
