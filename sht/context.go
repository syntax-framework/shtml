package sht

import "github.com/syntax-framework/shtml/cmn"

// Context simple framework for accessing the compile and run context.
//
// Context can be used by directives to exchange or expose execution information
type Context struct {
	Data   map[string]interface{}
	Timing *cmn.ServerTiming
}

func NewContext() *Context {
	return &Context{
		Data:   map[string]interface{}{},
		Timing: &cmn.ServerTiming{},
	}
}

// Get some value from the context
func (s *Context) Get(key string) interface{} {
	value, exists := s.Data[key]
	if !exists {
		return nil
	}
	return value
}

func (s *Context) GetOrDefault(key string, dfault interface{}) interface{} {
	value, exists := s.Data[key]
	if !exists {
		return dfault
	}
	return value
}

// Set Save some data in context
func (s *Context) Set(key string, value interface{}) {
	s.Data[key] = value
}
