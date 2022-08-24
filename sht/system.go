package sht

import "github.com/syntax-framework/shtml/cmn"

type TemplateSystem struct {
	Loader     func(filepath string) (string, error)
	Directives *Directives
}

// Load load an html file
func (s *TemplateSystem) Load(filepath string) (string, error) {
	return s.Loader(filepath)
}

func (s *TemplateSystem) Compile(filepath string) (*Compiled, *Context, error) {

	var err error
	var content string
	if content, err = s.Load(filepath); err != nil {
		return nil, nil, err
	}

	compiler := NewCompiler(s)

	var compiled *Compiled
	if compiled, err = compiler.Compile(content, filepath); err != nil {
		return nil, nil, err
	}

	var assets []*cmn.Asset
	for asset, _ := range compiler.Assets {
		assets = append(assets, asset)
	}

	compiled.Assets = assets

	return compiled, compiler.Context, err
}

// NewScope creates a new scope that can be used to render a compiled
func (s *TemplateSystem) NewScope() *Scope {
	return NewRootScope()
}
