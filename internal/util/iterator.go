package util

type Iterator[T any] interface {
	ForEach(cb func(i T) (err error)) (err error)
}
type IteratorFunc[T any] func(cb func(i T) error) (err error)

func (fn IteratorFunc[T]) ForEach(cb func(i T) error) (err error) {
	if fn == nil {
		return nil
	}
	return fn(cb)
}
