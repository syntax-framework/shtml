package sht

import (
	"bytes"
	"crypto/md5"
	"crypto/sha512"
	"encoding/base64"
	"encoding/hex"
	"github.com/cespare/xxhash"
	"regexp"
	"strconv"
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

// Sequence identifier generator utility qeu ensures that all executions have the same result
type Sequence struct {
	Salt string
	seq  int
}

const numbers = "0123456789"

func (s *Sequence) NextHash() string {
	s.seq++
	hash := HashXXH64([]byte(s.Salt + strconv.Itoa(s.seq)))
	if strings.Contains(numbers, hash[:1]) {
		// add _ if hash starts with number
		return "_" + hash
	}
	return hash
}

func (s *Sequence) NextInt() int {
	s.seq++
	return s.seq
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

// HashMD5 computing the MD5 checksum of strings
func HashMD5(text string) string {
	h := md5.New()
	h.Write([]byte(text))
	return hex.EncodeToString(h.Sum(nil))
}

// HashSha512Base64 return a base64-encoded cryptographic hash of a resource using SHA-512 hash algorithm
func HashSha512Base64(content []byte) string {
	h := sha512.New()
	h.Write(content)
	return base64.StdEncoding.EncodeToString(h.Sum(nil))
}

// HashXXH64Hex computing the XXH64 checksum of strings
func HashXXH64Hex(text string) string {
	h := xxhash.New()
	h.Write([]byte(text))
	return hex.EncodeToString(h.Sum(nil))
}

// HashXXH64 Make a fingerprint. Compute a non-cryptographic XXH64 checksum of strings
//
// Encoded using Base64 URLSafe
func HashXXH64(content []byte) string {
	h := xxhash.New()
	h.Write(content)
	return base64.RawURLEncoding.EncodeToString(h.Sum(nil))
}

func CreateBoolMap(arr []string) map[string]bool {
	out := map[string]bool{}
	for _, s := range arr {
		out[s] = true
	}
	return out
}

func init() {
	for _, tag := range voidElements {
		HtmlVoidElements[tag] = true
	}
	for _, attr := range booleanAtributes {
		HtmlBooleanAttributes[attr] = true
	}
}
