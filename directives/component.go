package directives

import (
	"github.com/syntax-framework/shtml/cmn"
	"github.com/syntax-framework/shtml/jsc"
	"github.com/syntax-framework/shtml/sht"
)

var errorCompNested = cmn.Err(
	"component:nested",
	"It is not allowed for a component to be defined inside another.", "Outer: %s", "Inner: %s",
)

var errorCompStyleSingle = cmn.Err(
	"component:style:single",
	"A component can only have a single style element.", "First: %s", "Second: %s",
)

var errorCompStyleLocation = cmn.Err(
	"component:style:location",
	"Style element must be an immediate child of the component.", "Component: %s", "Style: %s",
)

var errorCompScriptSingle = cmn.Err(
	"component:script:single",
	"A component can only have a single script element.", "First: %s", "Second: %s",
)

var errorCompScriptLocation = cmn.Err(
	"component:script:location",
	"Script element must be an immediate child of the component.", "Component: %s", "Script: %s",
)

// Component Responsible for creating components declaratively
//
// @TODO: Javascript directives?
var Component = &sht.Directive{
	Name:       "component",
	Restrict:   sht.ELEMENT,
	Priority:   1000,
	Terminal:   true,
	Transclude: true,
	Compile: func(node *sht.Node, attrs *sht.Attributes, t *sht.Compiler) (methods *sht.DirectiveMethods, einlineJsErrr error) {

		// @TODO: Parse include?

		var style *sht.Node
		var script *sht.Node

		node.Transverse(func(child *sht.Node) (stop bool) {
			stop = false
			if child == node || child.Type != sht.ElementNode {
				return
			}

			switch child.Data {
			case "component":
				// It is not allowed for a component to be defined inside another
				einlineJsErrr = errorCompNested(node.DebugTag(), child.DebugTag())

			case "style":
				if style != nil {
					// a component can only have a single style tag
					einlineJsErrr = errorCompStyleSingle(style.DebugTag(), child.DebugTag())

				} else if child.Parent != node {
					// when it has style, it must be an immediate child of the component
					einlineJsErrr = errorCompStyleLocation(node.DebugTag(), child.DebugTag())

				} else {
					style = child
				}

			case "script":
				if script != nil {
					// a component can only have a single script tag
					einlineJsErrr = errorCompScriptSingle(script.DebugTag(), child.DebugTag())

				} else if child.Parent != node {
					// when it has script, it must be an immediate child of the component
					einlineJsErrr = errorCompScriptLocation(node.DebugTag(), child.DebugTag())

				} else {
					script = child
				}
			}

			if einlineJsErrr != nil {
				stop = true
				return
			}

			return
		})

		if einlineJsErrr != nil {
			return
		}

		// parse attributes
		//goAttrs := map[string]*sht.Attribute{}
		//
		//for name, attr := range attrs.Map {
		//	if strings.HasPrefix(name, "param-") {
		//		goAttrs[strings.Replace(name, "param-", "", 1)] = attr
		//		delete(attrs.Map, name)
		//	}
		//}

		inlineJs, inlineJsErr := jsc.Compile(node, script, t.Sequence)
		if inlineJsErr != nil {
			return nil, inlineJsErr
		}
		if inlineJs != nil {
			t.RegisterAssetJsContent(inlineJs.Content)
		}

		// @TODO: Registrar o componente no contexto de compilação
		//t.RegisterComponent(&sht.Component{
		//
		//})

		// quando possui expr parametro live, expr componente não pode ter transclude
		// Quando um script existir, todos os eventos DOM/Javascript serão substituidos por addEventListener
		return
	},
}
