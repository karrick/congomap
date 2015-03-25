package congomap

import (
	"testing"
)

func TestSyncAtomicLoadBeforeStore(t *testing.T) {
	a, _ := NewSyncAtomicMap()
	actual, _ := a.Load("foo")
	if actual != nil {
		t.Errorf("Actual: %#v; Expected: %#v", actual, nil)
	}
}

func TestSyncAtomicLoadAfterStore(t *testing.T) {
	a, _ := NewSyncAtomicMap()
	a.Store("foo", "bar")
	actual, ok := a.Load("foo")
	if ok != true {
		t.Errorf("Actual: %#v; Expected: %#v", ok, true)
	}
	if actual != "bar" {
		t.Errorf("Actual: %#v; Expected: %#v", actual, "bar")
	}
}

func TestSyncAtomicDelete(t *testing.T) {
	cgm, _ := NewSyncAtomicMap()
	cgm.Store("foo", 13)

	cgm.Delete("foo")

	actual, ok := cgm.Load("foo")
	if actual != nil {
		t.Errorf("Actual: %#v; Expected: %#v", actual, nil)
	}
	if ok != false {
		t.Errorf("Actual: %#v; Expected: %#v", ok, false)
	}
}

func TestSyncAtomicLazyLoadingInvokesSetter(t *testing.T) {
	var setterInvoked bool
	fn := func(key string) (interface{}, error) {
		setterInvoked = true
		return len(key), nil
	}
	cgm, err := NewSyncAtomicMap(Lookup(fn))
	if err != nil {
		t.Fatalf("Actual: %#v; Expected: %#v", err, nil)
	}

	value, err := cgm.LoadStore("someKey")

	if value != 7 {
		t.Errorf("Actual: %#v; Expected: %#v", value, 7)
	}
	if err != nil {
		t.Errorf("Actual: %#v; Expected: %#v", err, nil)
	}
	if !setterInvoked {
		t.Errorf("Actual: %#v; Expected: %#v", setterInvoked, true)
	}
}

func TestSyncAtomicNotLazyLoadingDoesNotInvokeSetter(t *testing.T) {
	var setterInvoked bool
	fn := func(key string) (interface{}, error) {
		setterInvoked = true
		return len(key), nil
	}
	cgm, err := NewSyncAtomicMap(Lookup(fn))
	if err != nil {
		t.Fatalf("Actual: %#v; Expected: %#v", err, nil)
	}

	cgm.Store("someKey", 42)

	value, err := cgm.LoadStore("someKey")

	if value != 42 {
		t.Errorf("Actual: %#v; Expected: %#v", value, 42)
	}
	if err != nil {
		t.Errorf("Actual: %#v; Expected: %#v", err, nil)
	}
	if setterInvoked {
		t.Errorf("Actual: %#v; Expected: %#v", setterInvoked, false)
	}
}

func TestSyncAtomicPairs(t *testing.T) {
	cgm, _ := NewSyncAtomicMap()
	cgm.Store("foo", "FOO")
	cgm.Store("bar", "BAR")
	keys := make([]string, 0)
	values := make([]interface{}, 0)

	for pair := range cgm.Pairs() {
		keys = append(keys, pair.key)
		values = append(values, pair.value)
	}

	if len(keys) != 2 {
		t.Errorf("Actual: %#v; Expected: %#v", len(keys), 2)
	}
	if len(values) != 2 {
		t.Errorf("Actual: %#v; Expected: %#v", len(values), 2)
	}
}
