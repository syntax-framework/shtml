package directives

import (
  "bytes"
  "fmt"
  "github.com/iancoleman/strcase"
  "github.com/syntax-framework/shtml/sht"
  "strings"
)

var errorCompNested = sht.Err(
  "component:nested",
  "It is not allowed for a component to be defined inside another.", "Outer: %s", "Inner: %s",
)

var errorCompStyleSingle = sht.Err(
  "component:style:single",
  "A component can only have a single style element.", "1#: %s", "2#: %s",
)

var errorCompStyleLocation = sht.Err(
  "component:style:location",
  "Style element must be an immediate child of the component.", "Component: %s", "Style: %s",
)

var errorCompScriptSingle = sht.Err(
  "component:script:single",
  "A component can only have a single script element.", "1#: %s", "2#: %s",
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
    goAttrs := map[string]*sht.Attribute{}

    for name, attr := range attrs.Map {
      if strings.HasPrefix(name, "param-") {
        goAttrs[strings.Replace(name, "param-", "", 1)] = attr
        delete(attrs.Map, name)
      }
    }

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
  "component:js:location",
  "There are two elements with the same JS reference", "1#: %s", "2#: %s",
)

// compileJavascript does all the necessary handling to link the template with javascript
func compileJavascript(node *sht.Node, t *sht.Compiler, script *sht.Node) (asset *Javascript, err error) {

  // parse attributes
  var params []JsParam

  for name, attr := range node.Attributes.Map {
    if strings.HasPrefix(name, "js-param-") {
      paramName := strcase.ToLowerCamel(strings.Replace(name, "js-param-", "", 1))
      if paramName != "" {
        param := JsParam{
          Name:     paramName,
          Required: true,
        }
        paramType := strings.TrimSpace(attr.Value)
        if strings.HasPrefix(paramType, "?") {
          param.Required = false
          paramType = paramType[1:]
        }
        param.Type = paramType

        params = append(params, param)
      }
      node.Attributes.Remove(attr)
    }
  }

  // remove script from render
  if script != nil {
    script.Remove()
  }

  //jsRefs := map[string]jsRef{}
  hasRef := false
  refVarNodes := map[string]*sht.Node{}
  refVarAttrs := map[string]*sht.Attribute{}

  // Parse content
  t.Transverse(node, func(child *sht.Node) (stop bool) {
    stop = false
    if child == node || child.Type != sht.ElementNode {
      return
    }

    for nameN, attr := range child.Attributes.Map {
      // is a reference that can be used in JS
      if nameN == "ref" {
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
      }
    }

    if err != nil {
      stop = true
      return
    }

    return true
  })

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
  // let variavel = $params['variavel'];
  // let callback = $params['callback'];

  // component code
  if script != nil {
    js.WriteString("\n  // START\n")
    js.WriteString(script.FirstChild.Data)
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

  //fmt.Println(node.Render())

  jsCode := &Javascript{
    Code:   js.String(),
    Params: params,
  }

  return jsCode, nil
}
