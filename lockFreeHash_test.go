package congomap

import (
	"fmt"
	"testing"
)

func TestLFHLoad(t *testing.T) {
	lfh, err := NewLockFreeHash()
	if err != nil {
		t.Fatal(err)
	}
	expected := "some value"
	lfh.Store("key", expected)

	fmt.Printf("after store: %v\n", lfh.Dump())

	retrieved, ok := lfh.Load("key")
	if !ok {
		t.Fatalf("Actual: %#v; Expected: %#v", retrieved, expected)
	}
	actual, ok := retrieved.(string)
	if !ok {
		t.Fatalf("Actual: %v; Expected: %v", ok, true)
	}
	if actual != expected {
		t.Errorf("Actual: %v; Expected: %v", actual, expected)
	}
}

func TestLFHDelete(t *testing.T) {
	lfh, err := NewLockFreeHash()
	if err != nil {
		t.Fatal(err)
	}

	lfh.Store("key1", "value1")
	lfh.Store("key2", "value2")
	lfh.Store("key3", "value3")
	lfh.Delete("key2")

	if actual, ok := lfh.Load("key2"); ok {
		t.Errorf("Actual: %#v; Expected: %#v", actual, nil)
	}
	if _, ok := lfh.Load("key4"); ok {
		t.Errorf("Actual: %#v; Expected: %#v", ok, false)
	}
}

func TestLFHStore(t *testing.T) {
	lfh, err := NewLockFreeHash()
	if err != nil {
		t.Fatal(err)
	}

	lfh.Store("key1", "superman")
	lfh.Store("key1", "wonder woman")
	lfh.Store("key2", "batman")
	fmt.Printf("size: %d; count: %d; %v\n", lfh.Size(), lfh.Count(), lfh.Dump())
}

func TestLFHStoreExactlyAvailable(t *testing.T) {
	t.Skip("growing hash not implemented")

	lfh, err := NewLockFreeHash()
	if err != nil {
		t.Fatal(err)
	}

	limit := lfh.Size()
	for i := uint64(0); i < limit; i++ {
		lfh.Store(fmt.Sprintf("key%d", i), fmt.Sprintf("superman%d", i))
	}
	fmt.Printf("map: %v\n", lfh.Dump())
}

func TestLFHStoreMoreThanAvailable(t *testing.T) {
	t.Skip("growing hash not implemented")

	lfh, err := NewLockFreeHash()
	if err != nil {
		t.Fatal(err)
	}

	limit := lfh.Size() * 2
	for i := uint64(0); i < limit; i++ {
		lfh.Store(fmt.Sprintf("key%d", i), fmt.Sprintf("superman%d", i))
	}
	fmt.Printf("map: %v\n", lfh.Dump())
}
