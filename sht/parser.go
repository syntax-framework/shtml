package sht

import (
	"golang.org/x/net/html"
	"io"
	"strings"
)

// HtmlNodeStack is a Stack of nodes.
type HtmlNodeStack []*html.Node

// Pop pops the Stack. It will panic if s is empty.
func (s *HtmlNodeStack) Pop() *html.Node {
	i := len(*s)
	n := (*s)[i-1]
	*s = (*s)[:i-1]
	return n
}

// Top returns the most recently pushed node, or nil if s is empty.
func (s *HtmlNodeStack) Top() *html.Node {
	if i := len(*s); i > 0 {
		return (*s)[i-1]
	}
	return nil
}

type HtmlParser struct {
	Root  *html.Node
	Stack HtmlNodeStack
}

func (p *HtmlParser) Top() *html.Node {
	if n := p.Stack.Top(); n != nil {
		return n
	}
	return p.Root
}

// AddChild adds a child node n to the Top element, and pushes n onto the Stack
// of open elements if it is an element node.
func (p *HtmlParser) AddChild(n *html.Node) {
	p.Top().AppendChild(n)

	if n.Type == html.ElementNode {
		p.Stack = append(p.Stack, n)
	}
}

// AddElement adds a child element based on the current token.
func (p *HtmlParser) AddElement(token html.Token) {
	p.AddChild(&html.Node{
		Type:     html.ElementNode,
		DataAtom: token.DataAtom,
		Data:     token.Data,
		Attr:     token.Attr,
	})
}

// AddText adds text to the preceding node if it is a text node, or else it
// calls AddChild with a new text node.
func (p *HtmlParser) AddText(text string) {
	if text == "" {
		return
	}

	t := p.Top()
	if n := t.LastChild; n != nil && n.Type == html.TextNode {
		n.Data += text
		return
	}
	p.AddChild(&html.Node{
		Type: html.TextNode,
		Data: text,
	})
}

func ParseHtml(content string) ([]*html.Node, error) {

	tokenizer := html.NewTokenizer(strings.NewReader(content))

	// Iterate until EOF. Any other error will cause an early return.
	p := &HtmlParser{
		Root: &html.Node{Type: html.DocumentNode},
	}

	var err error
	for err != io.EOF {
		// CDATA sections are allowed only in foreign transcludeSlots.
		n := p.Stack.Top()
		tokenizer.AllowCDATA(n != nil && n.Namespace != "")
		// Read and parse the transverse token.
		tokenizer.Next()
		token := tokenizer.Token()
		if token.Type == html.ErrorToken {
			err = tokenizer.Err()
			if err != nil && err != io.EOF {
				return nil, err
			}
		}
		switch token.Type {
		case html.TextToken:
			p.AddText(token.Data)
		case html.StartTagToken:
			p.AddElement(token)
			if HtmlVoidElements[token.Data] {
				p.Stack.Pop()
			}
		case html.EndTagToken:
			p.Stack.Pop()
		case html.SelfClosingTagToken:
			p.AddElement(token)
			p.Stack.Pop()
		case html.CommentToken:
			p.AddChild(&html.Node{
				Type: html.CommentNode,
				Data: token.Data,
			})
		}
	}

	var nodes []*html.Node
	// deixa todos os filhos órfãos, para não serem renderizados
	for n := p.Root.FirstChild; n != nil; n = n.NextSibling {
		n.Parent = nil
		nodes = append(nodes, n)
	}

	return nodes, nil
}
