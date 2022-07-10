package util

type Set[T comparable] map[T]struct{}

func (s Set[T]) Add(values ...T) {
	for _, i := range values {
		s[i] = struct{}{}
	}
}
func (s Set[T]) Remove(values ...T) {
	for _, i := range values {
		delete(s, i)
	}
}
func (s Set[T]) Clear() {
	for k := range s {
		delete(s, k)
	}
}
func (s Set[T]) Has(v T) bool {
	_, ok := s[v]
	return ok
}

func NewSet[T comparable](s []T) Set[T] {
	var m = make(Set[T], len(s))
	m.Add(s...)
	return m
}
