package jsc

import (
	"bytes"
	"fmt"
	"github.com/syntax-framework/shtml/sht"
	"github.com/tdewolff/parse/v2"
	"github.com/tdewolff/parse/v2/js"
	"golang.org/x/net/html/atom"
	"strconv"
	"strings"
)

var errorCompJsRedeclaration = sht.Err(
	"component.js.redeclaration",
	"SyntaxError: Identifier has already been declared.", "Identifier: %s", "Context: %s", "Component: %s",
)

// @TODO: Usar https://github.com/evanw/esbuild?

// jsWatchInvalidateBlock
//
// ALGORITMO DE _$watches e _$invalidate
//  Definições
//    1. Atribuir valor: Qualquer expressão que modifica uma variável observável, disparando um evento
//    2. Variável observável: Uma variável global usada dentro de uma expressão watch
//    2. Watch: Uma expressão que faz referencia a uma variável observável e é executada sempre que essa variável sofre alteração
//
//  Quem atribui valor / modifica variável / dispara evento?
//    - Código javascript (Ex. function onClick(){ myVar += 1 })
//    - Eventos html (Ex. <element onclick="myvar += 1" />)
//
//  Onde a variável/evento é observado?
//    - Bloco de interpolação (Ex. {{myVar}})
//    - Directivas js/blocos de controle  (Ex. <element if="myVar == 3" />)
//
//  Algoritmo de Watch.
//
//  1 - Mapear todas as expressoes $watch
//  2 - Para cada expressão, mapear suas variáveis observáveis, associando um índice a cada uma
//  3 - Para cada expressão de atribuição de valor de variáveis observáveis que ocorram foram do escopo global (segundo
//      nível em diante), encapsular com expr comando de invalidate (_$i(index, currentValue, expression))
//

// ALGORITMO DE _$watches e _$invalidate
//  Definições
//    1. Atribuir valor: Qualquer expressão que modifica uma variável observável, disparando um evento
//    2. Variável observável: Uma variável global usada dentro de uma expressão watch
//    2. Watch: Uma expressão que faz referencia a uma variável observável e é executada sempre que essa variável sofre alteração
//
//  Quem atribui valor / modifica variável / dispara evento?
//    - Código javascript (Ex. function onClick(){ myVar += 1 })
//    - Eventos html (Ex. <element onclick="myvar += 1" />)
//
//  Onde a variável/evento é observado?
//    - Bloco de interpolação de texto (Ex. ${myVar} | <element class="class-a ${myVar == 3 ? 'new-class' : ''}" />)
//    - Directivas js/blocos de controle  (Ex. <element if="${myVar == 3}" /> | <if cond="${myVar == 3}">)
//
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
//
//  Algoritmo de Watch.
//
//  1 - Mapear todas as expressoes $watch
//  2 - Para cada expressão, mapear suas variáveis observáveis, associando um índice a cada uma
//  3 - Para cada expressão de atribuição de valor de variáveis observáveis que ocorram foram do escopo global (segundo
//      nível em diante), encapsular com expr comando de invalidate (_$i(index, currentValue, watcher))
//
// JAVASCRIPT INTERPOLATION ( ${value} or  #{value} )
//
// <element attribute="${return value}">
// <element attribute="xpto ${escape safe}">
// <element attribute="xpto #{escape unsafe}">
// <element attribute="#{escape unsafe}">
// <element>${escape safe}</element>
// <element>#{escape unsafe}</element>

// eventVariableScope escopo usado para referenciar a variável e. Ex. (e) => { myCallback(e); }
var eventVariableScope *js.Scope

func init() {
	ast, err := js.Parse(parse.NewInputString("let e;"), js.Options{})
	if err != nil {
		panic(err)
	}
	eventVariableScope = &ast.BlockStmt.Scope
}

