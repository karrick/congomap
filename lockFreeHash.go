package congomap

import (
	"fmt"
	"hash/fnv"
	"sync/atomic"
	"unsafe"
)

// TODO
// move keys and values into same cache line for better speed
// might be able to eliminate stored type if implement logic atomic.Value using CAS
// AND rather than MOD (30x speed improvement)

type lockFreeHashConfig struct {
	size uint64
}

const defaultInitialSize = 32

type LockFreeHashConfigurator func(*lockFreeHashConfig) error

// // ??? basically a chain of simple lock hashes, with a head pointer
// type LockFreeHash struct {
// 	bunch []*lockFreeHashBasic
// }

type lockFreeHashBasic struct {
	count  int64
	size   uint64
	keys   []interface{} // nil means no key; cannot use empty string for no key because then client could never have empty string as a key
	hashes []uint64
	values []atomic.Value
}

func NewLockFreeHash(setters ...LockFreeHashConfigurator) (*lockFreeHashBasic, error) {
	lfhc := &lockFreeHashConfig{
		size: defaultInitialSize,
	}
	for _, setter := range setters {
		if err := setter(lfhc); err != nil {
			return nil, err
		}
	}
	// TODO: if lfhc.size not power of 2, round up to next power of 2
	lfh := &lockFreeHashBasic{
		size:   lfhc.size,
		keys:   make([]interface{}, lfhc.size),
		hashes: make([]uint64, lfhc.size),
		values: make([]atomic.Value, lfhc.size),
	}
	return lfh, nil
}

func (lfh *lockFreeHashBasic) Count() uint64 {
	return uint64(atomic.AddInt64(&lfh.count, 0))
}

func (lfh *lockFreeHashBasic) getHash(index uint64) uint64 {
	return lfh.hashes[index]
}

func (lfh *lockFreeHashBasic) setHash(index uint64, hash uint64) {
	lfh.hashes[index] = hash
}

func (lfh *lockFreeHashBasic) getKey(index uint64) (string, bool) {
	if key, ok := lfh.keys[index].(string); ok {
		return key, true
	}
	return "", false
}

func (lfh *lockFreeHashBasic) setKey(index uint64, key string) {
	lfh.keys[index] = key
}

type sv struct {
	ptr                        unsafe.Pointer
	prime, sentinel, tombstone bool
}

func (lfh *lockFreeHashBasic) setValue(index uint64, value interface{}) {
	lfh.values[index].Store(sv{ptr: unsafe.Pointer(&value)})
}

func (lfh *lockFreeHashBasic) setValuePrime(index uint64, value interface{}) {
	// ??? not sure how deal with present value
	lfh.values[index].Store(sv{ptr: unsafe.Pointer(&value), prime: true})
}

func (lfh *lockFreeHashBasic) setValueSentinel(index uint64) {
	lfh.values[index].Store(sv{sentinel: true})
}

func (lfh *lockFreeHashBasic) setValueTombstone(index uint64) {
	fmt.Printf("key tombstone: %d\n", index)
	lfh.values[index].Store(sv{tombstone: true})
}

func (lfh *lockFreeHashBasic) getValue(index uint64) (interface{}, bool) {
	maybeValue := lfh.values[index].Load()
	if value, ok := maybeValue.(sv); ok {
		if value.tombstone {
			return nil, false // key has been deleted but not released
		} else if value.prime {
			// ???
		} else if value.sentinel {
			// resolve by asking next table
		} else {
			return *(*interface{})(value.ptr), true
		}
	}
	return nil, false // key was never set in this table
}

// WARNING: not concurrency safe; temp debugging function
func (lfh *lockFreeHashBasic) Dump() map[string]interface{} {
	m := make(map[string]interface{})
	for i := uint64(0); i < lfh.size; i++ {
		if key, ok := lfh.getKey(i); ok {
			if value, ok := lfh.getValue(i); ok {
				m[key] = value
				fmt.Printf("index %d; key: %q; value: %#v\n", i, key, value)
			}
		}
	}
	return m
}

func (lfh *lockFreeHashBasic) Delete(key string) {
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
		if memo := lfh.getHash(index); hash == memo && k == key {
			lfh.setValueTombstone(index)
			// TODO might need to percolate up if sentinel or prime is there
			return
		}
		index++
	}
}

func (lfh *lockFreeHashBasic) Load(key string) (interface{}, bool) {
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
		if memo := lfh.getHash(index); hash == memo && k == key {
			return lfh.getValue(index)
		}
		index++
	}
}

func (lfh *lockFreeHashBasic) Store(key string, value interface{}) {
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
			fmt.Printf("key value new: %d\n", index)
			lfh.setHash(index, hash)
			lfh.setKey(index, key)
			lfh.setValue(index, value)

			newCount := uint64(atomic.AddInt64(&lfh.count, 1))
			_ = newCount // we will need this below...

			if distance<<4 > lfh.size || newCount<<2 > lfh.size {
				lfh.grow()
			}
			return
		}
		if memo := lfh.getHash(index); hash == memo && k == key {
			// update value at this index
			fmt.Printf("key value update: %d\n", index)
			lfh.setValue(index, value)
			return
		}
		index++
		distance++
	}
}

func (lfh *lockFreeHashBasic) grow() {
	fmt.Printf("TODO: implement grow\n")
}
