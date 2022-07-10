// Copyright 2012 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package cookiejar implements http.CookieJar and saves cookie entries to a Repository.
package cookiejar

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/NateScarlet/cookiejar/internal/ascii"
	"golang.org/x/net/publicsuffix"
)

// PublicSuffixList provides the public suffix of a domain. For example:
//      - the public suffix of "example.com" is "com",
//      - the public suffix of "foo1.foo2.foo3.co.uk" is "co.uk", and
//      - the public suffix of "bar.pvt.k12.ma.us" is "pvt.k12.ma.us".
//
// Implementations of PublicSuffixList must be safe for concurrent use by
// multiple goroutines.
//
// An implementation that always returns "" is valid and may be useful for
// testing but it is not secure: it means that the HTTP server for foo.com can
// set a cookie for bar.com.
//
// A public suffix list implementation is in the package
// golang.org/x/net/publicsuffix.
type PublicSuffixList interface {
	// PublicSuffix returns the public suffix of domain.
	//
	// TODO: specify which of the caller and callee is responsible for IP
	// addresses, for leading and trailing dots, for case sensitivity, and
	// for IDN/Punycode.
	PublicSuffix(domain string) string

	// String returns a description of the source of this public suffix
	// list. The description will typically contain something like a time
	// stamp or version number.
	String() string
}

type Jar interface {
	http.CookieJar
}

// jar implements the http.CookieJar interface from the net/http package.
type jar struct {
	psList PublicSuffixList

	entryRepo           EntryRepository
	ctx                 context.Context
	errorCB             func(err error)
	creationIndexOffset int
}

// Options are the options for creating a new Jar.
type Options struct {
	publicSuffixList PublicSuffixList
	entryRepository  EntryRepository
	onError          func(err error)
}

// OptionPublicSuffixList is the public suffix list that determines whether
// an HTTP server can set a cookie for a domain.
//
// defaults to golang.org/x/net/publicsuffix.List
func OptionPublicSuffixList(v PublicSuffixList) Option {
	if v == nil {
		panic("nil public suffix list")
	}
	return func(opts *Options) {
		opts.publicSuffixList = v
	}
}

// OptionEntryRepository specify repository implementation, defaults to in-memory repository.
func OptionEntryRepository(v EntryRepository) Option {
	if v == nil {
		panic("nil entry repository")
	}
	return func(opts *Options) {
		opts.entryRepository = v
	}
}

// OptionOnError defines error callback,
// defaults to panic(err).
func OptionOnError(v func(err error)) Option {
	return func(opts *Options) {
		opts.onError = v
	}
}

func newOptions(options ...Option) *Options {
	var opts = new(Options)
	opts.onError = func(err error) {
		panic(err)
	}
	opts.entryRepository = NewInMemoryEntryRepository()
	opts.publicSuffixList = publicsuffix.List
	for _, i := range options {
		i(opts)
	}
	return opts
}

type Option func(opts *Options)

// New returns a new cookie jar.
func New(ctx context.Context, options ...Option) (Jar, error) {
	var opts = newOptions(options...)
	jar := &jar{
		ctx:       ctx,
		psList:    opts.publicSuffixList,
		entryRepo: opts.entryRepository,
		errorCB:   opts.onError,
	}
	return jar, nil
}

func (j *jar) onError(err error) {
	if err == nil {
		return
	}
	if j.errorCB != nil {
		j.errorCB(err)
	}
}

// hasDotSuffix reports whether s ends in "."+suffix.
func hasDotSuffix(s, suffix string) bool {
	return len(s) > len(suffix) && s[len(s)-len(suffix)-1] == '.' && s[len(s)-len(suffix):] == suffix
}

// Cookies implements the Cookies method of the http.CookieJar interface.
//
// It returns an empty slice if the URL's scheme is not HTTP or HTTPS.
func (j *jar) Cookies(u *url.URL) (cookies []*http.Cookie) {
	cookies, err := j.cookies(u, time.Now())
	j.onError(err)
	return
}

