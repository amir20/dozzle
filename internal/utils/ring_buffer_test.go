package utils

import (
	"reflect"
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
