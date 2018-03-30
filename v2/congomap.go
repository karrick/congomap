package congomap

import "time"

// Congomap is the interface implemented by an object that acts as a concurrent go map to store data
// in a key-value data store.
type Congomap interface {
	// Close releases resources used by the Congomap.
	Close() error

	// Delete removes a key value pair from a Congomap.
	Delete(string)

	// GC forces elimination of keys in Congomap with values that have expired.
	GC()

	// Keys returns an array of key-values stored in the map.
	Keys() []string

	// Load gets the value associated with the given key. When the key is in the map, it returns
	// the value associated with the key and true. Otherwise it returns nil for the value and
	// false.
	Load(string) (interface{}, bool)

	// LoadStore gets the value associated with the given key if it's in the map. If it's not in
	// the map, it calls the lookup function, and sets the value in the map to that returned by
	// the lookup function.
	LoadStore(string) (interface{}, error)

	// Pairs returns a channel through which key value pairs are read. Pairs will lock the
	// Congomap so that no other accessors can be used until the returned channel is closed.
	//
	// TODO: In next version, should return a channel of Pair structures, rather than channel of
	// pointers to Pair structures.
	Pairs() <-chan *Pair

	// Store sets the value associated with the given key.
	Store(string, interface{})

	Lookup(func(string) (interface{}, error)) error
	Reaper(func(interface{})) error
	TTL(time.Duration) error
}

// Pair objects represent a single key-value pair and are passed through the channel returned by the
// Pairs() method while enumerating through the keys and values stored in a Congomap.
type Pair struct {
	Key   string
	Value interface{}
}

// Setter declares the type of function used when creating a Congomap to change the instance's
// behavior.
type Setter func(Congomap) error

// Lookup is used to specify what function is to be called to retrieve the value for a key when the
// LoadStore() method is invoked for a key not found in a Congomap.
//
// Some maps are used as a lazy lookup device. When a key is not already found in the map, the
// callback function is invoked with the specified key. If the callback function returns an error,
// then a nil value and the error is returned from LoadStore. If the callback function returns no
// error, then the returned value is stored in the Congomap and returned from LoadStore.
func Lookup(lookup func(string) (interface{}, error)) Setter {
	return func(cgm Congomap) error {
		return cgm.Lookup(lookup)
	}
}

// Reaper is used to specify what function is to be called when garbage collecting item from the
// Congomap.
func Reaper(reaper func(interface{})) Setter {
	return func(cgm Congomap) error {
		return cgm.Reaper(reaper)
	}
}

// TTL is used to specify the time-to-live for a key-value pair in the Congomap. Pairs that have
// expired are not immediately Garbage Collected until replaced by a new value, or the GC() method
// is invoked either manually or periodically.
func TTL(duration time.Duration) Setter {
	return func(cgm Congomap) error {
		return cgm.TTL(duration)
	}
}

// ExpiringValue couples a value with an expiry time for the value. The zero value for time.Time
// implies no expiry for this value. If the Store or Lookup method return an ExpiringValue then the
// value will expire with the specified Expiry time.
type ExpiringValue struct {
	Value  interface{}
	Expiry time.Time
}

// helper function to wrap non ExpiringValue items as ExpiringValue items.
func newExpiringValue(value interface{}, defaultDuration time.Duration) *ExpiringValue {
	switch val := value.(type) {
	case *ExpiringValue:
		return val
	default:
		if defaultDuration > 0 {
			return &ExpiringValue{Value: value, Expiry: time.Now().Add(defaultDuration)}
		}
		return &ExpiringValue{Value: value}
	}
}

// ErrNoLookupDefined is returned by LoadStore method when a key is not found in a Congomap for
// which there has been no lookup function declared.
type ErrNoLookupDefined struct{}

func (e ErrNoLookupDefined) Error() string {
	return "congomap: no lookup callback function set"
}

// ErrInvalidDuration is returned by TTL function when a time-to-live of less than or equal to zero
// is specified.
type ErrInvalidDuration time.Duration

func (e ErrInvalidDuration) Error() string {
	return "congomap: duration must be greater than 0: " + time.Duration(e).String()
}
