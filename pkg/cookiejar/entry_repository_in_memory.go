package cookiejar

import (
	"context"
	"sync"
)

type entryRepositoryInMemory struct {
	mu      sync.Mutex
	m       map[string]map[string]Entry
	keyByID map[string]string
}

// Delete implements EntryRepository
func (r *entryRepositoryInMemory) Delete(ctx context.Context, id string) (err error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	var key = r.keyByID[id]
	var m = r.m[key]
	delete(m, id)
	delete(r.keyByID, id)
	return
}

// DeleteMany implements EntryRepository
func (r *entryRepositoryInMemory) DeleteMany(ctx context.Context, id []string) (err error) {
	for _, i := range id {
		err = r.Delete(ctx, i)
		if err != nil {
			return
		}
	}
	return
}

// Find implements EntryRepository
func (r *entryRepositoryInMemory) Find(ctx context.Context, key string) EntryIterator {
	r.mu.Lock()
	defer r.mu.Unlock()
	return EntryIteratorFunc(func(cb func(i Entry) (err error)) (err error) {
		for _, i := range r.m[key] {
			err = cb(i)
			if err != nil {
				return
			}
		}
		return
	})
}

// Save implements EntryRepository
func (r *entryRepositoryInMemory) Save(ctx context.Context, e Entry) (err error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	m := r.m[e.key]
	if m == nil {
		m = make(map[string]Entry)
		r.m[e.key] = m
	}
	if old, ok := m[e.ID()]; ok {
		e.creation = old.creation
		e.creationIndex = old.creationIndex
	}
	m[e.ID()] = e
	r.keyByID[e.ID()] = e.key
	return
}

func NewInMemoryEntryRepository() EntryRepository {
	return &entryRepositoryInMemory{
		m:       make(map[string]map[string]Entry),
		keyByID: make(map[string]string),
	}
}
