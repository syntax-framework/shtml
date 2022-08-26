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
	AttributeNames     *sht.IndexedMap
	EventNames         *sht.IndexedMap
	Events             *sht.IndexedMap
	Elements           *sht.IndexedMap
	Watchers           *sht.IndexedMap
	Writers            *sht.IndexedMap
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
			if textNodeErr := p.parseTextNode(child); textNodeErr != nil {
				err = textNodeErr
				stop = true
				return
			}
		} else if child.Type == sht.ElementNode {
			// busca interpolação nos atributos
			for attrNameNormalized, attr := range child.Attributes.Map {
				if strings.HasPrefix(attrNameNormalized, "on") {
					//
				} else {
					if attributeErr := p.parseAttribute(child, attr); attributeErr != nil {
						err = attributeErr
						stop = true
						return
					}
				}
			}
		}
		return
	})

	return err
}

// AddDispatcers(interpolationJsAst, contextAstScope, contextVariables, nil)

// parseAttribute faz processamento dos bindings de atributos (writers e eventos para two way data binding)
func (p *ExpressionsParser) parseAttribute(child *sht.Node, attr *sht.Attribute) error {
	sequence := p.Sequence
	contextAst := p.ContextAst
	contextAstScope := p.ContextAstScope
	contextVariables := p.ContextVariables
	expressions := p.Expressions
	attributes := p.AttributeNames
	elements := p.Elements
	watchers := p.Watchers
	//writers := p.Writers
	events := p.Events
	eventNames := p.EventNames
	getNodeIdentifier := p.NodeIdentifierFunc

	watchersByVar := p.WatchersByVar

	// Check for expressions (${value} or #{value})
	//
	// <element attr="text ${escape safe}">
	// <element attr="text #{escape unsafe}">
	// <element attr="${value}"></element>
	// <element attr="${return value}">
	// <element attr="#{escape unsafe}">
	attrValue, interpolations, interErr := Interpolate(attr.Value, sequence)
	if interErr != nil {
		return interErr
	}
	if interpolations == nil {
		// content has no js expressions (${value} or #{value})
		return nil
	}

	var err error

	elementIndex := strconv.Itoa(elements.Add(getNodeIdentifier(child)))
	attributeIndex := strconv.Itoa(attributes.Add(attr.Normalized))

	// ['content static 1', expressionIndex, 'content static 2']
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

		// resolve references to global scope (component <script> source code and client-param-*)
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

		// A writer applies the result of an expression to something (text, attribute, component, directive), has three forms
		//
		//  A) JS: Array<key: writerIndex, value: [elementIndex, expressionIndex]>
		//    Apply the result of an expression to an element ($(el).innerHtml = value)
		//
		//  B) JS: Array<key: writerIndex, value: [elementIndex, attributeIndex, expressionIndex]>
		//    Applies the result of the expression to an attribute ($(el).setAttribute(value))
		//
		//  C) JS: Array<key: writerIndex, value: [elementIndex, attributeIndex, [string, expressionIndex, string, ...]]>
		//    Apply the (dynamic) template to an attribute, allowing you to check for later changes to the attribute
		//    $(el).setAttribute(parse(template))

		if interpolation.IsFullContent {
			if interpolation.IsSafeSignal {
				// form inputs, can be two-way data-binding
				nodeTagName := child.Data
				if nodeTagName == "input" || nodeTagName == "select" || nodeTagName == "textarea" {
					// 1) Node cannot have "onchange" or "oninput" event defined
					//    onchange             [input, select, textarea] (occurs when the element loses focus)
					//    oninput              [input, select, textarea] (occurs when an element gets user input)
					//    on [paste|cut|drop]
					if child.Attributes.Get("onchange") == "" && child.Attributes.Get("oninput") == "" {
						// 2) Must be reference to a global variable (<input type="text" value="{myVariable}" >)
						//    2.1) Variable must be "let" or "var"
						//    2.2) Cannot be "function" or "arrow function"
						if isSingleRef, jsVar := IsContextSingleLetOrVarReference(interpolationJsAst, contextAst); isSingleRef {
							// add event handler
							variableIndex := strconv.Itoa(contextVariables.Add(jsVar))
							expressionIndex := strconv.Itoa(expressions.Add(
								"(e) => { $.c(" + variableIndex + ", " + elementIndex + ", e); }",
							))
							// JS: Array<[elementIndex, eventNameIndex, expressionIndex]>
							events.Add(
								// https://developer.mozilla.org/en-US/docs/Web/API/HTMLElement/change_event
								"[ " + elementIndex + ", " + strconv.Itoa(eventNames.Add("change")) + ", " + expressionIndex + " ]",
							)
							events.Add(
								// https://developer.mozilla.org/en-US/docs/Web/API/HTMLElement/input_event
								"[ " + elementIndex + ", " + strconv.Itoa(eventNames.Add("input")) + ", " + expressionIndex + " ]",
							)
						}
					}
				}

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
		expressionIndex := strconv.Itoa(expressions.Add("() => { return " + interpolationJs + "; }"))

		// splitting text, to check for modifications on client side
		attrValueParts = strings.Replace(attrValueParts, interpolationId, "', "+expressionIndex+", '", 1)

		watcherIndex := watchers.Add(
			// Create a watcher that bind element attribute to expression result
			// _$bind_prop(elementIndex, attributeIndex, expressionIndex )
			"_$bindToExpression(" + elementIndex + ", " + attributeIndex + ", " + expressionIndex + ") /* " + interpolation.Debug() + " */",
		)

		js.Walk(VisitorEnterFunc(func(node js.INode) bool {
			// só estamos interessados em descobrir as expressoes que acessam variáveis de contexto
			//
			// Uma variável do context só é classificada como observada quando alguma expressão faz uso dessa variável
			// para realizar alguma lógica.
			if jsVar, isVar := node.(*js.Var); isVar {
				if isDeclared, jsVarGlobal := IsDeclaredOnScope(jsVar, contextAstScope); isDeclared {
					// associate watcher to variable
					variableIndex := contextVariables.Add(jsVarGlobal)
					watchersByVar[variableIndex] = append(watchersByVar[variableIndex], watcherIndex)
				}
			}
			return true
		}), interpolationJsAst)
	}

	// Criate a watcher that bind element attribute to expression result
	watchers.Add(
		// _$bind_prop_tpl(elementIndex, attributeIndex, ['content static 1', expressionIdx, 'content static 2'] )
		"_$bind_prop_tpl(" + elementIndex + ", " + attributeIndex + ", ['" + attrValueParts + "'])",
	)

	attr.Value = attrValue

	return err
}

// parseTextNode extrai as expressoes e watchers de um TextNode
func (p *ExpressionsParser) parseTextNode(child *sht.Node) error {
	node := p.Node
	sequence := p.Sequence
	contextAst := p.ContextAst
	contextAstScope := p.ContextAstScope
	contextVariables := p.ContextVariables
	expressions := p.Expressions
	elements := p.Elements
	writers := p.Writers
	watchers := p.Watchers

	// Check for innerText expressions (${value} or #{value})
	//
	// <element>${escape safe}</element>
	// <element>#{escape unsafe}</element>
	innerText, interpolations, textInterErr := Interpolate(child.Data, sequence)
	if textInterErr != nil {
		return textInterErr
	}
	if interpolations == nil {
		// content has no js expressions (${value} or #{value})
		return nil
	}

	var err error
	for elementId, interpolation := range interpolations {

		// replace biding location by <embed hidden class="_xxx">
		innerText = strings.Replace(innerText, elementId, `<embed hidden class="`+elementId+`">`, 1)

		// eventJsCode = "(e) => { " + eventJsCode + " }"
		interpolationJs := interpolation.Expression
		interpolationJsAst, interpolationJsAstErr := js.Parse(parse.NewInputString(interpolationJs), js.Options{})
		if interpolationJsAstErr != nil {
			err = interpolationJsAstErr // @TODO: Custom error or Warning
			break
		}

		// contextVariables
		interpolationJsAstScope := interpolationJsAst.BlockStmt.Scope

		// resolve references to global scope (component <script> source code and client-param-*)
		undeclaredBackup := contextAstScope.Undeclared
		interpolationJsAstScope.Parent = contextAstScope
		interpolationJsAstScope.HoistUndeclared()
		contextAstScope.Undeclared = undeclaredBackup

		// is not allowed to a writer have a side effect (Ex. value++, value = other + 1)
		if hasSideEffect, sideEffectJs := HasSideEffect(interpolationJsAst, contextAst); hasSideEffect {
			err = errorJsInterpolationSideEffect(sideEffectJs, interpolationJsAst.JS(), child.DebugTag(), node.DebugTag())
		}

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

		// identical expressions are reused throughout the code
		expressionIndex := strconv.Itoa(expressions.Add("() => { return " + interpolationJs + "; }"))

		elementIndex := strconv.Itoa(elements.Add(elementId))

		// Apply the result of an expression to an element (innerHtml)
		// JS: Array<key: writerIndex, value: [elementIndex, expressionIndex]>
		writerIndex := strconv.Itoa(writers.Add(
			"[ " + elementIndex + ", " + expressionIndex + "] /* " + interpolation.Debug() + " */",
		))

		// add the watchers for the variables watched by this writer
		js.Walk(VisitorEnterFunc(func(node js.INode) bool {
			if jsVar, isVar := node.(*js.Var); isVar {
				if isDeclared, jsVarContext := IsDeclaredOnScope(jsVar, contextAstScope); isDeclared {
					variableIndex := strconv.Itoa(contextVariables.Add(jsVarContext))
					// JS: Array<key: _, value: [type, variableIndex, writerIndex]>
					//    type 1 = schedule(writerIndex)
					watchers.Add(
						"[ 1, " + variableIndex + ", " + writerIndex + " ] /* var: " + jsVarContext.JS() + ", writer: " + interpolation.Debug() + " */",
					)
				}
			}
			return true
		}), interpolationJsAst)
	}

	child.Data = innerText

	return err
}
