package jsc

import (
	"github.com/syntax-framework/shtml/sht"
	"github.com/tdewolff/parse/v2"
	"github.com/tdewolff/parse/v2/js"
	"strconv"
	"strings"
)

// ExpressionsParser
//

type ExpressionsParser struct {
	Node               *sht.Node
	Compiler           *sht.Compiler
	Sequence           *sht.Sequence
	ContextAst         *js.AST
	ContextAstScope    *js.Scope
	ContextVariables   *sht.IndexedMap
	Expressions        *sht.IndexedMap
	BindedAttributes   *sht.IndexedMap
	ElementIdentifiers *sht.IndexedMap
	Watchers           *sht.IndexedMap
	WatchersByVar      map[int][]int
	NodeIdentifierFunc func(node *sht.Node) string
}

var errorJsInterpolationSideEffect = sht.Err(
	"js:interpolation:sideeffect",
	"Expressions with Side Effect in text interpolation block or attributes are not allowed.",
	"Side Effect: (%s)",
	"Expression: (%s)",
	"Element: %s",
	"Component: %s",
)

// Parse Faz o processamento e validação de todas as expressões existente no código HTML do template
//
// Check for expressions (${value} or #{value})
//
// Text escaping
// <element>${escape safe}</element>
// <element>#{escape unsafe}</element>
//
// Attributes escaping
// <element attr="text ${escape safe}">
// <element attr="text #{escape unsafe}">
// <element attr="${return value}">
// <element attr="#{escape unsafe}">
//
// Bindings
// <input value="${value}"></input>
func (p *ExpressionsParser) Parse() error {
	node := p.Node
	t := p.Compiler

	var err error
	t.Transverse(node, func(child *sht.Node) (stop bool) {
		stop = false
		if child == node {
			return
		}

		if child.Type == sht.TextNode {
			// Check for innerText expressions (${value} or #{value})
			//
			// <element>${escape safe}</element>
			// <element>#{escape unsafe}</element>
			innerTextErr := p.parseTextNode(child)
			if innerTextErr != nil {
				err = innerTextErr
				stop = true
				return
			}

		} else if child.Type == sht.ElementNode {
			// busca interpolação nos atributos
			for _, attr := range child.Attributes.Map {
				//if strings.HasPrefix(attrNameNormalized, "on") {
				//
				//} else { }
				innerTextErr := p.parseAttribute(child, attr)
				if innerTextErr != nil {
					err = innerTextErr
					stop = true
					return
				}
			}
		}

		// para permitir expr transclude e não perder expr escopo de execução do javascript:
		// 1. Interpolação em texto deve ser feito criando um node html temporário e removendo-expr em tempo de execução.
		// 2. Interpolação em atributos, deve-se adicionar um identificador no atributo e, em tempo de execução, verificar
		// se houve mudança do template do atributo.
		// Obs. Se algum marcador tiver sido removido, não registrar expr $watch
		return
	})

	return err
}

