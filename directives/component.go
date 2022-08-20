package directives

import (
	"bytes"
	"fmt"
	"github.com/iancoleman/strcase"
	"github.com/syntax-framework/shtml/sht"
	"github.com/tdewolff/parse/v2"
	"github.com/tdewolff/parse/v2/js"
	"io"
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
// @TODO: Javascript directives?
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

		jsCode, err := compileJavascript(node, t, script)
		println(jsCode)

		// quando possui o parametro live, o componente não pode ter transclude
		// Quando um script existir, todos os eventos DOM/Javascript serão substituidos por addEventListener
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

// jsRef referencia para elementos e componentes existentes no template
//
// Exemplo:
// <component>
//   <span data-ref="mySpan"></span>
//   <script>
//      mySpan.innerText = "Hello World!"
//   </script>
// </component>
type jsRef struct {
	name      string
	component *sht.Component
}

type Javascript struct {
	Code   string
	Params []JsParam
}

var errorCompJsRefDuplicated = sht.Err(
	"component:js:ref:duplicated",
	"There are two elements with the same JS reference.", "First: %s", "Second: %s",
)

var errorCompJsParamRefNotFound = sht.Err(
	"component:js:param:ref:notfound",
	"The referenced parameter does not exist.", `Param: js-param-%s="@%s"`, "Component: %s",
)

var errorCompJsParamInvalidName = sht.Err(
	"component:js:param:name",
	"The parameter name is invalid.", `Param: js-param-%s="@%s"`, "Component: %s",
)

var errorCompJsRefInvalidName = sht.Err(
	"component:js:ref:name",
	"The reference name is invalid.", "Variable: %s", "Element: %s", "Component: %s",
)

var errorCompJsRedeclaration = sht.Err(
	"component:js:redeclaration",
	"SyntaxError: Identifier has already been declared.", "Identifier: %s", "Context: %s", "Component: %s",
)

func createBoolMap(arr []string) map[string]bool {
	out := map[string]bool{}
	for _, s := range arr {
		out[s] = true
	}
	return out
}

// htmlEventsPush list of events that are enabled by default to push to server
var htmlEventsPush = createBoolMap([]string{
	// Form Event Attributes
	"onblur", "onchange", "oncontextmenu", "onfocus", "oninput", "oninvalid", "onreset", "onsearch", "onselect", "onsubmit",
	// Mouse Event Attributes
	"onclick", "ondblclick", "onmousedown", "onmousemove", "onmouseout", "onmouseover", "onmouseup", "onwheel",
})

// jsComponentFields The methods and attributes that are part of the structure of a JS component
var jsComponentFields = []string{
	// used now
	"api",          // Object - Allows the component to expose an API for external consumption. see ref
	"onMount",      // () => void - A method that runs after initial render and elements have been mounted
	"beforeUpdate", // () => void -
	"afterUpdate",  // () => void -
	"onCleanup",    // () => void - A cleanup method that executes on disposal or recalculation of the current reactive scope.
	"onDestroy",    // () => void - After unmount
	"onConnect",    // () => void - Invoked when the component has connected/reconnected to the server
	"onDisconnect", // () => void - Executed when the component is disconnected from the server
	"onError",      // (err: any) => void - Error handler method that executes when child scope errors
	// for future use
}

// jsInvalidVariables reserved variable names, cannot be used in parameters or references
var jsInvalidVariables = createBoolMap([]string{
	"STX", "$", "$params", "$watches", "$watches_by_var", "$dirty", "$invalidate", "$line", "$file", "push",
	"api", "onMount", "beforeUpdate", "afterUpdate", "onCleanup", "onDestroy", "onConnect", "onDisconnect", "onError",
})

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
//      nível em diante), encapsular com o comando de invalidate (_$i(index, currentValue, expression))
var jsWatchInvalidateBlock = `
  // todas as funcoes reativas do template (ex {{time}}, {{count}}, js-hide="count == 0")
  const $watches = [];

  // cada variável possui um indice, essa lista faz o cruzamento entre variavel e watches no formato
  // [ $varIndex = [$watchIndexA, $watchIndexB]]
  const $watches_by_var = [];

  // indices de variáveis que sofreram alteração
  const $dirty_vars = new Set();

  // invalida uma variável e agenda aplicação no próximo tick
  const $invalidate = (index, old, nue) => {
    if (old != old ? nue == nue : old !== nue || ((old && typeof old === 'object') || typeof old === 'function')){
      $dirty.add(index);
      $.schedule($dirty, $watches_by_var, $watches);
    }    
    return ret;
  };
`

type IndexedMap struct {
	size   int
	values map[interface{}]int
}

func (m *IndexedMap) Add(value interface{}) int {
	index, exists := m.values[value]
	if !exists {
		index = m.size
		if m.values == nil {
			m.values = map[interface{}]int{}
		}
		m.values[value] = m.size
		m.size++
	}
	return index
}

func (m *IndexedMap) Get(value interface{}) int {
	index, exists := m.values[value]
	if !exists {
		return -1
	}
	return index
}

func (m *IndexedMap) ToArray() []interface{} {
	arr := make([]interface{}, m.size)
	for value, index := range m.values {
		arr[index] = value
	}
	return arr
}

// Sequence identifier generator utility qeu ensures that all executions have the same result
type Sequence struct {
	salt string
	seq  int
}

func (s *Sequence) NextHash(prefix string) string {
	if prefix == "" {
		prefix = "_"
	}
	s.seq++
	return prefix + sht.HashXXH64(s.salt+strconv.Itoa(s.seq))
}

func (s *Sequence) NextInt() int {
	s.seq++
	return s.seq
}

type jsInterpolation struct {
	expression    string
	isSafeSignal  bool
	isFullContent bool
}

// _astVisitorAddWatch visita cada elemento de um javascript compilado para adicionar o watch (_$W) em todos os setters
// https://www.w3schools.com/js/js_assignment.asp
type _astVisitorAddWatch struct {
}

func (i *_astVisitorAddWatch) Enter(node js.INode) js.IVisitor {
	if exprStmt, isExprStmt := node.(*js.ExprStmt); isExprStmt {
		s := exprStmt.JS()
		println(s + " || " + exprStmt.String())
	}
	return i
}

func (i *_astVisitorAddWatch) Exit(node js.INode) {

}

// jsAstVisitorEnterFunc use function as AST Visitor
//
// Each INode encountered by `Walk` is passed to func, children nodes will be ignored if return false
type jsAstVisitorEnterFunc func(node js.INode) (visitChildren bool)

func (f jsAstVisitorEnterFunc) Enter(node js.INode) js.IVisitor {
	if f(node) {
		return f
	}
	return nil
}

func (f jsAstVisitorEnterFunc) Exit(node js.INode) {
}

// eventVariableScope escopo usado para referenciar a variável e. Ex. (e) => { myCallback(e); }
var eventVariableScope *js.Scope

func init() {
	ast, err := js.Parse(parse.NewInputString("let e;"), js.Options{})
	if err != nil {
		panic(err)
	}
	eventVariableScope = &ast.BlockStmt.Scope
}

// isDeclaredOnScope check if this expression is declared on specified scope
func isDeclaredOnScope(expr *js.Var, scope *js.Scope) (bool, *js.Var) {
	for _, d := range scope.Declared {
		if d == expr {
			return true, d
		}

		if d == expr.Link {
			return true, expr.Link
		}
	}
	return false, nil
}

// compileJavascript does all the necessary handling to link the template with javascript
func compileJavascript(node *sht.Node, t *sht.Compiler, script *sht.Node) (asset *Javascript, err error) {

	sequence := &Sequence{}

	// classe única
	getNodeIdClass := createNodeIdentifierFunc(node)

	// parse component params
	params, paramsErr := parseNodeParams(node)
	if paramsErr != nil {
		return nil, paramsErr
	}

	jsParams, attrsToRemove := params.jsParams, params.attrsToRemove

	// parse references to elements within the template
	hasRef, refVarNodes, refVarAttrs, refErr := jsParseReferences(node, t)
	if refErr != nil {
		return nil, refErr
	}

	if hasRef {
		for _, attribute := range refVarAttrs {
			// validates reference names against parameter names, avoids double definition of variables
			identifier := attribute.Normalized
			if params.jsParamsByName[identifier] != nil {
				return nil, errorCompJsRedeclaration(identifier, "reference -> js-param", node.DebugTag())
			}
		}
	}

	// initialize the parameters (need to be visible in global scope to be indexed)
	jsBuf := &bytes.Buffer{}
	if len(jsParams) > 0 {
		jsBuf.WriteString("\n    // parameters (define)\n")
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

	//jsSource := jsBuf.String()

	jsSource := ""

	jsDeclarations := map[string]bool{} // all declared variables in script global scope

	//
	//  Obs. Não é necessário interceptar as atribuições de nível global pois entende-se que essa atribuição está sendo
	//  realizada na instanciação do componente, portanto, os observers já irão receber o valor correto neste momento
	// @TODO: parse content to get all watch variables (Ex. {{time}}, {{count}}, <element js-hide="count == 0">)

	// All expressions created in the code are idexed, to allow removing duplicates
	jsIndexedExpressions := &IndexedMap{}

	// all script global variables being used
	jsIndexedUsedGlobalVars := &IndexedMap{}

	// All watches. Represent expressions that will react when a variable changes.
	//    - Text interpolation block
	//      - <element>${value} #{value} ${value + other}</element>
	//      - <element class="class-a ${myVar == 3 ? 'new-class' : ''}" />
	//    - js directives or control blocks
	//      - <element if="${myVar == 3}" />
	//      - <if cond="${myVar == 3}">
	jsIndexedWatches := &IndexedMap{}

	// identifies which watches are candidates for replay when a variable changes
	// [ $varIndex = [$watchIndexA, $watchIndexB]]
	jsIndexedWatchersByVar := map[int][]int{}

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

	globalJsAst, globalJsAstErr := js.Parse(parse.NewInputString(jsSource), js.Options{})
	if globalJsAstErr != nil {
		err = globalJsAstErr // @TODO: Custom error or Warning
		return
	}

	globalJsAstScope := &globalJsAst.BlockStmt.Scope

	// quando encontrar uma atribuição do escopo global, adiciona a invocação de _$D(changedVarIndex)

	// @TODO: Problema, como fazer alteração no JS mantendo a formataçao e comentários original?
	// fazer fork do projeto https://github.com/tdewolff/parse/tree/master/js e adicionar feature para manter
	// formatação original (e comentários)
	//println(globalJsAst.JS())

	v := &_astVisitorAddWatch{}
	js.Walk(v, globalJsAst)

	// extracts the variables declared in the script's global scope
	// @TODO: REMOVER?
	for _, v := range globalJsAstScope.Declared {
		//jsIndexedUsedGlobalVars.Add(v)
		//jsIndexedVarName.Add(v.String())
		jsDeclarations[v.String()] = true
	}

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
	//  Algoritmo de Watch.
	//
	//  1 - Mapear todas as expressoes $watch
	//  2 - Para cada expressão, mapear suas variáveis observáveis, associando um índice a cada uma
	//  3 - Para cada expressão de atribuição de valor de variáveis observáveis que ocorram foram do escopo global (segundo
	//      nível em diante), encapsular com o comando de invalidate (_$i(index, currentValue, watcher))
	//
	// JAVASCRIPT INTERPOLATION ( ${value} or  #{value} )
	//
	// <element attribute="${return value}">
	// <element attribute="xpto ${escape safe}">
	// <element attribute="xpto #{escape unsafe}">
	// <element attribute="#{escape unsafe}">
	// <element>${escape safe}</element>
	// <element>#{escape unsafe}</element>

	// 1 - Mapear todas as expressoes $watch
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
			innerText, interpolations, textInterErr := InterpolateJs(child.Data, sequence)
			if textInterErr != nil {
				err = textInterErr
				stop = true
				return
			}
			if interpolations == nil {
				//content has no js expressions (${value} or #{value})
				return
			}

			for id, interpolation := range interpolations {
				// replace biding location by <embed hidden id="_$i_xxx">
				innerText = strings.Replace(innerText, id, `<embed hidden id="`+id+`">`, 1)

				// eventJsCode = "(e) => { " + eventJsCode + " }"
				interpolationJs := interpolation.expression
				interpolationJsAst, interpolationJsAstErr := js.Parse(parse.NewInputString(interpolationJs), js.Options{})
				if interpolationJsAstErr != nil {
					err = interpolationJsAstErr // @TODO: Custom error or Warning
					break
				}

				// jsIndexedUsedGlobalVars
				interpolationJsAstScope := interpolationJsAst.BlockStmt.Scope

				// resolve references to global scope (component <script> source code and js-param-*)
				undeclaredBackup := globalJsAstScope.Undeclared
				interpolationJsAstScope.Parent = globalJsAstScope
				interpolationJsAstScope.HoistUndeclared()
				globalJsAstScope.Undeclared = undeclaredBackup

				interpolationJs = interpolationJsAst.JS()
				if strings.HasSuffix(interpolationJs, "; ") {
					interpolationJs = interpolationJs[:len(interpolationJs)-2]
				}
				if interpolation.isSafeSignal {
					// <element>${escape safe}</element>
					interpolationJs = "$.e(" + interpolationJs + ")"
				} else {
					// <element>#{escape unsafe}</element>
					interpolationJs = "(" + interpolationJs + ")"
				}

				expressionId := jsIndexedExpressions.Add(
					// identical expressions are reused throughout the code
					//
					// $e.push(
					//    () => { return $.e(count); },
					//	  () => { return (time); },
					//	  () => { return $.e(x + y); }
					// )
					"() => { return " + interpolationJs + "; }",
				)

				watcherIndex := jsIndexedWatches.Add(
					// watchers
					// binding to element watcher - $.b(elementId, expressionIndex)
					//
					// $.b('_$i_b7b41276360564d4', 0) // 0
					// $.b('_$i_6021b5621680598b', 1) // 1
					// $.b('_$i_26167c2af5162ca4', 2) // 2
					// $.b('_$i_913914322ca46b89', 0) // 3
					// $.b('_$i_6a81b47405b648ed', 1) // 4
					"$.b('" + id + "', " + strconv.Itoa(expressionId) + ")",
				)

				js.Walk(jsAstVisitorEnterFunc(func(node js.INode) bool {
					if jsVar, isVar := node.(*js.Var); isVar {
						if isDeclared, jsVarGlobal := isDeclaredOnScope(jsVar, globalJsAstScope); isDeclared {
							// associate watcher to variable
							varIndex := jsIndexedUsedGlobalVars.Add(jsVarGlobal)
							jsIndexedWatchersByVar[varIndex] = append(jsIndexedWatchersByVar[varIndex], watcherIndex)
						}
					}
					return true
				}), interpolationJsAst)
			}
		} else if child.Type == sht.ElementNode {
			// busca interpolação nos atributos
			//for attrNameNormalized, attr := range child.Attributes.Map {
			//
			//}
		}

		// para permitir o transclude e não perder o escopo de execução do javascript:
		// 1. Interpolação em texto deve ser feito criando um node html temporário e removendo-o em tempo de execução.
		// 2. Interpolação em atributos, deve-se adicionar um identificador no atributo e, em tempo de execução, verificar
		// se houve mudança do template do atributo.
		// Obs. Se algum marcador tiver sido removido, não registrar o $watch
		return
	})

	// Html events
	//
	// Source: <element onclick="onClickFn(e.MouseX)">)
	// Transpiled to JS: (e) => { onClickFn(e.MouseX) }
	jsEventHandlers := []string{}

	// Parse content
	t.Transverse(node, func(child *sht.Node) (stop bool) {
		stop = false
		if child == node || child.Type != sht.ElementNode {
			return
		}

		// @TODO: Quando child é um Component registrado, faz o processamento adequado
		// isComponent := false

		for attrNameNormalized, attr := range child.Attributes.Map {
			if strings.HasPrefix(attrNameNormalized, "on") {

				// html events
				eventJsCode := strings.TrimSpace(attr.Value)

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

					// resolve reference to event ("e") variable. Ex. "(e) => { myCallback(e.MouseX) }"
					eventJsAstScope.Parent = eventVariableScope
					eventJsAstScope.HoistUndeclared()
					eventVariableScope.Undeclared = nil

					// resolve references to global scope (component <script> source code and js-param-*)
					undeclaredBackup := globalJsAstScope.Undeclared
					eventJsAstScope.Parent = globalJsAstScope
					eventJsAstScope.HoistUndeclared()
					globalJsAstScope.Undeclared = undeclaredBackup

					stmt := eventJsAst.BlockStmt.List[0]
					if exprStmt, isExprStmt := stmt.(*js.ExprStmt); isExprStmt {
						if callExpr, isCallExpr := exprStmt.Value.(*js.CallExpr); isCallExpr {
							// someFunc(arg1, arg2, argn...)
							if jsVar, isVar := callExpr.X.(*js.Var); isVar {
								if isDeclared, _ := isDeclaredOnScope(jsVar, globalJsAstScope); isDeclared {
									// is a custom javascript function or "js-param-name"
									// Ex. <button onclick="onClick()">
									eventJsCode = callExpr.JS()
									//eventJsCode = fmt.Sprintf("(e) => { %s }", eventJsCode)
								} else {
									// considers it to be a remote event call (push)
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
						} else if jsVar, isVar := exprStmt.Value.(*js.Var); isVar {
							// someVariable
							varName := jsVar.String()
							if jsDeclarations[varName] == true /*|| jsParamsByName[varName] != nil*/ {
								// is a custom javascript variable or "js-param-name"
								// Ex. <button onclick="callback"></button>
								eventJsCode = fmt.Sprintf("(e) => { %s(e) }", varName)
							} else {
								// considers it to be a remote event call (push)
								// <button onclick="increment"></button>
								eventJsCode = fmt.Sprintf("(e) => { STX.push('%s', $, e) }", varName)
							}
						} else if arrowFunc, isArrowFunc := exprStmt.Value.(*js.ArrowFunc); isArrowFunc {
							// (e) => doSomething
							eventJsCode = arrowFunc.JS()
						} else {
							println("o que poderia ser?")
							eventJsCode = fmt.Sprintf("(e) => { %s }", eventJsCode)
						}
					}

					println(eventJsAst)
					// Se existir "js-param-*" ou uma variável com mesmo nome no inicio, não processa
					//    <button onclick="onClick()">
					//    <button onclick="callback">
					// Se tiver o padrão "NOME_EVENTO" ou "NOME_EVENTO(...)" ou  "push(NOME_EVENTO, ...)", considera que é push para o server
					//    <button onclick="increment">
					//    <button onclick="increment(count, time, e.MouseX)">
					//    <button onclick="push('increment', count, time, e.MouseX)" data-ref="mySpan">
					// Para todos os outros casos, não processa
				}

				// add event handler
				// $.on('click', '.cba51d52w', (e) => onClick());
				eventName := strings.Replace(attrNameNormalized, "on", "", 1)
				jsEventHandlers = append(
					jsEventHandlers,
					fmt.Sprintf(`    $.on('%s', '.%s', (e) => { %s });`, eventName, getNodeIdClass(child), eventJsCode),
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
	bjs.WriteString(fmt.Sprintf("STX.r('%s', function (STX, $) {", node.Data))
	bjs.WriteString("\n  // constants\n")
	if script != nil {
		bjs.WriteString(fmt.Sprintf(`  const $line = %d;`, script.Line))
	} else {
		bjs.WriteString(fmt.Sprintf(`  const $line = %d;`, node.Line))
	}
	bjs.WriteRune('\n')
	bjs.WriteString(fmt.Sprintf(`  const $file = "%s";`, node.File))
	bjs.WriteRune('\n')

	// to track which variable has changed
	bjs.WriteString(jsWatchInvalidateBlock)

	// initialize references (need to be visible in global scope to be indexed)
	if hasRef {
		bjs.WriteString("\n  // references (define)\n")
		for refVar, _ := range refVarNodes {
			bjs.WriteString(fmt.Sprintf(`  let %s;`, refVar))
			bjs.WriteRune('\n')
		}
	}

	// START
	bjs.WriteString("\n  // component\n  (() => {")
	// component code
	if jsSource != "" {
		bjs.WriteString(jsSource)
	}

	// Configure hooks
	bjs.WriteString("\n    // hooks\n    $.c({ ")
	count := 0
	for _, field := range jsComponentFields {
		if jsDeclarations[field] == true {
			count = count + 1
			if count > 1 {
				bjs.WriteString(", ")
			}
			bjs.WriteString(field)
		}
	}
	bjs.WriteString(" })\n")

	// register watchers

	// all script global variables being used
	//jsIndexedUsedGlobalVars := &IndexedMap{}
	//jsIndexedVarName := &IndexedMap{}

	// All watches. Represent expressions that will react when a variable changes.
	//    - Text interpolation block
	//      - <element>${value} #{value} ${value + other}</element>
	//      - <element class="class-a ${myVar == 3 ? 'new-class' : ''}" />
	//    - js directives or control blocks
	//      - <element if="${myVar == 3}" />
	//      - <if cond="${myVar == 3}">
	//jsIndexedWatches := &IndexedMap{}

	// initialize references
	if hasRef {
		bjs.WriteString("\n    // references (initialize)\n")
		for refVar, refNode := range refVarNodes {
			isComponent := false

			className := "r-" + sht.HashXXH64(refVar)
			refNode.Attributes.AddClass(className)

			// if is component
			if isComponent {
				bjs.WriteString(fmt.Sprintf(`    %s = STX.init('otherComponent', $('.%s'), {callback: () => fazAlgumaCoisa()})`, refVar, className))
			} else {
				bjs.WriteString(fmt.Sprintf(`    %s = $('.%s');`, refVar, className))
			}
			bjs.WriteRune('\n')

			// remove attribute from node (to not be rendered anymore)
			refNode.Attributes.Remove(refVarAttrs[refVar])
		}
	}

	// all expressions
	bjs.WriteString("\n    // expressions")
	bjs.WriteString("\n    $e.push(")
	for _, expression := range jsIndexedExpressions.ToArray() {
		bjs.WriteString("\n      " + expression.(string))
	}
	bjs.WriteString("\n    );\n")

	// watchers
	bjs.WriteString("\n    // watchers")
	for _, watcher := range jsIndexedWatches.ToArray() {
		bjs.WriteString("\n    " + watcher.(string))
	}
	bjs.WriteRune('\n')

	// identifies which watches are candidates for replay when a variable changes
	// [ $varIndex = [$watchIndexA, $watchIndexB]]
	//jsIndexedWatchersByVar := map[int][]int{}

	// watchers by vars
	bjs.WriteString("\n    // watchers by vars")
	bjs.WriteString("\n    $wv = [")
	for i, gvar := range jsIndexedUsedGlobalVars.ToArray() {
		jsVar := gvar.(*js.Var)
		bjs.WriteString("\n      [")
		if watchersIds, exists := jsIndexedWatchersByVar[i]; exists {
			for j, watcherId := range watchersIds {
				if j > 0 {
					bjs.WriteString(", ")
				}
				bjs.WriteString(strconv.Itoa(watcherId))
			}
		}
		bjs.WriteString("], /* " + jsVar.JS() + " */")
	}
	bjs.WriteString("\n    ];\n")

	// initialize component
	bjs.WriteString("\n    // initialize\n    $.i()\n")

	// see https://hexdocs.pm/phoenix_live_view/bindings.html
	// Inicializa os eventos desse componente
	// Se o evento for
	if len(jsEventHandlers) > 0 {
		bjs.WriteString("\n    // events\n")
		// $.on('click', '.cba51d52w', (e) => onClick());
		for _, jsEventHandler := range jsEventHandlers {
			bjs.WriteString(jsEventHandler)
			bjs.WriteRune('\n')
		}
	}

	// END
	bjs.WriteString("  })()\n")

	// close
	bjs.WriteString("})")

	// to no longer be rendered
	for _, attr := range attrsToRemove {
		node.Attributes.Remove(attr)
	}

	println(bjs.String())

	jsCode := &Javascript{
		Code: bjs.String(),
		//Params: jsParams,
	}

	return jsCode, nil
}

// InterpolateJs processa as interpolações javascript em um texto
//
// JAVASCRIPT INTERPOLATION ( ${value} or  #{value} )
//
// <element attribute="${return value}">
// <element attribute="xpto ${escape safe}">
// <element attribute="xpto #{escape unsafe}">
// <element attribute="#{escape unsafe}">
// <element>${escape safe}</element>
// <element>#{escape unsafe}</element>
// #{serverExpressionUnescaped}
//
// @TODO: Filters/Pipe. Ex. ${ myValue | upperCase}
//
// newText, watches, err = InterpolateJs('Hello ${name}!');
// newText == "Hello _j$_i15151ffacb"
// interpolations == {"_j$_i15151ffacb": {expression: "name", isScape: true}}
// exp.Exec({name:'Syntax'}).String() == "Hello Syntax!"
func InterpolateJs(text string, sequence *Sequence) (string, map[string]jsInterpolation, error) {

	if !strings.ContainsRune(text, '{') || !strings.ContainsAny(text, "$#") {
		return text, nil, nil
	}

	// always trim, is still valid html. Syntax has no intention of working with other media
	text = strings.TrimSpace(text)

	interpolations := map[string]jsInterpolation{}

	// Allows you to discover the number of open braces within an expression
	innerBrackets := 0

	// Is processing an expression (started with "!{" or "#{")
	inExpression := false

	//   Safe: ${expr}
	// Unsafe: #{expr}
	isSafeSignal := true

	content := &bytes.Buffer{}

	expressionId := ""
	expressionContent := &bytes.Buffer{}

	reader := strings.NewReader(text)

	for {
		currChar, _, err := reader.ReadRune()
		if err != nil {
			if err == io.EOF {
				break
			} else {
				return text, nil, err
			}
		}
		nextChar, _, err := reader.ReadRune()
		if err != nil && err != io.EOF {
			return text, nil, err
		}

		if err != io.EOF {
			err = reader.UnreadRune()
			if err != nil {
				return text, nil, err
			}
		}

		if !inExpression {
			// ${value} or #{value}
			if (currChar == '$' || currChar == '#') && nextChar == '{' {
				inExpression = true
				isSafeSignal = currChar == '$'

				expressionId = sequence.NextHash("_$i_")
				content.WriteString(expressionId)

				expressionContent = &bytes.Buffer{}
			} else {
				content.WriteRune(currChar)
			}
		} else {
			if currChar == '{' {
				if expressionContent.Len() > 0 {
					innerBrackets++
					expressionContent.WriteRune(currChar)
				}
			} else {
				if currChar == '}' {
					if innerBrackets > 0 {
						innerBrackets--
					} else {
						inExpression = false

						interpolations[expressionId] = jsInterpolation{
							expression:   expressionContent.String(),
							isSafeSignal: isSafeSignal,
						}
						continue
					}
				}
				expressionContent.WriteRune(currChar)
			}
		}
	}

	if inExpression {
		// invalid content, will probably pop JS error
		interpolations[expressionId] = jsInterpolation{
			expression:   expressionContent.String(),
			isSafeSignal: isSafeSignal,
		}
	}

	text = content.String()

	if text == expressionId {
		interpolation := interpolations[expressionId]
		interpolation.isFullContent = true
	}

	return text, interpolations, nil
}

// createNodeIdentifierFunc creates a function that when invoked, adds a class so that a node can be identified
func createNodeIdentifierFunc(node *sht.Node) func(node *sht.Node) string {

	cache := map[*sht.Node]string{}
	sequence := &Sequence{salt: node.Attributes.Get("name")}

	return func(other *sht.Node) string {
		if className, exists := cache[other]; exists {
			return className
		}

		className := sequence.NextHash("_$c_")
		cache[other] = className
		other.Attributes.AddClass(className)
		return className
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
					// param name is valid?
					if _, isInvalid := jsInvalidVariables[paramName]; isInvalid {
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

// jsParseReferences handles references made available to JS (<element ref="myJsVariable">)
func jsParseReferences(node *sht.Node, t *sht.Compiler) (bool, map[string]*sht.Node, map[string]*sht.Attribute, error) {
	// references to elements within the template
	hasRef := false
	refVarNodes := map[string]*sht.Node{}
	refVarAttrs := map[string]*sht.Attribute{}

	var err error

	// Parse content
	t.Transverse(node, func(child *sht.Node) (stop bool) {
		stop = false
		if child == node || child.Type != sht.ElementNode {
			return
		}

		// @TODO: Quando child é um Component registrado, faz o processamento adequado
		// isComponent := false

		if attr := child.Attributes.GetAttribute("ref"); attr != nil {
			// is a reference that can be used in JS
			if refVar := strcase.ToLowerCamel(attr.Value); refVar != "" {

				// ref name is valid?
				if _, isInvalid := jsInvalidVariables[refVar]; isInvalid {
					err = errorCompJsRefInvalidName(refVar, node.DebugTag(), child.DebugTag())
					stop = true
					return
				}

				// ref name is duplicated?
				if firstNode, firstNodeExists := refVarNodes[refVar]; firstNodeExists {
					err = errorCompJsRefDuplicated(firstNode.DebugTag(), child.DebugTag())
					stop = true
					return
				}

				hasRef = true
				refVarNodes[refVar] = child
				refVarAttrs[refVar] = attr
			}
		}

		return
	})

	return hasRef, refVarNodes, refVarAttrs, err
}
