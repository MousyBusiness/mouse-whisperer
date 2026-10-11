package eye_tracking

import (
	"time"

	"github.com/mousybusiness/mouse_whisperer/pkg/util"
)

type PositionEvent struct {
	Position  util.Vector
	Timestamp time.Time
}

type EyePositionQueue struct {
	items []PositionEvent
	size  int
}

func NewQueue(size int) *EyePositionQueue {
	return &EyePositionQueue{
		items: make([]PositionEvent, 0, size),
		size:  size,
	}
}
func (q *EyePositionQueue) All() []PositionEvent {
	return q.items
}

func (q *EyePositionQueue) Enqueue(v PositionEvent) {
	if len(q.items) >= q.size {
		q.Dequeue()
	}
	q.items = append(q.items, v)
}

func (q *EyePositionQueue) Dequeue() (PositionEvent, bool) {
	if len(q.items) == 0 {
		return PositionEvent{}, false
	}
	item := q.items[0]
	q.items = q.items[1:]
	return item, true
}

func (q *EyePositionQueue) PeekLast() (PositionEvent, bool) {
	if len(q.items) == 0 {
		return PositionEvent{}, false
	}
	item := q.items[len(q.items)-1]
	return item, true
}

func (q *EyePositionQueue) FindNearest(t time.Time) (PositionEvent, bool) {
	// Find the event closest to the given time
	if len(q.items) == 0 {
		return PositionEvent{}, false
	}

	var nearest PositionEvent
	var nearestTime time.Duration
	for _, v := range q.items {
		diff := t.Sub(v.Timestamp)
		if nearestTime == 0 || diff.Abs() < nearestTime.Abs() {
			nearest = v
			nearestTime = diff
		}
	}
	return nearest, true
}
