package util

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type AtomicOptions struct {
	tmpSuffix            string
	backupSuffix         string
	testForceRenameError error
}

func newAtomicOptions(options ...AtomicOption) *AtomicOptions {
	var opts = new(AtomicOptions)
	opts.tmpSuffix = ".tmp"
	opts.backupSuffix = "~"
	for _, i := range options {
		i(opts)
	}
	return opts
}

type AtomicOption func(opts *AtomicOptions)

func AtomicSave(name string, write func(file *os.File) (err error), options ...AtomicOption) (err error) {
	var opts = newAtomicOptions(options...)
	if opts.tmpSuffix == "" {
		err = fmt.Errorf("empty tmpSuffix")
		return
	}
	var nameTmp string
	var nameBackup = name + opts.backupSuffix
	err = func() (err error) {
		var dir = filepath.Dir(name)
		var tmpPattern = filepath.Base(name)
		if index := strings.Index(tmpPattern, "."); index >= 0 {
			// only keep first part
			tmpPattern = tmpPattern[:index]
		}
		if len([]rune(tmpPattern)) > 16 {
			// truncate filename if too long
			tmpPattern = string([]rune(tmpPattern)[:16])
		}
		tmpPattern += "~*" + opts.tmpSuffix
		f, err := os.CreateTemp(dir, tmpPattern)
		if err != nil {
			return
		}
		defer f.Close()
		nameTmp = f.Name()
		err = write(f)
		if err != nil {
			return
		}
		return
	}()
	if err != nil {
		return
	}

	if nameBackup != name {
		err = os.Link(name, nameBackup)
		if errors.Is(err, os.ErrNotExist) {
			err = nil
		} else if errors.Is(err, os.ErrExist) {
			err = os.Remove(nameBackup)
			if err != nil {
				return
			}
			err = os.Link(name, nameBackup)
			if errors.Is(err, os.ErrNotExist) {
				err = nil
			}
		}
		if err != nil {
			return
		}
		defer func() {
			var origErr = err
			err = os.Remove(nameBackup)
			if errors.Is(err, os.ErrNotExist) {
				err = nil
			}
			err = errors.Join(origErr, err)
		}()
	}
	if opts.testForceRenameError != nil {
		return opts.testForceRenameError
	}
	err = os.Rename(nameTmp, name)
	if err != nil {
		return
	}
	return
}

func AtomicOptionBackupSuffix(v string) AtomicOption {
	return func(opts *AtomicOptions) {
		opts.backupSuffix = v
	}
}
