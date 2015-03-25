package congomap

type channelMap struct {
	db     map[string]interface{}
	lookup func(string) (interface{}, error)
	queue  chan func()
	quit   chan struct{}
}

// NewChannelMap returns a map that uses channels to serialize
// access. Note that it is important to call the Halt method on the
// returned data structure when it's no longer needed to free CPU and
// channel resources back to the runtime.
func NewChannelMap(setters ...CongomapSetter) (Congomap, error) {
	cgm := &channelMap{
		db:    make(map[string]interface{}),
		queue: make(chan func()),
		quit:  make(chan struct{}),
	}
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
	go cgm.run_queue()
	return cgm, nil
}

// Lookup sets the lookup callback function for this Congomap for use
// when `LoadStore` is called and a requested key is not in the map.
func (cgm *channelMap) Lookup(lookup func(string) (interface{}, error)) error {
	cgm.lookup = lookup
	return nil
}

// Delete removes a key value pair from a Congomap.
func (cgm *channelMap) Delete(key string) {
	cgm.queue <- func() {
		delete(cgm.db, key)
	}
}

// Load gets the value associated with the given key. When the key is
// in the map, it returns the value associated with the key and
// true. Otherwise it returns nil for the value and false.
func (cgm *channelMap) Load(key string) (interface{}, bool) {
	rq := make(chan result)
	cgm.queue <- func() {
		var res result
		res.value, res.ok = cgm.db[key]
		rq <- res
	}
	res := <-rq
	return res.value, res.ok
}

// Store sets the value associated with the given key.
func (cgm *channelMap) Store(key string, value interface{}) {
	cgm.queue <- func() {
		cgm.db[key] = value
	}
}

// LoadStore gets the value associated with the given key if it's in
// the map. If it's not in the map, it calls the lookup function, and
// sets the value in the map to that returned by the lookup function.
func (cgm *channelMap) LoadStore(key string) (interface{}, error) {
	rq := make(chan result)
	cgm.queue <- func() {
		var res result
		res.value, res.ok = cgm.db[key]
		if !res.ok {
			res.value, res.err = cgm.lookup(key)
		}
		rq <- res
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
		for k, v := range cgm.db {
			pairs <- &Pair{k, v}
		}
		close(pairs)
	}
	return pairs
}

// Halt releases resources used by the Congomap.
func (cgm *channelMap) Halt() {
	cgm.quit <- struct{}{}
}

type result struct {
	value interface{}
	ok    bool
	err   error
}

func (cgm *channelMap) run_queue() {
	for {
		select {
		case fn := <-cgm.queue:
			fn()
		case <-cgm.quit:
			break
		}
	}
}
