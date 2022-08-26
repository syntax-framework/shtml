package sht

import (
	"crypto/md5"
	"encoding/hex"
	"github.com/syntax-framework/shtml/cmn"
	"strings"
)

// Compiled structure representing a Compiled template
type Compiled struct {
	Assets      []*cmn.Asset // Reference to all resources that can be used by this compiled
	static      []string     // a list of literal strings
	dynamics    []Dynamic    // functions to get the dynamics part of this Compiled
	fingerprint string       // used to identify the static part of this Compiled
	root        bool
}

// Exec the rendering of the Compiled, applying the informed scope
func (c *Compiled) Exec(scope *Scope) *Rendered {
	out := &Rendered{
		Root:        c.root,
		Static:      &c.static, // never change
		Dynamics:    make([]interface{}, len(c.dynamics)),
		Fingerprint: c.Fingerprint(),
	}

	for i, dynamic := range c.dynamics {
		// nil, string, Compiled, Rendered
		result := dynamic.Exec(scope)
		if result == nil {
			out.Dynamics[i] = result
		} else if value, ok := result.(string); ok {
			out.Dynamics[i] = value
		} else if rendered, isRendered := result.(*Rendered); isRendered && rendered != nil {
			out.Dynamics[i] = rendered
			out.Assets = append(out.Assets, rendered.Assets...)
			rendered.Assets = nil
		} else if compiled, isCompiled := result.(*Compiled); isCompiled && compiled != nil {
			out.Dynamics[i] = compiled.Exec(scope)
		}
	}

	return out
}

// Fingerprint Get static fingerprint
func (c *Compiled) Fingerprint() string {
	if c.fingerprint == "" {
		h := md5.New()
		h.Write([]byte(strings.Join(c.static, "")))
		c.fingerprint = hex.EncodeToString(h.Sum(nil))
	}
	return c.fingerprint
}

//func (c *Compiled) Write(buffer *bytes.Buffer) {
//  for i := 0; i < len(c.Static); i++ {
//    if i == 0 {
//      buffer.WriteString(c.Static[i])
//    } else {
//      dynamic := c.Dynamics[i-1]
//
//      // nil, string, Compiled, Rendered
//      result := dynamic.Process()
//
//      if result != nil {
//        if value, ok := result.(string); ok {
//          buffer.WriteString(value)
//        } else if rendered, ok := result.(*Rendered); ok {
//          rendered.Write(buffer)
//        }
//      }
//      buffer.WriteString(c.Static[i])
//    }
//  }
//}
