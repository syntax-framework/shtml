package directives

import (
	"github.com/syntax-framework/shtml/sht"
)

var errorCompNested = sht.ErrorTemplate(
	"component:nested",
	"It is not allowed for a component to be defined inside another.", "Inner: %s", "Outer: %s",
)

// Component Responsável pela criação de componentes de forma declarativa
var Component = &sht.Directive{
	Name:       "component",
	Restrict:   sht.ELEMENT,
	Priority:   1000,
	Terminal:   true,
	Transclude: true,
	Compile: func(node *sht.Node, attrs *sht.Attributes, t *sht.Compiler) (methods *sht.DirectiveMethods, err error) {
		// It is not allowed for a component to be defined inside another
		t.Transverse(node, func(other *sht.Node) (stop bool) {
			if other == node || other.Type != sht.ElementNode {
				stop = false
				return
			}

			if other.Data == "component" {
				stop = true
				err = errorCompNested(other.DebugTag(), node.DebugTag())
			}

			return
		})

		if err != nil {
			return
		}

		// um componente só pode ter uma única tag style
		// um componente só pode ter uma única tag script
		// quando possui o parametro live, o componente não pode ter transclude
		// Quando um script existir, todos os eventos DOM/Javascript serão substituidos por addEventListener
		return
	},
}
