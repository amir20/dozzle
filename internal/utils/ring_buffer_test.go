package utils

import (
	"reflect"
	"testing"
)

func TestRingBuffer(t *testing.T) {
	rb := NewRingBuffer[int](3)

	if rb.Len() != 0 {
		t.Errorf("Expected length to be 0, got %d", rb.Len())
	}

	rb.Push(1)
	rb.Push(2)
	rb.Push(3)

	if rb.Len() != 3 {
		t.Errorf("Expected length to be 3, got %d", rb.Len())
	}

	if !rb.Full() {
		t.Errorf("Expected buffer to be full")
	}

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
