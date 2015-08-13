package congomap

import (
	"time"
)

// Congomap objects are useful when you need a concurrent go map.
type Congomap interface {
	Delete(string)
	GC()
	Halt()
	Pairs() <-chan *Pair
	Keys() []string
	Load(string) (interface{}, bool)
	LoadStore(string) (interface{}, error)
	Lookup(func(string) (interface{}, error)) error
	Store(string, interface{})
	TTL(time.Duration) error
}

// Pair objects represent a single key-value pair and are passed
// through the channel returned by the Pairs() method while
// enumerating through the keys and values stored in a Congomap.
type Pair struct {
	Key   string
	Value interface{}
}

// Setter declares the type of function used when creating a
// Congomap to change the instance's behavior.
type Setter func(Congomap) error

// Lookup is used to specify what function is to be called to retrieve
// the value for a key when the LoadStore() method is invoked for a
// key not found in a Congomap.
func Lookup(lookup func(string) (interface{}, error)) Setter {
	return func(cgm Congomap) error {
		return cgm.Lookup(lookup)
	}
}

// TTL is used to specify the time-to-live for a key-value pair in the
// Congomap. Pairs that have expired are not immediately Garbage
// Collected until replaced by a new value, or the GC() method is
// invoked either manually or periodically.
func TTL(duration time.Duration) Setter {
	return func(cgm Congomap) error {
		return cgm.TTL(duration)
	}
}

type expiringValue struct {
	value  interface{}
	expiry int64
}

// ErrNoLookupDefined is returned by LoadStore() method when a key is
// not found in a Congomap for which there has been no lookup function
// declared.
type ErrNoLookupDefined struct{}

func (e ErrNoLookupDefined) Error() string {
	return "congomap: no lookup callback function set"
}

// ErrInvalidDuration is returned by TTL() function when a
// time-to-live of less than or equal to zero is specified.
type ErrInvalidDuration time.Duration

func (e ErrInvalidDuration) Error() string {
	return "congomap: duration must be greater than 0: " + time.Duration(e).String()
}
