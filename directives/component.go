package directives

import (
	"bytes"
	"fmt"
	"github.com/iancoleman/strcase"
	"github.com/syntax-framework/shtml/jsc"
	"github.com/syntax-framework/shtml/sht"
	"github.com/tdewolff/parse/v2"
	"github.com/tdewolff/parse/v2/js"
	"strconv"
	"strings"
)

var errorCompNested = sht.Err(
	"component:nested",
	"It is not allowed for a component to be defined inside another.", "Outer: %s", "Inner: %s",
)

var errorCompStyleSingle = sht.Err(
	"component:style:single",
	"A component can only have a single style element.", "First: %s", "Second: %s",
)

var errorCompStyleLocation = sht.Err(
	"component:style:location",
	"Style element must be an immediate child of the component.", "Component: %s", "Style: %s",
)

var errorCompScriptSingle = sht.Err(
	"component:script:single",
	"A component can only have a single script element.", "First: %s", "Second: %s",
)

var errorCompScriptLocation = sht.Err(
	"component:script:location",
	"Script element must be an immediate child of the component.", "Component: %s", "Script: %s",
)

// Component Responsible for creating components declaratively
//
// @TODO: AssetJavascript directives?
var Component = &sht.Directive{
	Name:       "component",
	Restrict:   sht.ELEMENT,
	Priority:   1000,
	Terminal:   true,
	Transclude: true,
	Compile: func(node *sht.Node, attrs *sht.Attributes, t *sht.Compiler) (methods *sht.DirectiveMethods, err error) {

		// @TODO: Parse include?

		var style *sht.Node
		var script *sht.Node

		t.Transverse(node, func(child *sht.Node) (stop bool) {
			stop = false
			if child == node || child.Type != sht.ElementNode {
				return
			}

			switch child.Data {
			case "component":
				// It is not allowed for a component to be defined inside another
				err = errorCompNested(node.DebugTag(), child.DebugTag())

			case "style":
				if style != nil {
					// a component can only have a single style tag
					err = errorCompStyleSingle(style.DebugTag(), child.DebugTag())

				} else if child.Parent != node {
					// when it has style, it must be an immediate child of the component
					err = errorCompStyleLocation(node.DebugTag(), child.DebugTag())

				} else {
					style = child
				}

			case "script":
				if script != nil {
					// a component can only have a single script tag
					err = errorCompScriptSingle(script.DebugTag(), child.DebugTag())

				} else if child.Parent != node {
					// when it has script, it must be an immediate child of the component
					err = errorCompScriptLocation(node.DebugTag(), child.DebugTag())

				} else {
					script = child
				}
			}

			if err != nil {
				stop = true
				return
			}

			return
		})

		if err != nil {
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

		assetJavascript, err := compileJavascript(node, t, script)
		println(assetJavascript)

		// @TODO: Registrar o componente no contexto de compilação
		//t.RegisterComponent(&sht.Component{
		//
		//})

		// quando possui expr parametro live, expr componente não pode ter transclude
		// Quando um script existir, todos os eventos DOM/AssetJavascript serão substituidos por addEventListener
		return
	},
}

// JsParam details of a JS parameter of this component
type JsParam struct {
	Name        string
	Description string
	Type        string
	Required    bool
}

// AssetJavascript Representa um recurso necessário por um componente
type AssetJavascript struct {
	Code   string
	Params []JsParam
}

var errorCompJsParamRefNotFound = sht.Err(
	"component:js:param:ref:notfound",
	"The referenced parameter does not exist.", `Param: js-param-%s="@%s"`, "Component: %s",
)

var errorCompJsParamInvalidName = sht.Err(
	"component:js:param:name",
	"The parameter name is invalid.", `Param: js-param-%s="@%s"`, "Component: %s",
)

var errorCompJsRedeclaration = sht.Err(
	"component:js:redeclaration",
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

// compileJavascript does all the necessary handling to link the template with javascript
func compileJavascript(node *sht.Node, t *sht.Compiler, script *sht.Node) (asset *AssetJavascript, err error) {

	sequence := &sht.Sequence{Salt: node.Attributes.Get("name")}

	// classe única
	getNodeIdentifier := createNodeIdentifierFunc(sequence)

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
	params, paramsErr := parseNodeParams(node)
	if paramsErr != nil {
		return nil, paramsErr
	}

	jsParams, attrsToRemove := params.jsParams, params.attrsToRemove

	// parse references to elements within the template
	references, refErr := jsc.ParseReferences(node, t, elementIdentifiers)
	if refErr != nil {
		return nil, refErr
	}

	if len(references) > 0 {
		// initialize ref elements class id
		for _, reference := range references {
			elementIdentifiers.Add(getNodeIdentifier(reference.Node))

			// validates reference names against parameter names, avoids double definition of variables
			identifier := reference.Attr.Normalized
			if params.jsParamsByName[identifier] != nil {
				return nil, errorCompJsRedeclaration(identifier, "reference -> js-param", node.DebugTag())
			}
		}
	}

	// initialize the parameters (need to be visible in global scope to be indexed)
	jsBuf := &bytes.Buffer{}
	if len(jsParams) > 0 {
		jsBuf.WriteString("\n    // parameters (define)")
		jsBuf.WriteString("\n    let _$params = $.params;\n")
		for _, jsParam := range jsParams {
			name := jsParam.Name
			jsBuf.WriteString(fmt.Sprintf("    let %s = _$params['%s'];\n", name, name))
		}
		jsBuf.WriteString("\n    // parameters (watch)")
		jsBuf.WriteString("\n    $.onChangeParams(() => {\n")
		for _, jsParam := range jsParams {
			name := jsParam.Name
			jsBuf.WriteString(fmt.Sprintf("      %s = _$params['%s'];\n", name, name))
		}
		jsBuf.WriteString("    });\n")
	}

	jsSource := jsBuf.String()
	//jsSource := ""

	// @TODO: Mapear todos os watches que na verdade são estáticos (a variável nunca sofre alteração pelo js)
	// Nestes casos, exibir warning para desenvolvedor?

	if script != nil {
		if script.FirstChild != nil {
			// original source code
			jsSource = jsSource + script.FirstChild.Data
		}

		// remove script from render
		script.Remove()
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
	expressionsErr := (&jsc.ExpressionsParser{
		Node:               node,
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
	t.Transverse(node, func(child *sht.Node) (stop bool) {
		stop = false
		if child == node || child.Type != sht.ElementNode {
			return
		}

		// @TODO: Quando child é um Component registrado, faz expr processamento adequado
		// isComponent := false

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

					// resolve references to global scope (component <script> source code and js-param-*)
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
								if isDeclared, _ := jsc.IsDeclaredOnScope(jsVar, contextAstScope); isDeclared {
									// is a custom javascript function or "js-param-name"
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
							if isDeclared, _ := jsc.IsDeclaredOnScope(jsVar, contextAstScope); isDeclared {
								// is a custom javascript variable or "js-param-name"
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
							println("expr que poderia ser?")
							eventJsCode = fmt.Sprintf("(e) => { %s }", eventJsCode)
						}
					}

					//println(eventJsAst)
					// Se existir "js-param-*" ou uma variável com mesmo nome no inicio, não processa
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
	bjs.WriteString(fmt.Sprintf("STX.r('%s', function (STX) {\n", node.Data))
	// component constants
	if script != nil {
		bjs.WriteString(fmt.Sprintf(`  const _$line = %d;`, script.Line))
	} else {
		bjs.WriteString(fmt.Sprintf(`  const _$line = %d;`, node.Line))
	}
	bjs.WriteRune('\n')
	bjs.WriteString(fmt.Sprintf(`  const _$file = "%s";`, node.File))
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

	// identifies which watches are candidates for replay when a variable changes
	// [ $varIndex = [$watchIndexA, $watchIndexB]]
	//watchersByVar := map[int][]int{}

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
		if jsc.ComponentFieldsMap[varName] == true {
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

			// remove attribute from node (to not be rendered anymore)
			ref.Node.Attributes.Remove(ref.Attr)
		}
	}

	// see https://hexdocs.pm/phoenix_live_view/bindings.html
	// Inicializa os eventos desse componente
	// Se expr evento for

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
	for _, attr := range attrsToRemove {
		node.Attributes.Remove(attr)
	}

	println(bjs.String())

	jsCode := &AssetJavascript{
		Code: bjs.String(),
		//Params: jsParams,
	}

	return jsCode, nil
}

// createNodeIdentifierFunc creates a function that when invoked, adds a class so that a node can be identified
func createNodeIdentifierFunc(sequence *sht.Sequence) func(node *sht.Node) string {
	cache := map[*sht.Node]string{}

	return func(other *sht.Node) string {
		if identifier, exists := cache[other]; exists {
			return identifier
		}

		identifier := other.Attributes.Get("id")
		if identifier != "" {
			identifier = "#" + identifier
			cache[other] = identifier
		} else {
			// generate a class "identifier" (unique class)
			identifier = sequence.NextHash("_")
			cache[other] = identifier
			other.Attributes.AddClass(identifier)
		}
		return identifier
	}
}

type componentNodeParams struct {
	goParams       []sht.ComponentParam // server params
	jsParams       []sht.ComponentParam // javascript params
	attrsToRemove  []*sht.Attribute
	goParamsByName map[string]*sht.ComponentParam
	jsParamsByName map[string]*sht.ComponentParam
}

// parseNodeParams Processes the parameters of a component
func parseNodeParams(node *sht.Node) (*componentNodeParams, error) {
	// parse attributes
	var goParams []sht.ComponentParam // server params
	var jsParams []sht.ComponentParam // javascript params
	var attrsToRemove []*sht.Attribute
	goParamsByName := map[string]*sht.ComponentParam{}
	jsParamsByName := map[string]*sht.ComponentParam{}

	refParamsValueOrig := map[string]string{}
	jsParamsToResolve := map[string]*sht.ComponentParam{}

	for name, attr := range node.Attributes.Map {
		isParam, isJsParam, paramName := strings.HasPrefix(name, "param-"), false, ""
		if isParam {
			paramName = strcase.ToLowerCamel(strings.Replace(name, "param-", "", 1))
		} else {
			isJsParam = strings.HasPrefix(name, "js-param-")
			if isJsParam {
				paramName = strcase.ToLowerCamel(strings.Replace(name, "js-param-", "", 1))
			}
		}

		if isParam || isJsParam {
			if paramName != "" {
				param := sht.ComponentParam{
					Name:     paramName,
					Required: true,
					IsJs:     isJsParam,
				}
				paramTypeName := strings.TrimSpace(attr.Value)
				if strings.HasPrefix(paramTypeName, "?") {
					param.Required = false
					paramTypeName = paramTypeName[1:]
				}

				if isJsParam && strings.HasPrefix(paramTypeName, "@") {
					// is exposing a parameter to JS, by reference
					// Ex. <component param-name="string" js-param-name="@name" />
					referenceName := strcase.ToLowerCamel(paramTypeName[1:])
					refParamsValueOrig[referenceName] = paramTypeName[1:]

					serverParam, serverParamFound := goParamsByName[referenceName]
					if serverParamFound {
						param.Type = serverParam.Type
						param.TypeName = serverParam.TypeName
						param.Reference = serverParam
					} else {
						// will solve further below
						jsParamsToResolve[referenceName] = &param
					}
				} else {
					paramType, paramTypeFound := sht.ParamTypeNames[paramTypeName]
					if !paramTypeFound {
						paramType = sht.ParamTypeUnknown
					}

					param.Type = paramType
					param.TypeName = paramTypeName
				}

				if isJsParam {
					// @TODO: Valid param types (string, number, bool, array, object)
					// param name is valid?
					if _, isInvalid := jsc.InvalidParamsAndRefs[paramName]; isInvalid || strings.HasPrefix(paramName, "_$") {
						return nil, errorCompJsParamInvalidName(strcase.ToKebab(paramName), node.DebugTag())
					}

					jsParams = append(jsParams, param)
					jsParamsByName[paramName] = &param
				} else {
					goParams = append(goParams, param)
					goParamsByName[paramName] = &param
				}
			}

			attrsToRemove = append(attrsToRemove, attr)
		}
	}

	// resolve jsParams reference
	for referenceName, jsParam := range jsParamsToResolve {
		serverParam, serverParamFound := goParamsByName[referenceName]
		if serverParamFound {
			jsParam.Type = serverParam.Type
			jsParam.TypeName = serverParam.TypeName
			jsParam.Reference = serverParam
		} else {
			// Error, is referencing a non-existent parameter
			return nil, errorCompJsParamRefNotFound(
				strcase.ToKebab(jsParam.Name), strcase.ToKebab(refParamsValueOrig[referenceName]), node.DebugTag(),
			)
		}
	}

	return &componentNodeParams{
		goParams:       goParams,
		jsParams:       jsParams,
		attrsToRemove:  attrsToRemove,
		goParamsByName: goParamsByName,
		jsParamsByName: jsParamsByName,
	}, nil
}
