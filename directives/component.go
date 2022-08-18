package directives

import (
	"bytes"
	"fmt"
	"github.com/iancoleman/strcase"
	"github.com/syntax-framework/shtml/sht"
	"github.com/tdewolff/parse/v2"
	"github.com/tdewolff/parse/v2/js"
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
	"component:js:ref",
	"There are two elements with the same JS reference.", "First: %s", "Second: %s",
)

var errorCompParamJsInvalidReference = sht.Err(
	"component:param:js:ref",
	"The referenced parameter does not exist.", `Param: js-param-%s="@%s"`, "Component: %s",
)

// htmlEventsPush list of events that are enabled by default to push to server
var htmlEventsPush = map[string]bool{
	// Form Event Attributes
	"onblur": true, "onchange": true, "oncontextmenu": true, "onfocus": true, "oninput": true, "oninvalid": true,
	"onreset": true, "onsearch": true, "onselect": true, "onsubmit": true,
	// Mouse Event Attributes
	"onclick": true, "ondblclick": true, "onmousedown": true, "onmousemove": true, "onmouseout": true, "onmouseover": true,
	"onmouseup": true, "onwheel": true,
}

// compileJavascript does all the necessary handling to link the template with javascript
func compileJavascript(node *sht.Node, t *sht.Compiler, script *sht.Node) (asset *Javascript, err error) {

	// parse attributes
	var goParams []sht.ComponentParam // server params
	var jsParams []sht.ComponentParam // javascript params
	var attrsToRemove []*sht.Attribute
	jsParamsToResolve := map[string]*sht.ComponentParam{}
	serverParamsByName := map[string]*sht.ComponentParam{}
	refParamsValueOrig := map[string]string{}

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

					serverParam, serverParamFound := serverParamsByName[referenceName]
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
					jsParams = append(jsParams, param)
				} else {
					goParams = append(goParams, param)
					serverParamsByName[paramName] = &param
				}
			}

			attrsToRemove = append(attrsToRemove, attr)
		}
	}

	// resolve goParams reference
	for referenceName, jsParam := range jsParamsToResolve {
		serverParam, serverParamFound := serverParamsByName[referenceName]
		if serverParamFound {
			jsParam.Type = serverParam.Type
			jsParam.TypeName = serverParam.TypeName
			jsParam.Reference = serverParam
		} else {
			// Error, is referencing a non-existent parameter
			err = errorCompParamJsInvalidReference(
				strcase.ToKebab(jsParam.Name), strcase.ToKebab(refParamsValueOrig[referenceName]), node.DebugTag(),
			)
			break
		}
	}

	if err != nil {
		return
	}

	var jsSource string                 // original source code
	jsDeclarations := map[string]bool{} // all declared variables in global scope

	if script != nil {
		if script.FirstChild != nil {
			jsSource = script.FirstChild.Data

			ast, jsErrr := js.Parse(parse.NewInputString(jsSource), js.Options{})
			if jsErrr != nil {
				// @TODO: Custom error
				err = jsErrr
				return
			}

			// extracts the variables declared in the script's global scope
			for _, v := range ast.BlockStmt.Scope.Declared {
				jsDeclarations[v.String()] = true
			}
		}

		// remove script from render
		script.Remove()
	}

	// jsRefs := map[string]jsRef{}
	hasRef := false
	refVarNodes := map[string]*sht.Node{}
	refVarAttrs := map[string]*sht.Attribute{}

	// Parse content
	t.Transverse(node, func(child *sht.Node) (stop bool) {
		stop = false
		if child == node || child.Type != sht.ElementNode {
			return
		}

		// @TODO: Quando child é um Component registrado, faz o processamento adequado
		// isComponent := false

		for attrNameNormalized, attr := range child.Attributes.Map {
			if attrNameNormalized == "ref" {
				// is a reference that can be used in JS
				refVar := strcase.ToLowerCamel(attr.Value)
				if refVar != "" {
					firstNode, exists := refVarNodes[refVar]
					if exists {
						err = errorCompJsRefDuplicated(firstNode.DebugTag(), child.DebugTag())
						break
					}
					hasRef = true
					refVarNodes[refVar] = child
					refVarAttrs[refVar] = attr
				}
			} else if strings.HasPrefix(attrNameNormalized, "on") {
				// html events
				content := strings.TrimSpace(attr.Value)
				if strings.HasPrefix(content, "js:") {
					// If it has the prefix "js:", it does not process
					//    Ex. <button onclick="js: doSomeThing && doOtherThing">
					content = strings.Replace(content, "js:", "", 1)
				} else {
					ast, jsErrr := js.Parse(parse.NewInputString(content), js.Options{})
					if jsErrr != nil {
						// @TODO: Custom error or Warning
						err = jsErrr
						break
					}

					// @TODO: Só processa de ast.BlockStmt.List.Len == 1
					for _, stmt := range ast.BlockStmt.List {
						if exprStmt, isExprStmt := stmt.(*js.ExprStmt); isExprStmt {
							if callExpr, isCallExpr := exprStmt.Value.(*js.CallExpr); isCallExpr {
								// someFunc(arg1, arg2, argn)
								fnName := callExpr.X.String()
								// verifica se variável existe
								println(fnName)
							} else if jsVar, isVar := exprStmt.Value.(*js.Var); isVar {
								// someVariable
								fnName := jsVar.String()
								println(fnName)
							}
						}
					}
					println(ast)
					// Se existir "js-param-*" ou uma variável com mesmo nome no inicio, não processa
					//    <button onclick="onClick()">
					//    <button onclick="callback">
					// Se tiver o padrão "NOME_EVENTO" ou "NOME_EVENTO(...)" ou  "push(NOME_EVENTO, ...)", considera que é push para o server
					//    <button onclick="increment">
					//    <button onclick="increment(count, time, e.MouseX)">
					//    <button onclick="push('increment', count, time, e.MouseX)" data-ref="mySpan">
					// Para todos os outros casos, não processa
				}
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
	js := &bytes.Buffer{}
	js.WriteString(fmt.Sprintf("STX.r('%s', function (STX, $, $export, $params) {", node.Data))
	js.WriteString("\n  // constants\n")
	js.WriteString(fmt.Sprintf(`  const $file = "%s";`, node.File))
	js.WriteRune('\n')
	if script != nil {
		js.WriteString(fmt.Sprintf(`  const $line = %d;`, script.Line))
	} else {
		js.WriteString(fmt.Sprintf(`  const $line = %d;`, node.Line))
	}
	js.WriteRune('\n')

	if hasRef {
		js.WriteString("\n  // define refs\n")
		for refVar, _ := range refVarNodes {
			js.WriteString(fmt.Sprintf(`  let %s;`, refVar))
			js.WriteRune('\n')
		}
	}

	// initialize the parameters
	if len(jsParams) > 0 {
		js.WriteString("\n  // initialize parameters\n")
		for _, jsParam := range jsParams {
			js.WriteString(fmt.Sprintf(`  let %s = $params['%s'];`, jsParam.Name, jsParam.Name))
			js.WriteRune('\n')
		}
	}

	// component code
	if jsSource != "" {
		js.WriteString("\n  // START\n")
		js.WriteString(jsSource)
		js.WriteString("\n  // END\n")
	}

	js.WriteString("\n  // register this instance")
	js.WriteString("\n  $export(() => api, () => onInit, () => onUpdate, () => onExit)\n")

	// see https://hexdocs.pm/phoenix_live_view/bindings.html
	// Inicializa os eventos desse componente
	// Se o evento for
	// $('span.cba51d52w').on('click', (e) => onClick()) // span 1
	// $('span.cba51d525').on('click', (e) => callback()) // span 2

	if hasRef {
		js.WriteString("\n  // initialize refs\n")
		for refVar, refNode := range refVarNodes {
			isComponent := false

			className := "_ref_" + sht.HashXXH64(refVar)
			refNode.Attributes.AddClass(className)

			// if is component
			if isComponent {
				js.WriteString(fmt.Sprintf(`  %s = STX.init('otherComponent', $('.%s'), {callback: () => fazAlgumaCoisa()})`, refVar, className))
			} else {
				js.WriteString(fmt.Sprintf(`  %s = $('.%s');`, refVar, className))
			}
			js.WriteRune('\n')

			// remove attribute from node (to not be rendered anymore)
			refNode.Attributes.Remove(refVarAttrs[refVar])
		}
	}

	// close
	js.WriteString("})")

	// to no longer be rendered
	for _, attr := range attrsToRemove {
		node.Attributes.Remove(attr)
	}

	println(js.String())

	jsCode := &Javascript{
		Code: js.String(),
		//Params: jsParams,
	}

	return jsCode, nil
}
