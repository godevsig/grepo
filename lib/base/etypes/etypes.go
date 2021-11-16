package etypes

// IntMap is a map with int keys kept in insertion order,
// the result is stable when iterating the map.
type IntMap struct {
	ks []int // in insertion order
	mp map[int]interface{}
}

// NewIntMap returns an empty IntMap with size capacity.
func NewIntMap(size int) *IntMap {
	return &IntMap{make([]int, 0, size), make(map[int]interface{}, size)}
}

// Len reports the len of the map
func (im *IntMap) Len() int {
	return len(im.ks)
}

// Keys returns a int array that contains all keys of the map.
func (im *IntMap) Keys() []int {
	return im.ks
}

// Set sets a value into the map.
// The value will be overwritten if the key is already in the map.
func (im *IntMap) Set(k int, v interface{}) {
	_, has := im.mp[k]
	if !has {
		im.ks = append(im.ks, k)
	}

	im.mp[k] = v
}

// Get gets a value from the map.
// Returns nil, false if the key is not in the map.
func (im *IntMap) Get(k int) (interface{}, bool) {
	v, has := im.mp[k]
	return v, has
}

// Has tests if the map has the key.
func (im *IntMap) Has(k int) bool {
	_, has := im.mp[k]
	return has
}

// Del deletes the key and its corresponding value from the map.
func (im *IntMap) Del(k int) {
	_, has := im.mp[k]
	if !has {
		return
	}
	delete(im.mp, k)
	nks := make([]int, 0, len(im.ks)-1)
	for _, e := range im.ks {
		if e != k {
			nks = append(nks, e)
		}
	}
	im.ks = nks
}
