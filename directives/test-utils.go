package directives

import "github.com/syntax-framework/shtml/sht"

// testGDs test global directives
var testGDs = &sht.Directives{}

func init() {
	testGDs.Add(DirectiveIFElement)
	testGDs.Add(DirectiveIFAttribute)
}
