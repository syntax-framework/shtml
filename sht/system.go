package sht

import (
	"github.com/syntax-framework/shtml/cmn"
	"net/url"
	"path"
	"strings"
)

type TemplateSystem struct {
	Loader     func(filepath string) (string, error)
	Directives *Directives
	Assets     map[*cmn.Asset]bool // All Assets that referenced in this system
}

// Register a global directive
func (s *TemplateSystem) Register(directive *Directive) {
	s.Directives.Add(directive)
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

// RegisterAsset register an asset
func (s *TemplateSystem) RegisterAsset(asset *cmn.Asset) {
	if s.Assets == nil {
		s.Assets = map[*cmn.Asset]bool{}
	}

	if asset.Content != nil && len(asset.Content) > 0 {
		asset.Size = int64(len(asset.Content))

		if asset.Integrity == "" {
			asset.Integrity = "sha512-" + HashSha512Base64(asset.Content)
		}

		if asset.Etag == "" {
			asset.Etag = HashXXH64(asset.Content)
		}
	}

	if strings.HasSuffix(asset.Name, ".js") {
		asset.Name = asset.Name[:len(asset.Name)-3]
	} else if strings.HasSuffix(asset.Name, ".css") {
		asset.Name = asset.Name[:len(asset.Name)-4]
	}

	// avoid duplicated names, on first add
	if s.Assets[asset] != true {
		names := map[string]bool{}
		for oAsset, _ := range s.Assets {
			names[oAsset.Name] = true
		}
		name := asset.Name
		if names[asset.Name] {
			// asset name conflict, if there is duplication, resolve it by adding a suffix this does not create a problem
			// because this process is being carried out at compile time
			asset.Name = name + "-" + HashXXH64([]byte(asset.Name))
		}
	}
	s.Assets[asset] = true
}

// RegisterAssetJsURL register an javascript asset by url
func (s *TemplateSystem) RegisterAssetJsURL(src string) (*cmn.Asset, error) {
	jsUrl, err := url.Parse(src)
	if err != nil {
		return nil, err
	}
	println(jsUrl)
	name := HashXXH64([]byte(src))
	asset := &cmn.Asset{
		Url:  src,
		Name: name,
		Type: cmn.Javascript,
	}

	s.RegisterAsset(asset)

	return asset, nil
}

// RegisterAssetJsFilepath registers an existing javascript on the filesystem being used by this system
func (s *TemplateSystem) RegisterAssetJsFilepath(filepath string) (*cmn.Asset, error) {

	// check if is loaded
	for asset, _ := range s.Assets {
		if asset.Filepath == filepath {
			return asset, nil
		}
	}

	content, err := s.Load(filepath)
	if err != nil {
		return nil, err
	}

	cbytes := []byte(content)
	asset := &cmn.Asset{
		Content:  cbytes,
		Name:     path.Base(filepath),
		Type:     cmn.Javascript,
		Filepath: filepath,
	}

	s.RegisterAsset(asset)

	return asset, nil
}

// RegisterAssetJsContent register an anonymous javascript
func (s *TemplateSystem) RegisterAssetJsContent(content string) *cmn.Asset {
	cbytes := []byte(content)
	asset := &cmn.Asset{
		Content: cbytes,
		Name:    HashXXH64(cbytes),
		Type:    cmn.Javascript,
	}

	s.RegisterAsset(asset)

	return asset
}
