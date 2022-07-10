# cookiejar

[![godev](https://img.shields.io/static/v1?label=godev&message=reference&color=00add8)](https://pkg.go.dev/github.com/NateScarlet/cookiejar/pkg)
[![build status](https://github.com/NateScarlet/cookiejar/workflows/Go/badge.svg)](https://github.com/NateScarlet/cookiejar/actions)

A fork of `net/http/cookiejar` that saves cookie entries to a Repository.

Supports:

- in-memory Repository (default)
- file Repository (package `cookiejar_file` )
- custom Repository (implements `cookiejar.EntryRepository` yourself)
- multi Repository (use `cookiejar.NewMultiEntryRepository` for cache)