func (p *ExpressionsParser) parseAttribute(child *sht.Node, attr *sht.Attribute) error {
	sequence := p.Sequence
	contextAst := p.ContextAst
	contextAstScope := p.ContextAstScope
	contextVariables := p.ContextVariables
	expressions := p.Expressions
	attributes := p.BindedAttributes
	identifiers := p.ElementIdentifiers
	watchers := p.Watchers
	watchersByVar := p.WatchersByVar
	getNodeIdentifier := p.NodeIdentifierFunc

	// Check for expressions (${value} or #{value})
	//
	// <element attr="text ${escape safe}">
	// <element attr="text #{escape unsafe}">
	// <element attr="${value}"></element>
	// <element attr="${return value}">
	// <element attr="#{escape unsafe}">
	attrValue, interpolations, interErr := InterpolateJs(attr.Value, sequence)
	if interErr != nil {
		return interErr
	}
	if interpolations == nil {
		// content has no js expressions (${value} or #{value})
		return nil
	}

	var err error

	attrIdx := strconv.Itoa(attributes.Add(attr.Normalized))
	elementIdx := strconv.Itoa(identifiers.Add(getNodeIdentifier(child)))

	// ['content static 1', expressionIdx, 'content static 2']
	attrValueParts := strings.ReplaceAll(attrValue, "'", "\\'")

	for interpolationId, interpolation := range interpolations {

		interpolationJs := interpolation.Expression
		interpolationJsAst, interpolationJsAstErr := js.Parse(parse.NewInputString(interpolationJs), js.Options{})
		if interpolationJsAstErr != nil {
			err = interpolationJsAstErr // @TODO: Custom error or Warning
			break
		}

		// contextVariables
		interpolationJsAstScope := interpolationJsAst.BlockStmt.Scope

		// resolve references to global scope (component <script> source code and js-param-*)
		undeclaredBackup := contextAstScope.Undeclared
		interpolationJsAstScope.Parent = contextAstScope
		interpolationJsAstScope.HoistUndeclared()
		contextAstScope.Undeclared = undeclaredBackup

		if hasSideEffect, sideEffectJs := HasSideEffect(interpolationJsAst, contextAst); hasSideEffect {
			err = errorJsInterpolationSideEffect(sideEffectJs, interpolationJsAst.JS(), child.DebugTag(), p.Node.DebugTag())
		}

		interpolationJs = interpolationJsAst.JS()
		if strings.HasSuffix(interpolationJs, "; ") {
			interpolationJs = interpolationJs[:len(interpolationJs)-2]
		}

		if interpolation.IsFullContent {
			if interpolation.IsSafeSignal {
				// <element attr="${value}"></element>
				// <element attr="${return value}">
				if p.Compiler.IsComponente(child) {
					interpolationJs = "(" + interpolationJs + ")"
				} else {
					interpolationJs = "_$escape(" + interpolationJs + ")"
				}
			} else {
				// <element attr="#{escape unsafe}">
				interpolationJs = "(" + interpolationJs + ")"
			}
		} else {
			if interpolation.IsSafeSignal {
				// <element attr="text ${escape safe}">
				interpolationJs = "_$escape(" + interpolationJs + ")"
			} else {
				// <element attr="text #{escape unsafe}">
				interpolationJs = "(" + interpolationJs + ")"
			}
		}

		// identical expressions are reused throughout the code
		expressionIdx := strconv.Itoa(expressions.Add("() => { return " + interpolationJs + "; }"))

		// splitting text, to check for modifications on client side
		attrValueParts = strings.Replace(attrValueParts, interpolationId, "', "+expressionIdx+", '", 1)

		watcherIndex := watchers.Add(
			// Criate a watcher that bind element attribute to expression result
			// _$bind_prop(elementIndex, attributeIndex, expressionIdx )
			"_$bind_prop(" + elementIdx + ", " + attrIdx + ", " + expressionIdx + ") /* " + interpolation.Debug() + " */",
		)

		js.Walk(VisitorEnterFunc(func(node js.INode) bool {
			// só estamos interessados em descobrir as expressoes que acessam variáveis de contexto
			//
			// Uma variável do context só é classificada como observada quando alguma expressão faz uso dessa variável
			// para realizar alguma lógica.
			if jsVar, isVar := node.(*js.Var); isVar {
				if isDeclared, jsVarGlobal := IsDeclaredOnScope(jsVar, contextAstScope); isDeclared {
					// associate watcher to variable
					varIndex := contextVariables.Add(jsVarGlobal)
					watchersByVar[varIndex] = append(watchersByVar[varIndex], watcherIndex)
				}
			}
			return true
		}), interpolationJsAst)
	}

	// Criate a watcher that bind element attribute to expression result
	watchers.Add(
		// _$bind_prop_tpl(elementIndex, attributeIndex, ['content static 1', expressionIdx, 'content static 2'] )
		"_$bind_prop_tpl(" + elementIdx + ", " + attrIdx + ", ['" + attrValueParts + "'])",
	)

	attr.Value = attrValue

	return err
}

