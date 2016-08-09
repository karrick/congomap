package congomap

import (
	"sync"
	"time"
)

type channelMap struct {
	db    map[string]*ExpiringValue
	queue chan func()

	halt   chan struct{}
	lookup func(string) (interface{}, error)
	reaper func(interface{})

	ttl time.Duration
}

// NewChannelMap returns a map that uses channels to serialize access.
//
// Note that it is important to call the Close method on the returned data structure when it's no
// longer needed to free CPU and channel resources back to the runtime.
//
//	cgm, err := congomap.NewChannelMap()
//	if err != nil {
//	    panic(err)
//	}
//	defer cgm.Close()
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

func (cgm *channelMap) Lookup(lookup func(string) (interface{}, error)) error {
	cgm.lookup = lookup
	return nil
}

func (cgm *channelMap) Reaper(reaper func(interface{})) error {
	cgm.reaper = reaper
	return nil
}

func (cgm *channelMap) TTL(duration time.Duration) error {
	if duration <= 0 {
		return ErrInvalidDuration(duration)
	}
	cgm.ttl = duration
	return nil
}

func (cgm *channelMap) Delete(key string) {
	cgm.queue <- func() {
		ev, ok := cgm.db[key]
		if ok && cgm.reaper != nil {
			cgm.reaper(ev.Value)
		}
		delete(cgm.db, key)
	}
}

func (cgm *channelMap) GC() {
	var wg sync.WaitGroup

	cgm.queue <- func() {
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
	}
	wg.Wait()
}

func (cgm *channelMap) Load(key string) (interface{}, bool) {
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

func (cgm *channelMap) LoadStore(key string) (interface{}, error) {
	var wg sync.WaitGroup
	rq := make(chan result)
	cgm.queue <- func() {
		ev, ok := cgm.db[key]
		if ok && (ev.Expiry.IsZero() || ev.Expiry.After(time.Now())) {
			rq <- result{value: ev.Value, ok: true}
			return
		}
		// key not there or expired
		value, err := cgm.lookup(key)
		if err != nil {
			rq <- result{value: nil, ok: false, err: err}
			return
		}

		if ok && cgm.reaper != nil {
			wg.Add(1)
			go func(value interface{}) {
				cgm.reaper(value)
				wg.Done()
			}(ev.Value)
		}

		cgm.db[key] = newExpiringValue(value, cgm.ttl)
		rq <- result{value: value, ok: true}
	}
	res := <-rq
	wg.Wait() // must be after receive from rq to ensure Add had a chance to run
	return res.value, res.err
}

func (cgm *channelMap) Store(key string, value interface{}) {
	var wg sync.WaitGroup
	wg.Add(1)
	cgm.queue <- func() {
		ev, ok := cgm.db[key]

		if ok && cgm.reaper != nil {
			wg.Add(1)
			go func(value interface{}) {
				cgm.reaper(value)
				wg.Done()
			}(ev.Value)
		}

		cgm.db[key] = newExpiringValue(value, cgm.ttl)
		wg.Done()
	}
	wg.Wait()
}

func (cgm channelMap) Keys() []string {
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

func (cgm *channelMap) Pairs() <-chan *Pair {
	pairs := make(chan *Pair)
	cgm.queue <- func() {
		now := time.Now()
		for key, ev := range cgm.db {
			if ev.Expiry.IsZero() || (ev.Expiry.After(now)) {
				pairs <- &Pair{key, ev.Value}
			}
		}
		close(pairs)
	}
	return pairs
}

func (cgm *channelMap) Close() error {
	close(cgm.halt)
	return nil
}

type result struct {
	value interface{}
	ok    bool
	err   error
}

func (cgm *channelMap) run() {
	gcPeriodicity := 15 * time.Minute
	if cgm.ttl > 0 && cgm.ttl <= time.Second {
		gcPeriodicity = time.Minute
	}

	active := true
	for active {
		select {
		case fn := <-cgm.queue:
			fn()
		case <-time.After(gcPeriodicity):
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
			go func(value interface{}) {
				cgm.reaper(value)
				wg.Done()
			}(ev.Value)
		}
		wg.Wait()
	}
}
