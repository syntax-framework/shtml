package sht

import (
	"testing"
)

func Test_Page_Directive(t *testing.T) {

	template := `
    <page title="My First Syntax Page - {title}" layout="{layout}"/>
    <div>{valueOne ? 'value-true' : 'value-false'}</div>
  `

	static := []string{
		"<page",
		"></page><div>",
		"</div>",
	}

	expected := `<div>value-true</div>`

	values := map[string]interface{}{
		"valueOne": true,
		"title":    "My Page Title",
		"layout":   "custom.html",
	}

	directives := &Directives{}
	//directives.Add(PageDirective)

	compiled, _ := TestCompile(t, template, static, directives)
	rendered, _ := TestRender(t, compiled, values, expected)
	println(rendered)
}

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
