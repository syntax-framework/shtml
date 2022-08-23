package jsc

import (
	"github.com/iancoleman/strcase"
	"github.com/syntax-framework/shtml/sht"
	"strings"
)

var errorCompJsRefInvalidName = sht.Err(
	"component:js:ref:name",
	"The reference name is invalid.", "Variable: %s", "Element: %s", "Component: %s",
)

var errorCompJsRefDuplicated = sht.Err(
	"component:js:ref:duplicated",
	"There are two elements with the same JS reference.", "First: %s", "Second: %s",
)

type NodeRef struct {
	VarName string
	Node    *sht.Node
	Attr    *sht.Attribute
}

// ParseReferences handles references made available to JS (<element ref="myJsVariable">)
func ParseReferences(node *sht.Node, t *sht.Compiler, elementClassIds *sht.IndexedMap) ([]*NodeRef, error) {
	// references to elements within the template
	refVarNodes := map[string]*sht.Node{}

	var references []*NodeRef

	var err error

	// Parse content
	t.Transverse(node, func(child *sht.Node) (stop bool) {
		stop = false
		if child == node || child.Type != sht.ElementNode {
			return
		}

		// @TODO: Quando child é um Component registrado, faz expr processamento adequado
		// isComponent := false

		if attr := child.Attributes.GetAttribute("ref"); attr != nil {
			// is a reference that can be used in JS
			if varName := strcase.ToLowerCamel(attr.Value); varName != "" {

				// ref name is valid?
				if _, isInvalid := InvalidParamsAndRefs[varName]; isInvalid || strings.HasPrefix(varName, "_$") {
					err = errorCompJsRefInvalidName(varName, node.DebugTag(), child.DebugTag())
					stop = true
					return
				}

				// ref name is duplicated?
				if firstNode, isDuplicated := refVarNodes[varName]; isDuplicated {
					err = errorCompJsRefDuplicated(firstNode.DebugTag(), child.DebugTag())
					stop = true
					return
				}

				refVarNodes[varName] = child

				references = append(references, &NodeRef{
					VarName: varName,
					Node:    child,
					Attr:    attr,
				})
			}
		}

		return
	})

	return references, err
}
