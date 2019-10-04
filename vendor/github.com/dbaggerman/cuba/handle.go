package cuba

type Handle struct {
	pool  *Pool
	item  interface{}
	items []interface{}
}

func (handle *Handle) Item() interface{} {
	return handle.item
}

func (handle *Handle) Push(item interface{}) {
	handle.items = append(handle.items, item)
}

func (handle *Handle) Sync() {
	// PushAll can return PoolAbortedErr, but we deliberately ignore it
	// silently here.
	handle.pool.PushAll(handle.items)
	handle.items = handle.items[:0]
}
