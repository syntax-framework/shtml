package directives

import (
	"github.com/syntax-framework/shtml/sht"
	"testing"
)

func Test_IF_Element(t *testing.T) {

	template := `
    <div>
      <if cond="valueTrue">A</if>
      <if cond="valueFalse">B</if>
      <if cond="!valueTrue">C</if>
      <if cond="!valueFalse">D</if>
      <if cond="valueTrue and valueFalse">E</if>
      <if cond="valueTrue or valueFalse">F</if>
      <if cond="valueTrue">G
        <if cond="valueTrue">G.1</if>
        <if cond="valueFalse">G.2</if>
      </if>
      <if cond="valueNotFound">H</if>
    </div>`

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