// cookies is like Cookies but takes the current time as a parameter.
func (j *jar) cookies(u *url.URL, now time.Time) (cookies []*http.Cookie, err error) {
	if u.Scheme != "http" && u.Scheme != "https" {
		return
	}
	host, err := canonicalHost(u.Host)
	if err != nil {
		return
	}
	key := jarKey(host, j.psList)

	https := u.Scheme == "https"
	path := u.Path
	if path == "" {
		path = "/"
	}

	var selected []Entry
	var deleteIDs []string
	err = j.entryRepo.Find(j.ctx, key).ForEach(func(e Entry) (err error) {
		if e.IsExpiredAt(now) {
			deleteIDs = append(deleteIDs, e.ID())
			return
		}
		if !e.shouldSend(https, host, path) {
			return
		}
		selected = append(selected, e)
		return
	})
	if err != nil {
		return
	}
	if len(deleteIDs) > 0 {
		err = j.entryRepo.DeleteMany(j.ctx, deleteIDs)
		if err != nil {
			return
		}
	}

	// sort according to RFC 6265 section 5.4 point 2: by longest
	// path and then by earliest creation time.
	sort.Slice(selected, func(i, j int) bool {
		s := selected
		if len(s[i].path) != len(s[j].path) {
			return len(s[i].path) > len(s[j].path)
		}
		if !s[i].creation.Equal(s[j].creation) {
			return s[i].creation.Before(s[j].creation)
		}
		return s[i].creationIndex < s[j].creationIndex
	})
	for _, e := range selected {
		cookies = append(cookies, &http.Cookie{Name: e.name, Value: e.value})
	}

	return
}

// SetCookies implements the SetCookies method of the http.CookieJar interface.
//
// It does nothing if the URL's scheme is not HTTP or HTTPS.
func (j *jar) SetCookies(u *url.URL, cookies []*http.Cookie) {
	err := j.setCookies(u, cookies, time.Now())
	j.onError(err)
}

// setCookies is like SetCookies but takes the current time as parameter.
func (j *jar) setCookies(u *url.URL, cookies []*http.Cookie, now time.Time) (err error) {
	if len(cookies) == 0 {
		return
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return
	}
	host, err := canonicalHost(u.Host)
	if err != nil {
		return
	}
	key := jarKey(host, j.psList)
	defPath := defaultPath(u.Path)

	for index, cookie := range cookies {
		err = func() (err error) {
			e, remove, err := j.newEntry(cookie, now, defPath, host)
			if err != nil {
				return
			}
			e.key = key
			id := e.ID()
			if remove {
				err = j.entryRepo.Delete(j.ctx, id)
				return
			}
			e.creation = now
			e.creationIndex = j.creationIndexOffset + index
			err = j.entryRepo.Save(j.ctx, e)
			if err != nil {
				return
			}
			return
		}()
		if err != nil {
			return
		}
	}
	j.creationIndexOffset += len(cookies)

	return
}

// canonicalHost strips port from host if present and returns the canonicalized
// host name.
func canonicalHost(host string) (string, error) {
	var err error
	if hasPort(host) {
		host, _, err = net.SplitHostPort(host)
		if err != nil {
			return "", err
		}
	}
	if strings.HasSuffix(host, ".") {
		// Strip trailing dot from fully qualified domain names.
		host = host[:len(host)-1]
	}
	encoded, err := toASCII(host)
	if err != nil {
		return "", err
	}
	// We know this is ascii, no need to check.
	lower, _ := ascii.ToLower(encoded)
	return lower, nil
}

// hasPort reports whether host contains a port number. host may be a host
// name, an IPv4 or an IPv6 address.
func hasPort(host string) bool {
	colons := strings.Count(host, ":")
	if colons == 0 {
		return false
	}
	if colons == 1 {
		return true
	}
	return host[0] == '[' && strings.Contains(host, "]:")
}

// jarKey returns the key to use for a jar.
func jarKey(host string, psl PublicSuffixList) string {
	if isIP(host) {
		return host
	}

	var i int

	suffix := psl.PublicSuffix(host)
	if suffix == host {
		return host
	}
	i = len(host) - len(suffix)
	if i <= 0 || host[i-1] != '.' {
		// The provided public suffix list psl is broken.
		// Storing cookies under host is a safe stopgap.
		return host
	}
	// Only len(suffix) is used to determine the jar key from
	// here on, so it is okay if psl.PublicSuffix("www.buggy.psl")
	// returns "com" as the jar key is generated from host.

	prevDot := strings.LastIndex(host[:i-1], ".")
	return host[prevDot+1:]
}

// isIP reports whether host is an IP address.
func isIP(host string) bool {
	return net.ParseIP(host) != nil
}

// defaultPath returns the directory part of an URL's path according to
// RFC 6265 section 5.1.4.
func defaultPath(path string) string {
	if len(path) == 0 || path[0] != '/' {
		return "/" // Path is empty or malformed.
	}

	i := strings.LastIndex(path, "/") // Path starts with "/", so i != -1.
	if i == 0 {
		return "/" // Path has the form "/abc".
	}
	return path[:i] // Path is either of form "/abc/xyz" or "/abc/xyz/".
}

