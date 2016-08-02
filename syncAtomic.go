package congomap

import (
	"sync"
	"sync/atomic"
	"time"
)

type syncAtomicMap struct {
	db     atomic.Value
	dbLock sync.Mutex // used only by writers

	halt   chan struct{}
	lookup func(string) (interface{}, error)
	reaper func(interface{})

	ttlEnabled  bool
	ttlDuration time.Duration
}

// NewSyncAtomicMap returns a map that uses sync/atomic.Value to serialize
// access.
func NewSyncAtomicMap(setters ...Setter) (Congomap, error) {
	cgm := &syncAtomicMap{halt: make(chan struct{})}
	cgm.db.Store(make(map[string]*ExpiringValue))
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

// Reaper is used to specify what function is to be called when
// garbage collecting item from the Congomap.
func (cgm *syncAtomicMap) Reaper(reaper func(interface{})) error {
	cgm.reaper = reaper
	return nil
}

// TTL sets the time-to-live for values stored in the Congomap.
func (cgm *syncAtomicMap) TTL(duration time.Duration) error {
	if duration <= 0 {
		return ErrInvalidDuration(duration)
	}
	cgm.ttlDuration = duration
	cgm.ttlEnabled = true
	return nil
}

// Delete removes a key value pair from a Congomap.
func (cgm *syncAtomicMap) Delete(key string) {
	cgm.dbLock.Lock() // synchronize with other potential writers
	m := cgm.copyNonExpiredData(nil)
	if cgm.reaper != nil {
		if ev, ok := m[key]; ok {
			cgm.reaper(ev.Value)
		}
	}
	delete(m, key)  // remove the specified item
	cgm.db.Store(m) // atomically replace the current object with the new one
	// At this point all new readers start working with the new version.
	// The old version will be garbage collected once the existing readers
	// (if any) are done with it.
	cgm.dbLock.Unlock()
}

// GC forces elimination of keys in Congomap with values that have
// expired.
func (cgm *syncAtomicMap) GC() {
	cgm.dbLock.Lock()
	m := cgm.copyNonExpiredData(nil)
	cgm.db.Store(m) // atomically replace the current object with the new one
	cgm.dbLock.Unlock()
}

// Load gets the value associated with the given key. When the key is in the map, it returns the
// value associated with the key and true. Otherwise it returns nil for the value and false.
func (cgm *syncAtomicMap) Load(key string) (interface{}, bool) {
	ev, ok := cgm.db.Load().(map[string]*ExpiringValue)[key]
	if ok && (ev.Expiry == zeroTime || ev.Expiry.After(time.Now())) {
		return ev.Value, true
	}
	return nil, false
}

// LoadStore gets the value associated with the given key if it's in the map. If it's not in the
// map, it calls the lookup function, and sets the value in the map to that returned by the lookup
// function.
func (cgm *syncAtomicMap) LoadStore(key string) (interface{}, error) {
	cgm.dbLock.Lock() // synchronize with other potential writers

	m1 := cgm.db.Load().(map[string]*ExpiringValue) // load current value of the data structure

	ev, ok := m1[key]
	if ok && (ev.Expiry == zeroTime || ev.Expiry.After(time.Now())) {
		cgm.dbLock.Unlock()
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
		cgm.dbLock.Unlock()
		return nil, err
	}

	m2 := cgm.copyNonExpiredData(m1)
	m2[key] = cgm.ensureExpiringValue(value) // do the update that we need
	cgm.db.Store(m2)                         // atomically replace the current object with the new one
	cgm.dbLock.Unlock()

	return value, nil
}

// Store sets the value associated with the given key.
func (cgm *syncAtomicMap) Store(key string, value interface{}) {
	cgm.dbLock.Lock() // synchronize with other potential writers

	m := cgm.copyNonExpiredData(nil)

	ev, ok := m[key]

	var wg sync.WaitGroup
	if ok && cgm.reaper != nil {
		wg.Add(1)
		go func(value interface{}) {
			cgm.reaper(value)
			wg.Done()
		}(ev.Value)
	}

	m[key] = cgm.ensureExpiringValue(value) // do the update that we need
	cgm.db.Store(m)                         // atomically replace the current object with the new one
	cgm.dbLock.Unlock()
	wg.Wait()
}

// Keys returns an array of key values stored in the map.
func (cgm *syncAtomicMap) Keys() []string {
	var keys []string
	m1 := cgm.db.Load().(map[string]*ExpiringValue) // load current value of the data structure
	for k := range m1 {
		keys = append(keys, k)
	}
	return keys
}

// Pairs returns a channel through which key value pairs are
// read. Pairs will lock the Congomap so that no other accessors can
// be used until the returned channel is closed.
func (cgm *syncAtomicMap) Pairs() <-chan *Pair {
	pairs := make(chan *Pair)
	go func(pairs chan<- *Pair) {
		cgm.dbLock.Lock()
		defer cgm.dbLock.Unlock()

		m1 := cgm.db.Load().(map[string]*ExpiringValue) // load current value of the data structure
		now := time.Now()
		for k, v := range m1 {
			if v.Expiry == zeroTime || v.Expiry.After(now) {
				pairs <- &Pair{k, v.Value}
			}
		}
		close(pairs)
	}(pairs)
	return pairs
}

// Close releases resources used by the Congomap.
func (cgm *syncAtomicMap) Close() error {
	close(cgm.halt)
	return nil
}

func (cgm *syncAtomicMap) copyNonExpiredData(m1 map[string]*ExpiringValue) map[string]*ExpiringValue {
	var now time.Time
	if cgm.ttlEnabled {
		now = time.Now()
	}
	if m1 == nil {
		m1 = cgm.db.Load().(map[string]*ExpiringValue) // load current value of the data structure
	}
	m2 := make(map[string]*ExpiringValue) // create a new value
	for k, v := range m1 {
		if v.Expiry == zeroTime || v.Expiry.After(now) {
			m2[k] = v // copy non-expired data from the current object to the new one
		} else if cgm.reaper != nil {
			cgm.reaper(v.Value)
		}
	}
	return m2
}

func (cgm *syncAtomicMap) ensureExpiringValue(value interface{}) *ExpiringValue {
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

func (cgm *syncAtomicMap) run() {
	var duration time.Duration
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
		m1 := cgm.db.Load().(map[string]*ExpiringValue) // load current value of the data structure
		var wg sync.WaitGroup
		wg.Add(len(m1))
		for _, ev := range m1 {
			go func(value interface{}) {
				cgm.reaper(value)
				wg.Done()
			}(ev.Value)
		}
		wg.Wait()
	}
}
