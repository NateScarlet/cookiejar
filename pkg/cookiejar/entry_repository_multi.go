package cookiejar

import "context"

// MultiEntryRepository write to all, read from first non-empty result.
type MultiEntryRepository interface {
	EntryRepository
}

type multiEntryRepository struct {
	targets []EntryRepository
}

func (r multiEntryRepository) parallel(cb func(repo EntryRepository) (err error)) (err error) {
	var errCh = make(chan error, len(r.targets))
	var remains = 0
	for _, target := range r.targets {
		remains++
		go func(repo EntryRepository) {
			errCh <- cb(repo)
		}(target)
	}
	for err := range errCh {
		if err != nil {
			return err
		}
		remains--
		if remains == 0 {
			return nil
		}
	}
	return
}

// Delete implements EntryRepository
func (r multiEntryRepository) Delete(ctx context.Context, id string) (err error) {
	return r.parallel(func(repo EntryRepository) (err error) {
		return repo.Delete(ctx, id)
	})
}

// DeleteMany implements EntryRepository
func (r multiEntryRepository) DeleteMany(ctx context.Context, id []string) (err error) {
	return r.parallel(func(repo EntryRepository) (err error) {
		return repo.DeleteMany(ctx, id)
	})
}

// Find implements EntryRepository
func (r multiEntryRepository) Find(ctx context.Context, key string) EntryIterator {
	return EntryIteratorFunc(func(cb func(i Entry) (err error)) (err error) {
		for index, repo := range r.targets {
			var ok bool
			err = repo.Find(ctx, key).ForEach(func(i Entry) (err error) {
				ok = true
				return cb(i)
			})
			if err != nil {
				return
			}
			if !ok {
				continue
			}
			if index > 0 {
				err = repo.Find(ctx, key).ForEach(func(i Entry) (err error) {
					var repos = multiEntryRepository{r.targets[:index]}
					return repos.Save(ctx, i)
				})
			}
			return
		}
		return
	})
}

// Save implements EntryRepository
func (r multiEntryRepository) Save(ctx context.Context, entry Entry) (err error) {
	return r.parallel(func(repo EntryRepository) (err error) {
		return repo.Save(ctx, entry)
	})
}

func NewMultiEntryRepository(targets ...EntryRepository) EntryRepository {
	if len(targets) == 0 {
		panic("empty targets")
	}

	return multiEntryRepository{
		targets: targets,
	}
}
