package congomap

import "time"

// Congomap is the interface implemented by an object that acts as a concurrent go map to store data
// in a key-value data store.
type Congomap interface {
	// Close releases resources used by the Congomap.
	//
	//	cgm, err := congomap.NewTwoLevelMap(congomap.Lookup(lookup))
	//	if err != nil {
	//	    log.Fatal(err)
	//	}
	//	defer cgm.Close()
	Close() error

	// Delete removes a key value pair from a Congomap.
	//
	//	cgm, err := congomap.NewTwoLevelMap()
	//	if err != nil {
	//	    log.Fatal(err)
	//	}
	//	defer cgm.Close()
	//
	//	cgm.Store("someKey", 42)
	//	cgm.Delete("someKey") // if declared, reaper is called during this delete.
	Delete(string)

	// GC forces elimination of keys in Congomap with values that have expired.
	//
	//	// Note no default TTL is defined, so values will never expire by default.
	//	cgm, err := congomap.NewTwoLevelMap()
	//	if err != nil {
	//	    log.Fatal(err)
	//	}
	//	defer cgm.Close()
	//
	//	cgm.Store("someKey", &ExpiringValue{Value: 42, Expiry: time.Now().Add(time.Millisecond)})
	//	cgm.GC() // if declared, the reaper would have been called with 42 as its arguement
	GC()

	// Keys returns an array of key-values stored in the map.
	//
	//	cgm, err := congomap.NewTwoLevelMap(congomap.Lookup(lookup))
	//	if err != nil {
	//	    log.Fatal(err)
	//	}
	//	defer cgm.Close()
	//
	//	cgm.Store("abc", 123)
	//	cgm.Store("def", 456)
	//	keys := cgm.Keys() // returns: []string{"abc", "def"}
	Keys() []string

	// Load gets the value associated with the given key. When the key is in the map, it returns
	// the value associated with the key and true. Otherwise it returns nil for the value and
	// false.
	//
	//	cgm, err := congomap.NewTwoLevelMap()
	//	if err != nil {
	//	    log.Fatal(err)
	//	}
	//	defer cgm.Close()
	//
	//	cgm.Store("someKey", 42)
	//	value, ok := cgm.Load("someKey")
	//	// value is set to 42, and ok is set to false
	Load(string) (interface{}, bool)

	// LoadStore gets the value associated with the given key if it's in the map. If it's not in
	// the map, it calls the lookup function, and sets the value in the map to that returned by
	// the lookup function.
	//
	//	// Define a lookup function to be invoked when LoadStore is called for a key not stored in the
	//	// Congomap.
	//	lookup := func(key string) (interface{}, error) {
	//	    return someLenghyComputation(key), nil
	//	}
	//
	//	// Create a Congomap, specifying what the lookup callback function is.
	//	cgm, err := congomap.NewTwoLevelMap(congomap.Lookup(lookup))
	//	if err != nil {
	//	    log.Fatal(err)
	//	}
	//	defer cgm.Close()
	//
	//	// When you use the LoadStore function, and the key is not in the Congomap, the lookup funciton is
	//	// invoked, and the return value is stored in the Congomap and returned to the program.
	//	value, err := cgm.LoadStore("someKey")
	//	if err != nil {
	//	    // if lookup function returns with an error
	//	}
	LoadStore(string) (interface{}, error)

	// Pairs returns a channel through which key value pairs are read. Pairs will lock the
	// Congomap so that no other accessors can be used until the returned channel is closed.
	//
	// TODO: In next version, should return a channel of Pair structures, rather than channel of
	// pointers to Pair structures.
	//
	//	cgm, err := congomap.NewTwoLevelMap(congomap.Lookup(lookup))
	//	if err != nil {
	//	    log.Fatal(err)
	//	}
	//	defer cgm.Close()
	//
	//	cgm.Store("abc", 123)
	//	cgm.Store("def", 456)
	//
	//	for pair := range cgm.Pairs() {
	//		fmt.Println(pair.Key, pair.Value)
	//	}
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
//
//	// Define a lookup function to be invoked when LoadStore is called for a key not stored in the
//	// Congomap.
//	lookup := func(key string) (interface{}, error) {
//	    return someLenghyComputation(key), nil
//	}
//
//	// Create a Congomap, specifying what the lookup callback function is.
//	cgm, err := congomap.NewTwoLevelMap(congomap.Lookup(lookup))
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer cgm.Close()
//
//	// You can still use the regular Load and Store functions, which will not invoke the lookup
//	// function.
//	cgm.Store("someKey", 42)
//	value, ok := cgm.Load("someKey")
//	if !ok {
//	    // key is not in the Congomap
//	}
//
//	// When you use the LoadStore function, and the key is not in the Congomap, the lookup funciton is
//	// invoked, and the return value is stored in the Congomap and returned to the program.
//	value, err := cgm.LoadStore("someKey")
//	if err != nil {
//	    // lookup function returned an error
//	}
func Lookup(lookup func(string) (interface{}, error)) Setter {
	return func(cgm Congomap) error {
		return cgm.Lookup(lookup)
	}
}

// Reaper is used to specify what function is to be called when garbage collecting item from the
// Congomap.
//
//	// Create a Congomap, specifying what the reaper callback function is.
//	cgm, err := congomap.NewTwoLevelMap(congomap.Reaper(func(value interface{}){
//		fmt.Println("value", value, "expired")
//	}))
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer cgm.Close()
//
//	cgm.Store("someKey", 42) // reaper is not called because nothing was replaced
//	cgm.Store("someKey", 13) // reaper is called with 42 as its argument
func Reaper(reaper func(interface{})) Setter {
	return func(cgm Congomap) error {
		return cgm.Reaper(reaper)
	}
}

// TTL is used to specify the time-to-live for a key-value pair in the Congomap. Pairs that have
// expired are not immediately Garbage Collected until replaced by a new value, or the GC() method
// is invoked either manually or periodically.
//
//	// Create a Congomap, specifying what the default TTL is.
//	cgm, err := congomap.NewTwoLevelMap(congomap.TTL(5 * time.Minute))
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer cgm.Close()
//
//	// While the below key-value pair will expire 5 minutes from now, there is no guarantee when
//	// the Reaper, if declared, would be called.
//	cgm.Store("someKey", 42)
//
// In the next example, no default TTL is defined, so values will never expire by default. A
// particular value can still have an expiry and will be invalidated after its particular expiry
// passes.
//
//	cgm, err := congomap.NewTwoLevelMap()
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer cgm.Close()
//
//	cgm.Store("someKey", &ExpiringValue{Value: 42, Expiry: time.Now().Add(time.Millisecond)})
//
//	time.Sleep(2 * time.Millisecond)
//
//	value, ok := cgm.Load("someKey")
//	// At this point, the Load would have triggered the Reaper function, if it was declared when
//	// the Congomap was created. Furthermore, value will be nil, and ok will be false.
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

// ErrNoLookupDefined is returned by LoadStore method when a key is
// not found in a Congomap for which there has been no lookup function
// declared.
type ErrNoLookupDefined struct{}

func (e ErrNoLookupDefined) Error() string {
	return "congomap: no lookup callback function set"
}

// ErrInvalidDuration is returned by TTL function when a
// time-to-live of less than or equal to zero is specified.
type ErrInvalidDuration time.Duration

func (e ErrInvalidDuration) Error() string {
	return "congomap: duration must be greater than 0: " + time.Duration(e).String()
}
