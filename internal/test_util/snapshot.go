package test_util

import (
	"time"

	"github.com/NateScarlet/snapshot/pkg/snapshot"
)

func SnapshotOptionCleanDate() snapshot.Option {
	var now = time.Now().UTC()
	var patternLayout = `2006-01-02(?: |T)15:\d\d:.+Z`
	return snapshot.OptionCleanRegex(
		snapshot.CleanAs(`*now*`),
		now.Format(patternLayout),
		now.Add(-time.Minute).Format(patternLayout),
	)
}