func (p *ExpressionsParser) parseTextNode(child *sht.Node) error {
	node := p.Node
	sequence := p.Sequence
	contextAst := p.ContextAst
	contextAstScope := p.ContextAstScope
	contextVariables := p.ContextVariables
	expressions := p.Expressions
	identifiers := p.ElementIdentifiers
	watchers := p.Watchers
	watchersByVar := p.WatchersByVar

	// Check for innerText expressions (${value} or #{value})
	//
	// <element>${escape safe}</element>
	// <element>#{escape unsafe}</element>
	innerText, interpolations, textInterErr := InterpolateJs(child.Data, sequence)
	if textInterErr != nil {
		return textInterErr
	}
	if interpolations == nil {
		// content has no js expressions (${value} or #{value})
		return nil
	}

	var err error
	for id, interpolation := range interpolations {

		// replace biding location by <embed hidden class="_xxx">
		innerText = strings.Replace(innerText, id, `<embed hidden class="`+id+`">`, 1)

		// eventJsCode = "(e) => { " + eventJsCode + " }"
		interpolationJs := interpolation.Expression
		interpolationJsAst, interpolationJsAstErr := js.Parse(parse.NewInputString(interpolationJs), js.Options{})
		if interpolationJsAstErr != nil {
			err = interpolationJsAstErr // @TODO: Custom error or Warning
			break
		}

		// contextVariables
		interpolationJsAstScope := interpolationJsAst.BlockStmt.Scope

		// resolve references to global scope (component <script> source code and js-param-*)
		undeclaredBackup := contextAstScope.Undeclared
		interpolationJsAstScope.Parent = contextAstScope
		interpolationJsAstScope.HoistUndeclared()
		contextAstScope.Undeclared = undeclaredBackup

		if hasSideEffect, sideEffectJs := HasSideEffect(interpolationJsAst, contextAst); hasSideEffect {
			err = errorJsInterpolationSideEffect(sideEffectJs, interpolationJsAst.JS(), child.DebugTag(), node.DebugTag())
		}

		// AddDispatcers(interpolationJsAst, contextAstScope, contextVariables, nil)

		interpolationJs = interpolationJsAst.JS()
		if strings.HasSuffix(interpolationJs, "; ") {
			interpolationJs = interpolationJs[:len(interpolationJs)-2]
		}
		if interpolation.IsSafeSignal {
			// <element>${escape safe}</element>
			interpolationJs = "_$escape(" + interpolationJs + ")"
		} else {
			// <element>#{escape unsafe}</element>
			interpolationJs = "(" + interpolationJs + ")"
		}

		expressionId := expressions.Add(
			// identical expressions are reused throughout the code
			"() => { return " + interpolationJs + "; }",
		)

		elementIdx := strconv.Itoa(identifiers.Add(id))
		watcherIndex := watchers.Add(
			// Criate a Watcher that bind element content to Expression result
			"_$bind(" + elementIdx + ", " + strconv.Itoa(expressionId) + ") /* " + interpolation.Debug() + " */",
		)

		js.Walk(VisitorEnterFunc(func(node js.INode) bool {
			// só estamos interessados em descobrir as expressoes que acessam variáveis de contexto
			//
			// Uma variável do context só é classificada como observada quando alguma expressão faz uso dessa variável
			// para realizar alguma lógica.
			if jsVar, isVar := node.(*js.Var); isVar {
				if isDeclared, jsVarGlobal := IsDeclaredOnScope(jsVar, contextAstScope); isDeclared {
					// associate watcher to variable
					varIndex := contextVariables.Add(jsVarGlobal)
					watchersByVar[varIndex] = append(watchersByVar[varIndex], watcherIndex)
				}
			}
			return true
		}), interpolationJsAst)

		//    - Quando um sideeffect acontece durante a renderização, os watchers só serão informados no proximo tick.
		//      No exemplo abaixo, `count++` (disparado durante expr render), apesar de modificar a variável, não dispara os
		//      eventos imediatamente.
		//      `<component name="test">
		//        <span>${ (count %2 == 0 && count++), name }</span>
		//        <script>
		//          let count = 0;
		//          let name = 'Gabriel';
		//        </script>
		//      </component>`

		//if _, isExprStmt := node.(*js.JsDispatchUnaryExpr); isExprStmt {
		//  // JsDispatchUnaryExpr is an update or unary Expression.
		//  // value++, value--, value*=2
		//  return false
		//}
		//if exprStmt, isExprStmt := node.(*js.ExprStmt); isExprStmt {
		//  s := exprStmt.JS()
		//  println(s + " || " + exprStmt.String())
		//}
	}

	child.Data = innerText

	return err
}
