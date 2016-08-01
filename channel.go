package congomap

import (
	"sync"
	"time"
)

type channelMap struct {
	db     map[string]*ExpiringValue
	halt   chan struct{}
	lookup func(string) (interface{}, error)
	queue  chan func()
	reaper func(interface{})

	ttlEnabled  bool
	ttlDuration time.Duration
}

// NewChannelMap returns a map that uses channels to serialize
// access. Note that it is important to call the Halt method on the
// returned data structure when it's no longer needed to free CPU and
// channel resources back to the runtime.
func NewChannelMap(setters ...Setter) (Congomap, error) {
	cgm := &channelMap{
		db:    make(map[string]*ExpiringValue),
		halt:  make(chan struct{}),
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
	go cgm.run()
	return cgm, nil
}

// Lookup sets the lookup callback function for this Congomap for use
// when `LoadStore` is called and a requested key is not in the map.
func (cgm *channelMap) Lookup(lookup func(string) (interface{}, error)) error {
	cgm.lookup = lookup
	return nil
}

// Reaper is used to specify what function is to be called when
// garbage collecting item from the Congomap.
func (cgm *channelMap) Reaper(reaper func(interface{})) error {
	cgm.reaper = reaper
	return nil
}

// TTL sets the time-to-live for values stored in the Congomap.
func (cgm *channelMap) TTL(duration time.Duration) error {
	if duration <= 0 {
		return ErrInvalidDuration(duration)
	}
	cgm.ttlDuration = duration
	cgm.ttlEnabled = true
	return nil
}

// Delete removes a key value pair from a Congomap.
func (cgm *channelMap) Delete(key string) {
	cgm.queue <- func() {
		ev, ok := cgm.db[key]
		if ok && cgm.reaper != nil {
			cgm.reaper(ev.Value)
		}
		delete(cgm.db, key)
	}
}

// GC forces elimination of keys in Congomap with values that have
// expired.
func (cgm *channelMap) GC() {
	if cgm.ttlEnabled {
		cgm.queue <- func() {
			now := time.Now()
			var keysToRemove []string
			for key, ev := range cgm.db {
				if ev.Expiry != zeroTime && now.After(ev.Expiry) {
					keysToRemove = append(keysToRemove, key)
				}
			}
			for _, key := range keysToRemove {
				if cgm.reaper != nil {
					cgm.reaper(cgm.db[key].Value)
				}
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
		if ok && (ev.Expiry == zeroTime || ev.Expiry.After(time.Now())) {
			rq <- result{value: ev.Value, ok: true}
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
		var newValue *ExpiringValue

		switch ev := value.(type) {
		case *ExpiringValue:
			newValue = ev
		default:
			var expiry time.Time
			if cgm.ttlEnabled {
				expiry = time.Now().Add(cgm.ttlDuration)
			}
			newValue = &ExpiringValue{Value: value, Expiry: expiry}
		}

		cgm.db[key] = newValue
	}
}

// LoadStore gets the value associated with the given key if it's in
// the map. If it's not in the map, it calls the lookup function, and
// sets the value in the map to that returned by the lookup function.
func (cgm *channelMap) LoadStore(key string) (interface{}, error) {
	rq := make(chan result)
	cgm.queue <- func() {
		ev, ok := cgm.db[key]
		if ok && (ev.Expiry == zeroTime || ev.Expiry.After(time.Now())) {
			rq <- result{value: ev.Value, ok: true}
			return
		}
		// key not there or expired
		value, err := cgm.lookup(key)
		if err != nil {
			rq <- result{value: nil, ok: false, err: err}
			return
		}

		var newValue *ExpiringValue

		switch ev := value.(type) {
		case *ExpiringValue:
			newValue = ev
		default:
			var expiry time.Time
			if cgm.ttlEnabled {
				expiry = time.Now().Add(cgm.ttlDuration)
			}
			newValue = &ExpiringValue{Value: value, Expiry: expiry}
		}

		cgm.db[key] = newValue
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

// Pairs returns a channel through which key value pairs are
// read. Pairs will lock the Congomap so that no other accessors can
// be used until the returned channel is closed.
func (cgm *channelMap) Pairs() <-chan *Pair {
	pairs := make(chan *Pair)
	cgm.queue <- func() {
		now := time.Now()
		for k, v := range cgm.db {
			if v.Expiry == zeroTime || (v.Expiry.After(now)) {
				pairs <- &Pair{k, v.Value}
			}
		}
		close(pairs)
	}
	return pairs
}

// Close releases resources used by the Congomap.
func (cgm *channelMap) Close() error {
	close(cgm.halt)
	return nil
}

// Halt releases resources used by the Congomap.
func (cgm *channelMap) Halt() {
	close(cgm.halt)
}

type result struct {
	value interface{}
	ok    bool
	err   error
}

func (cgm *channelMap) run() {
	var duration time.Duration
	if !cgm.ttlEnabled {
		duration = time.Hour
	} else if duration < time.Second {
		duration = time.Minute
	}
	active := true
	for active {
		select {
		case fn := <-cgm.queue:
			fn()
		case <-time.After(duration):
			cgm.GC()
		case <-cgm.halt:
			active = false
		}
	}
	if cgm.reaper != nil {
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
	}
}
