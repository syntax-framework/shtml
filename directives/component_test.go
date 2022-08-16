package directives

import (
	"github.com/syntax-framework/shtml/sht"
	"testing"
)

func Test_Should_Not_Allow_Defining_Nested_Components(t *testing.T) {
	template := `
    <component name="out">
      <div>
        <component name="inner">
          content
        </component>
      </div>
    </component>
  `

	template = sht.TestUnindentedTemplate(template)
	compiler := sht.NewCompiler(&sht.TemplateSystem{Directives: testGDs.NewChild()})
	_, err := compiler.Compile(template)

	if err == nil {
		t.Errorf("compiler.Compile(template) | expect to receive compilation error")
	}
}

func Test_Component(t *testing.T) {

	template := `
    <component
      name="clock"
      element="div"
      p:variavel="js"
      p:callback="js"
      p:name="string"
      p:other="?map"
      todo="@TODO: Parametros que deverão ser suportados no futuro"
      controller="ControllerOpcional"
    >
      <span onclick="onClick()">...</span>
      <span onclick="callback">js input</span>
    
      <style>
        span {
          font-family: Roboto
        }
      </style>
    
      <script type="text/javascript">
        let $span = $('span')
        let time = new Date()
    
        const interval = setInterval(() => {
          time = new Date()
          $span.innerText = time.toString()
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
