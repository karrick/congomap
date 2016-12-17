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
	ttl    time.Duration
}

// NewSyncAtomicMap returns a map that uses atomic.Value to serialize access, using a copy-on-write
// method of atomically updating the data store.
//
// Because write speeds are O(n) based on the size of the keys in this Congomap, this type of
// Congomap is particularly well suited for scenarios with a very large read to write ratio, and a
// small corpus of keys in the Congomap. This type of Congomap also uses a mutex to guard all
// mutations to the data store.
//
// Note that it is important to call the Close method on the returned data structure when it's no
// longer needed to free CPU and channel resources back to the runtime.
//
//	cgm,_ := congomap.NewSyncAtomicMap()
//	if err != nil {
//	    panic(err)
//	}
//	defer func() { _ = cgm.Close() }()
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

func (cgm *syncAtomicMap) Lookup(lookup func(string) (interface{}, error)) error {
	cgm.lookup = lookup
	return nil
}

func (cgm *syncAtomicMap) Reaper(reaper func(interface{})) error {
	cgm.reaper = reaper
	return nil
}

func (cgm *syncAtomicMap) TTL(duration time.Duration) error {
	if duration <= 0 {
		return ErrInvalidDuration(duration)
	}
	cgm.ttl = duration
	return nil
}

func (cgm *syncAtomicMap) Delete(key string) {
	cgm.dbLock.Lock()
	m := cgm.copyNonExpiredData(nil)
	if cgm.reaper != nil {
		if ev, ok := m[key]; ok {
			cgm.reaper(ev.Value)
		}
	}
	delete(m, key)
	cgm.db.Store(m)
	cgm.dbLock.Unlock()
}

func (cgm *syncAtomicMap) GC() {
	cgm.dbLock.Lock()
	m := cgm.copyNonExpiredData(nil)
	cgm.db.Store(m)
	cgm.dbLock.Unlock()
}

func (cgm *syncAtomicMap) Load(key string) (interface{}, bool) {
	ev, ok := cgm.db.Load().(map[string]*ExpiringValue)[key]
	if ok && (ev.Expiry.IsZero() || ev.Expiry.After(time.Now())) {
		return ev.Value, true
	}
	return nil, false
}

func (cgm *syncAtomicMap) LoadStore(key string) (interface{}, error) {
	cgm.dbLock.Lock() // synchronize with other potential writers

	m1 := cgm.db.Load().(map[string]*ExpiringValue) // load current value of the data structure

	ev, ok := m1[key]
	if ok && (ev.Expiry.IsZero() || ev.Expiry.After(time.Now())) {
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
	m2[key] = newExpiringValue(value, cgm.ttl)
	cgm.db.Store(m2)
	cgm.dbLock.Unlock()

	return value, nil
}

func (cgm *syncAtomicMap) Store(key string, value interface{}) {
	cgm.dbLock.Lock()

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

	m[key] = newExpiringValue(value, cgm.ttl)
	cgm.db.Store(m)
	cgm.dbLock.Unlock()
	wg.Wait()
}

func (cgm *syncAtomicMap) Keys() []string {
	var keys []string
	m1 := cgm.db.Load().(map[string]*ExpiringValue) // load current value of the data structure
	for k := range m1 {
		keys = append(keys, k)
	}
	return keys
}

func (cgm *syncAtomicMap) Pairs() <-chan *Pair {
	pairs := make(chan *Pair)
	go func(pairs chan<- *Pair) {
		cgm.dbLock.Lock()
		defer cgm.dbLock.Unlock()

		m1 := cgm.db.Load().(map[string]*ExpiringValue) // load current value of the data structure
		now := time.Now()
		for k, v := range m1 {
			if v.Expiry.IsZero() || v.Expiry.After(now) {
				pairs <- &Pair{k, v.Value}
			}
		}
		close(pairs)
	}(pairs)
	return pairs
}

func (cgm *syncAtomicMap) Close() error {
	close(cgm.halt)
	return nil
}

func (cgm *syncAtomicMap) copyNonExpiredData(m1 map[string]*ExpiringValue) map[string]*ExpiringValue {
	now := time.Now()
	if m1 == nil {
		m1 = cgm.db.Load().(map[string]*ExpiringValue) // load current value of the data structure
	}
	m2 := make(map[string]*ExpiringValue) // create a new value

	var wg sync.WaitGroup

	for k, v := range m1 {
		if v.Expiry.IsZero() || v.Expiry.After(now) {
			m2[k] = v // copy non-expired data from the current object to the new one
		} else if cgm.reaper != nil {
			wg.Add(1)
			go func(value interface{}) {
				cgm.reaper(value)
				wg.Done()
			}(v.Value)
		}
	}

	wg.Wait()
	return m2
}

func (cgm *syncAtomicMap) run() {
	gcPeriodicity := 15 * time.Minute
	if cgm.ttl > 0 && cgm.ttl <= time.Second {
		gcPeriodicity = time.Minute
	}

	active := true
	for active {
		select {
		case <-time.After(gcPeriodicity):
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
