package congomap

import (
	"time"
)

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

type Pair struct {
	Key   string
	Value interface{}
}

type CongomapSetter func(Congomap) error

func Lookup(lookup func(string) (interface{}, error)) CongomapSetter {
	return func(cgm Congomap) error {
		return cgm.Lookup(lookup)
	}
}

func TTL(duration time.Duration) CongomapSetter {
	return func(cgm Congomap) error {
		return cgm.TTL(duration)
	}
}

type expiringValue struct {
	value  interface{}
	expiry int64
}

type ErrNoLookupDefined struct{}

func (e ErrNoLookupDefined) Error() string {
	return "congomap: no lookup callback function set"
}

type ErrInvalidDuration time.Duration

func (e ErrInvalidDuration) Error() string {
	return "congomap: duration must be greater than 0: " + time.Duration(e).String()
}
