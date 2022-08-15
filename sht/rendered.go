package sht

import "bytes"

// Rendered structure of a Compiled
type Rendered struct {
	Static      *[]string     `json:"s"` // a list of literal strings
	Dynamics    []interface{} `json:"d"` // nil, string, Rendered
	Fingerprint string        `json:"f"`
	Root        bool          `json:"r"`
}

// Write the output to the given buffer
func (r *Rendered) Write(buffer *bytes.Buffer) {
	static := *r.Static
	for i := 0; i < len(static); i++ {
		if i == 0 {
			buffer.WriteString(static[i])
		} else {
			// nil, string, Rendered
			dynamic := r.Dynamics[i-1]
			if dynamic != nil {
				if value, ok := dynamic.(string); ok {
					buffer.WriteString(value)
				} else if rendered, ok := dynamic.(*Rendered); ok && rendered != nil {
					rendered.Write(buffer)
				}
			}
			buffer.WriteString(static[i])
		}
	}
}

// String convert the result to string
func (r *Rendered) String() string {
	buf := &bytes.Buffer{}
	r.Write(buf)
	return buf.String()
}
