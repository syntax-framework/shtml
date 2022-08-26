package sht

import (
	"bytes"
	"fmt"
	"github.com/antonmedv/expr"
	"github.com/antonmedv/expr/vm"
	"io"
	"log"
	"strings"
)

type Expression struct {
	program *vm.Program
}

func (e *Expression) Exec(scope *Scope) interface{} {
	output, err := expr.Run(e.program, scope)
	if err != nil {
		log.Print(err)
		return nil
	}
	return output
}

func (e *Expression) EvalBool(scope *Scope) bool {
	if e == nil {
		return false
	}
	result := e.Exec(scope)
	if result == nil || result == false || result == -1 || result == 0 || result == "false" || result == "" {
		return false
	}
	return true
}

func (e *Expression) EvalString(scope *Scope) string {
	if e == nil {
		return ""
	}
	result := e.Exec(scope)
	if result == nil {
		return ""
	}
	return fmt.Sprintf("%v", result)
}

var _cachedExpressions = map[string]*Expression{}

// ParseExpression process a single expression
func ParseExpression(exp string) (*Expression, error) {
	exp = strings.TrimSpace(exp)
	expression, exists := _cachedExpressions[exp]
	if exists {
		return expression, nil
	}
	program, err := expr.Compile(exp)
	if err != nil {
		return nil, err
	}
	expression = &Expression{program: program}
	_cachedExpressions[exp] = expression
	return expression, nil
}

// DynamicInterpolate parte dinamica de execução de uma expressão
type DynamicInterpolate struct {
	expression *Expression
}

func (d *DynamicInterpolate) Exec(scope *Scope) interface{} {
	return d.expression.EvalString(scope)
}

// DynamicInterpolateEscaped parte dinamica de execução de uma expressão
type DynamicInterpolateEscaped struct {
	expression *Expression
}

func (d *DynamicInterpolateEscaped) Exec(scope *Scope) interface{} {
	return HtmlEscape(d.expression.EvalString(scope))
}

// Interpolate Compiles a string with markup into an interpolation function.
//
// GO INTERPOLATION ( {value} or  !{value} )
//
// <element attribute="{return value}">
// <element attribute="xpto {escape safe}">
// <element attribute="xpto !{escape unsafe}">
// <element attribute="!{escape unsafe}">
// <element>{escape safe}</element>
// <element>!{escape unsafe}</element>
//
// exp = Interpolate('Hello {name}!');
// exp.Exec({name:'Syntax'}).String() == "Hello Syntax!"
func Interpolate(text string) (*Compiled, error) {

	if !strings.ContainsRune(text, '{') {
		return nil, nil
	}

	compiled := &Compiled{}

	// Allows you to discover the number of open braces within an expression
	innerBrackets := 0

	// Is processing an expression (started with "{" or "!{")
	inExpression := false

	//   Safe:  {expr}
	// Unsafe: !{expr}
	isSafeSignal := true

	content := &bytes.Buffer{}

	reader := strings.NewReader(text)
	for {
		currChar, _, err := reader.ReadRune()
		if err != nil {
			if err == io.EOF {
				break
			} else {
				return nil, err
			}
		}
		nextChar, _, err := reader.ReadRune()
		if err != nil && err != io.EOF {
			return nil, err
		}

		if err != io.EOF {
			// unred nextChar
			err = reader.UnreadRune()
			if err != nil {
				return nil, err
			}
		}

		if !inExpression {
			if currChar == '{' || (currChar == '!' && nextChar == '{') {
				// {value} or !{value}
				compiled.static = append(compiled.static, content.String())

				inExpression = true
				isSafeSignal = currChar == '{'

				content = &bytes.Buffer{}
			} else {
				content.WriteRune(currChar)
			}
		} else {
			if currChar == '{' {
				// is not first "{"
				if content.Len() > 0 {
					innerBrackets++
					content.WriteRune(currChar)
				}
			} else {
				if currChar == '}' {
					if innerBrackets > 0 {
						innerBrackets--
					} else {
						inExpression = false

						value := content.String()
						program, programErr := ParseExpression(value)
						if programErr != nil {
							return nil, programErr
						}

						if isSafeSignal {
							compiled.dynamics = append(compiled.dynamics, &DynamicInterpolateEscaped{expression: program})
						} else {
							compiled.dynamics = append(compiled.dynamics, &DynamicInterpolate{expression: program})
						}

						//prev = currChar
						content = &bytes.Buffer{}
						continue
					}
				}
				content.WriteRune(currChar)
			}
		}

		//prev = currChar
	}

	compiled.static = append(compiled.static, content.String())

	return compiled, nil
}
