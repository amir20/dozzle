package utils

import (
	"fmt"
	"reflect"
	"sync"
	"testing"
)

func TestRingBuffer(t *testing.T) {
	rb := NewRingBuffer[int](3)

	rb.Push(1)
	rb.Push(2)
	rb.Push(3)

	data := rb.Data()
	expectedData := []int{1, 2, 3}
	if !reflect.DeepEqual(data, expectedData) {
		t.Errorf("Expected data to be %v, got %v", expectedData, data)
	}

	rb.Push(4)
	data = rb.Data()
	expectedData = []int{2, 3, 4}
	if !reflect.DeepEqual(data, expectedData) {
		t.Errorf("Expected data to be %v, got %v", expectedData, data)
	}
}

func TestRingBuffer_DataSnapshot(t *testing.T) {
	for _, pushed := range []int{2, 3, 4} {
		t.Run(fmt.Sprintf("after_%d_pushes", pushed), func(t *testing.T) {
			rb := NewRingBuffer[int](3)
			for i := 1; i <= pushed; i++ {
				rb.Push(i)
			}
			snapshot := rb.Data()
			want := append([]int(nil), snapshot...)
			for i := pushed + 1; i <= pushed+3; i++ {
				rb.Push(i)
			}
			if !reflect.DeepEqual(snapshot, want) {
				t.Fatalf("snapshot changed after Push: got %v, want %v", snapshot, want)
			}
			snapshot[0] = -1
			if got := rb.Data(); got[0] == -1 {
				t.Fatalf("editing snapshot changed buffer: %v", got)
			}
		})
	}
}

func TestRingBuffer_ConcurrentMarshalJSON(t *testing.T) {
	rb := NewRingBuffer[int](3)
	for i := range 3 {
		rb.Push(i)
	}
	var wg sync.WaitGroup
	wg.Go(func() {
		for i := range 1000 {
			rb.Push(i)
		}
	})
	defer wg.Wait()
	for range 1000 {
		if _, err := rb.MarshalJSON(); err != nil {
			t.Fatalf("MarshalJSON: %v", err)
		}
	}
}

func TestRingBuffer_MarshalJSON(t *testing.T) {
	rb := NewRingBuffer[int](3)

	rb.Push(1)
	rb.Push(2)
	rb.Push(3)

	data, err := rb.MarshalJSON()
	if err != nil {
		t.Errorf("Expected error to be nil, got %v", err)
	}

	expectedData := []byte("[1,2,3]")
	if !reflect.DeepEqual(data, expectedData) {
		t.Errorf("Expected data to be %v, got %v", expectedData, data)
	}
}

func TestRingBuffer_ZeroSize(t *testing.T) {
	// A zero-capacity buffer must hold nothing rather than panic. This is
	// reachable from the logs endpoint (?min=0), which builds NewRingBuffer(0).
	rb := NewRingBuffer[int](0)

	rb.Push(1)
	rb.Push(2)

	if rb.Len() != 0 {
		t.Errorf("Expected len to be 0, got %d", rb.Len())
	}

	data := rb.Data()
	if len(data) != 0 {
		t.Errorf("Expected data to be empty, got %v", data)
	}
}

func TestRingBuffer_Clear(t *testing.T) {
	rb := NewRingBuffer[int](3)

	rb.Push(1)
	rb.Push(2)
	rb.Push(3)

	rb.Clear()
	data := rb.Data()
	expectedData := []int{}
	if !reflect.DeepEqual(data, expectedData) {
		t.Errorf("Expected data to be %v, got %v", expectedData, data)
	}
}

func TestRingBufferFrom_CopiesInput(t *testing.T) {
	src := make([]int, 2, 10)
	src[0], src[1] = 1, 2
	rb := RingBufferFrom(3, src)
	rb.Push(3)
	src[0] = -1
	if got := rb.Data(); !reflect.DeepEqual(got, []int{1, 2, 3}) {
		t.Fatalf("buffer shares storage with input: got %v", got)
	}
}