// Compile does all the necessary handling to link the template with javascript
func Compile(nodeParent *sht.Node, nodeScript *sht.Node, t *sht.Compiler) (asset *Javascript, err error) {

	sequence := &sht.Sequence{}
	nodeParentIsComponent := false
	if nodeParent.Data == "component" {
		nodeParentIsComponent = true
		sequence.Salt = nodeParent.Attributes.Get("name")
	} else {
		if nodeScript != nil && nodeScript.FirstChild != nil {
			sequence.Salt = sht.HashXXH64(nodeScript.FirstChild.Data)
		} else {
			sequence.Salt = t.NextHash()
		}
	}

	// Unique node identifier (#id or .class-name)
	// @TODO: make global to prevent a node from having more than one identifier
	getNodeIdentifier := sht.CreateNodeIdentifier(sequence)

	// All script global variables being used
	contextVariables := &sht.IndexedMap{}

	// All expressions created in the code are idexed, to allow removing duplicates
	// JS: Array<key: expressionIndex, value: Function>
	expressions := &sht.IndexedMap{}

	// The elements used by the script and which therefore must be referenced in the code
	// JS: Array<key: elementIndex, value: string(#id|.class-name)>
	elements := &sht.IndexedMap{}

	// All html attributeNames referencied in writers (Ex. src, href, value)
	// JS: Array<key: attributeIndex, value: string>
	attributeNames := &sht.IndexedMap{}

	// All html events referenced in code (Ex. click, change)
	// JS: Array<key: eventNameIndex, value: string>
	eventNames := &sht.IndexedMap{}

	// Mapping of html events that are fired in code
	// JS: Array<[elementIndex, eventNameIndex, expressionIndex]>
	events := &sht.IndexedMap{}

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
	writers := &sht.IndexedMap{}

	// All watches. Represent expressions that will react when a variable changes.
	// JS: Array<key: _, value: [type, variableIndex, expressionIndex|writerIndex]>
	//    type 0 = action(expressionIndex)
	//    type 1 = schedule(writerIndex)
	watchers := &sht.IndexedMap{}

	// parse component params (<element param-name="type" client-param-name="type">)
	var componentParams *NodeComponentParams
	if nodeParentIsComponent {
		if componentParams, err = ParseComponentParams(nodeParent); err != nil {
			return nil, err
		}
	}

	// parse references to elements within the template (<element ref="myJsVariable">)
	references, referenceErr := ParseReferences(nodeParent, t)
	if referenceErr != nil {
		return nil, referenceErr
	}

	if len(references) > 0 {
		// initialize ref elements class id
		for _, reference := range references {
			elements.Add(getNodeIdentifier(reference.Node))

			// validates reference names against parameter names, avoids double definition of variables
			if componentParams != nil {
				identifier := reference.Attr.Normalized
				if componentParams.ClientParamsByName[identifier] != nil {
					return nil, errorCompJsRedeclaration(identifier, "reference -> client-param", nodeParent.DebugTag())
				}
			}
		}
	}

	// initialize the parameters (need to be visible in global scope to be indexed)
	var jsSource string
	if componentParams != nil && len(componentParams.ClientParams) > 0 {
		compJsParams := &bytes.Buffer{}
		compJsParams.WriteString("\n    // parameters")
		compJsParams.WriteString("\n    const _$params = $.params;\n")
		for _, jsParam := range componentParams.ClientParams {
			name := jsParam.Name
			compJsParams.WriteString(fmt.Sprintf("    let %s = _$params['%s'];\n", name, name))
		}
		compJsParams.WriteString("\n    $.p(() => {\n")
		for _, jsParam := range componentParams.ClientParams {
			name := jsParam.Name
			compJsParams.WriteString(fmt.Sprintf("      %s = _$params['%s'];\n", name, name))
		}
		compJsParams.WriteString("    });\n")
		jsSource = compJsParams.String()
	}

	// @TODO: Map all watches that are actually static (the variable is never changed by js) in these cases, display warning to developer?

	if nodeScript != nil {
		if nodeScript.FirstChild != nil {
			// original source code
			jsSource = jsSource + nodeScript.FirstChild.Data
		}

		if nodeParentIsComponent {
			// when component, remove script from render
			nodeScript.Remove()
		} else {
			// for page scripts, remove only content, will be initialized by syntax using this element
			nodeScript.FirstChild = nil
			nodeScript.LastChild = nil
		}
	}

	contextJsAst, contextJsAstErr := js.Parse(parse.NewInputString(jsSource), js.Options{})
	if contextJsAstErr != nil {
		err = contextJsAstErr // @TODO: Custom error or Warning
		return
	}
	contextAstScope := &contextJsAst.BlockStmt.Scope

	// @TODO: fork the project https://github.com/tdewolff/parse/tree/master/js and add feature to keep original formatting
	jsSource = contextJsAst.JS()

	// @DEPRECATED
	idxEventHandlers := &sht.IndexedMap{}

	// @DEPRECATED
	// identifies which watches are candidates for replay when a variable changes
	// [ $varIndex = [$watchIndexA, $watchIndexB]]
	watchersByVar := map[int][]int{}

	// 1 - Map all watch expressions. After that, all the variables that are being watched will have been mapped
	expressionsErr := (&ExpressionsParser{
		Node:               nodeParent,
		Compiler:           t,
		Sequence:           sequence,
		ContextAst:         contextJsAst,
		ContextAstScope:    contextAstScope,
		ContextVariables:   contextVariables,
		Elements:           elements,
		Events:             events,
		EventNames:         eventNames,
		AttributeNames:     attributeNames,
		Expressions:        expressions,
		Writers:            writers,
		Watchers:           watchers,
		WatchersByVar:      watchersByVar,
		NodeIdentifierFunc: getNodeIdentifier,
	}).Parse()
	if expressionsErr != nil {
		err = expressionsErr // @TODO: Custom error or Warning
		return
	}

	// Parse Events
	t.Transverse(nodeParent, func(child *sht.Node) (stop bool) {
		stop = false
		if child == nodeParent || child.Type != sht.ElementNode {
			return
		}

		// @TODO: When child is a registered component, do proper expr processing
		// nodeParentIsComponent := false

		for attrNameNormalized, attr := range child.Attributes.Map {
			if strings.HasPrefix(attrNameNormalized, "on") {

				// html events
				eventJsCode := strings.TrimSpace(attr.Value)

				elements.Add(getNodeIdentifier(child))

				if strings.HasPrefix(eventJsCode, "js:") {
					// If it has the prefix "js:", it does not process
					//    Ex. <button onclick="js: doSomeThing && doOtherThing">
					eventJsCode = strings.Replace(eventJsCode, "js:", "", 1)
				} else if strings.HasPrefix(eventJsCode, "javascript:") {
					// If it has the prefix "javascript:", it does not process
					//    Ex. <button onclick="javascript: doSomeThing && doOtherThing">
					eventJsCode = strings.Replace(eventJsCode, "javascript:", "", 1)
				} else {

					//eventJsCode = "(e) => { " + eventJsCode + " }"
					eventJsAst, eventJsAstErr := js.Parse(parse.NewInputString(eventJsCode), js.Options{})
					if eventJsAstErr != nil {
						err = eventJsAstErr // @TODO: Custom error or Warning
						break
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
									eventJsCode = "(e) => { " + callExpr.JS() + " }"
								} else {
									// @TODO: STX.push
									// considers it to be a remote eventIdx call (push)
									functionName := jsVar.String()
									eventName := functionName
									eventPayload := ""
									if functionName == "push" {
										// <button onclick="push('increment', count, time, e.MouseX)" data-ref="mySpan">
									} else {
										// <button onclick="increment(count, time, e.MouseX)">

									}
									eventJsCode = fmt.Sprintf("(e) => { STX.push('%s', $, e, %s) }", eventName, eventPayload)
								}
							} else {
								panic("@TODO: what to do? I didn't find a scenario")
							}
						case *js.Var:
							// <element onclick="someVariable">
							jsVar := exprStmt.Value.(*js.Var)
							if isDeclared, _ := IsDeclaredOnScope(jsVar, contextAstScope); isDeclared {
								// is a custom javascript variable or "client-param-name"
								// Ex. <button onclick="callback"></button>
								eventJsCode = fmt.Sprintf("(e) => { %s(e) }", jsVar.String())
							} else {
								// considers it to be a remote eventIdx call (push)
								// <button onclick="increment"></button>
								eventJsCode = fmt.Sprintf("(e) => { STX.push('%s', $, e) }", jsVar.String())
							}
						case *js.ArrowFunc:
							// <element onclick="(e) => doSomething">
							eventJsCode = exprStmt.Value.(*js.ArrowFunc).JS()
						case *js.FuncDecl:
							// <element onclick="function xpto(e){ doSomething() }">
							eventJsCode = "(e) => { (" + exprStmt.Value.(*js.FuncDecl).JS() + ")() }"
						default:
							eventJsCode = fmt.Sprintf("(e) => { %s }", eventJsCode)
						}
					}

					//println(eventJsAst)
					// Se existir "client-param-*" ou uma variável com mesmo nome no inicio, não processa
					//    <button onclick="onClick()">
					//    <button onclick="callback">
					// Se tiver expr padrão "NOME_EVENTO" ou "NOME_EVENTO(...)" ou  "push(NOME_EVENTO, ...)", considera que é push para expr server
					//    <button onclick="increment">
					//    <button onclick="increment(count, time, e.MouseX)">
					//    <button onclick="push('increment', count, time, e.MouseX)" data-ref="mySpan">
					// Para todos os outros casos, não processa
				}

				// add event handler
				// _$on(_$event_names[0], _$elements[1], _$event_handlers[5]);
				eventIdx := strconv.Itoa(eventNames.Add(strings.Replace(attrNameNormalized, "on", "", 1)))
				handlerIdx := strconv.Itoa(idxEventHandlers.Add(eventJsCode))
				elementIdx := strconv.Itoa(elements.Add(getNodeIdentifier(child)))
				events.Add(
					"[" + eventIdx + ", " + elementIdx + ", " + handlerIdx + "]",
				)
			}
		}

		if err != nil {
			stop = true
		}

		return
	})

	if err != nil {
		return
	}

	// write the component JS
	bjs := &bytes.Buffer{}

	if nodeParentIsComponent {
		bjs.WriteString("STX.c('" + nodeParent.Data + "', function (STX) {\n")
	} else {
		var anchorId string
		if nodeScript != nil {
			//nodeScript.Attributes.Set("data-syntax-s", "")
			anchorId = getNodeIdentifier(nodeScript)
		} else {
			anchorScript := &sht.Node{
				Data:       "script",
				DataAtom:   atom.Script,
				File:       nodeParent.File,
				Line:       nodeParent.Line,
				Column:     nodeParent.Column,
				Attributes: &sht.Attributes{Map: map[string]*sht.Attribute{}},
			}
			//anchorScript.Attributes.Set("data-syntax-s", "")
			nodeParent.AppendChild(anchorScript)
			anchorId = getNodeIdentifier(anchorScript)
		}
		bjs.WriteString("STX.s('" + anchorId + "', function (STX) {\n")
	}

	// component constants
	if nodeScript != nil {
		bjs.WriteString(fmt.Sprintf(`  const _$line = %d;`, nodeScript.Line))
	} else {
		bjs.WriteString(fmt.Sprintf(`  const _$line = %d;`, nodeParent.Line))
	}
	bjs.WriteRune('\n')
	bjs.WriteString(fmt.Sprintf(`  const _$file = "%s";`, nodeParent.File))
	bjs.WriteRune('\n')

	if !watchers.IsEmpty() {
		bjs.WriteString("\n  const _$bind = STX.bindToText;\n")
	}

	bjs.WriteString("\n  return {\n    f: _$file,\n    l: _$line,")

	// Elements class ids
	if !elements.IsEmpty() {
		bjs.WriteString("\n    e: [ /* $elements */")
		for i, id := range elements.ToArray() {
			if i > 0 {
				bjs.WriteString(",")
			}
			bjs.WriteString("\n      '" + id.(string) + "'")
		}
		bjs.WriteString("\n    ],")
	}

	// Attributes used
	if !attributeNames.IsEmpty() {
		bjs.WriteString("\n    a : [")
		for i, name := range attributeNames.ToArray() {
			if i > 0 {
				bjs.WriteString(", ")
			}
			bjs.WriteString("'" + name.(string) + "'")
		}
		bjs.WriteString("] /* $attributeNames */,")
	}

	// Event names
	if !eventNames.IsEmpty() {
		bjs.WriteString("\n    n : [")
		for i, name := range eventNames.ToArray() {
			if i > 0 {
				bjs.WriteString(", ")
			}
			bjs.WriteString("'" + name.(string) + "'")
		}
		bjs.WriteString("] /* $event_names */,")
	}

	if !events.IsEmpty() {
		bjs.WriteString("\n    o : [ /* $events */")
		// _$on(_$event_names[0], _$elements[5], _$event_handlers[1]);
		for j, jsEventHandler := range events.ToArray() {
			if j > 0 {
				bjs.WriteString(", ")
			}
			bjs.WriteString("\n      " + jsEventHandler.(string))
		}
		bjs.WriteString("\n    ],")
	}

	// watchers
	if !watchers.IsEmpty() {
		bjs.WriteString("\n    w : [ /* $watchers */")
		for i, watcher := range watchers.ToArray() {
			if i > 0 {
				bjs.WriteString(",")
			}
			bjs.WriteString("\n      " + watcher.(string))
		}
		bjs.WriteString("\n    ],")
	}

	// START - Instance Function
	bjs.WriteString("\n    i : function ($) {\n")
	bjs.WriteString("      const [_$escape ] = [STX.escape];\n")

	// initialize references (need to be visible in global scope to be indexed)
	if len(references) > 0 {
		bjs.WriteString("\n      // references (define)\n")
		for _, reference := range references {
			bjs.WriteString(fmt.Sprintf(`      let %s;`, reference.VarName))
			bjs.WriteRune('\n')
		}
	}

	// component code
	bjs.WriteString("\n      // component\n")
	if jsSource != "" {
		bjs.WriteString("      " + jsSource + "\n")
	}

	// Configure hooks (lifecycle callback and api)
	var jsHooks []string
	for _, v := range contextAstScope.Declared {
		varName := v.String()
		if ClientComponentFieldsMap[varName] == true {
			jsHooks = append(jsHooks, varName)
		}
	}
	if len(jsHooks) > 0 {
		bjs.WriteString("\n      // hooks\n      $.hooks({ ")
		for i, hook := range jsHooks {
			if i > 0 {
				bjs.WriteString(", ")
			}
			bjs.WriteString(hook)
		}
		bjs.WriteString(" })\n")
	}

	// initialize references
	if len(references) > 0 {
		bjs.WriteString("\n      // references (initialize)\n")
		for _, ref := range references {
			isComponent := false

			elementIdx := strconv.Itoa(elements.Get(getNodeIdentifier(ref.Node)))

			// if is component
			bjs.WriteString("      " + ref.VarName + " = ")
			if isComponent {
				bjs.WriteString("STX.init('otherComponent', $(" + elementIdx + "), { callback: () => fazAlgumaCoisa() });")
			} else {
				bjs.WriteString("$(" + elementIdx + ");")
			}
			bjs.WriteRune('\n')

			// remove attribute from nodeParent (to not be rendered anymore)
			ref.Node.Attributes.Remove(ref.Attr)
		}
	}

	// START - Return instance
	bjs.WriteString("\n      return {")

	// all expressions
	if !expressions.IsEmpty() {
		bjs.WriteString("\n        e : [ /* $expressions */")
		for i, expression := range expressions.ToArray() {
			if i > 0 {
				bjs.WriteString(",")
			}
			bjs.WriteString("\n          " + expression.(string))
		}
		bjs.WriteString("\n        ],")
	}

	// Event names
	if !idxEventHandlers.IsEmpty() {
		bjs.WriteString("\n        h : [ /* $event_handlers */")
		for i, handler := range idxEventHandlers.ToArray() {
			if i > 0 {
				bjs.WriteString(",")
			}
			bjs.WriteString("\n          " + handler.(string))
		}
		bjs.WriteString("\n        ]")
	}

	bjs.WriteString("\n      };\n") // END - Return instance
	bjs.WriteString("    }\n")      // END - Instance Function
	bjs.WriteString("  }\n")        // END - Return constructor
	bjs.WriteString("})")           // END

	// to no longer be rendered
	if componentParams != nil {
		for _, attr := range componentParams.AttrsToRemove {
			nodeParent.Attributes.Remove(attr)
		}
	}

	println(bjs.String())

	jsCode := &Javascript{
		Content: bjs.String(),
		//ComponentParams: ClientParams,
	}

	return jsCode, nil
}
