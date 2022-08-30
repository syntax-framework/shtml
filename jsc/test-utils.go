package jsc

import (
	"github.com/syntax-framework/shtml/sht"
	"github.com/tdewolff/parse/v2"
	"github.com/tdewolff/parse/v2/js"
	"github.com/tdewolff/test"
	"strings"
	"testing"
)

func testCompileJs(t *testing.T, template string, expected string, getNodes func([]*sht.Node) (*sht.Node, *sht.Node)) {
	nodeList, err := sht.Parse(strings.TrimSpace(template), "template.html")
	if err != nil {
		t.Error(err)
	} else {
		if getNodes == nil {
			getNodes = func(nodeList []*sht.Node) (*sht.Node, *sht.Node) {
				nodeParent := nodeList[0].FirstChild.NextSibling
				nodeScript := nodeParent.NextSibling.NextSibling
				return nodeParent, nodeScript
			}
		}
		nodeParent, nodeScript := getNodes(nodeList)

		asset, jsErr := Compile(nodeParent, nodeScript, &sht.Sequence{})
		if jsErr != nil {
			t.Error(jsErr)
		}

		assetJsAst, assetJsAstErr := js.Parse(parse.NewInputString(asset.Content), js.Options{})
		if assetJsAstErr != nil {
			t.Error(assetJsAstErr)
		} else {
			expectedJsAst, expectedJsAstErr := js.Parse(parse.NewInputString(expected), js.Options{})
			if expectedJsAstErr != nil {
				t.Error(expectedJsAstErr)
			} else {
				assetJs := assetJsAst.JS()
				expectedJs := expectedJsAst.JS()
				test.String(t, assetJs, expectedJs, "jsc.Compile(nodeParent, nodeScript, Sequence) | invalid output")
			}
		}
	}
}
