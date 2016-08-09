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
	ttl    time.Duration
}

// NewSyncMutexMap returns a map that uses sync.RWMutex to serialize access to the data store.
//
// Note that it is important to call the Close method on the returned data structure when it's no
// longer needed to free CPU and channel resources back to the runtime.
//
//	cgm, err := cmap.NewSyncMutexMap()
//	if err != nil {
//	    panic(err)
//	}
//	defer cgm.Close()
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

func (cgm *syncMutexMap) Lookup(lookup func(string) (interface{}, error)) error {
	cgm.lookup = lookup
	return nil
}

func (cgm *syncMutexMap) Reaper(reaper func(interface{})) error {
	cgm.reaper = reaper
	return nil
}

func (cgm *syncMutexMap) TTL(duration time.Duration) error {
	if duration <= 0 {
		return ErrInvalidDuration(duration)
	}
	cgm.ttl = duration
	return nil
}

func (cgm *syncMutexMap) Delete(key string) {
	cgm.dbLock.Lock()
	ev, ok := cgm.db[key]
	delete(cgm.db, key)
	cgm.dbLock.Unlock()

	if ok && cgm.reaper != nil {
		cgm.reaper(ev.Value)
	}
}

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

func (cgm *syncMutexMap) Load(key string) (interface{}, bool) {
	cgm.dbLock.RLock()
	ev, ok := cgm.db[key]
	cgm.dbLock.RUnlock()

	if ok && (ev.Expiry.IsZero() || ev.Expiry.After(time.Now())) {
		return ev.Value, true
	}

	return nil, false
}

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

	cgm.db[key] = newExpiringValue(value, cgm.ttl)
	return value, nil
}

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

	cgm.db[key] = newExpiringValue(value, cgm.ttl)
	cgm.dbLock.Unlock()
	wg.Wait()
}

func (cgm *syncMutexMap) Keys() (keys []string) {
	cgm.dbLock.RLock()
	defer cgm.dbLock.RUnlock()
	keys = make([]string, 0, len(cgm.db))
	for k := range cgm.db {
		keys = append(keys, k)
	}
	return
}

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

func (cgm *syncMutexMap) Close() error {
	close(cgm.halt)
	return nil
}

func (cgm *syncMutexMap) run() {
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
