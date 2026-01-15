package expiryheap

import "time"

type ExpiryItem struct {
	Key       string
	ExpiresAt time.Time
	Index     int
}

type ExpiryHeap []*ExpiryItem

func (h ExpiryHeap) Len() int           { return len(h) }
func (h ExpiryHeap) Less(i, j int) bool { return h[i].ExpiresAt.Before(h[j].ExpiresAt) }
func (h ExpiryHeap) Swap(i, j int) {
	h[i], h[j] = h[j], h[i]
	h[i].Index = i
	h[j].Index = j
}

func (h *ExpiryHeap) Push(x interface{}) {
	n := len(*h)
	item := x.(*ExpiryItem)
	item.Index = n
	*h = append(*h, item)
}

func (h *ExpiryHeap) Pop() interface{} {
	old := *h
	n := len(old)
	item := old[n-1]
	old[n-1] = nil
	item.Index = -1
	*h = old[0 : n-1]
	return item
}
