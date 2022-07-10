package cookiejar_file

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"sync"
	"time"

	"github.com/NateScarlet/cookiejar/internal/util"
	"github.com/NateScarlet/cookiejar/pkg/cookiejar"
)

type EntryRepository interface {
	cookiejar.EntryRepository
	Compact() (err error)
	Filename() string
}

type entryRepository struct {
	filename string
	mu       sync.Mutex
}

func (*entryRepository) write(f io.Writer, e ...Entry) (err error) {
	var encoder = json.NewEncoder(f)
	for _, i := range e {
		err = encoder.Encode(i)
		if err != nil {
			return
		}
	}
	return
}

func (r *entryRepository) forEachRaw(cb func(i Entry) (err error)) (err error) {
	f, err := os.Open(r.filename)
	if errors.Is(err, os.ErrNotExist) {
		err = nil
		return
	}
	if err != nil {
		return
	}
	defer f.Close()

	var s = bufio.NewScanner(f)
	for s.Scan() {
		var i = new(Entry)
		err = json.Unmarshal(s.Bytes(), i)
		if err != nil {
			return
		}
		err = cb(*i)
		if err != nil {
			return
		}
	}
	return

}

// Delete implements EntryRepository
func (r *entryRepository) Delete(ctx context.Context, id string) (err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("cookiejar_file: entryRepository.Delete('%s'): %w", id, err)
		}
	}()
	return r.DeleteMany(ctx, []string{id})
}

// DeleteMany implements EntryRepository
func (r *entryRepository) DeleteMany(ctx context.Context, id []string) (err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("cookiejar_file: entryRepository.DeleteMany(%s): %w", id, err)
		}
	}()
	r.mu.Lock()
	defer r.mu.Unlock()
	f, err := os.OpenFile(r.filename, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0600)
	if err != nil {
		return
	}
	defer f.Close()
	var encoder = json.NewEncoder(f)
	for _, i := range id {
		err = encoder.Encode(Entry{
			ID:      i,
			Deleted: nullTime{time.Now()}.PtrValue(),
		})
		if err != nil {
			return
		}
	}
	return
}

func (r *entryRepository) forEach(filter func(i Entry) bool, cb func(i Entry) (err error)) (err error) {
	var m = make(map[string]Entry)
	err = r.forEachRaw(func(i Entry) (err error) {
		if !filter(i) {
			return
		}
		if !newNullTime(i.Deleted).IsNull() {
			delete(m, i.ID)
			return
		}
		if old, ok := m[i.ID]; ok {
			i.Creation = old.Creation
			i.CreationIndex = old.CreationIndex
		}
		m[i.ID] = i
		return
	})
	if err != nil {
		return
	}
	for _, i := range m {
		err = cb(i)
		if err != nil {
			return
		}
	}
	return
}

// Find implements EntryRepository
func (r *entryRepository) Find(ctx context.Context, key string) cookiejar.EntryIterator {
	r.mu.Lock()
	defer r.mu.Unlock()
	return cookiejar.EntryIteratorFunc(func(cb func(i cookiejar.Entry) (err error)) (err error) {
		defer func() {
			if err != nil {
				err = fmt.Errorf("cookiejar_file: entryRepository.Find('%s'): %w", key, err)
			}
		}()
		return r.forEach(func(v Entry) bool { return v.Key == key }, func(i Entry) (err error) {
			do, err := i.DomainObject()
			if err != nil {
				return
			}
			err = cb(*do)
			return
		})
	})
}

// Save implements EntryRepository
func (r *entryRepository) Save(ctx context.Context, entry cookiejar.Entry) (err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("cookiejar_file: entryRepository.Save: %w", err)
		}
	}()
	r.mu.Lock()
	defer r.mu.Unlock()
	f, err := os.OpenFile(r.filename, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0600)
	if err != nil {
		return
	}
	defer f.Close()
	var encoder = json.NewEncoder(f)
	po := NewEntry(entry)
	err = encoder.Encode(po)
	if err != nil {
		return
	}
	return
}

func (r *entryRepository) Compact() (err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("cookiejar_file: entryRepository.Compact: %w", err)
		}
	}()
	r.mu.Lock()
	defer r.mu.Unlock()
	return util.AtomicSave(r.filename, func(name string) (err error) {
		f, err := os.OpenFile(name, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
		if err != nil {
			return
		}
		defer f.Close()

		var encoder = json.NewEncoder(f)
		err = r.forEach(func(i Entry) bool {
			return true
		}, func(i Entry) (err error) {
			return encoder.Encode(i)
		})
		return
	})
}

// NewEntryRepository use filename to store cookies
// will use `.tmp` as tmp file suffix, and `~` as backupSuffix
func NewEntryRepository(filename string) EntryRepository {
	if filename == "" {
		panic("empty filename")
	}
	return &entryRepository{filename: filename}
}

func (obj *entryRepository) Filename() string {
	return obj.filename
}
