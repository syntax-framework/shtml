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

	// unique node identifier.
	// @TODO: make global to prevent a node from having more than one identifier
	getNodeIdentifier := sht.CreateNodeIdentifier(sequence)

	//
	//  Obs. Não é necessário interceptar as atribuições de nível global pois entende-se que essa atribuição está sendo
	//  realizada na instanciação do componente, portanto, os observers já irão receber expr valor correto neste momento
	// @TODO: parse content to get all watch variables (Ex. {{time}}, {{count}}, <element js-hide="count == 0">)

	// all script global variables being used
	contextVariables := &sht.IndexedMap{}

	// All expressions created in the code are idexed, to allow removing duplicates
	expressions := &sht.IndexedMap{}

	// All html attributes referencied in watchers
	attributes := &sht.IndexedMap{}

	// Html events
	//
	// Source: <element onclick="onClickFn(e.MouseX)">)
	// Transpiled to JS: (e) => { onClickFn(e.MouseX) }
	idxEventNames := &sht.IndexedMap{}    // click, change
	idxEventHandlers := &sht.IndexedMap{} // console.log(e.MouseX)
	idxEvents := &sht.IndexedMap{}        // _$on(_$event_names[0], _$elements[1], _$event_handlers[1])

	// All watches. Represent expressions that will react when a variable changes.
	//    - Text interpolation block
	//      - <element>${value} #{value} ${value + other}</element>
	//      - <element class="class-a ${myVar == 3 ? 'new-class' : ''}" />
	//    - js directives or control blocks
	//      - <element if="${myVar == 3}" />
	//      - <if cond="${myVar == 3}">
	watchers := &sht.IndexedMap{}

	// identifies which watches are candidates for replay when a variable changes
	// [ $varIndex = [$watchIndexA, $watchIndexB]]
	watchersByVar := map[int][]int{}

	// Os elementos usados pelo script e que portanto devem ser referenciados no código
	elementIdentifiers := &sht.IndexedMap{}

	// parse component params
	var componentParams *NodeComponentParams
	if nodeParentIsComponent {
		if componentParams, err = ParseComponentParams(nodeParent); err != nil {
			return nil, err
		}
	}

	// parse references to elements within the template
	references, refErr := ParseReferences(nodeParent, t, elementIdentifiers)
	if refErr != nil {
		return nil, refErr
	}

	if len(references) > 0 {
		// initialize ref elements class id
		for _, reference := range references {
			elementIdentifiers.Add(getNodeIdentifier(reference.Node))

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
		compJsParams.WriteString("\n    // parameters (define)")
		compJsParams.WriteString("\n    let _$params = $.params;\n")
		for _, jsParam := range componentParams.ClientParams {
			name := jsParam.Name
			compJsParams.WriteString(fmt.Sprintf("    let %s = _$params['%s'];\n", name, name))
		}
		compJsParams.WriteString("\n    // parameters (watch)")
		compJsParams.WriteString("\n    $.onChangeParams(() => {\n")
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

	contextJsAst, globalJsAstErr := js.Parse(parse.NewInputString(jsSource), js.Options{})
	if globalJsAstErr != nil {
		err = globalJsAstErr // @TODO: Custom error or Warning
		return
	}
	contextAstScope := &contextJsAst.BlockStmt.Scope

	// @TODO: fork the project https://github.com/tdewolff/parse/tree/master/js and add feature to keep original formatting
	jsSource = contextJsAst.JS()

	// 1 - Map all watch expressions. After that, all the variables that are being watched will have been mapped
	expressionsErr := (&ExpressionsParser{
		Node:               nodeParent,
		Compiler:           t,
		Sequence:           sequence,
		ContextAst:         contextJsAst,
		ContextAstScope:    contextAstScope,
		ContextVariables:   contextVariables,
		ElementIdentifiers: elementIdentifiers,
		Expressions:        expressions,
		Watchers:           watchers,
		BindedAttributes:   attributes,
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

				elementIdentifiers.Add(getNodeIdentifier(child))

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
				eventIdx := strconv.Itoa(idxEventNames.Add(strings.Replace(attrNameNormalized, "on", "", 1)))
				handlerIdx := strconv.Itoa(idxEventHandlers.Add(eventJsCode))
				elementIdx := strconv.Itoa(elementIdentifiers.Add(getNodeIdentifier(child)))
				idxEvents.Add(
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
			nodeScript.Attributes.Set("data-syntax-s", "")
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
			anchorScript.Attributes.Set("data-syntax-s", "")
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
	if !elementIdentifiers.IsEmpty() {
		bjs.WriteString("\n    e: [ /* $elements */")
		for i, id := range elementIdentifiers.ToArray() {
			if i > 0 {
				bjs.WriteString(",")
			}
			bjs.WriteString("\n      '" + id.(string) + "'")
		}
		bjs.WriteString("\n    ],")
	}

	// Attributes used
	if !attributes.IsEmpty() {
		bjs.WriteString("\n    t : [")
		for i, name := range attributes.ToArray() {
			if i > 0 {
				bjs.WriteString(", ")
			}
			bjs.WriteString("'" + name.(string) + "'")
		}
		bjs.WriteString("] /* $attributes */,")
	}

	// Event names
	if !idxEventNames.IsEmpty() {
		bjs.WriteString("\n    o : [")
		for i, name := range idxEventNames.ToArray() {
			if i > 0 {
				bjs.WriteString(", ")
			}
			bjs.WriteString("'" + name.(string) + "'")
		}
		bjs.WriteString("] /* $event_names */,")
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

	// watchers by vars
	if !contextVariables.IsEmpty() {
		bjs.WriteString("\n    v: [ /* $watchers_by_vars */")
		for i, gvar := range contextVariables.ToArray() {
			jsVar := gvar.(*js.Var)
			if i > 0 {
				bjs.WriteString(", ")
			}
			bjs.WriteString("\n      [")
			if watchersIds, exists := watchersByVar[i]; exists {
				for j, watcherId := range watchersIds {
					if j > 0 {
						bjs.WriteString(", ")
					}
					bjs.WriteString(strconv.Itoa(watcherId))
				}
			}
			bjs.WriteString("] /* " + jsVar.JS() + " */")
		}
		bjs.WriteString("\n    ],")
	}

	if !idxEvents.IsEmpty() {
		bjs.WriteString("\n    a : [ /* $events */")
		// _$on(_$event_names[0], _$elements[5], _$event_handlers[1]);
		for _, jsEventHandler := range idxEvents.ToArray() {
			bjs.WriteString("\n      " + jsEventHandler.(string))
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

			elementIdx := strconv.Itoa(elementIdentifiers.Get(getNodeIdentifier(ref.Node)))

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
