package directives

import (
	"github.com/syntax-framework/shtml/sht"
	"log"
	"strings"
)

func createIfDirective(attrs *sht.Attributes, getValue func(attrs *sht.Attributes) string) *sht.DirectiveMethods {
	cond := getValue(attrs)
	if strings.TrimSpace(cond) == "" {
		log.Fatal("Atributo cond não encontrado para elemento if")
	}

	// @TODO: https://github.com/antonmedv/expr/blob/master/docs/Visitor-and-Patch.md
	expression, err := sht.ParseExpression(cond)
	if err != nil {
		// @TODO: Remover todos os log.Fatal e simplesmente fazer log de warning
		log.Fatal(err)
	}

	return &sht.DirectiveMethods{
		Process: func(scope *sht.Scope, attrs *sht.Attributes, transclude sht.TranscludeFunc) *sht.Rendered {
			// If the attribute has changed since last Interpolate()
			newCond := getValue(attrs)

			if newCond != cond {
				// we need to interpolate again since the attribute value has been updated
				// (e.g. by another directive's compile function)
				// ensure unset/empty values make expression falsy
				if newCond != "" {
					newExpression, err := sht.ParseExpression(newCond)
					if err != nil {
						// @TODO: Log.Warning
						log.Print(err)
						expression = nil
					} else {
						expression = newExpression
					}
				} else {
					expression = nil
				}
				cond = newCond
			}

			if expression.EvalBool(scope) {
				return transclude("", nil)
			} else {
				return nil
			}
		},
	}
}

func attrCOND(attrs *sht.Attributes) string {
	return attrs.Get("cond")
}

func attrIF(attrs *sht.Attributes) string {
	return attrs.Get("if")
}

// IFElement `<if cond="true"/>`
// @TODO `<if cond="true"></if> <else-if cond="true"></else-if> <else></else>`
var IFElement = &sht.Directive{
	Name:       "if",
	Restrict:   sht.ELEMENT,
	Priority:   600,
	Terminal:   true,
	Transclude: true,
	Compile: func(node *sht.Node, attrs *sht.Attributes, t *sht.Compiler) (*sht.DirectiveMethods, error) {
		return createIfDirective(attrs, attrCOND), nil

		//conditions := &_CommandConditions{BreakOnFirstValid: true}
		//
		//nextSibling := node.NextSibling
		//
		//// if
		//conditions.Add(CompileCondAttr(node, t), t.Transverse(t.ExtractRoot(node)))
		//
		//if nextSibling != nil && nextSibling.Type == html.ElementNode {
		//	for nextNode := nextSibling; nextNode != nil; {
		//		breakLoop := false
		//		tag := strings.TrimSpace(nextNode.Data)
		//		switch tag {
		//		case "else-if":
		//			aux := nextNode.NextSibling
		//			conditions.Add(CompileCondAttr(nextNode, t), t.Transverse(t.ExtractRoot(nextNode)))
		//			nextNode = aux
		//			continue
		//
		//		case "else":
		//			breakLoop = true
		//			conditions.Add(trueBoolFunc, t.Transverse(t.ExtractRoot(nextNode)))
		//			break
		//
		//		default:
		//			breakLoop = true
		//			break
		//		}
		//
		//		if breakLoop {
		//			break
		//		}
		//	}
		//}
		//
		//t.replaceNodeByDynamic(node, conditions)
	},
}

// IFAttribute `<element if="true"/>`
var IFAttribute = &sht.Directive{
	Name:       "if",
	Restrict:   sht.ATTRIBUTE,
	Priority:   599,
	Terminal:   true,
	Transclude: "element",
	Compile: func(node *sht.Node, attrs *sht.Attributes, t *sht.Compiler) (*sht.DirectiveMethods, error) {
		return createIfDirective(attrs, attrIF), nil
	},
}
