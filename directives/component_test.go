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

func Test_Component(t *testing.T) {

	template := `
    <component
      name="clock"
      element="div"
    
      param-name="string"
      param-other="?map"
    
      js:param-callback="string"
      js:param-variavel="string"
    
      todo="@TODO: Parametros que deverão ser suportados no futuro"
      controller="RegisteredController"
    >
      <span onclick="onClick()">...</span>
      <span onclick="callback" data-ref="mySpan">js input</span>
    
      {{count}}
    
      <style>
        span {
          font-family: Roboto
        }
      </style>
    
      <script>
        const [count, setCount] = STX.state(0)
    
        let time = new Date()
    
        const interval = setInterval(() => {
          time = new Date()
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
    
        const clear = () => clearInterval(interval)
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
