package congomap

import "errors"

type TemplateMap struct {
	config *Config
}

func NewTemplateMap(config *Config) (*TemplateMap, error) {
	if config == nil {
		config = &Config{}
	}
	return &TemplateMap{config: config}, nil
}

func (cgm *TemplateMap) Close() error {
	return nil
}

func (cgm *TemplateMap) Delete(key string) {
}

func (cgm *TemplateMap) GC() {
}

func (cgm *TemplateMap) Keys() []string {
	return nil
}

func (cgm *TemplateMap) Load(key string) (interface{}, bool) {
	return nil, false
}

func (cgm *TemplateMap) LoadStore(key string) (interface{}, error) {
	return nil, errors.New("TODO")
}

func (cgm *TemplateMap) Pairs() <-chan Pair {
	ch := make(chan Pair)
	go func(ch chan<- Pair) {
		close(ch)
	}(ch)
	return ch
}

func (cgm *TemplateMap) Store(key string, value interface{}) {
}
