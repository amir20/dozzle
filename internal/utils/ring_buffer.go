package utils

import "encoding/json"

type RingBuffer[T any] struct {
	Size  int
	data  []T
	start int
}

func NewRingBuffer[T any](size int) *RingBuffer[T] {
	return &RingBuffer[T]{
		Size: size,
		data: make([]T, 0, size),
	}
}

func (r *RingBuffer[T]) Push(data T) {
	if len(r.data) == r.Size {
		r.data[r.start] = data
		r.start = (r.start + 1) % r.Size
	} else {
		r.data = append(r.data, data)
	}
}

func (r *RingBuffer[T]) Data() []T {
	if len(r.data) == r.Size {
		return append(r.data[r.start:], r.data[:r.start]...)
	} else {
		return r.data
	}
}

func (r *RingBuffer[T]) Len() int {
	return len(r.data)
}

func (r *RingBuffer[T]) Full() bool {
	return len(r.data) == r.Size
}

func (r *RingBuffer[T]) MarshalJSON() ([]byte, error) {
	return json.Marshal(r.Data())
}
