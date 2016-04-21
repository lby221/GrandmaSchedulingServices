package queue

import (
	"conf"
	"message"
	"sync"
)

type GrandmaQueue struct {
	queue     []*message.Obj
	front     int
	pointer   int
	varLock   *sync.RWMutex
	queueLock *sync.Cond
}

var Main_Queue *GrandmaQueue

func init() {
	Main_Queue = new(GrandmaQueue)
	Main_Queue.queue = make([]*message.Obj, conf.GetQueueLength())
	Main_Queue.varLock = new(sync.RWMutex)
	Main_Queue.queueLock = sync.NewCond(new(sync.Mutex))
}

func increase(i int) int {

	if i == conf.GetQueueLength()-1 {
		return 0
	} else {
		return i + 1
	}
}

func decrease(i int) int {
	if i == 0 {
		return conf.GetQueueLength() - 1
	} else {
		return i - 1
	}
}

func NewQueue() *GrandmaQueue {
	q := new(GrandmaQueue)
	q.queue = make([]*message.Obj, conf.GetQueueLength())
	q.varLock = new(sync.RWMutex)
	q.queueLock = sync.NewCond(new(sync.Mutex))

	return q
}

func (q *GrandmaQueue) PushMessage(obj *message.Obj) {
	if q.IsFull() {
		panic("GrandmaQueue full exception")
		return
	}

	q.varLock.Lock()
	q.queue[q.pointer] = obj
	q.pointer = increase(q.pointer)
	q.varLock.Unlock()

	q.queueLock.Signal()
}

func (q *GrandmaQueue) PushFront(obj *message.Obj) {
	if q.IsFull() {
		panic("GrandmaQueue full exception")
		return
	}

	q.varLock.Lock()
	q.front = decrease(q.front)
	q.queue[q.front] = obj
	q.varLock.Unlock()

	q.queueLock.Signal()
}

func (q *GrandmaQueue) PopMessage() *message.Obj {
	q.queueLock.L.Lock()
	for q.IsEmpty() {
		q.queueLock.Wait()
	}

	q.varLock.RLock()
	ret := q.queue[q.front]
	q.varLock.RUnlock()

	q.varLock.Lock()
	q.front = increase(q.front)
	q.varLock.Unlock()

	q.queueLock.L.Unlock()
	return ret
}

func (q *GrandmaQueue) IsEmpty() bool {
	q.varLock.RLock()
	ret := q.front == q.pointer
	q.varLock.RUnlock()

	return ret
}

func (q *GrandmaQueue) IsFull() bool {
	q.varLock.RLock()
	ret := q.front == increase(q.pointer)
	q.varLock.RUnlock()

	return ret
}
