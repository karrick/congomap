package congomap

import (
	"fmt"
	"hash/fnv"
	"sync/atomic"
	"time"
	"unsafe"
)

// TODO
// move keys and values into same cache line for better speed
// might be able to eliminate stored type if implement logic atomic.Value using CAS

type lockFreeHashConfig struct {
	size uint64
}

// const defaultInitialSize = 32 // ideal initial size
const defaultInitialSize = 128 // for debugging until can grow

type LockFreeHashConfigurator func(*lockFreeHashConfig) error

type lockFreeHash struct {
	count  int64
	size   uint64
	keys   []interface{} // nil means no key; cannot use empty string for no key because then client could never have empty string as a key
	hashes []uint64
	values []atomic.Value
}

func newLockFreeHash(setters ...LockFreeHashConfigurator) (*lockFreeHash, error) {
	lfhc := &lockFreeHashConfig{size: defaultInitialSize}
	for _, setter := range setters {
		if err := setter(lfhc); err != nil {
			return nil, err
		}
	}
	// TODO: if lfhc.size not power of 2, round up to next power of 2
	lfh := &lockFreeHash{
		size:   lfhc.size,
		keys:   make([]interface{}, lfhc.size),
		hashes: make([]uint64, lfhc.size),
		values: make([]atomic.Value, lfhc.size),
	}
	return lfh, nil
}

func (lfh *lockFreeHash) Count() uint64 {
	return uint64(atomic.AddInt64(&lfh.count, 0))
}

func (lfh *lockFreeHash) getKey(index uint64) (string, bool) {
	if key, ok := lfh.keys[index].(string); ok {
		return key, true
	}
	return "", false
}

type sv struct {
	ptr                        unsafe.Pointer
	prime, sentinel, tombstone bool
}

func (lfh *lockFreeHash) getValue(index uint64) (interface{}, bool) {
	maybeValue := lfh.values[index].Load()
	if value, ok := maybeValue.(sv); ok {
		if value.tombstone {
			return nil, false // key has been deleted but not released
		} else if value.prime {
			// ???
		} else if value.sentinel {
			// resolve by asking new table
		} else {
			return *(*interface{})(value.ptr), true
		}
	}
	return nil, false // key was never set in this table
}

func (lfh *lockFreeHash) setValue(index uint64, value interface{}) {
	lfh.values[index].Store(sv{ptr: unsafe.Pointer(&value)})
}

func (lfh *lockFreeHash) setValuePrime(index uint64, value interface{}) {
	// fmt.Printf("key prime: %d\n", index)
	// ??? not sure how deal with present value
	lfh.values[index].Store(sv{ptr: unsafe.Pointer(&value), prime: true})
}

func (lfh *lockFreeHash) setValueSentinel(index uint64) {
	// fmt.Printf("key sentinel: %d\n", index)

	// TODO: we don't need to CAS here, can be blind overwrite
	lfh.values[index].Store(sv{sentinel: true})
}

func (lfh *lockFreeHash) setValueTombstone(index uint64) {
	// fmt.Printf("key tombstone: %d\n", index)
	lfh.values[index].Store(sv{tombstone: true})
}

// WARNING: not concurrency safe; temp debugging function
func (lfh *lockFreeHash) Dump() map[string]interface{} {
	m := make(map[string]interface{})
	for i := uint64(0); i < lfh.size; i++ {
		if key, ok := lfh.getKey(i); ok {
			if value, ok := lfh.getValue(i); ok {
				m[key] = value
				// fmt.Printf("index %d; key: %q; value: %#v\n", i, key, value)
			}
		}
	}
	return m
}

func (lfh *lockFreeHash) Delete(key string) {
	hasher := fnv.New64a()
	hasher.Write([]byte(key))
	hash := hasher.Sum64()
	index := hash

	var k string
	var ok bool
	for {
		index &= (lfh.size - 1)
		if k, ok = lfh.getKey(index); !ok {
			return
		}
		if memo := lfh.hashes[index]; hash == memo && k == key {
			lfh.setValueTombstone(index)
			// TODO might need to percolate up if sentinel or prime is there
			return
		}
		index++
	}
}

func (lfh *lockFreeHash) Load(key string) (interface{}, bool) {
	hasher := fnv.New64a()
	hasher.Write([]byte(key))
	index := hasher.Sum64()
	hash := index

	var k string
	var ok bool
	for {
		index &= (lfh.size - 1)
		if k, ok = lfh.getKey(index); !ok {
			return nil, false
		}
		if memo := lfh.hashes[index]; hash == memo && k == key {
			return lfh.getValue(index)
		}
		index++
	}
}

func (lfh *lockFreeHash) Store(key string, value interface{}) {
	hasher := fnv.New64a()
	hasher.Write([]byte(key))
	index := hasher.Sum64()
	hash := index

	var k string
	var ok bool
	var distance uint64
	for {
		index &= (lfh.size - 1)
		if k, ok = lfh.getKey(index); !ok {
			// found a place to store the pair
			// fmt.Printf("key value new: %d\n", index)
			lfh.hashes[index] = hash
			lfh.keys[index] = key
			lfh.setValue(index, value)

			// FIXME: race condition might increment count twice

			count := uint64(atomic.AddInt64(&lfh.count, 1))
			if count<<1 > lfh.size || distance<<4 > lfh.size {
				lfh.grow()
			}
			return
		}
		if memo := lfh.hashes[index]; hash == memo && k == key {
			// update value at this index
			// fmt.Printf("key value update: %d\n", index)
			lfh.setValue(index, value)
			return
		}
		index++
		distance++
	}
}

func (lfh *lockFreeHash) grow() {
	// fmt.Printf("TODO: implement grow\n")
}

func (lfh *lockFreeHash) GC() {
}

func (lfh *lockFreeHash) LoadStore(key string) (interface{}, error) {
	return nil, fmt.Errorf("TODO: implement LFH LoadStore")
}

func (lfh *lockFreeHash) Keys() []string {
	var keys []string
	for i := uint64(0); i < lfh.size; i++ {
		if key, ok := lfh.getKey(i); ok {
			keys = append(keys, key)
		}
	}
	return keys
}

func (lfh *lockFreeHash) Pairs() <-chan *Pair {
	pairs := make(chan *Pair)

	go func(pairs chan<- *Pair) {
		// now := time.Now().UnixNano()

		// for i := uint64(0); i < lfh.size; i++ {
		// 	if key, ok := lfh.getKey(i); ok {
		// 		if value, ok := lfh.getValue(i); ok {
		// 			if !cgm.ttl || (v.expiry > now) {
		// 				pairs <- &Pair{key, v.value}
		// 			}
		// 		}
		// 	}
		// }
		close(pairs)
	}(pairs)
	return pairs
}

func (lfh *lockFreeHash) Close() error {
	return nil
}

func (lfh *lockFreeHash) Halt() {
}

func (lfh *lockFreeHash) Lookup(lookup func(string) (interface{}, error)) error {
	return nil
}

func (lfh *lockFreeHash) Reaper(reaper func(interface{})) error {
	return nil
}

func (cgm *lockFreeHash) TTL(duration time.Duration) error {
	if duration <= 0 {
		return ErrInvalidDuration(duration)
	}
	// cgm.duration = duration
	// cgm.ttl = true
	return nil
}
