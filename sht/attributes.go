package sht

import (
	"bytes"
	"sort"
	"strings"
)

// Attribute representa um atributo html
type Attribute struct {
	Name       string
	Value      string
	Namespace  string
	Normalized string // name normalized
}

// Attributes abstração dos atributos de um Node html
type Attributes struct {
	Map map[string]*Attribute
}

func NewAttribute(name string, value string, namespace string) *Attribute {
	return &Attribute{
		Name:       name,
		Value:      value,
		Namespace:  namespace,
		Normalized: NormalizeName(name),
	}
}

// Clone Uso interno, faz uma cópia dos atributos para ser usado em tempo de execução
func (a *Attributes) Clone() *Attributes {
	attributes := map[string]*Attribute{}
	if a.Map != nil {
		for nameN, attr := range a.Map {
			attributes[nameN] = &Attribute{
				Name:       attr.Name,
				Value:      attr.Value,
				Namespace:  attr.Namespace,
				Normalized: attr.Normalized,
			}
		}
	}

	return &Attributes{Map: attributes}
}

func (a *Attributes) GetAttribute(name string) *Attribute {
	return a.Map[name]
}

func (a *Attributes) Get(name string) (value string) {
	attribute, exists := a.Map[name]
	if exists {
		value = attribute.Value
	}
	return
}

func (a *Attributes) GetOrDefault(name string, dfault string) (value string) {
	attribute, exists := a.Map[name]
	if exists {
		value = attribute.Value
	} else {
		value = dfault
	}
	return
}

func (a *Attributes) Set(key string, value string) {
	attribute, exists := a.Map[NormalizeName(key)]
	if exists {
		attribute.Value = value
	} else {
		attr := NewAttribute(key, value, "")
		if attr.Normalized != "" {
			a.Map[attr.Normalized] = attr
		}
	}
}

// HasClass Determine whether the element is assigned the given class.
func (a *Attributes) HasClass(value string) bool {
	current := a.Get("class")
	if current == "" {
		return false
	}
	return strings.Contains(current, value)
}

// AddClass Adds the specified class(es) to element
func (a *Attributes) AddClass(value string) {
	current := a.Get("class")
	changed := false
	buf := &bytes.Buffer{}
	for _, c := range strings.Split(value, " ") {
		if !strings.Contains(current, c) {
			buf.WriteRune(' ')
			buf.WriteString(c)
			changed = true
		}
	}
	if changed {
		a.Set("class", strings.TrimSpace(current+buf.String()))
	}
}

// RemoveClass Remove a single class or multiple classes from element
func (a *Attributes) RemoveClass(value string) {
	trimValue := strings.TrimSpace(value)
	if trimValue == "" {
		return
	}
	oldValue := a.Get("class")
	if oldValue == "" {
		return
	}

	newValue := oldValue
	for _, c := range strings.Split(trimValue, " ") {
		newValue = strings.TrimSpace(strings.ReplaceAll(newValue, c, ""))
	}
	if newValue != oldValue {
		a.Set("class", newValue)
	}
}

// Render the attributes sorted by name. Boolean attributes (no value) are always rendered at the end
//  &Rendered{
//    Static: *[]string{
//      ` attribute-one="`,
//      `"  attribute-two="`,
//      `" class="`,
//      `" boolean-attr-one boolean-attr-2`,
//    }
//    Dynamics: []interface{}{
//      "value-1",
//      "value-2"
//      "class-1 class-",
//    }
//}
func (a *Attributes) Render() *Rendered {
	if a.Map == nil {
		return nil
	}

	var static []string
	var dynamics []interface{}

	// Attributes sorted
	//  [
	//    ["attr-one", "value"],
	//    ["attr-two", "value"],
	//    ["attr-bool"],
	//  ]
	var sortedAttributes [][]string
	for _, a := range a.Map {
		attrName := a.Name
		if a.Namespace != "" {
			attrName = a.Namespace + ":" + attrName
		}

		if HtmlBooleanAttributes[attrName] == true {
			// https://html.spec.whatwg.org/#boolean-attribute
			if a.Value != "false" {
				sortedAttributes = append(sortedAttributes, []string{attrName})
			}
		} else {
			if a.Value != "" {
				// attr-name, value
				sortedAttributes = append(sortedAttributes, []string{attrName, HtmlEscape(a.Value)})
			} else {
				sortedAttributes = append(sortedAttributes, []string{attrName})
			}
		}
	}

	sort.Slice(sortedAttributes, func(i, j int) bool {
		a := sortedAttributes[i]
		b := sortedAttributes[j]
		// boolean attributes at the end
		if len(b) < len(a) {
			// a != boolean
			// b == boolean
			// a first
			return true
		}
		if len(a) < len(b) {
			// a == boolean
			// b != boolean
			// b first
			return false
		}

		return strings.Compare(a[0], b[0]) <= 0
	})

	prevHasValue := false
	staticCurr := &bytes.Buffer{}
	for _, part := range sortedAttributes {
		if prevHasValue {
			staticCurr.WriteRune('"')
		}
		staticCurr.WriteByte(' ')
		if len(part) == 2 {
			// attr-name="value"
			prevHasValue = true
			staticCurr.WriteString(part[0] + `="`)
			static = append(static, staticCurr.String())
			// reset buffer (next Static)
			staticCurr = &bytes.Buffer{}
			dynamics = append(dynamics, part[1])
		} else {
			// attr-name
			prevHasValue = false
			staticCurr.WriteString(part[0])
		}
	}

	if prevHasValue {
		staticCurr.WriteString(`"`)
	}
	static = append(static, staticCurr.String())

	// @TODO: Cache processing, if Context hasn't changed since template, keep it
	return &Rendered{
		Static:   &static,
		Dynamics: dynamics,
	}
}

func (a *Attributes) Remove(attr *Attribute) {
	if attr != nil {
		delete(a.Map, attr.Normalized)
	}
}
