package shtml

import (
	"github.com/syntax-framework/shtml/directives"
	"github.com/syntax-framework/shtml/sht"
)

// TemplateSystem interface for configuration, loading and compilation of templates
type TemplateSystem interface {
	Load(filepath string) (string, error)
	Compile(filepath string) (*sht.Compiled, error)
}

var globalDirectives = &sht.Directives{}

// Register a global directive
func Register(directive *sht.Directive) {
	globalDirectives.Add(directive)
}

// New create a new TemplateSystem
func New(loader func(filepath string) (string, error)) TemplateSystem {
	return &sht.TemplateSystem{Loader: loader, Directives: globalDirectives.NewChild()}
}

func init() {
	Register(directives.IFElement)
	Register(directives.IFAttribute)
}
