package congomap

import (
	"testing"
)

func TestSyncMutexLoadBeforeStore(t *testing.T) {
	a, _ := NewSyncMutexMap()
	actual, _ := a.Load("foo")
	if actual != nil {
		t.Errorf("Actual: %#v; Expected: %#v", actual, nil)
	}
}

func TestSyncMutexLoadAfterStore(t *testing.T) {
	a, _ := NewSyncMutexMap()
	a.Store("foo", "bar")
	actual, ok := a.Load("foo")
	if ok != true {
		t.Errorf("Actual: %#v; Expected: %#v", ok, true)
	}
	if actual != "bar" {
		t.Errorf("Actual: %#v; Expected: %#v", actual, "bar")
	}
}

func TestSyncMutexDelete(t *testing.T) {
	cgm, _ := NewSyncMutexMap()
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

func TestSyncMutexLazyLoadingInvokesSetter(t *testing.T) {
	var setterInvoked bool
	fn := func(key string) (interface{}, error) {
		setterInvoked = true
		return len(key), nil
	}
	cgm, err := NewSyncMutexMap(Lookup(fn))
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

func TestSyncMutexNotLazyLoadingDoesNotInvokeSetter(t *testing.T) {
	var setterInvoked bool
	fn := func(key string) (interface{}, error) {
		setterInvoked = true
		return len(key), nil
	}
	cgm, err := NewSyncMutexMap(Lookup(fn))
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
