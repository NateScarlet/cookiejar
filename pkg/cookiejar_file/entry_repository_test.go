package cookiejar_file

import (
	"context"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path"
	"strings"
	"testing"
	"time"

	"github.com/NateScarlet/cookiejar/internal/test_util"
	"github.com/NateScarlet/cookiejar/pkg/cookiejar"
	"github.com/NateScarlet/snapshot/pkg/snapshot"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func snapshotEntryRepository(t *testing.T, repo EntryRepository) {
	data, err := ioutil.ReadFile(repo.Filename())
	require.NoError(t, err)
	snapshot.Match(t, string(data),
		snapshot.OptionExt(".jsonl"),
		test_util.SnapshotOptionCleanDate(),
		snapshot.OptionSkip(1),
	)
}

func TestEntryRepository(t *testing.T) {
	var ctx = context.Background()
	url1, _ := url.Parse("http://example.com")
	var useJar = func(t *testing.T) (cookiejar.Jar, EntryRepository) {
		t.Parallel()
		dir, err := os.MkdirTemp("", strings.Replace(t.Name(), "/", "-", -1))
		require.NoError(t, err)
		t.Cleanup(func() {
			require.NoError(t, os.RemoveAll(dir))
		})
		var filename = path.Join(dir, "cookies.jsonl")
		var repo = NewEntryRepository(filename)
		jar, err := cookiejar.New(ctx, cookiejar.OptionEntryRepository(repo))
		require.NoError(t, err)
		return jar, repo
	}
	t.Run("should able to save", func(t *testing.T) {
		var jar, repo = useJar(t)
		u, _ := url.Parse("http://example.com")
		jar.SetCookies(u, []*http.Cookie{
			{Name: "a", Value: "1", Path: "/"},
		})
		assert.Len(t, jar.Cookies(u), 1)
		snapshotEntryRepository(t, repo)
	})

	t.Run("should able to delete", func(t *testing.T) {
		var jar, repo = useJar(t)
		jar.SetCookies(url1, []*http.Cookie{
			{Name: "a", Value: "1", Path: "/", Expires: time.Now().Add(time.Second)},
		})
		assert.Len(t, jar.Cookies(url1), 1)
		time.Sleep(time.Second + 1)
		assert.Len(t, jar.Cookies(url1), 0)
		snapshotEntryRepository(t, repo)
	})

	t.Run("should remove deleted item after compact", func(t *testing.T) {
		var jar, repo = useJar(t)
		jar.SetCookies(url1, []*http.Cookie{
			{Name: "a", Value: "1", Path: "/", Expires: time.Now().Add(time.Second)},
		})
		assert.Len(t, jar.Cookies(url1), 1)
		time.Sleep(time.Second + 1)
		assert.Len(t, jar.Cookies(url1), 0)
		require.NoError(t, repo.Compact())
		snapshotEntryRepository(t, repo)
	})

	t.Run("should keep latest item after compact", func(t *testing.T) {
		var jar, repo = useJar(t)
		jar.SetCookies(url1, []*http.Cookie{
			{Name: "a", Value: "1", Path: "/", Expires: time.Now().Add(time.Second)},
		})
		jar.SetCookies(url1, []*http.Cookie{
			{Name: "a", Value: "2", Path: "/"},
		})
		require.NoError(t, repo.Compact())
		snapshotEntryRepository(t, repo)
	})

	t.Run("should able to read", func(t *testing.T) {
		var jar, repo = useJar(t)
		jar.SetCookies(url1, []*http.Cookie{
			{Name: "a", Value: "1", Path: "/"},
		})
		jar2, err := cookiejar.New(ctx, cookiejar.OptionEntryRepository(repo))
		require.NoError(t, err)
		assert.Len(t, jar2.Cookies(url1), 1)
	})

	t.Run("should able to read before write", func(t *testing.T) {
		var jar, _ = useJar(t)
		assert.Len(t, jar.Cookies(url1), 0)
	})
}
