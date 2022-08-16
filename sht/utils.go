package sht

import (
	"bytes"
	"fmt"
	"regexp"
	"strings"
)

type StringSet map[string]bool

func (p StringSet) Contains(key string) (exists bool) {
	_, exists = p[key]
	return
}

func (p StringSet) Clone(others ...string) StringSet {
	nmap := StringSet{}
	for k, v := range p {
		nmap[k] = v
	}
	for _, other := range others {
		nmap[other] = true
	}
	return nmap
}

type RegexMatch struct {
	start   int      // The  0-based index of the search at which the result was found.
	end     int      // The  0-based index of the search at which the result was found.
	input   *string  // A copy of the search string.
	text    string   // The full string of characters matched
	group   []string // An array where each entry represents a substring group.
	indices [][2]int // An array where each entry represents a substring RegexMatch (start,end).
}

// RegexExec https://developer.mozilla.org/en-US/docs/Web/JavaScript/Reference/Global_Objects/RegExp/exec
func RegexExec(re *regexp.Regexp, input string) *RegexMatch {
	result := re.FindStringSubmatchIndex(input)
	if result == nil {
		return nil
	}

	return parseSubmatchIndex(result, input)
}

func parseSubmatchIndex(result []int, input string) *RegexMatch {
	size := len(result) / 2
	indices := make([][2]int, size)
	strings := make([]string, size)
	for i := 0; i < size; i++ {
		start, end := result[i*2], result[i*2+1]
		indices[i] = [2]int{start, end}
		if start >= 0 {
			strings[i] = input[start:end]
		}
	}

	//if 2*i < len(a) && a[2*i] >= 0 {
	//  ret[i] = s[a[2*i]:a[2*i+1]]
	//}

	return &RegexMatch{
		start:   result[0],
		end:     result[1],
		input:   &input,
		text:    strings[0],
		group:   strings,
		indices: indices,
	}
}

func RegexExecAll(re *regexp.Regexp, input string) []*RegexMatch {
	out := make([]*RegexMatch, 0)
	all := re.FindAllStringSubmatchIndex(input, -1)
	for _, result := range all {
		out = append(out, parseSubmatchIndex(result, input))
	}

	return out
}

const escapedChars = "&'<>\"\r"

func HtmlEscape(s string) string {
	w := &bytes.Buffer{}
	i := strings.IndexAny(s, escapedChars)
	for i != -1 {
		w.WriteString(s[:i])
		var esc string
		switch s[i] {
		case '&':
			esc = "&amp;"
		case '\'':
			// "&#39;" is shorter than "&apos;" and apos was not in HTML until HTML5.
			esc = "&#39;"
		case '<':
			esc = "&lt;"
		case '>':
			esc = "&gt;"
		case '"':
			// "&#34;" is shorter than "&quot;".
			esc = "&#34;"
		case '\r':
			esc = "&#13;"
		default:
			panic("unrecognized HtmlEscape character")
		}
		s = s[i+1:]
		w.WriteString(esc)
		i = strings.IndexAny(s, escapedChars)
	}
	w.WriteString(s)
	return w.String()
}

// HtmlVoidElements Void elements are those that can't have any contents.
var HtmlVoidElements = map[string]bool{}

var HtmlBooleanAttributes = map[string]bool{}

var voidElements = []string{
	"area", "base", "br", "col", "embed", "hr", "img", "input", "keygen", "link", "meta", "param", "source", "track", "wbr",
}

// https://html.spec.whatwg.org/#boolean-attribute
var booleanAtributes = []string{
	"allowfullscreen", "async", "autofocus", "autoplay", "checked", "controls", "default", "defer", "disabled",
	"formnovalidate", "ismap", "itemscope", "loop", "multiple", "muted", "nomodule", "novalidate", "open", "playsinline",
	"readonly", "required", "reversed", "selected", "truespeed",
}

var regPrefix = regexp.MustCompile(`^((?:x|data)[:\-_])`)

func NormalizeName(name string) string {
	return strings.ToLower(regPrefix.ReplaceAllString(strings.TrimSpace(name), ""))
}

// Returns the string representation of the element.
func startingTag(node *Node) {
	//element = jqLite(element).clone().empty()
	//var elemHtml = jqLite('<div></div>').append(element).html()
	//try{
	//  return element[0].nodeType == = NODE_TYPE_TEXT ? lowercase(elemHtml):
	//  elemHtml.
	//  match(/^(<[^>]+>)/)[1].
	//  replace(/^<([\w-]+)/, function(match, nodeName){return '<' + lowercase(nodeName)})
	//}
	//catch(e)
	//{
	//  return lowercase(elemHtml)
	//}
}

type ErrorFunc func(params ...interface{}) error

func ErrorTemplate(code string, template ...string) ErrorFunc {
	format := "[" + code + "] " + template[0]

	for i := 1; i < len(template); i++ {
		format = format + "\n    " + template[i]
	}

	return func(params ...interface{}) error {
		return fmt.Errorf(format, params...)
	}
}

func init() {
	for _, tag := range voidElements {
		HtmlVoidElements[tag] = true
	}
	for _, attr := range booleanAtributes {
		HtmlBooleanAttributes[attr] = true
	}
}
