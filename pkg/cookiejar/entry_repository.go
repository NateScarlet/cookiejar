package cookiejar

import (
	"context"
)

type EntryIterator interface {
	ForEach(cb func(i Entry) (err error)) (err error)
}

type EntryIteratorFunc func(cb func(i Entry) (err error)) (err error)

func (fn EntryIteratorFunc) ForEach(cb func(i Entry) (err error)) (err error) {
	return fn(cb)
}

type EntryRepository interface {
	Find(ctx context.Context, key string) EntryIterator
	DeleteMany(ctx context.Context, id []string) (err error)
	Delete(ctx context.Context, id string) (err error)
	// Save should keep CreationTime and CreationIndex form previously saved entry.
	Save(ctx context.Context, entry Entry) (err error)
}
