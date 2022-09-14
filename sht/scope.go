package sht

type Scope struct {
	Context   *Context // allows directives to save context information during execution
	root      *Scope
	parent    *Scope
	destroyed bool
	children  map[*Scope]bool
	data      map[string]interface{}
}

func NewRootScope() *Scope {
	scope := &Scope{
		Context: NewContext(),
		data:    map[string]interface{}{},
	}
	scope.root = scope
	return scope
}

// New Creates a new child scope
//
// The parent scope will propagate change events
func (s *Scope) New(isolate bool) *Scope {
	child := &Scope{
		Context: s.Context,
		root:    s.root,
		data:    map[string]interface{}{},
	}
	if !isolate {
		child.parent = s
	}
	return child
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
	target := s
	if target.root != s {
		found := false
		for target != nil {
			if _, exists := target.data[key]; exists {
				found = true
				break
			}
			target = target.parent
		}
		if !found {
			target = s
		}
	}

	target.data[key] = value
}

func (s *Scope) Destroy() {
	// We can't destroy a scope that has been already destroyed.
	if s.destroyed {
		return
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
