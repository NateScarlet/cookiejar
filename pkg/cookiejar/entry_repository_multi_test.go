package cookiejar

import (
	"context"
	"testing"
)

func TestMultiEntryRepository(t *testing.T) {

	var ctx = context.Background()
	t.Run("should write to all", func(t *testing.T) {
		var repo1 = NewInMemoryEntryRepository().(*entryRepositoryInMemory)
		var repo2 = NewInMemoryEntryRepository().(*entryRepositoryInMemory)

		var repo = NewMultiEntryRepository(repo1, repo2)
		err := repo.Save(ctx, Entry{key: "a"})
		if err != nil {
			t.Error(err)
		}
		if len(repo1.m) != 1 || len(repo1.m) != 1 {
			t.Error("should saved")
		}
	})

	t.Run("should read from first", func(t *testing.T) {
		var repo1 = NewInMemoryEntryRepository().(*entryRepositoryInMemory)
		var repo2 = NewInMemoryEntryRepository().(*entryRepositoryInMemory)

		var repo = NewMultiEntryRepository(repo1, repo2)
		err := repo.Save(ctx, Entry{key: "a"})
		if err != nil {
			t.Error(err)
		}
		var matchCount int
		err = repo.Find(ctx, "a").ForEach(func(i Entry) (err error) {
			matchCount++
			return
		})
		if err != nil {
			t.Error(err)
		}
		if matchCount != 1 {
			t.Error("should match")
		}
	})
	t.Run("should read from second", func(t *testing.T) {
		var repo1 = NewInMemoryEntryRepository().(*entryRepositoryInMemory)
		var repo2 = NewInMemoryEntryRepository().(*entryRepositoryInMemory)

		var repo = NewMultiEntryRepository(repo1, repo2)
		err := repo.Save(ctx, Entry{key: "a"})
		if err != nil {
			t.Error(err)
		}
		for i := range repo1.m {
			delete(repo1.m, i)
		}
		var matchCount int
		err = repo.Find(ctx, "a").ForEach(func(i Entry) (err error) {
			matchCount++
			return
		})
		if err != nil {
			t.Error(err)
		}
		if matchCount != 1 {
			t.Error("should match")
		}
		if len(repo1.m) != 1 {
			t.Error("should write back to first")
		}
	})
}
