package congomap

import "time"

// Congomap objects are useful when you need a concurrent go map.
type Congomap interface {
	Close() error
	Delete(string)
	GC()
	Keys() []string
	Load(string) (interface{}, bool)
	LoadStore(string) (interface{}, error)
	Pairs() <-chan Pair
	Store(string, interface{})
}

// Config specifies the configuration of a Congomap to be created.
type Config struct {
	// TTL specifies the default time-to-live for a key-value pair in the Congomap. Pairs that
	// have expired are not immediately garbage collected until replaced by a new value, or the
	// GC() method is invoked either manually or periodically. Using the zero value causes a
	// value to not have a TTL.
	TTL time.Duration

	// Reaper specifies the function to be called on values when a value is expired. Values
	// expire when they are replaced by a new value, either by invoking the Congomap's Store
	// method, or when a value's expiry passes.
	Reaper func(interface{})

	// Lookup specifies the callback function that the Congomap will invoke with the key of an
	// unknown or expired value. If the function returns a nil error, the value is stored in the
	// Congomap. If the function returns a non-nil error, nothing is stored in the Congomap, and
	// the error is passed by from the LoadStore method.
	Lookup func(string) (interface{}, error)
}

// ExpiringValue couples a value with an expiry time for the value. The zero value for time.Time
// implies no expiry for this value. If the Store or Lookup method return a pointer to an
// ExpiringValue then the value will expire with the specified Expiry time. Otherwise the default
// TTL is used for the value. If the Config.TTL was not setup when the Congomap creation, then the
// value will not expire until replaced by invoking the Congomap's Store method.
type ExpiringValue struct {
	// Value stores the raw datum value in the Congomap.
	Value interface{}

	// Expiry indicates the absolute time after which this value is no longer valid and will not
	// be returned by either the Load or the LoadStore methods. The zero value of time indicates
	// no expiry for this particular value and the value will not expire, unless superseded by
	// calling the Store method.
	Expiry time.Time
}

// helper function used internally to wrap non ExpiringValue items as ExpiringValue items.
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

// Pair objects represent a single key-value pair and are passed through the channel returned by the
// Pairs() method while enumerating through the keys and values stored in a Congomap.
type Pair struct {
	Key   string
	Value interface{}
}

// ErrNoLookupDefined is returned by LoadStore() method when a key is not found in a Congomap for
// which there has been no lookup function declared.
type ErrNoLookupDefined struct{}

func (e ErrNoLookupDefined) Error() string {
	return "congomap: no lookup callback function set"
}

// ErrInvalidDuration is returned by TTL() function when a time-to-live of less than or equal to
// zero is specified.
type ErrInvalidDuration time.Duration

func (e ErrInvalidDuration) Error() string {
	return "congomap: duration must be greater than 0: " + time.Duration(e).String()
}
