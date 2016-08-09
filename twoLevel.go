package congomap

import (
	"sync"
	"time"
)

type twoLevelMap struct {
	db     map[string]*lockingValue
	dbLock sync.RWMutex

	halt   chan struct{}
	lookup func(string) (interface{}, error)
	reaper func(interface{})
	ttl    time.Duration
}

// lockingValue is a pointer to a value and the lock that protects it. All access to the
// ExpiringValue ought to be protected by use of the lock.
type lockingValue struct {
	l  sync.RWMutex
	ev *ExpiringValue // nil means not present
}

// NewTwoLevelMap returns a map that uses two levels of locks to serialize access to a key-value
// map. The top-level lock guards insertion and removal of keys in the map. The values of those keys
// are locks that guard each individual datum value for that key.
//
// Note that it is important to call the Close method on the returned data structure when it's no
// longer needed to free CPU and channel resources back to the runtime.
//
//	cgm, err := congomap.NewTwoLevelMap()
//	if err != nil {
//	    panic(err)
//	}
//	defer cgm.Close()
func NewTwoLevelMap(setters ...Setter) (Congomap, error) {
	cgm := &twoLevelMap{
		db:   make(map[string]*lockingValue),
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

func (cgm *twoLevelMap) Lookup(lookup func(string) (interface{}, error)) error {
	cgm.lookup = lookup
	return nil
}

func (cgm *twoLevelMap) Reaper(reaper func(interface{})) error {
	cgm.reaper = reaper
	return nil
}

func (cgm *twoLevelMap) TTL(duration time.Duration) error {
	if duration <= 0 {
		return ErrInvalidDuration(duration)
	}
	cgm.ttl = duration
	return nil
}

func (cgm *twoLevelMap) Delete(key string) {
	cgm.dbLock.Lock()
	lv, ok := cgm.db[key]
	delete(cgm.db, key)
	cgm.dbLock.Unlock()

	if ok && cgm.reaper != nil {
		cgm.reaper(lv.ev.Value)
	}
}

func (cgm *twoLevelMap) GC() {
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
				if cgm.reaper != nil {
					cgm.reaper(lv.ev.Value)
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

func (cgm *twoLevelMap) Load(key string) (interface{}, bool) {
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

func (cgm *twoLevelMap) LoadStore(key string) (interface{}, error) {
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
	if ok && cgm.reaper != nil {
		wg.Add(1)
		go func(value interface{}) {
			defer wg.Done()
			cgm.reaper(value)
		}(lv.ev.Value)
	}

	value, err := cgm.lookup(key)
	if err != nil {
		lv.ev = nil
		return nil, err
	}

	lv.ev = newExpiringValue(value, cgm.ttl)
	wg.Wait()
	return value, nil
}

func (cgm *twoLevelMap) Store(key string, value interface{}) {
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
	if ok && cgm.reaper != nil {
		wg.Add(1)
		go func(value interface{}) {
			defer wg.Done()
			cgm.reaper(value)
		}(lv.ev.Value)
	}

	lv.ev = newExpiringValue(value, cgm.ttl)
	wg.Wait()
}

func (cgm *twoLevelMap) Keys() []string {
	cgm.dbLock.RLock()
	keys := make([]string, 0, len(cgm.db))
	for k := range cgm.db {
		keys = append(keys, k)
	}
	cgm.dbLock.RUnlock()
	return keys
}

func (cgm *twoLevelMap) Pairs() <-chan *Pair {
	keys := make([]string, 0, len(cgm.db))
	lockedValues := make([]*lockingValue, 0, len(cgm.db))

	cgm.dbLock.RLock()
	for key, lv := range cgm.db {
		keys = append(keys, key)
		lockedValues = append(lockedValues, lv)
	}
	cgm.dbLock.RUnlock()

	pairs := make(chan *Pair, len(keys))

	go func(pairs chan<- *Pair) {
		now := time.Now()

		var wg sync.WaitGroup
		wg.Add(len(keys))

		for i, key := range keys {
			go func(key string, lv *lockingValue) {
				lv.l.Lock()
				if lv.ev != nil && (lv.ev.Expiry.IsZero() || lv.ev.Expiry.After(now)) {
					pairs <- &Pair{key, lv.ev.Value}
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

func (cgm *twoLevelMap) Close() error {
	close(cgm.halt)
	return nil
}

func (cgm *twoLevelMap) run() {
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
		for key, lv := range cgm.db {
			delete(cgm.db, key)
			go func(value interface{}) {
				defer wg.Done()
				cgm.reaper(value)
			}(lv.ev.Value)
		}
		cgm.dbLock.Unlock()
		wg.Wait()
	}
}
