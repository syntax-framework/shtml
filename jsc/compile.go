package jsc

import (
	"bytes"
	"fmt"
	"github.com/syntax-framework/shtml/cmn"
	"github.com/syntax-framework/shtml/sht"
	"github.com/tdewolff/parse/v2"
	"github.com/tdewolff/parse/v2/js"
	"strconv"
)

var errorCompJsRedeclaration = cmn.Err(
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
func Compile(nodeParent *sht.Node, nodeScript *sht.Node, sequenceGlobal *sht.Sequence) (asset *Javascript, err error) {

	if nodeScript != nil && nodeScript.Attributes.Get("src") != "" {
		// jsc only works with inline scripts, any external script must be handled by specialized module
		return nil, err
	}

	sequence := &sht.Sequence{}
	nodeParentIsComponent := false
	if nodeParent.Data == "component" {
		nodeParentIsComponent = true
		sequence.Salt = nodeParent.Attributes.Get("name")
	} else {
		if nodeScript != nil && nodeScript.FirstChild != nil {
			sequence.Salt = sht.HashXXH64([]byte(nodeScript.FirstChild.Data))
		} else {
			sequence.Salt = sequenceGlobal.NextHash()
		}
	}

	// Unique node identifier (#id or data-syntax-id)
	getNodeIdentifier := sht.CreateNodeIdentifier(sequence)

	// All script global variables being used
	contextVariables := &cmn.IndexedSet{}

	// All expressions created in the code are idexed, to allow removing duplicates
	// JS: Array<key: expressionIndex, value: Function>
	expressions := &cmn.IndexedSet{}

	// The elements used by the script and which therefore must be referenced in the code
	// JS: Array<key: elementIndex, value: string(#id|data-syntax-id)>
	elements := &cmn.IndexedSet{}

	// All html attributeNames referencied in writers (Ex. src, href, value)
	// JS: Array<key: attributeIndex, value: string>
	attributeNames := &cmn.IndexedSet{}

	// All html events referenced in code (Ex. click, change)
	// JS: Array<key: eventNameIndex, value: string>
	eventNames := &cmn.IndexedSet{}

	// Mapping of html events that are fired in code
	// JS: Array<[elementIndex, eventNameIndex, expressionIndex]>
	events := &cmn.IndexedSet{}

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
	writers := &cmn.IndexedSet{}

	// All watches. Represent expressions that will react when a variable changes.
	// JS: Array<key: _, value: [type, variableIndex, expressionIndex|writerIndex]>
	//    type 0 = action(expressionIndex)
	//    type 1 = schedule(writerIndex)
	watchers := &cmn.IndexedSet{}

	// parse component params (<element param-name="type" client-param-name="type">)
	var componentParams *NodeComponentParams
	if nodeParentIsComponent {
		if componentParams, err = ParseComponentParams(nodeParent); err != nil {
			return nil, err
		}
	}

	// parse references to elements within the template (<element ref="myJsVariable">)
	references, referenceErr := ParseReferences(nodeParent)
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

		//
		//if nodeParentIsComponent {
		//	// when component, remove script from render
		//   nodeScript.Remove()
		//} else {
		//	// for page scripts, remove only content, will be used by the framework to initialize this content
		//	nodeScript.Data = "s"
		//	nodeScript.FirstChild = nil
		//	nodeScript.LastChild = nil
		//	nodeScript.Attributes.set("hidden", "hidden")
		//}
	}

	contextJsAst, contextJsAstErr := js.Parse(parse.NewInputString(jsSource), js.Options{})
	if contextJsAstErr != nil {
		return nil, contextJsAstErr // @TODO: Custom error or Warning
	}
	contextAstScope := &contextJsAst.BlockStmt.Scope

	// Parse all exports, change by varibles
	export, exportDefault := parseExportApi(contextJsAst)

	// @TODO: fork the project https://github.com/tdewolff/parse/tree/master/js and add feature to keep original formatting
	jsSource = contextJsAst.JS()

	expressionsErr := (&ExpressionsParser{
		Node:               nodeParent,
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
		NodeIdentifierFunc: getNodeIdentifier,
	}).Parse()
	if expressionsErr != nil {
		return nil, expressionsErr // @TODO: Custom error or Warning
	}

	// write the component JS
	bjs := &bytes.Buffer{}

	if nodeParentIsComponent {
		bjs.WriteString("STX.c('" + nodeParent.Data + "', function (STX) {\n")
	} else {
		anchorId := getNodeIdentifier(nodeParent)
		//if nodeScript != nil {
		//} else {
		//  anchor := &sht.NodeTest{
		//    Data:       "s",
		//    DataAtom:   atom.Script,
		//    File:       nodeParent.File,
		//    Line:       nodeParent.Line,
		//    Column:     nodeParent.Column,
		//    Attributes: &sht.Attributes{Map: map[string]*sht.Attribute{}},
		//  }
		//  anchor.Attributes.set("hidden", "hidden")
		//  nodeParent.AppendChild(anchor)
		//  anchorId = getNodeIdentifier(anchor)
		//}
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

	bjs.WriteString("\n  return {\n    f: _$file,\n    l: _$line,")

	// Elements class ids
	if !elements.IsEmpty() {
		bjs.WriteString("\n    // Elements")
		bjs.WriteString("\n    //   Array<key: elementIndex, value: string(#id|data-syntax-id)>")
		bjs.WriteString("\n    e: [")
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
		bjs.WriteString("\n    // Attribute Names")
		bjs.WriteString("\n    //   Array<key: attributeIndex, value: string>")
		bjs.WriteString("\n    a : [")
		for i, name := range attributeNames.ToArray() {
			if i > 0 {
				bjs.WriteString(", ")
			}
			bjs.WriteString("'" + name.(string) + "'")
		}
		bjs.WriteString("],")
	}

	// Event names
	if !eventNames.IsEmpty() {
		bjs.WriteString("\n    // Event Names")
		bjs.WriteString("\n    //   Array<key: eventNameIndex, value: string>")
		bjs.WriteString("\n    n : [")
		for i, name := range eventNames.ToArray() {
			if i > 0 {
				bjs.WriteString(", ")
			}
			bjs.WriteString("'" + name.(string) + "'")
		}
		bjs.WriteString("],")
	}

	if !events.IsEmpty() {
		bjs.WriteString("\n    // Events")
		bjs.WriteString("\n    //   Array<[elementIndex, eventNameIndex, expressionIndex]>")
		bjs.WriteString("\n    o : [")
		// _$on(_$event_names[0], _$elements[5], _$event_handlers[1]);
		for j, jsEventHandler := range events.ToArray() {
			if j > 0 {
				bjs.WriteString(", ")
			}
			bjs.WriteString("\n      " + jsEventHandler.(string))
		}
		bjs.WriteString("\n    ],")
	}

	// writers
	if !writers.IsEmpty() {
		bjs.WriteString("\n    // Writers")
		bjs.WriteString("\n    //   Array<key: writerIndex, value: [elementIndex, expressionIndex]>")
		bjs.WriteString("\n    //   Array<key: writerIndex, value: [elementIndex, attributeIndex, expressionIndex]>")
		bjs.WriteString("\n    //   Array<key: writerIndex, value: [elementIndex, attributeIndex, [string, expressionIndex, string, ...]]>")
		bjs.WriteString("\n    t : [")
		for i, writer := range writers.ToArray() {
			if i > 0 {
				bjs.WriteString(",")
			}
			bjs.WriteString("\n      " + writer.(string))
		}
		bjs.WriteString("\n    ],")
	}

	// watchers
	if !watchers.IsEmpty() {
		bjs.WriteString("\n    // Watchers")
		bjs.WriteString("\n    //   Array<key: _, value: [type, variableIndex, expressionIndex|writerIndex]>")
		bjs.WriteString("\n    //     type 0 = action(expressionIndex)")
		bjs.WriteString("\n    //     type 1 = schedule(writerIndex)")
		bjs.WriteString("\n    w : [")
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

	// initialize references (need to be visible in global scope to be indexed)
	if len(references) > 0 {
		bjs.WriteString("\n      // References (define)\n")
		for _, reference := range references {
			bjs.WriteString(fmt.Sprintf(`      let %s;`, reference.VarName))
			bjs.WriteRune('\n')
		}
	}

	// component code
	bjs.WriteString("\n      // Component\n")
	if jsSource != "" {
		bjs.WriteString("      " + jsSource + "\n")
	}

	// initialize references
	if len(references) > 0 {
		bjs.WriteString("\n      // References (initialize)\n")
		for _, ref := range references {
			elementIndex := strconv.Itoa(elements.GetIndex(getNodeIdentifier(ref.Node)))
			bjs.WriteString("      " + ref.VarName + " = $(" + elementIndex + ");\n")
		}
	}

	// START - Return instance
	bjs.WriteString("\n      return {")

	// LifeCycle callback
	i := 0
	for _, v := range contextAstScope.Declared {
		varName := v.String()
		if field, exist := ClientLifeCycleMap[varName]; exist {
			if i > 0 {
				bjs.WriteString(",")
			}
			i++
			bjs.WriteString("\n        " + field + " : " + varName)
		}
	}

	// All expressions
	if !expressions.IsEmpty() {
		bjs.WriteString("\n        // Expressions\n        x : [")
		for _, expression := range expressions.ToArray() {
			if i > 0 {
				bjs.WriteString(",")
			}
			i++
			bjs.WriteString("\n          " + expression.(string))
		}
		bjs.WriteString("\n        ],")
	}

	// Exports (component API)
	if exportDefault != nil {
		bjs.WriteString("\n        // API (exports)\n        z : " + exportDefault.JS())
	} else if !export.IsEmpty() {
		bjs.WriteString("\n        // API (exports)\n        z : {")
		for _, expression := range export.ToArray() {
			if i > 0 {
				bjs.WriteString(",")
			}
			i++
			bjs.WriteString("\n          " + expression.(string))
		}
		bjs.WriteString("\n        }")
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

	jsCode := &Javascript{
		Content: bjs.String(),
		//ComponentParams: ClientParams,
	}

	return jsCode, nil
}

// parseExportApi All exports are transformed in the component's API, and can be accessed by the "ref" attribute
//
// All exports from the JS file are collected and made available in the "return" of the instance
// If there is a default export, all other exports will be ignored and the default will be used in the "return"
func parseExportApi(contextJsAst *js.AST) (*cmn.IndexedSet, js.INode) {
	export := &cmn.IndexedSet{}
	var exportDefault js.INode
	for i, stmt := range contextJsAst.BlockStmt.List {
		if jsExportStmt, isExportStmt := stmt.(*js.ExportStmt); isExportStmt {
			if jsExportStmt.Default {
				// export default SOMETHING
				exportDefault = jsExportStmt.Decl
				contextJsAst.BlockStmt.List[i] = &js.EmptyStmt{}
			} else if jsExportStmt.List != nil {
				for _, alias := range jsExportStmt.List {
					binding := string(alias.Binding)
					if alias.Name == nil {
						// export { myVar, myVar2 }
						export.Add(binding + " : " + binding)
					} else {
						// export { myVar as myVarRenamed, myVar2 }
						export.Add(binding + " : " + string(alias.Name))
					}
				}
				// replace `export { myVar , myVar2 }`
				contextJsAst.BlockStmt.List[i] = &js.EmptyStmt{}
			} else if jsExportStmt.Decl != nil {
				if Stmt, isIStmt := jsExportStmt.Decl.(js.IStmt); isIStmt {
					switch Stmt.(type) {
					case *js.VarDecl:
						for _, binding := range Stmt.(*js.VarDecl).List {
							switch binding.Binding.(type) {
							case *js.Var:
								name := binding.Binding.(*js.Var).String()
								export.Add(name + " : " + name)
							}
						}
					case *js.FuncDecl:
						name := Stmt.(*js.FuncDecl).Name.String()
						export.Add(name + " : " + name)
					case *js.ClassDecl:
						name := Stmt.(*js.ClassDecl).Name.String()
						export.Add(name + " : " + name)
					}
					contextJsAst.BlockStmt.List[i] = Stmt
				}
			}
		}
	}
	return export, exportDefault
}
