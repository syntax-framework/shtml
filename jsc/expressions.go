package jsc

import (
	"github.com/syntax-framework/shtml/cmn"
	"github.com/syntax-framework/shtml/sht"
	"github.com/tdewolff/parse/v2"
	"github.com/tdewolff/parse/v2/js"
	"golang.org/x/net/html/atom"
	"log"
	"strconv"
	"strings"
)

type ExpressionsParser struct {
	Node               *sht.Node
	Sequence           *sht.Sequence
	ContextAst         *js.AST
	ContextAstScope    *js.Scope
	ContextVariables   *cmn.IndexedSet
	Elements           *cmn.IndexedSet
	AttributeNames     *cmn.IndexedSet
	Expressions        *cmn.IndexedSet
	EventNames         *cmn.IndexedSet
	Events             *cmn.IndexedSet
	Writers            *cmn.IndexedSet
	Watchers           *cmn.IndexedSet
	NodeIdentifierFunc func(node *sht.Node) string
}

var errorJsInterpolationSideEffect = cmn.Err(
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

	var err error
	node.Transverse(func(child *sht.Node) (stop bool) {
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
					if eventErr := p.parseAttributeEvent(child, attr); eventErr != nil {
						err = eventErr
						stop = true
						return
					}
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

// parseAttributeEvent handles the html events defined in an element
func (p *ExpressionsParser) parseAttributeEvent(child *sht.Node, attr *sht.Attribute) error {
	//contextAst := p.ContextAst
	contextAstScope := p.ContextAstScope
	//contextVariables := p.ContextVariables
	expressions := p.Expressions
	//attributes := p.AttributeNames
	elements := p.Elements
	//watchers := p.Watchers
	//writers := p.Writers
	events := p.Events
	eventNames := p.EventNames
	getNodeIdentifier := p.NodeIdentifierFunc

	eventJsCode := strings.TrimSpace(attr.Value)

	// eventJsCode = "(e) => { " + eventJsCode + " }"
	eventJsAst, eventJsAstErr := js.Parse(parse.NewInputString(eventJsCode), js.Options{})
	if eventJsAstErr != nil {
		return eventJsAstErr // @TODO: Custom error or Warning
	}

	eventJsAstScope := eventJsAst.BlockStmt.Scope

	// resolve reference to eventIdx ("e") variable. Ex. "(e) => { myCallback(e.MouseX) }"
	eventJsAstScope.Parent = eventVariableScope
	eventJsAstScope.HoistUndeclared()
	eventVariableScope.Undeclared = nil

	// resolve references to global scope (component <script> source code and client-param-*)
	undeclaredBackup := contextAstScope.Undeclared
	eventJsAstScope.Parent = contextAstScope
	eventJsAstScope.HoistUndeclared()
	contextAstScope.Undeclared = undeclaredBackup

	stmt := eventJsAst.BlockStmt.List[0]
	if exprStmt, isExprStmt := stmt.(*js.ExprStmt); isExprStmt {
		switch exprStmt.Value.(type) {
		case *js.CallExpr:
			// <element onclick="someFunc(arg1, arg2, argn...)">
			callExpr := exprStmt.Value.(*js.CallExpr)
			if jsVar, isVar := callExpr.X.(*js.Var); isVar {
				if isDeclared, _ := IsDeclaredOnScope(jsVar, contextAstScope); isDeclared {
					// is a custom javascript function or "client-param-name"
					// Ex. <button onclick="onClick()">
					// AddDispatcers(interpolationJsAst, contextAstScope, contextVariables, nil)
					eventJsCode = "(e) => { " + callExpr.JS() + " }"
				} else {
					// considers it to be a remote eventIdx call (push)
					functionName := jsVar.String()
					eventName := functionName
					eventPayload := ""
					if functionName == "push" {
						// <button onclick="push('increment', count, time, e.MouseX)" data-ref="mySpan">
					} else {
						// <button onclick="increment(count, time, e.MouseX)">

					}
					// AddDispatcers(interpolationJsAst, contextAstScope, contextVariables, nil)
					eventJsCode = "(e) => { push('" + eventName + "', e, " + eventPayload + ") }"
				}
			} else {
				log.Println("[@TODO] UNKNOWN: what to do? At jsc.parseAttributeEvent(*sht.NodeTest, *sht.Attribute)")
			}
		case *js.Var:
			// <element onclick="someVariable">
			jsVar := exprStmt.Value.(*js.Var)
			if isDeclared, _ := IsDeclaredOnScope(jsVar, contextAstScope); isDeclared {
				// is a custom javascript variable or "client-param-name"
				// Ex. <button onclick="callback"></button>
				eventJsCode = "(e) => { " + jsVar.String() + "(e) }"
			} else {
				// considers it to be a remote eventIdx call (push)
				// <button onclick="increment"></button>
				eventJsCode = "(e) => { push('" + jsVar.String() + "', e) }"
			}
		case *js.ArrowFunc:
			// <element onclick="(e) => doSomething">
			// AddDispatcers(interpolationJsAst, contextAstScope, contextVariables, nil)
			eventJsCode = exprStmt.Value.(*js.ArrowFunc).JS()
		case *js.FuncDecl:
			// <element onclick="function xpto(e){ doSomething() }">
			// AddDispatcers(interpolationJsAst, contextAstScope, contextVariables, nil)
			eventJsCode = "(e) => { (" + exprStmt.Value.(*js.FuncDecl).JS() + ")() }"
		default:
			// AddDispatcers(interpolationJsAst, contextAstScope, contextVariables, nil)
			eventJsCode = "(e) => { " + eventJsCode + " }"
		}
	}

	// add event handler
	elementIndex := strconv.Itoa(elements.Add(getNodeIdentifier(child)))
	eventNameIndex := strconv.Itoa(eventNames.Add(attr.Normalized[2:]))
	expressionIndex := strconv.Itoa(expressions.Add(eventJsCode))
	events.Add(
		// JS: Array<[elementIndex, eventNameIndex, expressionIndex]>
		"[ " + elementIndex + ", " + eventNameIndex + ", " + expressionIndex + " ]",
	)

	// remove html event
	child.Attributes.Remove(attr)

	return nil
}

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
	writers := p.Writers
	events := p.Events
	eventNames := p.EventNames
	getNodeIdentifier := p.NodeIdentifierFunc

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
	isTemplate := true
	var templateInterpolations []*js.AST
	// [string, expressionIndex, string, expressionIndex, string ...]
	templateExpressionsJsArr := "['" + strings.ReplaceAll(attrValue, "'", "\\'") + "']"

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
			break
		}

		interpolationJs = interpolationJsAst.JS()
		if strings.HasSuffix(interpolationJs, "; ") {
			interpolationJs = interpolationJs[:len(interpolationJs)-2]
		}

		if interpolation.IsFullContent {
			isTemplate = false
			if interpolation.IsSafeSignal {
				// form inputs, can be two-way data-binding
				nodeTagName := child.Data
				if nodeTagName == "input" || nodeTagName == "select" || nodeTagName == "textarea" {
					// 1) NodeTest cannot have "onchange" or "oninput" event defined
					//    onchange             [input, select, textarea] (occurs when the element loses focus)
					//    oninput              [input, select, textarea] (occurs when an element gets user input)
					//    on [paste|cut|drop]
					if child.Attributes.Get("onchange") == "" && child.Attributes.Get("oninput") == "" {
						// 2) Must be reference to a global variable (<input type="text" value="{myVariable}" >)
						//    2.1) Variable must be "let" or "var"
						//    2.2) Cannot be "function" or "arrow function"
						if isSingleRef, jsVar := IsContextSingleLetOrVarReference(interpolationJsAst, contextAst); isSingleRef {
							// add event handler
							variableIndex := strconv.Itoa(contextVariables.GetIndex(jsVar))
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
				if child.Data == "component" || child.Attributes.Get("data-syntax-component") == "true" {
					// is component
					interpolationJs = "(" + interpolationJs + ")"
				} else {
					interpolationJs = "$.e(" + interpolationJs + ")"
				}
			} else {
				// <element attr="#{escape unsafe}">
				interpolationJs = "(" + interpolationJs + ")"
			}
		} else {
			if interpolation.IsSafeSignal {
				// <element attr="text ${escape safe}">
				interpolationJs = "$.e(" + interpolationJs + ")"
			} else {
				// <element attr="text #{escape unsafe}">
				interpolationJs = "(" + interpolationJs + ")"
			}
		}

		// identical expressions are reused throughout the code
		expressionIndex := strconv.Itoa(expressions.Add("() => { return " + interpolationJs + "; }"))

		if isTemplate {
			// many expressions <element attr="text ${myVariable} text " >

			// splitting text, to check for modifications on client side
			templateInterpolations = append(templateInterpolations, interpolationJsAst)
			templateExpressionsJsArr = strings.Replace(templateExpressionsJsArr, interpolationId, "', "+expressionIndex+", '", 1)

		} else {
			// full content, single expression <element attr="${myVariable}" >

			// Applies the result of the expression to an attribute ($(el).setAttribute(value))
			// JS: Array<key: writerIndex, value: [elementIndex, attributeIndex, expressionIndex]>
			writerIndex := strconv.Itoa(writers.Add(
				"[ " + elementIndex + ", " + attributeIndex + ", " + expressionIndex + "] /* " + interpolation.Debug() + " */",
			))

			// add the watchers for the variables watched by this writer
			js.Walk(VisitorEnterFunc(func(node js.INode) bool {
				if jsVar, isVar := node.(*js.Var); isVar {
					if isDeclared, jsVarContext := IsDeclaredOnScope(jsVar, contextAstScope); isDeclared {
						variableIndex := strconv.Itoa(contextVariables.GetIndex(jsVarContext))
						// JS: Array<key: _, value: [type, variableIndex, writerIndex]>
						//    type 1 = schedule(writerIndex)
						watchers.Add(
							"[ 1, " + variableIndex + ", " + writerIndex + " ] /* " + jsVarContext.JS() + " -> " + interpolation.Debug() + " */",
						)
					}
				}
				return true
			}), interpolationJsAst)
		}
	}

	if err != nil {
		return err
	}

	if isTemplate {

		// Apply the (dynamic) template to an attribute, allowing you to check for later changes to the attribute
		// JS: Array<key: writerIndex, value: [elementIndex, attributeIndex, [string, expressionIndex, string, ...]]>
		// $(el).setAttribute(parse(template))
		writerIndex := strconv.Itoa(writers.Add(
			"[ " + elementIndex + ", " + attributeIndex + ", " + templateExpressionsJsArr + "",
		))

		for _, ast := range templateInterpolations {
			// add the watchers for the variables watched by this writer
			js.Walk(VisitorEnterFunc(func(node js.INode) bool {
				if jsVar, isVar := node.(*js.Var); isVar {
					if isDeclared, jsVarContext := IsDeclaredOnScope(jsVar, contextAstScope); isDeclared {
						variableIndex := strconv.Itoa(contextVariables.GetIndex(jsVarContext))
						// JS: Array<key: _, value: [type, variableIndex, writerIndex]>
						//    type 1 = schedule(writerIndex)
						watchers.Add(
							"[ 1, " + variableIndex + ", " + writerIndex + " ] /* " + jsVarContext.JS() + " */",
						)
					}
				}
				return true
			}), ast)
		}
	}

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

	nodeParent := child.Parent

	var err error
	for elementId, interpolation := range interpolations {

		parts := strings.Split(innerText, elementId)

		// pre text
		nodeParent.AppendChild(&sht.Node{
			Type:   sht.TextNode,
			Data:   parts[0],
			File:   child.File,
			Line:   child.Line,
			Column: child.Column,
		})

		// replace biding location by <embed hidden class="_xxx">
		// https://html.spec.whatwg.org/#the-embed-element
		anchor := &sht.Node{
			Type:       sht.ElementNode,
			Data:       "embed",
			DataAtom:   atom.S,
			File:       child.File,
			Line:       child.Line,
			Column:     child.Column,
			Attributes: &sht.Attributes{Map: map[string]*sht.Attribute{}},
		}
		anchor.Attributes.Set("hidden", "hidden")
		anchor.Attributes.Set("id", elementId)
		nodeParent.AppendChild(anchor)

		innerText = parts[1]

		//innerText = strings.Replace(innerText, elementId, `<embed hidden class="`+elementId+`">`, 1)

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
			break
		}

		interpolationJs = interpolationJsAst.JS()
		if strings.HasSuffix(interpolationJs, "; ") {
			interpolationJs = interpolationJs[:len(interpolationJs)-2]
		}
		if interpolation.IsSafeSignal {
			// <element>${escape safe}</element>
			interpolationJs = "$.e(" + interpolationJs + ")"
		} else {
			// <element>#{escape unsafe}</element>
			interpolationJs = "(" + interpolationJs + ")"
		}

		// identical expressions are reused throughout the code
		expressionIndex := strconv.Itoa(expressions.Add("() => { return " + interpolationJs + "; }"))

		elementIndex := strconv.Itoa(elements.Add("#" + elementId))

		// Apply the result of an expression to an element (innerHtml)
		// JS: Array<key: writerIndex, value: [elementIndex, expressionIndex]>
		writerIndex := strconv.Itoa(writers.Add(
			"[ " + elementIndex + ", " + expressionIndex + "] /* " + interpolation.Debug() + " */",
		))

		// add the watchers for the variables watched by this writer
		js.Walk(VisitorEnterFunc(func(node js.INode) bool {
			if jsVar, isVar := node.(*js.Var); isVar {
				if isDeclared, jsVarContext := IsDeclaredOnScope(jsVar, contextAstScope); isDeclared {
					variableIndex := strconv.Itoa(contextVariables.GetIndex(jsVarContext))
					// JS: Array<key: _, value: [type, variableIndex, writerIndex]>
					//    type 1 = schedule(writerIndex)
					watchers.Add(
						"[ 1, " + variableIndex + ", " + writerIndex + " ] /* " + jsVarContext.JS() + " -> " + interpolation.Debug() + " */",
					)
				}
			}
			return true
		}), interpolationJsAst)
	}

	// post text
	nodeParent.AppendChild(&sht.Node{
		Type:   sht.TextNode,
		Data:   innerText,
		File:   child.File,
		Line:   child.Line,
		Column: child.Column,
	})

	child.Remove()

	return err
}
