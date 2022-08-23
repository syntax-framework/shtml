package directives

import (
  "github.com/syntax-framework/shtml/sht"
  "testing"
)

func Test_Component_Should_Not_Allow_Nested_Components(t *testing.T) {
  template := `
    <component name="out">
      <div>
        <component name="inner">
          content
        </component>
      </div>
    </component>
  `
  testForErrorCode(t, template, "component:nested")
}

// a component can only have a single style tag
func Test_Component_Should_Not_Allow_Multiple_Style_Element(t *testing.T) {
  template := `
    <component name="test">
      <style>.my-class {color: #FFF}</style>
      <div><style>.my-class-2 {color: #FFF}</style></div>
    </component>
  `
  testForErrorCode(t, template, "component:style:single")
}

// a component can only have a single script tag
func Test_Component_Should_Not_Allow_Multiple_Script_Element(t *testing.T) {
  template := `
    <component name="test">
      <script>console.log("hello")</script>
      <div><script>console.log("world!")</script></div>
    </component>
  `
  testForErrorCode(t, template, "component:script:single")
}

// when it has style, it must be an immediate child of the component
func Test_Component_Style_Element_Must_Be_Immediate_Child(t *testing.T) {
  template := `
    <component name="test">
      <div><style>.my-class-2 {color: #FFF}</style></div>
    </component>
  `
  testForErrorCode(t, template, "component:style:location")
}

// when it has script, it must be an immediate child of the component
func Test_Component_Script_Element_Must_Be_Immediate_Child(t *testing.T) {
  template := `
    <component name="test">
      <div><script>console.log("world!")</script></div>
    </component>
  `
  testForErrorCode(t, template, "component:script:location")
}

// js-param is referencing a non-existent parameter
func Test_Component_JS_Param_Invalid_Reference(t *testing.T) {
  template := `
    <component name="test" param-server-name="string" js-param-name="@server-name-wrong">
      <div></div>
    </component>
  `
  testForErrorCode(t, template, "component:param:js:ref")
}

// Expressions with Side Effect in text interpolation block are not allowed.
func Test_Should_Not_Allow_Side_Effect_in_Interpolation(t *testing.T) {
  var tests = []string{
    `<component name="c"> <span>${ a = --a + 1, b }</span> <script>let a = 0; let b = '';</script></component>`,
    `<component name="c"> <span>${ a++, b }</span> <script>let a = 0; let b = '';</script></component>`,
    `<component name="c"> <span>${ ++a, b }</span> <script>let a = 0; let b = '';</script></component>`,
    `<component name="c"> <span>${ a--, b }</span> <script>let a = 0; let b = '';</script></component>`,
    `<component name="c"> <span>${ --a, b }</span> <script>let a = 0; let b = '';</script></component>`,
    `<component name="c"> <span>${ a = a + 1, b }</span> <script>let a = 0; let b = '';</script></component>`,
    `<component name="c"> <span>${ a = --a + a++, b }</span> <script>let a = 0; let b = '';</script></component>`,
    `<component name="c"> <span class="${ a = --a + a++, b }">text</span> <script>let a = 0; let b = '';</script></component>`,
  }
  for _, tt := range tests {
    testForErrorCode(t, tt, "js:interpolation:sideeffect")
  }
}

func Test_Todo_List(t *testing.T) {
  template := `
    <component name="todo">
      <form onsubmit="handleSubmit()" class="xpto ${inputValue ? 'sujo' : 'limpo'}">
        <label for="listItem">List Item: </label>
        <input id="listItem" value="${inputValue}" onchange="inputValue = e.target.value"/>
      </form>
    
      <p>Todo List:</p>
      <ul>
        <li each="item in todoList">{item}</li>
      </ul>
    
      <script>
        let inputValue = ''
        let todoList = [];
    
        const handleSubmit = (e) => {
          e.preventDefault()
          todoList = [...todoList, inputValue]
          inputValue = ''
        }
      </script>
    </component>
  `
  testForErrorCode(t, template, "component:param:js:ref")
}

func Test_Component(t *testing.T) {

  template := `
    <component
      name="clock"
      element="div"
    
      param-xxx="string"
      param-other-value="?map"
    
      js-param-callback="string"
      js-param-variavel="string"
      js-param-xxx="@other-value"
    
      todo="@TODO: Parametros que deverão ser suportados no futuro"
      controller="RegisteredController"
    >
      <!-- Custom variables -->
      <button onclick="onClick()"></button>
      <button onclick="callback"></button>

      <!-- Server push -->
      <button ref="buttonWithOnClick" onclick="increment"></button>
      <button onclick="increment(count, time, e.MouseX)"></button>    
      <button onclick="push('increment', count, time, e.MouseX)" data-ref="mySpan"></button>

      <!-- JS ignored -->
      <button onclick="doSomeThing && doOtherThing"></button>
      <button onclick="doSomeThing && push('increment', count, time, e.MouseX)"></button>

      <!-- Full content -->
      <button ref="buttonWithManyEvents" onclick="(e) => doSomeThing" onmousedown="increment(count, time, e.MouseX)">
        ${count} #{time}  ${x + y}
      </button>
      <button onclick="function xpto(e) { doSomeThing(e) }"></button>
    
      ${count} #{time}

      <span ref="mySpan2">${ renders = --renders + 1, name }</span>
	    <span>${ renders++, name }</span>
	    <span>${ ++renders, name }</span>
	    <span>${ renders--, name }</span>
	    <span>${ --renders, name }</span>
	    <span>${ renders = renders + 1, name }</span>
	    <span>${ renders = --renders + renders++, name }</span>
    
      <style>
        span {
          font-family: Roboto
        }
      </style>
    
      <script>
        let renders = 0;
        let name = 'alex';

        // https://www.w3schools.com/js/js_assignment.asp
        let x = 33, y = 25;
        x = y;
        x = y;
        x += y;
        x = x + y;
        x -= y;
        x = x - y;
        x *= y;
        x = x * y;
        x /= y;
        x = x / y;
        x %= y;
        x = x % y;
        x <<= y;
        x = x << y;
        x >>= y;
        x = x >> y;
        x >>>= y;
        x = x >>> y;
        x &= y;
        x = x & y;
        x ^= y;
        x = x ^ y;
        x |= y;
        x = x | y;
        x **= y;
        x = x ** y;
        x = (()=>{return x ** y})();
        x = 33 == true ? 25 : 88;
    
        /**
         * Esse é um comentário que deve ser ignorado
         */
        let count = 0;
    
        console.log("eita porra"); // deve ser ignorado
    
        let time = new Date()
        var xpto;
    
        "use strict"
    
        function mariaGabriela() {
          xpto = undefined;
        }
    
        // eita diabo
    
        maria = 33
    
        const interval = setInterval(() => {
          count++;
          time = new Date();
          mySpan.innerText = time.toString()
        }, 1000)
    
        const onClick = () => {
          alert(variavel)
        }
    
        const api = {
          GetTime: () => {
            return time
          }
        }
    
        const onDestroy = () => clearInterval(interval)
      </script>
    </component>
    
    <component name="dois">
      <clock
        ref="clockRef"
        callback="fazAlgumaCoisa()"
      />
      <script>
        const fazAlgumaCoisa = () => {
          console.log(clockRef.GetTime())
        }
      </script>
    </component>
    
    <dois/>
`

  values := map[string]interface{}{
    "valueTrue":  true,
    "valueFalse": false,
  }

  expected := `
    <div>
      A
      
      
      D
      
      F
      G
        G.1
        
      
      
    </div>`

  sht.TestTemplate(t, template, values, expected, testGDs)
}
