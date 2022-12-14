package jsc

import (
	"bytes"
	"github.com/syntax-framework/shtml/sht"
	"io"
	"strings"
)

type Interpolation struct {
	Expression    string
	IsSafeSignal  bool
	IsFullContent bool
}

func (i *Interpolation) Debug() string {
	if i.IsSafeSignal {
		return "${" + i.Expression + "}"
	} else {
		return "#{" + i.Expression + "}"
	}
}

// Interpolate processa as interpolações javascript em um texto
//
// JAVASCRIPT INTERPOLATION ( ${value} or  #{value} )
//
// <element attribute="${return value}">
// <element attribute="xpto ${escape safe}">
// <element attribute="xpto #{escape unsafe}">
// <element attribute="#{escape unsafe}">
// <element>${escape safe}</element>
// <element>#{escape unsafe}</element>
// #{serverExpressionUnescaped}
//
// @TODO: Filters/Pipe. Ex. ${ myValue | upperCase}
//
// newText, watches, err = Interpolate('Hello ${name}!');
// newText == "Hello _j$_i15151ffacb"
// interpolations == {"_j$_i15151ffacb": {Expression: "name", isScape: true}}
// exp.Exec({name:'Syntax'}).String() == "Hello Syntax!"
func Interpolate(text string, sequence *sht.Sequence) (string, map[string]*Interpolation, error) {

	if !strings.ContainsRune(text, '{') || !strings.ContainsAny(text, "$#") {
		return text, nil, nil
	}

	// always trim, is still valid html. Syntax has no intention of working with other media
	text = strings.TrimSpace(text)

	interpolations := map[string]*Interpolation{}

	// Allows you to discover the number of open braces within an Expression
	innerBrackets := 0

	// Is processing an Expression (started with "${" or "#{")
	inExpression := false

	//   Safe: ${expr}
	// Unsafe: #{expr}
	isSafeSignal := true

	content := &bytes.Buffer{}

	expressionId := ""
	expressionContent := &bytes.Buffer{}

	reader := strings.NewReader(text)

	for {
		currChar, _, err := reader.ReadRune()
		if err != nil {
			if err == io.EOF {
				break
			} else {
				return text, nil, err
			}
		}
		nextChar, _, err := reader.ReadRune()
		if err != nil && err != io.EOF {
			return text, nil, err
		}

		if err != io.EOF {
			err = reader.UnreadRune()
			if err != nil {
				return text, nil, err
			}
		}

		if !inExpression {
			if (currChar == '$' || currChar == '#') && nextChar == '{' {
				// ${value} or #{value}
				inExpression = true
				isSafeSignal = currChar == '$'

				expressionId = sequence.NextHash()
				content.WriteString(expressionId)

				expressionContent = &bytes.Buffer{}
			} else {
				content.WriteRune(currChar)
			}
		} else {
			if currChar == '{' {
				if expressionContent.Len() > 0 {
					innerBrackets++
					expressionContent.WriteRune(currChar)
				}
			} else {
				if currChar == '}' {
					if innerBrackets > 0 {
						innerBrackets--
					} else {
						inExpression = false

						interpolations[expressionId] = &Interpolation{
							Expression:   expressionContent.String(),
							IsSafeSignal: isSafeSignal,
						}
						continue
					}
				}
				expressionContent.WriteRune(currChar)
			}
		}
	}

	if inExpression {
		// invalid content, will probably pop JS error
		interpolations[expressionId] = &Interpolation{
			Expression:   expressionContent.String(),
			IsSafeSignal: isSafeSignal,
		}
	}

	text = content.String()

	if text == expressionId {
		interpolation := interpolations[expressionId]
		interpolation.IsFullContent = true
	}

	return text, interpolations, nil
}
