package directives

import (
	"github.com/syntax-framework/shtml/sht"
	"testing"
)

func Test_component_should_not_allow_nested_components(t *testing.T) {
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
func Test_component_should_not_allow_multiple_style_element(t *testing.T) {
	template := `
    <component name="test">
      <style>.my-class {color: #FFF}</style>
      <div><style>.my-class-2 {color: #FFF}</style></div>
    </component>
  `
	testForErrorCode(t, template, "component:style:single")
}

// a component can only have a single script tag
func Test_component_should_not_allow_multiple_script_element(t *testing.T) {
	template := `
    <component name="test">
      <script>console.log("hello")</script>
      <div><script>console.log("world!")</script></div>
    </component>
  `
	testForErrorCode(t, template, "component:script:single")
}

// when it has style, it must be an immediate child of the component
func Test_component_style_element_must_be_immediate_child(t *testing.T) {
	template := `
    <component name="test">
      <div><style>.my-class-2 {color: #FFF}</style></div>
    </component>
  `
	testForErrorCode(t, template, "component:style:location")
}

// when it has script, it must be an immediate child of the component
func Test_component_script_element_must_be_immediate_child(t *testing.T) {
	template := `
    <component name="test">
      <div><script>console.log("world!")</script></div>
    </component>
  `
	testForErrorCode(t, template, "component:script:location")
}

// client-param is referencing a non-existent parameter
func Test_component_js_param_invalid_reference(t *testing.T) {
	template := `
    <component name="test" param-server-name="string" client-param-name="@server-name-wrong">
      <div></div>
    </component>
  `
	testForErrorCode(t, template, "component.param.client.ref.notfound")
}

// client-param is referencing a non-existent parameter
func Test_two_way_data_binding(t *testing.T) {
	template := `
    <component name="test">
      <input type="text" value="${varx}" />
      <input type="text" value="${vary}" />
      <input type="text" value="${myVariable1}" />
      <input type="text" value="${myVariable2}" />
      <input type="text" value="${myVariable3}" />
      <input type="text" value="${variableX}" />
      <input type="text" value="${myFunction1}" />
      <input type="text" value="${myFunction2}" />
      <input type="text" value="${myFunction3}" />
      <input type="text" value="${unknownVariable}" />

      <script>
        let myVariable1 = "";
        var myVariable2 = "";
        let variableX;
        let [varx, vary] = [25, 'aoha'];
        const myVariable3 = "";
        let myFunction1 = () => {};
        var myFunction2 = () => {};
        const myFunction3 = () => {};

        // onChange
        validate(myVariable1, (value) => {
          return true
        })

        // to STATE
        set(myVariable1, (value) => {
          return Number.parseInt(value)
        })
        
        // to DOM
        get(myVariable1, (value) => {
          return value.ToFixed(2)
        })
      </script>
    </component>
  `
	template = sht.TestUnindentedTemplate(template)
	compiler := sht.NewCompiler(&sht.TemplateSystem{Directives: testGDs.NewChild()})
	compiled, err := compiler.Compile(template, "template.html")
	if err != nil {
		t.Fatal(err)
	}
	println(compiled)
}

// Expressions with Side Effect in text interpolation block are not allowed.
func Test_should_not_allow_side_effect_in_interpolation(t *testing.T) {
	var tests = []struct {
		name  string
		input string
	}{
		{"1", `<component name="c"> <span>${ a = --a + 1, b }</span> <script>let a = 0; let b = '';</script></component>`},
		{"2", `<component name="c"> <span>${ a++, b }</span> <script>let a = 0; let b = '';</script></component>`},
		{"3", `<component name="c"> <span>${ ++a, b }</span> <script>let a = 0; let b = '';</script></component>`},
		{"4", `<component name="c"> <span>${ a--, b }</span> <script>let a = 0; let b = '';</script></component>`},
		{"5", `<component name="c"> <span>${ --a, b }</span> <script>let a = 0; let b = '';</script></component>`},
		{"6", `<component name="c"> <span>${ a = a + 1, b }</span> <script>let a = 0; let b = '';</script></component>`},
		{"7", `<component name="c"> <span>${ a = --a + a++, b }</span> <script>let a = 0; let b = '';</script></component>`},
		{"8", `<component name="c"> <span class="${ a = --a + a++, b }">text</span> <script>let a = 0; let b = '';</script></component>`},
		{"9", `<component name="c"> 
      <span>${sideEffectFn()}</span> 
      <script>
        let a = 0; 
        let b = '';
        const sideEffectFn = () =>{
          return a = --a + a++, b 
        }
      </script>
    </component>`},
		{"10", `<component name="c"> 
      <span>${myFn()}</span> 
      <script>
        let a = 0; 
        let b = '';

        const sideEffectFn = () => {
            return a = --a + a++, b 
        }

        const myFn = () => {
          return sideEffectFn()
        }
      </script>
    </component>`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testForErrorCode(t, tt.input, "js:interpolation:sideeffect")
		})

	}
}
