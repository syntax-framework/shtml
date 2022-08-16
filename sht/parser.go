package sht

import (
	"bytes"
	"github.com/erinpentecost/byteline"
	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
	"io"
	"strconv"
	"strings"
)

// A NodeType is the type of a Node.
type NodeType uint32

const (
	ErrorNode NodeType = iota
	TextNode
	DocumentNode
	ElementNode
	CommentNode
	DoctypeNode
	// RawNode nodes are not returned by the parser, but can be part of the
	// Node tree passed to func Render to insert raw HTML (without escaping).
	// If so, this package makes no guarantee that the rendered HTML is secure
	// (from e.g. Cross Site Scripting attacks) or well-formed.
	RawNode
	scopeMarkerNode
)

// A Node consists of a NodeType and some Data (tag name for element nodes,
// content for text) and are part of a tree of Nodes. Element nodes may also
// have a Namespace and contain a slice of Attributes. Data is unescaped, so
// that it looks like "a<b" rather than "a&lt;b". For element nodes, DataAtom
// is the atom for Data, or zero if Data is not a known tag name.
//
// An empty Namespace implies a "http://www.w3.org/1999/xhtml" namespace.
// Similarly, "math" is short for "http://www.w3.org/1998/Math/MathML", and
// "svg" is short for "http://www.w3.org/2000/svg".
type Node struct {
	Parent, FirstChild, LastChild, PrevSibling, NextSibling *Node

	Type      NodeType
	Data      string
	DataAtom  atom.Atom
	Namespace string
	file      string
	line      int
	Attr      []Attribute
}

// clone returns a new node with the same type, data and attributes.
// The clone has no parent, no siblings and no children.
func (n *Node) clone() *Node {
	m := &Node{
		Type:     n.Type,
		DataAtom: n.DataAtom,
		Data:     n.Data,
		Attr:     make([]Attribute, len(n.Attr)),
	}
	copy(m.Attr, n.Attr)
	return m
}

// DebugTag Returns the string representation of the element.
func (n *Node) DebugTag() string {

	if n.Type == TextNode {
		return HtmlEscape(n.Data)
	}

	// Render the <xxx> opening tag.
	w := &bytes.Buffer{}
	w.WriteByte('<')
	w.WriteString(n.Data)
	for _, a := range n.Attr {
		w.WriteByte(' ')
		if a.Namespace != "" {
			w.WriteString(a.Namespace)
			w.WriteByte(':')
		}
		w.WriteString(a.Name)
		w.WriteString(`="`)
		w.WriteString(HtmlEscape(a.Value))
		w.WriteByte('"')
	}
	w.WriteByte('>')

	if n.file != "" {
		w.WriteString(" at ")
		w.WriteString(n.file)
		w.WriteByte(':')
		w.WriteString(strconv.Itoa(n.line))
	}
	return w.String()
}

// AppendChild adds a node c as a child of n.
//
// It will panic if c already has a parent or siblings.
func (n *Node) AppendChild(c *Node) {
	if c.Parent != nil || c.PrevSibling != nil || c.NextSibling != nil {
		panic("html: AppendChild called for an attached child Node")
	}
	last := n.LastChild
	if last != nil {
		last.NextSibling = c
	} else {
		n.FirstChild = c
	}
	n.LastChild = c
	c.Parent = n
	c.PrevSibling = last
}

func (n *Node) toHtmlNode() *html.Node {
	o := &html.Node{
		Data:      n.Data,
		DataAtom:  n.DataAtom,
		Namespace: n.Namespace,
	}
	switch n.Type {
	case ElementNode:
		o.Type = html.ElementNode
	case TextNode:
		o.Type = html.TextNode
	case ErrorNode:
		o.Type = html.ErrorNode
	case DocumentNode:
		o.Type = html.DocumentNode
	case CommentNode:
		o.Type = html.CommentNode
	case DoctypeNode:
		o.Type = html.DoctypeNode
	case RawNode:
		o.Type = html.RawNode
	}

	attributes := make([]html.Attribute, len(n.Attr))
	for _, attr := range n.Attr {
		attributes = append(attributes, html.Attribute{
			Key:       attr.Name,
			Val:       attr.Value,
			Namespace: attr.Namespace,
		})
	}
	o.Attr = attributes

	if n.FirstChild != nil {

		var prev *html.Node

		for c := n.FirstChild; c != nil; c = c.NextSibling {
			child := c.toHtmlNode()
			child.Parent = o

			if o.FirstChild == nil {
				o.FirstChild = child
			}

			if prev != nil {
				prev.NextSibling = child
				child.PrevSibling = prev
			}
			prev = child
		}

		o.LastChild = prev
	}

	return o
}

