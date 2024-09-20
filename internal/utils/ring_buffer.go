package utils

import (
	"sync"

	"encoding/json"
)

type RingBuffer[T any] struct {
	Size  int
	data  []T
	start int
	mutex sync.RWMutex
}

func NewRingBuffer[T any](size int) *RingBuffer[T] {
	return &RingBuffer[T]{
		Size: size,
		data: make([]T, 0, size),
	}
}

func RingBufferFrom[T any](size int, data []T) *RingBuffer[T] {
	if len(data) == 0 {
		return NewRingBuffer[T](size)
	}
	if len(data) > size {
		data = data[len(data)-size:]
	}
	return &RingBuffer[T]{
		Size:  size,
		data:  data,
		start: 0,
	}
}

func (r *RingBuffer[T]) Push(data T) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	if len(r.data) == r.Size {
		r.data[r.start] = data
		r.start = (r.start + 1) % r.Size
	} else {
		r.data = append(r.data, data)
	}
}

func (r *RingBuffer[T]) Len() int {
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	return len(r.data)
}

func (r *RingBuffer[T]) Clear() {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	r.data = r.data[:0]
	r.start = 0
}

func (r *RingBuffer[T]) Data() []T {
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	if len(r.data) == r.Size {
		return append(r.data[r.start:], r.data[:r.start]...)
	} else {
		return r.data
	}
}

func (r *RingBuffer[T]) MarshalJSON() ([]byte, error) {
	return json.Marshal(r.Data())
}
