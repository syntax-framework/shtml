package sht

import (
	"bytes"
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
		a.Set("class", current+buf.String())
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

func (a *Attributes) Render() *Rendered {
	if a.Map == nil {
		return nil
	}

	//return Rendered{
	//  Static: *[]string{
	//    ` class="`,
	//    `"  atributo-1="`,
	//    `"  atributo-2="`,
	//    `"`,
	//  }
	//  Dynamics: []interface{}{
	//    "dentro class-false class-2 class-1",
	//    "att1-valor",
	//    "scope-valor"
	//  }
	//}

	var static []string
	var dynamics []interface{}

	staticCurr := &bytes.Buffer{}
	prevHasValue := false

	for _, a := range a.Map {
		if prevHasValue {
			staticCurr.WriteRune('"')
		}
		staticCurr.WriteByte(' ')

		if a.Namespace != "" {
			staticCurr.WriteString(a.Namespace)
			staticCurr.WriteByte(':')
		}

		staticCurr.WriteString(a.Name)

		if a.Value != "" {
			prevHasValue = true
			staticCurr.WriteString(`="`)
			static = append(static, staticCurr.String())
			// reset buffer (next Static)
			staticCurr = &bytes.Buffer{}

			dynamics = append(dynamics, HtmlEscape(a.Value))
		} else {
			prevHasValue = false
		}
	}

	if prevHasValue {
		staticCurr.WriteString(`"`)
	}

	static = append(static, staticCurr.String())

	// @TODO: Cache processing, if data hasn't changed since template, keep it
	return &Rendered{
		Static:   &static,
		Dynamics: dynamics,
	}
}

func (a *Attributes) Remove(attr *Attribute) {
	delete(a.Map, attr.Normalized)
}