// HtmlNodeStack is a Stack of nodes.
type HtmlNodeStack []*Node

// Pop pops the Stack. It will panic if s is empty.
func (s *HtmlNodeStack) Pop() *Node {
	i := len(*s)
	n := (*s)[i-1]
	*s = (*s)[:i-1]
	return n
}

// Top returns the most recently pushed node, or nil if s is empty.
func (s *HtmlNodeStack) Top() *Node {
	if i := len(*s); i > 0 {
		return (*s)[i-1]
	}
	return nil
}

type HtmlParser struct {
	Root  *Node
	Stack HtmlNodeStack
}

func (p *HtmlParser) Top() *Node {
	if n := p.Stack.Top(); n != nil {
		return n
	}
	return p.Root
}

// AddChild adds a child node n to the Top element, and pushes n onto the Stack
// of open elements if it is an element node.
func (p *HtmlParser) AddChild(n *Node) {
	p.Top().AppendChild(n)

	if n.Type == ElementNode {
		p.Stack = append(p.Stack, n)
	}
}

// AddElement adds a child element based on the current token.
func (p *HtmlParser) AddElement(token html.Token) {
	attributes := make([]Attribute, len(token.Attr))
	for i, attr := range token.Attr {
		attributes[i] = Attribute{
			Name:      attr.Key,
			Value:     attr.Val,
			Namespace: attr.Namespace,
		}
	}
	p.AddChild(&Node{
		Type:     ElementNode,
		Data:     token.Data,
		DataAtom: token.DataAtom,
		Attr:     attributes,
	})
}

// AddText adds text to the preceding node if it is a text node, or else it
// calls AddChild with a new text node.
func (p *HtmlParser) AddText(text string) {
	if text == "" {
		return
	}

	t := p.Top()
	if n := t.LastChild; n != nil && n.Type == TextNode {
		n.Data += text
		return
	}
	p.AddChild(&Node{
		Type: TextNode,
		Data: text,
	})
}

// Multiple directives [{0}{1}, {2}{3}] asking for {4} on: {5}
var errorParseTokenizer = ErrorTemplate(
	"parse:tokenizer",
	"An unexpected error occurred while tokenizing the html.", "Line: %d", "Column: %d", "Caused by: %s",
)

var errorParseEndTag = ErrorTemplate(
	"parse:endingTag",
	"Mismatched ending tag.", "Expected: %d", "Found: %d", "Line: %d", "Column: %d",
)

func ParseHtml(content string) ([]*Node, error) {

	lineTracker := byteline.NewReader(strings.NewReader(content))
	tokenizer := html.NewTokenizer(lineTracker)

	// Iterate until EOF. Any other error will cause an early return.
	parser := &HtmlParser{Root: &Node{Type: DocumentNode}}

	prevCol := 0
	prevLine := 0

	var err error
	for err != io.EOF {
		// CDATA sections are allowed only in foreign transcludeSlots.
		n := parser.Stack.Top()
		tokenizer.AllowCDATA(n != nil && n.Namespace != "")
		// Read and parse the transverse token.
		tokenizer.Next()

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
			parser.AddText(token.Data)
		case html.StartTagToken:
			parser.AddElement(token)
			if HtmlVoidElements[token.Data] {
				parser.Stack.Pop()
			}
		case html.EndTagToken:
			lastPushed := parser.Stack.Pop()
			if lastPushed.DataAtom != token.DataAtom {
				return nil, errorParseEndTag(lastPushed.Data, token.Data, prevLine, prevCol)
			}
		case html.SelfClosingTagToken:
			parser.AddElement(token)
			parser.Stack.Pop()
		case html.CommentToken:
			parser.AddChild(&Node{
				Type: CommentNode,
				Data: token.Data,
			})
		}

		prevCol = curCol
		prevLine = curLine
	}

	var nodes []*Node

	// deixa todos os filhos órfãos, para não serem renderizados
	for n := parser.Root.FirstChild; n != nil; n = n.NextSibling {
		n.Parent = nil
		nodes = append(nodes, n)
	}

	return nodes, nil
}
