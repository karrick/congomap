package congomap

import (
	"sync"
	"time"
)

var zeroTime time.Time

type twoLevelMap struct {
	dbLock sync.RWMutex
	db     map[string]*lockedValue

	halt   chan struct{}
	lookup func(string) (interface{}, error)
	reaper func(interface{})

	// ignored, but here because of Congomap interface...
	ttlDuration time.Duration
	ttlEnabled  bool
}

// lockedValue is a pointer to a value and the lock that protects it. All access to the value ought
// to be protected by use of the lock.
type lockedValue struct {
	vlock sync.RWMutex
	ev    *ExpiringValue // nil means not present
}

// ExpiringValue couples a value with an expiry time for the value. The zero value for time.Time
// implies no expiry for this value. If the Store or Lookup method return an ExpiringValue the the
// value will expire with the specified Expiry time.
type ExpiringValue struct {
	Value  interface{}
	Expiry time.Time // zero value means no expiry
}

// NewTwoLevelMap returns a map that uses sync.RWMutex to serialize access. Keys must be
// strings.
func NewTwoLevelMap(setters ...Setter) (Congomap, error) {
	cgm := &twoLevelMap{
		db:   make(map[string]*lockedValue),
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

// Lookup sets the lookup callback function for this Congomap for use when `LoadStore` is called and
// a requested key is not in the map.
func (cgm *twoLevelMap) Lookup(lookup func(string) (interface{}, error)) error {
	cgm.lookup = lookup
	return nil
}

// Reaper is used to specify what function is to be called when garbage collecting item from the
// Congomap.
func (cgm *twoLevelMap) Reaper(reaper func(interface{})) error {
	cgm.reaper = reaper
	return nil
}

// TTL sets the time-to-live for values stored in the Congomap.
func (cgm *twoLevelMap) TTL(duration time.Duration) error {
	if duration <= 0 {
		return ErrInvalidDuration(duration)
	}
	cgm.ttlDuration = duration
	cgm.ttlEnabled = true
	return nil
}

// Delete removes a key value pair from a Congomap.
func (cgm *twoLevelMap) Delete(key string) {
	cgm.dbLock.Lock()
	lv, ok := cgm.db[key]
	delete(cgm.db, key)
	cgm.dbLock.Unlock()

	if ok && cgm.reaper != nil {
		lv.vlock.Lock()
		defer lv.vlock.Unlock() // NOTE: ensure unlock is called even if reaper panics
		cgm.reaper(lv.ev.Value)
	}
}

// GC forces elimination of keys in Congomap with values that have expired.
func (cgm *twoLevelMap) GC() {
	var lockedValuesToDelete []*lockedValue

	cgm.dbLock.Lock()
	now := time.Now()
	for key, lv := range cgm.db {
		if lv.ev.Expiry != zeroTime && now.After(lv.ev.Expiry) {
			lockedValuesToDelete = append(lockedValuesToDelete, lv)
			delete(cgm.db, key)
		}
	}
	cgm.dbLock.Unlock()

	if cgm.reaper != nil {
		var wg sync.WaitGroup
		wg.Add(len(lockedValuesToDelete))
		for _, lv := range lockedValuesToDelete {
			go func(lv *lockedValue) {
				lv.vlock.Lock()
				defer lv.vlock.Unlock() // NOTE: ensure unlock is called even if reaper panics
				cgm.reaper(lv.ev.Value)
				wg.Done()
			}(lv)
		}
		wg.Wait()
	}
}

// Load gets the value associated with the given key. When the key is in the map, it returns the
// value associated with the key and true. Otherwise it returns nil for the value and false.
func (cgm *twoLevelMap) Load(key string) (interface{}, bool) {
	cgm.dbLock.RLock()
	defer cgm.dbLock.RUnlock()

	lv, ok := cgm.db[key]
	if !ok {
		return nil, false
	}

	lv.vlock.Lock()
	defer lv.vlock.Unlock()

	if lv.ev != nil {
		if lv.ev.Expiry == zeroTime || lv.ev.Expiry.After(time.Now()) {
			return lv.ev.Value, true
		}
	}
	return nil, false
}

// Store sets the value associated with the given key.
func (cgm *twoLevelMap) Store(key string, value interface{}) {
	var newLockedValue lockedValue

	switch ev := value.(type) {
	case *ExpiringValue:
		newLockedValue.ev = ev
	default:
		newLockedValue.ev = &ExpiringValue{Value: value}
	}

	cgm.dbLock.Lock()
	if cgm.reaper != nil {
		if previousLockedValue, ok := cgm.db[key]; ok {
			go func(lv *lockedValue) {
				lv.vlock.Lock()
				defer lv.vlock.Unlock() // NOTE: ensure unlock is called even if reaper panics
				if lv.ev != nil {
					cgm.reaper(lv.ev.Value)
				}
			}(previousLockedValue)
		}
	}
	cgm.db[key] = &newLockedValue
	cgm.dbLock.Unlock()
}

// LoadStore gets the value associated with the given key if it's in the map. If it's not in the
// map, it calls the lookup function, and sets the value in the map to that returned by the lookup
// function.
func (cgm *twoLevelMap) LoadStore(key string) (interface{}, error) {
	cgm.dbLock.Lock()
	lv, ok := cgm.db[key]
	if !ok {
		lv = &lockedValue{}
		cgm.db[key] = lv
	}
	cgm.dbLock.Unlock()

	lv.vlock.Lock()
	defer lv.vlock.Unlock()

	// value might have been filled by another go-routine
	if lv.ev != nil && (lv.ev.Expiry == zeroTime || lv.ev.Expiry.After(time.Now())) {
		return lv.ev.Value, nil
	}

	// it's our job to fill it
	value, err := cgm.lookup(key)
	if err != nil {
		return nil, err
	}

	switch ev := value.(type) {
	case *ExpiringValue:
		lv.ev = ev
	default:
		lv.ev = &ExpiringValue{Value: value}
	}

	return value, nil
}

// Keys returns an array of key values stored in the map.
func (cgm *twoLevelMap) Keys() []string {
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
func (cgm *twoLevelMap) Pairs() <-chan *Pair {
	keys := make([]string, 0, len(cgm.db))
	values := make([]*lockedValue, 0, len(cgm.db))

	cgm.dbLock.RLock()
	for k, v := range cgm.db {
		keys = append(keys, k)
		values = append(values, v)
	}
	cgm.dbLock.RUnlock()

	pairs := make(chan *Pair)

	go func(pairs chan<- *Pair) {
		now := time.Now()

		var wg sync.WaitGroup
		wg.Add(len(keys))

		for i, key := range keys {
			go func(key string, lv *lockedValue) {
				if lv.ev != nil {
					if lv.ev.Expiry == zeroTime || lv.ev.Expiry.After(now) {
						pairs <- &Pair{key, lv.ev.Value}
					}
				}
				wg.Done()
			}(key, values[i])
		}

		wg.Wait()
		close(pairs)
	}(pairs)
	return pairs
}

// Close releases resources used by the Congomap.
func (cgm *twoLevelMap) Close() error {
	close(cgm.halt)
	return nil
}

// Halt releases resources used by the Congomap.
func (cgm *twoLevelMap) Halt() {
	close(cgm.halt)
}

func (cgm *twoLevelMap) run() {
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
	// ??? do we need to lock dbLock ???
	if cgm.reaper != nil {
		var wg sync.WaitGroup
		wg.Add(len(cgm.db))
		for key, lv := range cgm.db {
			go func(key string, lv *lockedValue) {
				delete(cgm.db, key)
				lv.vlock.Lock()
				defer lv.vlock.Unlock() // NOTE: ensure unlock is called even if reaper panics
				cgm.reaper(lv.ev.Value)
				wg.Done()
			}(key, lv)
		}
		wg.Wait()
	}
}
