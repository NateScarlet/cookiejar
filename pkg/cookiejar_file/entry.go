package cookiejar_file

import (
	"time"

	"github.com/NateScarlet/cookiejar/pkg/cookiejar"
)

type nullTime struct {
	v time.Time
}

var endOfTime = time.Date(9999, 12, 31, 23, 59, 59, 0, time.UTC)

func (t nullTime) IsNull() bool {
	if t.v.IsZero() {
		return true
	}
	if t.v.Equal(endOfTime) {
		return true
	}

	return false
}

func (t nullTime) Value() time.Time {
	return t.v.UTC()
}

func (t nullTime) PtrValue() *time.Time {
	if t.IsNull() {
		return nil
	}
	var v = t.Value()
	return &v
}

func (t nullTime) ValueOr(d time.Time) time.Time {
	if t.IsNull() {
		return d
	}
	return t.Value()
}

func newNullTime(t *time.Time) *nullTime {
	if t == nil {
		return &nullTime{}
	}
	return &nullTime{*t}
}

type entry struct {
	ID         string     `json:"id,omitempty"`
	Key        string     `json:"key,omitempty"`
	Name       string     `json:"name,omitempty"`
	Value      string     `json:"value,omitempty"`
	Domain     string     `json:"domain,omitempty"`
	Path       string     `json:"path,omitempty"`
	SameSite   string     `json:"sameSite,omitempty"`
	Secure     bool       `json:"secure,omitempty"`
	HttpOnly   bool       `json:"httpOnly,omitempty"`
	Persistent bool       `json:"persistent,omitempty"`
	HostOnly   bool       `json:"hostOnly,omitempty"`
	Expires    *time.Time `json:"expires,omitempty"`
	Creation   *time.Time `json:"creation,omitempty"`
	Deleted    *time.Time `json:"deleted,omitempty"`
	Order      int        `json:"order,omitempty"`
}

func newEntry(do cookiejar.Entry) *entry {
	return &entry{
		ID:         do.ID(),
		Key:        do.Key(),
		Name:       do.Name(),
		Value:      do.Value(),
		Domain:     do.Domain(),
		Path:       do.Path(),
		SameSite:   do.SameSite(),
		Secure:     do.Secure(),
		HttpOnly:   do.HttpOnly(),
		Persistent: do.Persistent(),
		HostOnly:   do.HostOnly(),
		Expires:    nullTime{do.Expires()}.PtrValue(),
		Creation:   nullTime{do.Creation()}.PtrValue(),
		Order:      do.Order(),
	}
}

func (obj entry) DomainObject() (_ *cookiejar.Entry, err error) {
	return cookiejar.EntryFromRepository(
		obj.Key,
		obj.Name,
		obj.Value,
		obj.Domain,
		obj.Path,
		obj.SameSite,
		obj.Secure,
		obj.HttpOnly,
		obj.Persistent,
		obj.HostOnly,
		newNullTime(obj.Expires).ValueOr(endOfTime),
		newNullTime(obj.Creation).Value(),
		obj.Order,
	)
}
