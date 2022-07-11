package cookiejar

import (
	"fmt"
	"strings"
	"time"
)

// Entry is the internal representation of a cookie.
//
// This struct type is not used outside of this package per se, but the exported
// fields are those of RFC 6265.
type Entry struct {
	key        string
	name       string
	value      string
	domain     string
	path       string
	sameSite   string
	secure     bool
	httpOnly   bool
	persistent bool
	hostOnly   bool
	expires    time.Time
	creation   time.Time
	order      int
}

// ID returns the domain;path;name triple of e as an ID.
func (e *Entry) ID() string {
	return fmt.Sprintf("%s;%s;%s;%s", e.key, e.domain, e.path, e.name)
}

// shouldSend determines whether e's cookie qualifies to be included in a
// request to host/path. It is the caller's responsibility to check if the
// cookie is expired.
func (e *Entry) shouldSend(https bool, host, path string) bool {
	return e.domainMatch(host) && e.pathMatch(path) && (https || !e.secure)
}

// domainMatch implements "domain-match" of RFC 6265 section 5.1.3.
func (e *Entry) domainMatch(host string) bool {
	if e.domain == host {
		return true
	}
	return !e.hostOnly && hasDotSuffix(host, e.domain)
}

// pathMatch implements "path-match" according to RFC 6265 section 5.1.4.
func (e *Entry) pathMatch(requestPath string) bool {
	if requestPath == e.path {
		return true
	}
	if strings.HasPrefix(requestPath, e.path) {
		if e.path[len(e.path)-1] == '/' {
			return true // The "/any/" matches "/any/path" case.
		} else if requestPath[len(e.path)] == '/' {
			return true // The "/any" matches "/any/path" case.
		}
	}
	return false
}

func (e Entry) IsExpiredAt(t time.Time) bool {
	return e.persistent && !e.expires.After(t)
}

func (obj Entry) Key() string {
	return obj.key
}

func (obj Entry) Name() string {
	return obj.name
}

func (obj Entry) Value() string {
	return obj.value
}

func (obj Entry) Domain() string {
	return obj.domain
}

func (obj Entry) Path() string {
	return obj.path
}

func (obj Entry) SameSite() string {
	return obj.sameSite
}

func (obj Entry) Secure() bool {
	return obj.secure
}

func (obj Entry) HttpOnly() bool {
	return obj.httpOnly
}

func (obj Entry) Persistent() bool {
	return obj.persistent
}

func (obj Entry) HostOnly() bool {
	return obj.hostOnly
}

func (obj Entry) Expires() time.Time {
	return obj.expires
}

func (obj Entry) Creation() time.Time {
	return obj.creation
}

// EntryFromRepository recreate object
// DO NOT use this as constructor
func EntryFromRepository(
	key string,
	name string,
	value string,
	domain string,
	path string,
	sameSite string,
	secure bool,
	httpOnly bool,
	persistent bool,
	hostOnly bool,
	expires time.Time,
	creation time.Time,
	order int,
) (obj *Entry, err error) {
	obj = &Entry{
		key:        key,
		name:       name,
		value:      value,
		domain:     domain,
		path:       path,
		sameSite:   sameSite,
		secure:     secure,
		httpOnly:   httpOnly,
		persistent: persistent,
		hostOnly:   hostOnly,
		expires:    expires,
		creation:   creation,
		order:      order,
	}
	return
}

// Order used when path length and creation time is same.
// Only unique when entry created by same jar object.
func (obj Entry) Order() int {
	return obj.order
}
