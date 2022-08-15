package sht

// Dynamic represents a dynamic part of a Compiled
type Dynamic interface {
	// Exec a function that takes a scope argument, and returns a list of dynamic content.
	// @return nil, string, *Compiled, *Rendered
	Exec(scope *Scope) interface{}
}

// DynamicFunc permite usar uma função como um Dynamic
type DynamicFunc func(scope *Scope) interface{}

func (f DynamicFunc) Exec(scope *Scope) interface{} {
	return f(scope)
}

// DynamicCompiled um dynamic que apenas executa um compiled
type DynamicCompiled struct {
	Compiled *Compiled
}

func (d *DynamicCompiled) Exec(scope *Scope) interface{} {
	return d.Compiled.Exec(scope)
}
