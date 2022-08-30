package sht

import (
	"github.com/erinpentecost/byteline"
	"github.com/syntax-framework/shtml/cmn"
	"golang.org/x/net/html"
	"io"
	"strings"
)

// Multiple directives [{0}{1}, {2}{3}] asking for {4} on: {5}
var errorParseTokenizer = cmn.Err(
	"parse.tokenizer",
	"An unexpected error occurred while tokenizing the html.", "Line: %d", "Column: %d", "Caused by: %s",
)

var errorParseEndTag = cmn.Err(
	"parse.endingTag",
	"Mismatched ending tag.", "Expected: %d", "Found: %d", "Line: %d", "Column: %d",
)

// Parse returns the parse tree for the HTML from the given content.
//
// The input is assumed to be UTF-8 encoded.
func Parse(template string, filepath string) ([]*Node, error) {

	lineTracker := byteline.NewReader(strings.NewReader(template))
	tokenizer := html.NewTokenizer(lineTracker)

	// Iterate until EOF. Any other error will cause an early return.
	p := &parser{root: &Node{Type: DocumentNode}}

	prevCol := 0
	prevLine := 1

	var err error
	for err != io.EOF {
		// CDATA sections are allowed only in foreign transcludeSlots.
		n := p.stack.top()
		tokenizer.AllowCDATA(n != nil && n.Namespace != "")

		// Read and parse the transverse token.
		tokenizer.Next()

		// https://github.com/erinpentecost/vugu/blob/74175693cd5bc1c30bfb603c266c96bf72cb4e3e/vugufmt/formatter.go#L99
		totalOffset, _ := lineTracker.GetCurrentOffset()
		tokenOffset := totalOffset - len(tokenizer.Buffered())
		curLine, curCol, _ := lineTracker.GetLineAndColumn(tokenOffset)

		token := tokenizer.Token()
		if token.Type == html.ErrorToken {
			if err = tokenizer.Err(); err != nil {
				if err != io.EOF {
					return nil, errorParseTokenizer(prevLine, prevCol, err.Error())
				}
			} else {
				return nil, errorParseTokenizer(prevLine, prevCol, "unknown html.ErrorToken")
			}
		}

		switch token.Type {
		case html.TextToken:
			p.addText(token.Data, filepath, prevLine, prevCol)
		case html.StartTagToken:
			p.addChild(createNode(token, filepath, prevLine, prevCol))
			if HtmlVoidElements[token.Data] {
				p.stack.pop()
			}
		case html.EndTagToken:
			lastPushed := p.stack.pop()
			if lastPushed.DataAtom != token.DataAtom {
				return nil, errorParseEndTag(lastPushed.Data, token.Data, prevLine, prevCol)
			}
		case html.SelfClosingTagToken:
			p.addChild(createNode(token, filepath, prevLine, prevCol))
			p.stack.pop()
		case html.CommentToken:
			p.addChild(createNode(token, filepath, prevLine, prevCol))
		}

		prevCol = curCol + 1
		prevLine = curLine + 1
	}

	var nodes []*Node

	// leaves all children orphaned, to not be rendered
	for n := p.root.FirstChild; n != nil; n = n.NextSibling {
		n.Parent = nil
		nodes = append(nodes, n)
	}

	return nodes, nil
}
