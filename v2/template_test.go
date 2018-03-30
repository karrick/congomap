package congomap

import (
	"errors"
	"time"
)

type Template struct {
	lookup func(string) (interface{}, error)
	reaper func(interface{})
	ttl    time.Duration
}

func (cgm *Template) Lookup(lookup func(string) (interface{}, error)) error {
	cgm.lookup = lookup
	return nil
}

func (cgm *Template) Reaper(reaper func(interface{})) error {
	cgm.reaper = reaper
	return nil
}

func (cgm *Template) TTL(duration time.Duration) error {
	if duration <= 0 {
		return ErrInvalidDuration(duration)
	}
	cgm.ttl = duration
	return nil
}

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
