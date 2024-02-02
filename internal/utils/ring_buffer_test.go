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