// newEntry creates an entry from a http.Cookie c. now is the current time and
// is compared to c.Expires to determine deletion of c. defPath and host are the
// default-path and the canonical host name of the URL c was received from.
//
// remove records whether the jar should delete this cookie, as it has already
// expired with respect to now. In this case, e may be incomplete, but it will
// be valid to call e.id (which depends on e's Name, Domain and Path).
//
// A malformed c.Domain will result in an error.
func (j *jar) newEntry(c *http.Cookie, now time.Time, defPath, host string) (e Entry, remove bool, err error) {
	e.name = c.Name

	if c.Path == "" || c.Path[0] != '/' {
		e.path = defPath
	} else {
		e.path = c.Path
	}

	e.domain, e.hostOnly, err = j.domainAndType(host, c.Domain)
	if err != nil {
		return e, false, err
	}

	// MaxAge takes precedence over Expires.
	if c.MaxAge < 0 {
		return e, true, nil
	} else if c.MaxAge > 0 {
		e.expires = now.Add(time.Duration(c.MaxAge) * time.Second)
		e.persistent = true
	} else {
		if c.Expires.IsZero() {
			e.expires = endOfTime
			e.persistent = false
		} else {
			if !c.Expires.After(now) {
				return e, true, nil
			}
			e.expires = c.Expires
			e.persistent = true
		}
	}

	e.value = c.Value
	e.secure = c.Secure
	e.httpOnly = c.HttpOnly

	switch c.SameSite {
	case http.SameSiteDefaultMode:
		e.sameSite = "SameSite"
	case http.SameSiteStrictMode:
		e.sameSite = "SameSite=Strict"
	case http.SameSiteLaxMode:
		e.sameSite = "SameSite=Lax"
	}

	return e, false, nil
}

var (
	errIllegalDomain   = errors.New("cookiejar: illegal cookie domain attribute")
	errMalformedDomain = errors.New("cookiejar: malformed cookie domain attribute")
	errNoHostname      = errors.New("cookiejar: no host name available (IP only)")
)

// endOfTime is the time when session (non-persistent) cookies expire.
// This instant is representable in most date/time formats (not just
// Go's time.Time) and should be far enough in the future.
var endOfTime = time.Date(9999, 12, 31, 23, 59, 59, 0, time.UTC)

// domainAndType determines the cookie's domain and hostOnly attribute.
func (j *jar) domainAndType(host, domain string) (string, bool, error) {
	if domain == "" {
		// No domain attribute in the SetCookie header indicates a
		// host cookie.
		return host, true, nil
	}

	if isIP(host) {
		// According to RFC 6265 domain-matching includes not being
		// an IP address.
		// TODO: This might be relaxed as in common browsers.
		return "", false, errNoHostname
	}

	// From here on: If the cookie is valid, it is a domain cookie (with
	// the one exception of a public suffix below).
	// See RFC 6265 section 5.2.3.
	if domain[0] == '.' {
		domain = domain[1:]
	}

	if len(domain) == 0 || domain[0] == '.' {
		// Received either "Domain=." or "Domain=..some.thing",
		// both are illegal.
		return "", false, errMalformedDomain
	}

	domain, isASCII := ascii.ToLower(domain)
	if !isASCII {
		// Received non-ASCII domain, e.g. "perché.com" instead of "xn--perch-fsa.com"
		return "", false, errMalformedDomain
	}

	if domain[len(domain)-1] == '.' {
		// We received stuff like "Domain=www.example.com.".
		// Browsers do handle such stuff (actually differently) but
		// RFC 6265 seems to be clear here (e.g. section 4.1.2.3) in
		// requiring a reject.  4.1.2.3 is not normative, but
		// "Domain Matching" (5.1.3) and "Canonicalized Host Names"
		// (5.1.2) are.
		return "", false, errMalformedDomain
	}

	// See RFC 6265 section 5.3 #5.
	if ps := j.psList.PublicSuffix(domain); ps != "" && !hasDotSuffix(domain, ps) {
		if host == domain {
			// This is the one exception in which a cookie
			// with a domain attribute is a host cookie.
			return host, true, nil
		}
		return "", false, errIllegalDomain
	}

	// The domain must domain-match host: www.mycompany.com cannot
	// set cookies for .ourcompetitors.com.
	if host != domain && !hasDotSuffix(host, domain) {
		return "", false, errIllegalDomain
	}

	return domain, false, nil
}
