package balancer

// Code is from https://github.com/mr-karan/balance
// Updated for our usecase

import (
	"strconv"
	"strings"
	"sync"

	"github.com/knadh/listmonk/models"
	"golang.org/x/exp/slices"
)

// Item represents the item in the list.
type item struct {
	id string
	// weight is the weight of the item that is given by the user.
	weight int
	// current is the current weight of the item.
	current int

	balancer *Balance
}

// Balance represents a smooth weighted round-robin load balancer.
type Balance struct {
	// items is the list of items to balance
	items []*item
	// next is the index of the next item to use.
	next *item

	mutex *sync.RWMutex

	ids []string
}

type MessengerFrom struct {
	UUID string
	From string
}

// NewBalance creates a new load balancer.
func NewBalance() *Balance {
	return &Balance{
		items: make([]*item, 0),
		mutex: &sync.RWMutex{},
	}
}

func (b *Balance) addWFrom(from string, weight int) *Balance {
	b.items = append(b.items, &item{
		id: from,
		weight: weight,
	})

	b.ids = append(b.ids, from)

	return b
}

func addWeightedFrom(messenger *models.CampaignMessenger) *Balance {
	parts := strings.Split(messenger.WFrom, ",")

	var lastKey string
	choiceVal := make(map[string]int)

	for idx, part := range parts {
		spart := strings.TrimSpace(part)

		if len(parts) == idx+1 && len(spart) == 0 {
			break
		}

		if len(lastKey) > 0 {
			if v, e := strconv.Atoi(spart); e == nil {
				choiceVal[lastKey] = v
				lastKey = ""
				continue
			}
		}

		choiceVal[spart] = 1
		lastKey = spart
	}

	balancer := NewBalance()

	for k, v := range choiceVal {
		balancer.addWFrom(k, v)
	}

	return balancer
}

// Allow chaining
func (b *Balance) Add(messenger *models.CampaignMessenger) *Balance {
	b.mutex.Lock()
	defer b.mutex.Unlock()

	if slices.Contains(b.ids, messenger.UUID) {
		return b
	}

	item := &item{
		id:      messenger.UUID,
		weight:  messenger.Weight,
		current: 0,
		balancer: addWeightedFrom(messenger),
	}

	if len(item.id) == 0 {
		return b
	}

	b.items = append(b.items, item)
	b.ids = append(b.ids, messenger.UUID)

	return b
}

func (b *Balance) All() []*MessengerFrom {
	items := make([]*MessengerFrom, 0)

	for _, item := range b.items {
		items = append(items, &MessengerFrom{
			UUID: item.id,
			From: item.balancer.get().id,
		})
	}

	return items
}

func (b *Balance) GetMF() *MessengerFrom {
	b.mutex.Lock()
	defer b.mutex.Unlock()

	item := b.get()

	return &MessengerFrom{
		UUID: item.id,
		From: item.balancer.get().id,
	}
}

func (b *Balance) get() *item {
	if len(b.items) == 1 {
		return b.items[0]
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

	return max
}

