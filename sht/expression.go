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
// exp = Interpolate('Hello {name}!');
// exp.Exec({name:'Syntax'}).String() == "Hello Syntax!"
func Interpolate(text string) (*Compiled, error) {

	if !strings.ContainsRune(text, '{') {
		return nil, nil
	}

	compiled := &Compiled{}

	// Allows you to discover the number of open braces within an expression
	innerBrackets := 0

	// Is processing an expression (started with "!{" or "#{")
	inExpression := false

	// String Unescaped: !{riskyBusiness}
	// String Escaped:   #{expression}
	isEscaped := true

	content := &bytes.Buffer{}

	prev := ' '
	z := strings.NewReader(text)
	for {
		c, _, err := z.ReadRune()
		if err != nil {
			if err == io.EOF {
				break
			} else {
				return nil, err
			}
		}
		next, _, err := z.ReadRune()
		if err != nil && err != io.EOF {
			return nil, err
		}

		if err != io.EOF {
			err = z.UnreadRune()
			if err != nil {
				return nil, err
			}
		}

		if !inExpression {
			if (c == '!' || c == '#') && next == '{' {
				if content.Len() > 0 {
					compiled.static = append(compiled.static, content.String())
				}
				isEscaped = c == '#'
				inExpression = true
				content = &bytes.Buffer{}
			}
		} else {
			if c == '{' && prev == '\\' {
				innerBrackets++
			} else if c == '}' {
				if innerBrackets > 0 {
					innerBrackets--
				} else {
					inExpression = false
					value := content.String()[2:]
					program, err := ParseExpression(value)
					if err != nil {
						log.Fatal(err)
					}

					if isEscaped {
						compiled.dynamics = append(compiled.dynamics, &DynamicInterpolateEscaped{expression: program})
					} else {
						compiled.dynamics = append(compiled.dynamics, &DynamicInterpolate{expression: program})
					}

					prev = c
					content = &bytes.Buffer{}
					continue
				}
			}
		}

		prev = c
		content.WriteRune(c)
	}

	compiled.static = append(compiled.static, content.String())

	return compiled, nil
}
