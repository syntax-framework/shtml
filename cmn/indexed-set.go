package cmn

// IndexedSet a set that maintains the order of the entered items
type IndexedSet struct {
	size   int
	values map[interface{}]int
}

// IsEmpty check if this set is empty
func (m *IndexedSet) IsEmpty() bool {
	return m.size == 0
}

// Cardinality Returns the number of elements in the set.
func (m *IndexedSet) Cardinality() int {
	return m.size
}

// Add an element to the set (if it doesn't exist yet). Returns the item index.
func (m *IndexedSet) Add(value interface{}) int {
	index, exists := m.values[value]
	if !exists {
		index = m.size
		if m.values == nil {
			m.values = map[interface{}]int{}
		}
		m.values[value] = m.size
		m.size++
	}
	return index
}

// GetIndex get the index of an item in the set, or -1 if the item is not in the set
func (m *IndexedSet) GetIndex(value interface{}) int {
	index, exists := m.values[value]
	if !exists {
		return -1
	}
	return index
}

// Contains checks if this set has the item informed
func (m *IndexedSet) Contains(value interface{}) bool {
	_, contains := m.values[value]
	return contains
}

// ToArray get all items keeping insertion order
func (m *IndexedSet) ToArray() []interface{} {
	arr := make([]interface{}, m.size)
	for value, index := range m.values {
		arr[index] = value
	}
	return arr
}

// Difference Returns the difference between this set and other. The returned set will contain all elements of this
//set that are not also elements of other.
func (m *IndexedSet) Difference(other IndexedSet) *IndexedSet {
	diff := &IndexedSet{}
	for value, _ := range m.values {
		if !other.Contains(value) {
			diff.Add(value)
		}
	}
	return diff
}
