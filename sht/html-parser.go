package sht

import "golang.org/x/net/html"

// nodeStack is a stack of nodes.
type nodeStack []*Node

// pop the stack. It will panic if s is empty.
func (s *nodeStack) pop() *Node {
	i := len(*s)
	n := (*s)[i-1]
	*s = (*s)[:i-1]
	return n
}

// top returns the most recently pushed node, or nil if s is empty.
func (s *nodeStack) top() *Node {
	if i := len(*s); i > 0 {
		return (*s)[i-1]
	}
	return nil
}

type parser struct {
	root  *Node
	stack nodeStack
}

func (p *parser) top() *Node {
	if n := p.stack.top(); n != nil {
		return n
	}
	return p.root
}

// addChild adds a child node n to the top element, and pushes n onto the stack
// of open elements if it is an element node.
func (p *parser) addChild(n *Node) {
	p.top().AppendChild(n)

	if n.Type == ElementNode {
		p.stack = append(p.stack, n)
	}
}

func createNode(token html.Token, file string, line int, column int) *Node {
	node := &Node{
		Data:     token.Data,
		DataAtom: token.DataAtom,
		File:     file,
		Line:     line,
		Column:   column,
	}

	if token.Type == html.TextToken {
		node.Type = TextNode
	} else if token.Type == html.CommentToken {
		node.Type = CommentNode
	} else if token.Type == html.StartTagToken || token.Type == html.SelfClosingTagToken {
		node.Type = ElementNode
		attrList := make([]*Attribute, len(token.Attr))
		for i, attr := range token.Attr {
			attrList[i] = NewAttribute(attr.Key, attr.Val, attr.Namespace)
		}

		attributes := &Attributes{Map: map[string]*Attribute{}}
		for _, attr := range attrList {
			if attr.Normalized != "" {
				attributes.Map[attr.Normalized] = attr
			}
		}

		//node.AttrList = attrList
		node.Attributes = attributes
	}

	return node
}

// addText adds text to the preceding node if it is a text node, or else it
// calls addChild with a new text node.
func (p *parser) addText(text string, file string, line int, column int) {
	if text == "" {
		return
	}

	t := p.top()
	if n := t.LastChild; n != nil && n.Type == TextNode {
		n.Data += text
		return
	}

	p.addChild(&Node{
		Type:   TextNode,
		Data:   text,
		File:   file,
		Line:   line,
		Column: column,
	})
}
