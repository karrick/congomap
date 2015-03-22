package congomap

type Congomap interface {
	Delete(string)
	Halt()
	Keys() []string
	Load(string) (interface{}, bool)
	LoadStore(string) (interface{}, error)
	Lookup(func(string) (interface{}, error)) error
	Store(string, interface{})
}

type CongomapSetter func(Congomap) error

func Lookup(lookup func(string) (interface{}, error)) CongomapSetter {
	return func(cgm Congomap) error {
		return cgm.Lookup(lookup)
	}
}
