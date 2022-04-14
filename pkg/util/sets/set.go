package sets

type Set[T comparable] struct {
	m map[T]struct{}
}

func New[T comparable]() Set[T] {
	return Set[T]{
		m: make(map[T]struct{}),
	}
}

func (set *Set[T]) Clear() {
	for key := range set.m {
		delete(set.m, key)
	}
}

func (set *Set[T]) Insert(element T) {
	set.m[element] = struct{}{}
}

func (set *Set[T]) Contains(element T) bool {
	_, present := set.m[element]
	return present
}

func (set *Set[T]) IsEmpty() bool {
	return len(set.m) == 0
}
