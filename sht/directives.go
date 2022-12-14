package sht

import (
	"github.com/syntax-framework/shtml/cmn"
	"log"
	"sort"
)

type DirectivesByPriority []*Directive

func (l DirectivesByPriority) Len() int { return len(l) }
func (l DirectivesByPriority) Less(i, j int) bool {
	a, b := l[i], l[j]
	if a.Priority == b.Priority && (a.Terminal || b.Terminal) {
		// terminal always at the end
		if a.Terminal {
			return true
		}
		return false
	}
	return a.Priority > b.Priority
}
func (l DirectivesByPriority) Swap(i, j int) { l[i], l[j] = l[j], l[i] }

// Directives group the list of registered directives
type Directives struct {
	parent *Directives
	list   []*Directive
	byName map[string][]*Directive
}

// Contains verifica se essa directiva já está registrada nessa lista
func (d *Directives) Contains(directive *Directive) bool {
	for _, o := range d.list {
		if o == directive {
			return true
		}
	}
	if d.parent != nil {
		return d.parent.Contains(directive)
	}
	return false
}

// Add a new directive.
func (d *Directives) Add(directive *Directive) {
	if !d.Contains(directive) {
		directive.Normalize()
		d.list = append(d.list, directive)
		_, exists := d.byName[directive.Name]
		if !exists {
			if d.byName == nil {
				d.byName = map[string][]*Directive{}
			}
			d.byName[directive.Name] = []*Directive{}
		}
		d.byName[directive.Name] = append(d.byName[directive.Name], directive)
	}
}

// NewChild cria uma nova lista, que mantém referencia para a lista atual
func (d *Directives) NewChild() *Directives {
	return &Directives{parent: d}
}

var errorAttrInterpolation = cmn.Err(
	"attr.interpolation",
	"Error while interpolating an attribute.", "AttrName: %s", "AttrValue: %s", "Element: %s", "Cause: %s",
)

// collect Looks for directives on the given node and adds them to the directive collection which is sorted.
func (d *Directives) collect(node *Node, attrs *Attributes, ignore *Directive) ([]*Directive, error) {

	ddMap := map[*Directive]bool{}

	// use the node name: <directive>
	d.collectInto(ddMap, NormalizeName(node.Data), ELEMENT, ignore)

	// iterate over the Map
	for _, attr := range attrs.Map {
		err := addAttrInterpolateDirective(ddMap, attr.Value, attr.Name)
		if err != nil {
			return nil, errorAttrInterpolation(attr.Name, attr.Value, node.DebugTag(), err.Error())
		}
		d.collectInto(ddMap, attr.Name, ATTRIBUTE, ignore)
	}
	//addTextInterpolateDirective(ddMap, node.Data)

	var directives DirectivesByPriority
	for dd, _ := range ddMap {
		directives = append(directives, dd)
	}

	sort.Sort(directives)

	return directives, nil
}

func (d *Directives) collectInto(ddMap map[*Directive]bool, name string, location DirectiveRestrict, ignore *Directive) {
	if directives, exist := d.byName[name]; exist {
		for _, directive := range directives {
			if directive.Restrict&location != 0 && directive != ignore {
				ddMap[directive] = true
			}
		}
	}
	if d.parent != nil {
		d.parent.collectInto(ddMap, name, location, ignore)
	}
}

func addAttrInterpolateDirective(directives map[*Directive]bool, value string, name string) error {
	interpolateFn, err := Interpolate(value)
	if err != nil {
		return err
	}

	// no interpolation found -> ignore
	if interpolateFn == nil {
		return nil
	}

	directive := Directive{
		Name:     "AttrInterpolateDirective",
		Priority: 300,
		Process: func(s *Scope, attr *Attributes, transclude TranscludeFunc) *Rendered {

			// If the attribute has changed since last Interpolate()
			newValue := attr.Get(name)
			if newValue != value {
				// we need to interpolate again since the attribute value has been updated
				// (e.g. by another directive's compile function)
				// ensure unset/empty values make interpolateFn falsy
				if newValue != "" {
					exp, err := Interpolate(newValue)
					if err != nil {
						// @TODO: Log.Warning
						log.Print(err)
						interpolateFn = nil
					} else {
						interpolateFn = exp
					}
				} else {
					interpolateFn = nil
				}
				value = newValue
			}

			// if attribute was updated so that there is no interpolation going on we don't want to
			// register any observers
			if interpolateFn != nil {
				// initialize attr object so that it's ready in case we need the value for isolate
				// scope initialization, otherwise the value would not be available from isolate
				// directive's linking fn during linking phase
				attr.Set(name, interpolateFn.Exec(s).String())
			}

			return nil
		},
	}
	directives[&directive] = true
	return nil
}
