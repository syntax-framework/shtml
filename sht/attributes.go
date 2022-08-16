package sht

import (
	"bytes"
	"strings"
)

// Attribute representa um atributo html
type Attribute struct {
	Name      string
	Value     string
	Namespace string

	// nodeName string
}

// Attributes abstração dos atributos de um Node html
type Attributes struct {
	Attrs          map[string]*Attribute
	DirectiveMap   map[string]string // Faz mapeamento de nomes de directivas. Esses são removidos
	staticContent  string
	hasDynamicAttr bool
	observers      map[string][]func(value string)
}

func NewAttribute(name string, value string, namespace string) *Attribute {
	return &Attribute{
		Name:      NormalizeName(name),
		Value:     value,
		Namespace: namespace,
	}
}

func AttributesFromHtmlNode(node *Node) *Attributes {
	attrs := &Attributes{
		DirectiveMap: map[string]string{},
		Attrs:        map[string]*Attribute{},
	}

	for _, nodeAttr := range node.Attr {
		attr := NewAttribute(nodeAttr.Name, nodeAttr.Value, nodeAttr.Namespace)
		if attr.Name == "" {
			continue
		}
		attrs.DirectiveMap[attr.Name] = nodeAttr.Name
		attrs.Attrs[attr.Name] = attr
	}

	return attrs
}

// Clone Uso interno, faz uma cópia dos atributos para ser usado em tempo de execução
func (a *Attributes) Clone() *Attributes {
	attributes := map[string]*Attribute{}
	if a.Attrs != nil {
		for name, a2 := range a.Attrs {
			attributes[name] = &Attribute{
				Name:  name,
				Value: a2.Value,
			}
		}
	}

	return &Attributes{
		Attrs: attributes,
	}
}

func (a *Attributes) Set(key string, value string) {
	attribute, exists := a.Attrs[key]
	if exists {
		attribute.Value = value
	} else {
		attr := NewAttribute(key, value, "")
		if attr.Name != "" {
			if a.DirectiveMap == nil {
				a.DirectiveMap = map[string]string{}
			}
			a.DirectiveMap[attr.Name] = key
			a.Attrs[attr.Name] = attr
		}
	}

	// fire observers
	listeners, exists := a.observers[key]
	if exists {
		for _, fn := range listeners {
			fn(value)
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

func (a *Attributes) Get(name string) (value string) {
	attribute, exists := a.Attrs[name]
	if exists {
		value = attribute.Value
	}
	return
}

func (a *Attributes) Exists(name string) (exists bool) {
	_, exists = a.Attrs[name]
	return
}

// GetStaticContent obtém o código html com todos os atributos estáticos da lista
func (a *Attributes) GetStaticContent() string {
	return a.staticContent
}

func (a *Attributes) GetDynamicAttrs() []Attribute {
	//if !a.hasDynamicAttr {
	//	return ""
	//}
	return nil
}

func (a *Attributes) Render() *Rendered {
	if a.Attrs == nil {
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

	for _, a := range a.Attrs {
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
