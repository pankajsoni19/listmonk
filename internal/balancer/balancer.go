package balancer

// Code is from https://github.com/mr-karan/balance
// Updated for our usecase

import (
	"errors"
	"sync"
)

// ErrDuplicateID error is thrown when attempt to add an ID
// which is already added to the balancer.
var ErrDuplicateID = errors.New("entry already added")

// Item represents the item in the list.
type item struct {
	// id is the id of the item.
	id string
	// weight is the weight of the item that is given by the user.
	weight int
	// current is the current weight of the item.
	current int
}

// Balance represents a smooth weighted round-robin load balancer.
type Balance struct {
	// items is the list of items to balance
	items []*item
	// next is the index of the next item to use.
	next *item

	mutex *sync.RWMutex
}

// NewBalance creates a new load balancer.
func NewBalance() *Balance {
	return &Balance{
		items: make([]*item, 0),
		mutex: &sync.RWMutex{},
	}
}

// Allow chaining
func (b *Balance) Add(id string, weight int) *Balance {
	b.mutex.Lock()
	defer b.mutex.Unlock()
	for _, v := range b.items {
		if v.id == id {
			return b
		}
	}

	b.items = append(b.items, &item{
		id:      id,
		weight:  weight,
		current: 0,
	})

	return b
}

func (b *Balance) Get() string {
	b.mutex.Lock()
	defer b.mutex.Unlock()

	if len(b.items) == 0 {
		return ""
	}

	if len(b.items) == 1 {
		return b.items[0].id
	}

	// Total weight of all items.
	var total int

	// Loop through the list of items and add the item's weight to the current weight.
	// Also increment the total weight counter.
	var max *item
	for _, item := range b.items {
		item.current += item.weight
		total += item.weight

		// Select the item with max weight.
		if max == nil || item.current > max.current {
			max = item
		}
	}

	// Select the item with the max weight.
	b.next = max
	// Reduce the current weight of the selected item by the total weight.
	max.current -= total

	return max.id
}
