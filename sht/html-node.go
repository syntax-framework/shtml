package sht

import (
	"bytes"
	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
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

	Type       NodeType
	Data       string
	DataAtom   atom.Atom
	Namespace  string
	Attributes *Attributes
	//AttrList   []*Attribute
	File   string
	Line   int
	Column int
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

// GetChildNodes returns a collection (list) of an elements's child nodes.
func (n *Node) GetChildNodes() []*Node {
	var childNodes []*Node
	for child := n.FirstChild; child != nil; child = child.NextSibling {
		childNodes = append(childNodes, child)
	}
	return childNodes
}

// Remove remove this node from parent node
func (n *Node) Remove() {
	if n.Parent != nil {
		n.Parent.RemoveChild(n)
	}
}

// RemoveChild removes a node c that is a child of n. Afterwards, c will have
// no parent and no siblings.
//
// It will panic if c's parent is not n.
func (n *Node) RemoveChild(c *Node) {
	if c.Parent != n {
		panic("html: RemoveChild called for a non-child Node")
	}
	if n.FirstChild == c {
		n.FirstChild = c.NextSibling
	}
	if c.NextSibling != nil {
		c.NextSibling.PrevSibling = c.PrevSibling
	}
	if n.LastChild == c {
		n.LastChild = c.PrevSibling
	}
	if c.PrevSibling != nil {
		c.PrevSibling.NextSibling = c.NextSibling
	}
	c.Parent = nil
	c.PrevSibling = nil
	c.NextSibling = nil
}

// ExtractChildren returns a new parent to the children of the current node
func (n *Node) ExtractChildren() *Node {

	newParent := &Node{Type: DocumentNode}

	newParent.FirstChild = n.FirstChild
	newParent.LastChild = n.LastChild

	for n := n.FirstChild; n != nil; n = n.NextSibling {
		n.Parent = newParent
	}

	n.FirstChild = nil
	n.LastChild = nil

	// remove referencias para filhos
	return newParent
}

// ReplaceByText used for element transclude, replace current element with plain text, extracting all node data to a
// separate element
func (n *Node) ReplaceByText() *Node {
	newNode := n.ExtractChildren()

	newNode.Type = n.Type
	newNode.Data = n.Data
	newNode.DataAtom = n.DataAtom
	newNode.Namespace = n.Namespace
	newNode.Attributes = n.Attributes
	//newNode.AttrList = n.AttrList
	newNode.File = n.File
	newNode.Line = n.Line
	newNode.Column = n.Column

	n.Type = TextNode
	n.Data = " "
	n.DataAtom = atom.Plaintext
	n.Namespace = " "
	n.Attributes = &Attributes{Map: map[string]*Attribute{}}
	//n.AttrList = []*Attribute{}

	return newNode
}

// Render renders the parse tree n to string.
func (n *Node) Render() (string, error) {
	w := bytes.NewBufferString("")
	err := html.Render(w, n.toXhtml())
	if err != nil {
		return "", err
	}
	return w.String(), nil
}

// Transverse run callback for node and all its children, until callback returns true
//
// @TODO: Alterar o algoritmo para o formato enter() exit().
// node.transverse(func(enter){ /*before*/ enter() /*after*/  })
func (n *Node) Transverse(callback func(node *Node) (stop bool)) {
	var f func(*Node)
	f = func(o *Node) {
		if callback(o) {
			return
		}
		for d := o.FirstChild; d != nil; d = d.NextSibling {
			f(d)
		}
	}
	f(n)
}

// DebugTag Returns the string representation of the element.
func (n *Node) DebugTag() string {
	w := &bytes.Buffer{}

	if n.Type == TextNode {
		w.WriteString(`<TextNode:`)
		w.WriteString(n.Data)
		w.WriteByte('>')
	} else {
		// Render the <xxx> opening tag.
		w.WriteByte('<')
		w.WriteString(n.Data)
		if n.Attributes != nil && n.Attributes.Map != nil {
			for _, a := range n.Attributes.Map {
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
		}
		w.WriteByte('>')
	}

	if n.File != "" {
		w.WriteString(", File: ")
		w.WriteByte('"')
		w.WriteString(n.File)
		w.WriteByte('"')
		w.WriteString(", Line: ")
		w.WriteString(strconv.Itoa(n.Line))
		w.WriteString(", Column: ")
		w.WriteString(strconv.Itoa(n.Column))
	}
	return w.String()
}

func (n *Node) toXhtml() *html.Node {
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

	attributes := []html.Attribute{}
	//if n.AttrList != nil {
	//	for _, attr := range n.AttrList {
	//		attributes = append(attributes, html.Attribute{
	//			Key:       attr.Name,
	//			Val:       attr.Value,
	//			Namespace: attr.Namespace,
	//		})
	//	}
	//}

	if n.Attributes != nil && n.Attributes.Map != nil {
		for _, attr := range n.Attributes.Map {
			attributes = append(attributes, html.Attribute{
				Key:       attr.Name,
				Val:       attr.Value,
				Namespace: attr.Namespace,
			})
		}
	}

	o.Attr = attributes

	if n.FirstChild != nil {

		var prev *html.Node

		for c := n.FirstChild; c != nil; c = c.NextSibling {
			child := c.toXhtml()
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

// CreateNodeIdentifier creates a function that when invoked, adds a class so that a node can be identified
func CreateNodeIdentifier(sequence *Sequence) func(*Node) string {
	cache := map[*Node]string{}

	return func(node *Node) string {
		if identifier, exists := cache[node]; exists {
			return identifier
		}

		identifier := node.Attributes.Get("id")
		if identifier == "" {
			// new "id"
			identifier = sequence.NextHash()
			node.Attributes.Set("id", identifier)
			identifier = "#" + identifier
			cache[node] = identifier
		} else if identifier != "" && !strings.ContainsRune(identifier, '{') {
			// original "id"
			identifier = "#" + identifier
			cache[node] = identifier
		} else if identifier = node.Attributes.Get("data-syntax-id"); identifier != "" {
			// original "data-syntax-id"
			cache[node] = identifier
		} else {
			// new "data-syntax-id"
			identifier = sequence.NextHash()
			cache[node] = identifier
			node.Attributes.Set("data-syntax-id", identifier)
		}
		return identifier
	}
}
