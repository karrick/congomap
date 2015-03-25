package congomap

import (
	"errors"
)

var errNoLookupCallbackSet = errors.New("congomap: no lookup callback function set")

type Congomap interface {
	Delete(string)
	Halt()
	Pairs() <-chan *Pair
	Keys() []string
	Load(string) (interface{}, bool)
	LoadStore(string) (interface{}, error)
	Lookup(func(string) (interface{}, error)) error
	Store(string, interface{})
}

type Pair struct {
	key   string
	value interface{}
}

type CongomapSetter func(Congomap) error

func Lookup(lookup func(string) (interface{}, error)) CongomapSetter {
	return func(cgm Congomap) error {
		return cgm.Lookup(lookup)
	}
}
