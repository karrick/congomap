package congomap

import (
	"fmt"
	"sync"
	"sync/atomic"
)

type syncAtomicMap struct {
	db     atomic.Value
	lookup func(string) (interface{}, error)
	lock   sync.Mutex // used only by writers
}

// NewSyncAtomicMap returns a map that uses sync/atomic.Value to serialize
// access.
func NewSyncAtomicMap(setters ...CongomapSetter) (Congomap, error) {
	cgm := &syncAtomicMap{}
	cgm.db.Store(make(map[string]interface{}))
	for _, setter := range setters {
		if err := setter(cgm); err != nil {
			return nil, err
		}
	}
	if cgm.lookup == nil {
		cgm.lookup = func(_ string) (interface{}, error) {
			return nil, fmt.Errorf("no lookup function set")
		}
	}
	return cgm, nil
}

// Lookup sets the lookup callback function for this Congomap for use
// when `LoadStore` is called and a requested key is not in the map.
func (cgm *syncAtomicMap) Lookup(lookup func(string) (interface{}, error)) error {
	cgm.lookup = lookup
	return nil
}

// Delete removes a key value pair from a Congomap.
func (cgm *syncAtomicMap) Delete(key string) {
	cgm.lock.Lock() // synchronize with other potential writers
	defer cgm.lock.Unlock()

	m1 := cgm.db.Load().(map[string]interface{}) // load current value of the data structure
	m2 := make(map[string]interface{})           // create a new value
	for k, v := range m1 {
		m2[k] = v // copy all data from the current object to the new one
	}
	delete(m2, key)  // remove the specified item
	cgm.db.Store(m2) // atomically replace the current object with the new one
	// At this point all new readers start working with the new version.
	// The old version will be garbage collected once the existing readers
	// (if any) are done with it.
}

// Load gets the value associated with the given key. When the key is
// in the map, it returns the value associated with the key and
// true. Otherwise it returns nil for the value and false.
func (cgm syncAtomicMap) Load(key string) (interface{}, bool) {
	v, ok := cgm.db.Load().(map[string]interface{})[key]
	return v, ok
}

// Store sets the value associated with the given key.
func (cgm *syncAtomicMap) Store(key string, value interface{}) {
	cgm.lock.Lock() // synchronize with other potential writers
	defer cgm.lock.Unlock()

	m1 := cgm.db.Load().(map[string]interface{}) // load current value of the data structure
	m2 := make(map[string]interface{})           // create a new value
	for k, v := range m1 {
		m2[k] = v // copy all data from the current object to the new one
	}
	m2[key] = value  // do the update that we need
	cgm.db.Store(m2) // atomically replace the current object with the new one
	// At this point all new readers start working with the new version.
	// The old version will be garbage collected once the existing readers
	// (if any) are done with it.
}

// LoadStore gets the value associated with the given key if it's in
// the map. If it's not in the map, it calls the lookup function, and
// sets the value in the map to that returned by the lookup function.
func (cgm *syncAtomicMap) LoadStore(key string) (interface{}, error) {
	cgm.lock.Lock() // synchronize with other potential writers
	defer cgm.lock.Unlock()

	m1 := cgm.db.Load().(map[string]interface{}) // load current value of the data structure

	// load
	v, ok := m1[key]
	if ok {
		return v, nil
	}

	// store
	value, err := cgm.lookup(key)
	if err != nil {
		return nil, err
	}

	m2 := make(map[string]interface{}) // create a new value
	for k, v := range m1 {
		m2[k] = v // copy all data from the current object to the new one
	}
	m2[key] = value  // do the update that we need
	cgm.db.Store(m2) // atomically replace the current object with the new one
	return value, nil
}

// Keys returns an array of key values stored in the map.
func (cgm syncAtomicMap) Keys() []string {
	keys := make([]string, 0)
	m1 := cgm.db.Load().(map[string]interface{}) // load current value of the data structure
	for k := range m1 {
		keys = append(keys, k)
	}
	return keys
}

// Halt releases resources used by the Congomap.
func (_ syncAtomicMap) Halt() {
	// no-operation
}
