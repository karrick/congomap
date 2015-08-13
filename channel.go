package congomap

import (
	"time"
)

type channelMap struct {
	db       map[string]expiringValue
	lookup   func(string) (interface{}, error)
	queue    chan func()
	duration time.Duration
	ttl      bool
	done     chan struct{}
}

// NewChannelMap returns a map that uses channels to serialize
// access. Note that it is important to call the Halt method on the
// returned data structure when it's no longer needed to free CPU and
// channel resources back to the runtime.
func NewChannelMap(setters ...CongomapSetter) (Congomap, error) {
	cgm := &channelMap{
		db:    make(map[string]expiringValue),
		done:  make(chan struct{}),
		queue: make(chan func()),
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
	go cgm.run_queue()
	return cgm, nil
}

// Lookup sets the lookup callback function for this Congomap for use
// when `LoadStore` is called and a requested key is not in the map.
func (cgm *channelMap) Lookup(lookup func(string) (interface{}, error)) error {
	cgm.lookup = lookup
	return nil
}

func (cgm *channelMap) TTL(duration time.Duration) error {
	if duration <= 0 {
		return ErrInvalidDuration(duration)
	}
	cgm.duration = duration
	cgm.ttl = true
	return nil
}

// Delete removes a key value pair from a Congomap.
func (cgm *channelMap) Delete(key string) {
	cgm.queue <- func() {
		delete(cgm.db, key)
	}
}

// GC forces elimination of keys in Congomap with values that have
// expired.
func (cgm *channelMap) GC() {
	if cgm.ttl {
		cgm.queue <- func() {
			now := time.Now().UnixNano()
			keysToRemove := make([]string, 0)
			for key, ev := range cgm.db {
				if ev.expiry < now {
					keysToRemove = append(keysToRemove, key)
				}
			}
			for _, key := range keysToRemove {
				delete(cgm.db, key)
			}
		}
	}
}

// Load gets the value associated with the given key. When the key is
// in the map, it returns the value associated with the key and
// true. Otherwise it returns nil for the value and false.
func (cgm *channelMap) Load(key string) (interface{}, bool) {
	rq := make(chan result)
	cgm.queue <- func() {
		ev, ok := cgm.db[key]
		if ok && (!cgm.ttl || ev.expiry > time.Now().UnixNano()) {
			rq <- result{value: ev.value, ok: true}
			return
		}
		rq <- result{value: nil, ok: false}
	}
	res := <-rq
	return res.value, res.ok
}

// Store sets the value associated with the given key.
func (cgm *channelMap) Store(key string, value interface{}) {
	cgm.queue <- func() {
		ev := expiringValue{value: value}
		if cgm.ttl {
			ev.expiry = time.Now().UnixNano() + int64(cgm.duration)
		}
		cgm.db[key] = ev
	}
}

// LoadStore gets the value associated with the given key if it's in
// the map. If it's not in the map, it calls the lookup function, and
// sets the value in the map to that returned by the lookup function.
func (cgm *channelMap) LoadStore(key string) (interface{}, error) {
	rq := make(chan result)
	cgm.queue <- func() {
		ev, ok := cgm.db[key]
		if ok && (!cgm.ttl || ev.expiry > time.Now().UnixNano()) {
			rq <- result{value: ev.value, ok: true}
			return
		}
		// key not there or expired
		value, err := cgm.lookup(key)
		if err != nil {
			rq <- result{value: nil, ok: false, err: err}
			return
		}
		ev = expiringValue{value: value}
		if cgm.ttl {
			ev.expiry = time.Now().UnixNano() + int64(cgm.duration)
		}
		cgm.db[key] = ev
		rq <- result{value: value, ok: true}
	}
	res := <-rq
	return res.value, res.err
}

// Keys returns an array of key values stored in the map.
func (cgm channelMap) Keys() []string {
	keys := make([]string, 0, len(cgm.db))
	for k := range cgm.db {
		keys = append(keys, k)
	}
	return keys
}

// Halt releases resources used by the Congomap.
func (cgm *channelMap) Halt() {
	cgm.done <- struct{}{}
}

type result struct {
	value interface{}
	ok    bool
	err   error
}

func (cgm *channelMap) run_queue() {
	duration := 5 * cgm.duration
	if duration < time.Second {
		duration = time.Hour
	}
	for {
		select {
		case fn := <-cgm.queue:
			fn()
		case <-time.After(duration):
			cgm.GC()
		case <-cgm.done:
			break
		}
	}
}
