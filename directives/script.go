package directives

import (
	"github.com/syntax-framework/shtml/jsc"
	"github.com/syntax-framework/shtml/sht"
	"path"
	"strings"
)

// Script It handles scripts that are not inside components, which must therefore be executed after page load
var Script = &sht.Directive{
	Name:       "script",
	Restrict:   sht.ELEMENT,
	Priority:   990,
	Terminal:   true,
	Transclude: true, // will remove <script tag>
	Compile: func(node *sht.Node, attrs *sht.Attributes, t *sht.Compiler) (*sht.DirectiveMethods, error) {

		var assets []string

		if src := node.Attributes.Get("src"); src != "" {
			// external src ("//" = Protocol-relative URL)
			if strings.HasPrefix(src, "http") || strings.HasPrefix(src, "//") {
				asset, err := t.RegisterAssetJsURL(src)
				if err != nil {
					return nil, err
				}

				if value := node.Attributes.Get("integrity"); value != "" {
					asset.Integrity = value
				}

				if value := node.Attributes.Get("crossorigin"); value != "" {
					asset.CrossOrigin = value
				}

				if value := node.Attributes.Get("referrerpolicy"); value != "" {
					asset.ReferrerPolicy = value
				}

				assets = append(assets, asset.Name)
			} else {
				asset, err := t.RegisterAssetJsFilepath(path.Join(path.Dir(node.File), src))
				if err != nil {
					return nil, err
				}
				assets = append(assets, asset.Name)
			}
		} else {
			inlineJs, inlineJsErr := jsc.Compile(node.Parent, node, t.Sequence)
			if inlineJsErr != nil {
				return nil, inlineJsErr
			}
			if inlineJs != nil {
				assets = append(assets, t.RegisterAssetJsContent(inlineJs.Content).Name)
			}
		}

		if assets == nil {
			return nil, nil
		}

		// removes content to no longer be processed, loading any script in syntax is via registered assets
		node.FirstChild = nil
		node.LastChild = nil

		return &sht.DirectiveMethods{
			Process: func(scope *sht.Scope, attrs *sht.Attributes, transclude sht.TranscludeFunc) *sht.Rendered {
				// This directive only tells Syntax that this script is required for rendering
				return &sht.Rendered{Assets: assets}
			},
		}, nil
	},
}
