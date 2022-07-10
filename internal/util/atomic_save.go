package util

import (
	"errors"
	"fmt"
	"os"
)

type AtomicOptions struct {
	tmpSuffix    string
	backupSuffix string
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

func AtomicSave(name string, write func(name string) (err error), options ...AtomicOption) (err error) {
	var opts = newAtomicOptions(options...)
	if opts.tmpSuffix == "" {
		err = fmt.Errorf("empty tmpSuffix")
		return
	}

	var nameTmp = name + opts.tmpSuffix
	var nameBackup = name + opts.backupSuffix
	err = write(nameTmp)
	if err != nil {
		return
	}
	var shouldRemoveBackup bool
	if nameBackup != name {
		err = os.Rename(name, nameBackup)
		shouldRemoveBackup = err == nil
		if errors.Is(err, os.ErrNotExist) {
			err = nil
		}
		if err != nil {
			return
		}
	}
	err = os.Rename(nameTmp, name)
	if err != nil {
		return
	}
	if shouldRemoveBackup {
		err = os.Remove(nameBackup)
		if err != nil {
			return
		}
	}
	return
}
