package omap

type Map[KEY comparable, V any] struct {
	entries [][]V
	index   map[KEY]int
}

func NewMap[K comparable, V any](slice [][]V) *Map[K, V] {
	if slice == nil {
		var buf [4][]V
		slice = buf[:0]
	}
	return &Map[K, V]{
		entries: slice,
		index:   make(map[K]int, cap(slice)),
	}
}

func (s *Map[K, V]) Get(key K) V {
	v, _ := s.GetOK(key)
	return v
}

func (s *Map[K, V]) GetOK(key K) (zero V, ok bool) {
	val, ok := s.GetAllOK(key)
	if !ok || len(val) == 0 {
		return zero, false
	}

	return val[0], true
}

func (s *Map[K, V]) GetAll(key K) []V {
	values, _ := s.GetAllOK(key)
	return values
}

func (s *Map[K, V]) GetAllOK(key K) ([]V, bool) {
	if s == nil {
		return nil, false
	}
	idx, ok := s.index[key]
	if !ok {
		return nil, false
	}

	return s.entries[idx], true
}

func (s *Map[K, V]) Set(key K, value V) (new bool) {
	if idx, ok := s.index[key]; ok {
		s.entries[idx] = append(s.entries[idx], value)
		return true
	}
	s.index[key] = len(s.entries)
	s.entries = append(s.entries, []V{value})
	return true
}

func (s *Map[K, V]) Len() int {
	return len(s.entries)
}
