package congomap

import (
	"sync"
	"time"
)

type ChannelMap struct {
	config *Config

	db    map[string]*ExpiringValue
	queue chan func()
	halt  chan struct{}
}

// NewChannelMap returns a map that uses channels to serialize access. Note that it is important to
// call the Halt method on the returned data structure when it's no longer needed to free CPU and
// channel resources back to the runtime.
func NewChannelMap(config *Config) (Congomap, error) {
	if config == nil {
		config = &Config{}
	}
	cgm := &ChannelMap{
		config: config,
		db:     make(map[string]*ExpiringValue),
		halt:   make(chan struct{}),
		queue:  make(chan func()),
	}
	if cgm.config.Lookup == nil {
		cgm.config.Lookup = func(_ string) (interface{}, error) {
			return nil, ErrNoLookupDefined{}
		}
	}
	go cgm.run()
	return cgm, nil
}

// Delete removes a key value pair from a Congomap.
func (cgm *ChannelMap) Delete(key string) {
	cgm.queue <- func() {
		ev, ok := cgm.db[key]
		if ok && cgm.config.Reaper != nil {
			cgm.config.Reaper(ev.Value)
		}
		delete(cgm.db, key)
	}
}

// GC forces elimination of keys in Congomap with values that have
// expired.
func (cgm *ChannelMap) GC() {
	var wg sync.WaitGroup

	cgm.queue <- func() {
		now := time.Now()
		for key, ev := range cgm.db {
			if !ev.Expiry.IsZero() && now.After(ev.Expiry) {
				delete(cgm.db, key)
				if cgm.config.Reaper != nil {
					wg.Add(1)
					go func(value interface{}) {
						cgm.config.Reaper(value)
						wg.Done()
					}(ev.Value)
				}
			}
		}
	}
	wg.Wait()
}

// Load gets the value associated with the given key. When the key is
// in the map, it returns the value associated with the key and
// true. Otherwise it returns nil for the value and false.
func (cgm *ChannelMap) Load(key string) (interface{}, bool) {
	rq := make(chan result)
	cgm.queue <- func() {
		ev, ok := cgm.db[key]
		if ok && (ev.Expiry.IsZero() || ev.Expiry.After(time.Now())) {
			rq <- result{value: ev.Value, ok: true}
			return
		}
		rq <- result{value: nil, ok: false}
	}
	res := <-rq
	return res.value, res.ok
}

// LoadStore gets the value associated with the given key if it's in
// the map. If it's not in the map, it calls the lookup function, and
// sets the value in the map to that returned by the lookup function.
func (cgm *ChannelMap) LoadStore(key string) (interface{}, error) {
	var wg sync.WaitGroup
	rq := make(chan result)
	cgm.queue <- func() {
		ev, ok := cgm.db[key]
		if ok && (ev.Expiry.IsZero() || ev.Expiry.After(time.Now())) {
			rq <- result{value: ev.Value, ok: true}
			return
		}
		// key not there or expired
		value, err := cgm.config.Lookup(key)
		if err != nil {
			rq <- result{value: nil, ok: false, err: err}
			return
		}

		if ok && cgm.config.Reaper != nil {
			wg.Add(1)
			go func(value interface{}) {
				cgm.config.Reaper(value)
				wg.Done()
			}(ev.Value)
		}

		cgm.db[key] = newExpiringValue(value, cgm.config.TTL)
		rq <- result{value: value, ok: true}
	}
	res := <-rq
	wg.Wait() // must be after receive from rq to ensure Add had a chance to run
	return res.value, res.err
}

// Store sets the value associated with the given key.
func (cgm *ChannelMap) Store(key string, value interface{}) {
	var wg sync.WaitGroup
	wg.Add(1)
	cgm.queue <- func() {
		ev, ok := cgm.db[key]

		if ok && cgm.config.Reaper != nil {
			wg.Add(1)
			go func(value interface{}) {
				cgm.config.Reaper(value)
				wg.Done()
			}(ev.Value)
		}

		cgm.db[key] = newExpiringValue(value, cgm.config.TTL)
		wg.Done()
	}
	wg.Wait()
}

// Keys returns an array of key values stored in the map.
func (cgm ChannelMap) Keys() []string {
	var wg sync.WaitGroup
	keys := make([]string, 0, len(cgm.db))
	wg.Add(1)
	cgm.queue <- func() {
		for k := range cgm.db {
			keys = append(keys, k)
		}
		wg.Done()
	}
	wg.Wait()
	return keys
}

// Pairs returns a channel through which key value pairs are
// read. Pairs will lock the Congomap so that no other accessors can
// be used until the returned channel is closed.
func (cgm *ChannelMap) Pairs() <-chan Pair {
	pairs := make(chan Pair, len(cgm.db))
	cgm.queue <- func() {
		now := time.Now()
		for key, ev := range cgm.db {
			if ev.Expiry.IsZero() || (ev.Expiry.After(now)) {
				pairs <- Pair{key, ev.Value}
			}
		}
		close(pairs)
	}
	return pairs
}

// Close releases resources used by the Congomap.
func (cgm *ChannelMap) Close() error {
	close(cgm.halt)
	return nil
}

type result struct {
	value interface{}
	ok    bool
	err   error
}

func (cgm *ChannelMap) run() {
	duration := 15 * time.Minute
	if cgm.config.TTL > 0 && cgm.config.TTL <= time.Second {
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

	if cgm.config.Reaper != nil {
		var wg sync.WaitGroup
		wg.Add(len(cgm.db))
		for key, ev := range cgm.db {
			delete(cgm.db, key)
			go func(value interface{}) {
				cgm.config.Reaper(value)
				wg.Done()
			}(ev.Value)
		}
		wg.Wait()
	}
}
