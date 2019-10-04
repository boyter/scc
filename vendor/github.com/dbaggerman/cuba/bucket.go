package cuba

import (
	"container/list"
)

type Bucket interface {
	Push(interface{})
	PushAll([]interface{})
	Pop() interface{}
	IsEmpty() bool
	Empty()
}

type Stack struct {
	data []interface{}
}

func NewStack() *Stack {
	return &Stack{}
}

func (stack *Stack) Push(item interface{}) {
	stack.data = append(stack.data, item)
}

func (stack *Stack) PushAll(items []interface{}) {
	stack.data = append(stack.data, items...)
}

func (stack *Stack) Pop() interface{} {
	item := stack.data[len(stack.data)-1]
	stack.data = stack.data[:len(stack.data)-1]
	return item
}

func (stack *Stack) IsEmpty() bool {
	return len(stack.data) == 0
}

func (stack *Stack) Empty() {
	stack.data = nil
}

type Queue struct {
	data *list.List
}

func NewQueue() *Queue {
	return &Queue{
		data: list.New().Init(),
	}
}

func (queue *Queue) Push(item interface{}) {
	queue.data.PushBack(item)
}

func (queue *Queue) PushAll(items []interface{}) {
	for _, item := range items {
		queue.Push(item)
	}
}

func (queue *Queue) Pop() interface{} {
	return queue.data.Remove(queue.data.Front())
}

func (queue *Queue) IsEmpty() bool {
	return queue.data.Len() == 0
}

func (queue *Queue) Empty() {
	queue.data = list.New().Init()
}
