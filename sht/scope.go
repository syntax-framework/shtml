package sht

type Scope struct {
	Context *Context // allows directives to save context information during execution
	root    *Scope
	parent  *Scope
	data    map[string]interface{}
}

func NewRootScope() *Scope {
	scope := &Scope{
		Context: NewContext(),
		data:    map[string]interface{}{},
	}
	scope.root = scope
	return scope
}

func (s *Scope) New(algo bool, containingScope *Scope) *Scope {
	return s
}

// Get a value from scope or parent scope
func (s *Scope) Get(key string) (value interface{}, exists bool) {
	value, exists = s.data[key]
	if !exists && s.parent != nil {
		value, exists = s.parent.Get(key)
	}
	return
}

// Set a value in scope
func (s *Scope) Set(key string, value interface{}) {
	if s.root == s {
		// fast
		s.data[key] = value
		return
	}

	found := false
	ref := s
	for ref != nil {
		_, exists := ref.data[key]
		if exists {
			found = true
			ref.data[key] = value
			break
		}
		ref = ref.parent
	}
	if !found {
		s.data[key] = value
	}
}

// Fetch used by expressions https://github.com/antonmedv/expr/blob/master/docs/Optimizations.md#reduced-use-of-reflect
func (s *Scope) Fetch(key interface{}) interface{} {
	if keyStr, ok := key.(string); ok {
		value, exists := s.Get(keyStr)
		if exists {
			return value
		}
	}
	return ""
}
