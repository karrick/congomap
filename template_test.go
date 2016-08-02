package congomap

import (
	"errors"
	"time"
)

type Template struct {
	lookup func(string) (interface{}, error)
	reaper func(interface{})

	ttlEnabled  bool
	ttlDuration time.Duration
}

// Lookup sets the lookup callback function for this Congomap for use when `LoadStore` is called and
// a requested key is not in the map.
func (cgm *Template) Lookup(lookup func(string) (interface{}, error)) error {
	cgm.lookup = lookup
	return nil
}

// Reaper is used to specify what function is to be called when garbage collecting item from the
// Congomap.
func (cgm *Template) Reaper(reaper func(interface{})) error {
	cgm.reaper = reaper
	return nil
}

// TTL sets the time-to-live for values stored in the Congomap.
func (cgm *Template) TTL(duration time.Duration) error {
	if duration <= 0 {
		return ErrInvalidDuration(duration)
	}
	cgm.ttlDuration = duration
	cgm.ttlEnabled = true
	return nil
}

////////////////////////////////////////

func TemplateTemplateMap() (*Template, error) {
	return &Template{}, nil
}

func (cgm *Template) Close() error {
	return nil
}

func (cgm *Template) Delete(key string) {
}

func (cgm *Template) GC() {
}

func (cgm *Template) Keys() []string {
	return nil
}

func (cgm *Template) Load(key string) (interface{}, bool) {
	return nil, false
}

func (cgm *Template) LoadStore(key string) (interface{}, error) {
	return nil, errors.New("TODO")
}

func (cgm *Template) Pairs() <-chan *Pair {
	ch := make(chan *Pair)
	go func(ch chan<- *Pair) {
		close(ch)
	}(ch)
	return ch
}

func (cgm *Template) Store(key string, value interface{}) {
}
