package sht

import (
	"testing"
)

func Test_Interpolation(t *testing.T) {

	template := `
    <div class="out {valueOne ? 'class-true' : 'class-false'}">
      !{valueOne ? 'value-true' : 'value-false' }
      Other text
      !{valueTwo ? 'value-true' : 'value-false' }
    
      <div class="in !{valueTwo ? 'class-true' : 'class-false' }">
        !{valueOne ? 'value-true' : 'value-false' }
        More text
        !{valueTwo ? 'value-true' : 'value-false' }
      </div>
    </div>`

	static := []string{
		"<div",
		">\n  ",
		"\n  Other text\n  ",
		"\n\n  <div",
		">\n    ",
		"\n    More text\n    ",
		"\n  </div>\n</div>",
	}

	expected := `
    <div class="out class-true">
      value-true
      Other text
      value-false
    
      <div class="in class-false">
        value-true
        More text
        value-false
      </div>
    </div>`

	values := map[string]interface{}{
		"valueOne": true,
		"valueTwo": false,
	}

	compiled, _ := TestCompile(t, template, static, &Directives{})
	TestRender(t, compiled, values, expected)
}

func Test_transclude_element(t *testing.T) {

	template := `
    <div>
      out
      <div checked="false" multiple="any" disabled empty-attr test param-1="x" param-2="y" param-3="z" class="xpto">
        inner
      </div>
    </div>`

	static := []string{
		"<div>\n  out\n  ",
		"\n</div>",
	}

	expected := `
    <div>
      out
      <div class="xpto" param-1-compile="true" param-2-process="true" param-3="z" disabled empty-attr multiple>
        inner
      </div>
    </div>`

	values := map[string]interface{}{
		"valueOne": true,
		"valueTwo": false,
	}

	directives := &Directives{}
	directives.Add(&Directive{
		Name:       "test",
		Restrict:   ATTRIBUTE,
		Priority:   200,
		Terminal:   false,
		Transclude: "element",
		Compile: func(node *Node, attrs *Attributes, t *Compiler) (methods *DirectiveMethods, err error) {

			attrs.Remove(attrs.GetAttribute("test"))
			attrs.Remove(attrs.GetAttribute("param-1"))
			attrs.Set("param-1-compile", "true")

			methods = &DirectiveMethods{
				Process: func(scope *Scope, attrs *Attributes, transclude TranscludeFunc) *Rendered {
					attrs.Remove(attrs.GetAttribute("param-2"))
					attrs.Set("param-2-process", "true")

					return transclude("", nil)
				},
			}

			return
		},
	})
	compiled, _ := TestCompile(t, template, static, directives)
	TestRender(t, compiled, values, expected)
}
