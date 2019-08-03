package cuba

import (
	"runtime"
	"sync"
	"sync/atomic"
)

type CubaFunc func(interface{}) []interface{}

type CubaStack struct {
	mutex      *sync.Mutex
	items      []interface{}
	cond       *sync.Cond
	numWorkers int32
	maxWorkers int32
	closed     bool
	workerFunc CubaFunc
	wg         *sync.WaitGroup
}

func NewStack(worker CubaFunc) *CubaStack {
	m := &sync.Mutex{}
	return &CubaStack{
		mutex:      m,
		cond:       sync.NewCond(m),
		workerFunc: worker,
		maxWorkers: int32(runtime.NumCPU()),
		wg:         &sync.WaitGroup{},
	}
}

func (ws *CubaStack) Close() {
	ws.mutex.Lock()
	defer ws.mutex.Unlock()

	ws.closed = true
	ws.cond.Broadcast()
}

func (ws *CubaStack) Run() {
	ws.Close()
	ws.wg.Wait()
}

func (ws *CubaStack) Push(items []interface{}) {
	ws.mutex.Lock()
	defer ws.mutex.Unlock()

	if ws.numWorkers < ws.maxWorkers {
		ws.wg.Add(1)
		go ws.runWorker()
	}

	ws.items = append(ws.items, items...)
	ws.cond.Signal()
}

func (ws *CubaStack) Next() (interface{}, bool) {
	ws.mutex.Lock()
	defer ws.mutex.Unlock()

	for !ws.closed && len(ws.items) == 0 {
		ws.cond.Wait()
	}

	if ws.closed && len(ws.items) == 0 {
		return nil, false
	}

	item := ws.items[len(ws.items)-1]
	ws.items = ws.items[:len(ws.items)-1]


	return item, ws.closed
}

func (ws *CubaStack) runWorker() {
	atomic.AddInt32(&ws.numWorkers, 1)
	for {
		item, ok := ws.Next()
		if !ok {
			break
		}

		newItems := ws.workerFunc(item)
		ws.Push(newItems)
	}
	atomic.AddInt32(&ws.numWorkers, -1)

	ws.wg.Done()
}
