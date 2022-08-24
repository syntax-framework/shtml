package directives

import (
	"github.com/syntax-framework/shtml/jsc"
	"github.com/syntax-framework/shtml/sht"
)

// Script It handles scripts that are not inside components, which must therefore be executed after page load
var Script = &sht.Directive{
	Name:       "script",
	Restrict:   sht.ELEMENT,
	Priority:   990,
	Terminal:   true,
	Transclude: nil,
	Compile: func(node *sht.Node, attrs *sht.Attributes, t *sht.Compiler) (*sht.DirectiveMethods, error) {
		javascript, err := jsc.Compile(node.Parent, node, t)
		if err != nil {
			return nil, err
		}

		assetJs := t.RegisterAssetJsContent(javascript.Content)
		assets := []string{assetJs.Name}
		assetJs = nil

		return &sht.DirectiveMethods{
			Process: func(scope *sht.Scope, attrs *sht.Attributes, transclude sht.TranscludeFunc) *sht.Rendered {
				return &sht.Rendered{Assets: assets}
			},
		}, nil
	},
}
