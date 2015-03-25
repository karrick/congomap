package congomap

import (
	"sync"
)

type syncMutexMap struct {
	db     map[string]interface{}
	lookup func(string) (interface{}, error)
	lock   sync.RWMutex
}

// NewMutexMap returns a map that uses sync.RWMutexMap to serialize
// access. Keys must be strings.
func NewSyncMutexMap(setters ...CongomapSetter) (Congomap, error) {
	cgm := &syncMutexMap{db: make(map[string]interface{})}
	for _, setter := range setters {
		if err := setter(cgm); err != nil {
			return nil, err
		}
	}
	if cgm.lookup == nil {
		cgm.lookup = func(_ string) (interface{}, error) {
			return nil, errNoLookupCallbackSet
		}
	}
	return cgm, nil
}

// Lookup sets the lookup callback function for this Congomap for use
// when `LoadStore` is called and a requested key is not in the map.
func (cgm *syncMutexMap) Lookup(lookup func(string) (interface{}, error)) error {
	cgm.lookup = lookup
	return nil
}

// Delete removes a key value pair from a Congomap.
func (cgm *syncMutexMap) Delete(key string) {
	cgm.lock.Lock()
	defer cgm.lock.Unlock()
	delete(cgm.db, key)
}

// Load gets the value associated with the given key. When the key is
// in the map, it returns the value associated with the key and
// true. Otherwise it returns nil for the value and false.
func (cgm *syncMutexMap) Load(key string) (interface{}, bool) {
	cgm.lock.RLock()
	defer cgm.lock.RUnlock()
	v, ok := cgm.db[key]
	return v, ok
}

// Store sets the value associated with the given key.
func (cgm *syncMutexMap) Store(key string, value interface{}) {
	cgm.lock.Lock()
	defer cgm.lock.Unlock()
	cgm.db[key] = value
}

// LoadStore gets the value associated with the given key if it's in
// the map. If it's not in the map, it calls the lookup function, and
// sets the value in the map to that returned by the lookup function.
func (cgm *syncMutexMap) LoadStore(key string) (interface{}, error) {
	cgm.lock.Lock()
	defer cgm.lock.Unlock()
	value, ok := cgm.db[key]
	if !ok {
		var err error
		value, err = cgm.lookup(key)
		if err != nil {
			return nil, err
		}
		cgm.db[key] = value
	}
	return value, nil
}

// Keys returns an array of key values stored in the map.
func (cgm *syncMutexMap) Keys() (keys []string) {
	cgm.lock.RLock()
	defer cgm.lock.RUnlock()
	keys = make([]string, 0, len(cgm.db))
	for k, _ := range cgm.db {
		keys = append(keys, k)
	}
	return
}

// Pairs returns a channel through which key value pairs are
// read. Pairs will lock the Congomap so that no other accessors can
// be used until the returned channel is closed.
func (cgm *syncMutexMap) Pairs() <-chan *Pair {
	cgm.lock.RLock()
	defer cgm.lock.RUnlock()

	pairs := make(chan *Pair)
	go func(pairs chan<- *Pair) {
		for k, v := range cgm.db {
			pairs <- &Pair{k, v}
		}
		close(pairs)
	}(pairs)
	return pairs
}

// Halt releases resources used by the Congomap.
func (_ syncMutexMap) Halt() {
	// no-operation
}
