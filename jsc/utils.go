package jsc

import (
	"github.com/syntax-framework/shtml/sht"
	"github.com/tdewolff/parse/v2/js"
	"strconv"
)

// Binary Assignment Operators
//
// https://tc39.es/ecma262/multipage/ecmascript-language-expressions.html#sec-assignment-operators
var binaryAssignmentOperators = map[js.TokenType]bool{
	js.EqToken:        true, // =    Assignment
	js.AddEqToken:     true, // +=   Addition assignment
	js.SubEqToken:     true, // -=   Subtraction assignment
	js.MulEqToken:     true, // *=   Multiplication assignment
	js.DivEqToken:     true, // /=   Division assignment
	js.ModEqToken:     true, // %=   Remainder assignment
	js.ExpEqToken:     true, // **=  Exponentiation assignment (ECMAScript 2016)
	js.LtLtEqToken:    true, // <<=  Left shift assignment
	js.GtGtEqToken:    true, // >>=  Right shift assignment
	js.GtGtGtEqToken:  true, // >>>= Unsigned right shift assignment
	js.BitAndEqToken:  true, // &=   Bitwise AND assignment
	js.BitXorEqToken:  true, // ^=   Bitwise XOR assignment
	js.BitOrEqToken:   true, // |=   Bitwise OR assignment
	js.AndEqToken:     true, // &&=  Logical AND assignment
	js.OrEqToken:      true, // ||=  Logical OR assignment
	js.NullishEqToken: true, // ??=  Logical nullish assignment
}

// IsBinaryAssignmentOperator check if is a JavaScript Assignment Operators
func IsBinaryAssignmentOperator(tokenType js.TokenType) bool {
	return binaryAssignmentOperators[tokenType] == true
}

func CallExpr(name string, args ...js.IExpr) js.CallExpr {

	var list []js.Arg

	for _, arg := range args {
		list = append(list, js.Arg{Value: arg})
	}

	return js.CallExpr{
		X:        &js.Var{Data: []byte(name)},
		Args:     js.Args{List: list},
		Optional: false,
	}
}

func IntegerExpr(value int) js.LiteralExpr {
	return js.LiteralExpr{
		TokenType: js.NumericToken,
		Data:      []byte(strconv.Itoa(value)),
	}
}

func StringExpr(value string) js.LiteralExpr {
	return js.LiteralExpr{
		TokenType: js.NumericToken,
		Data:      []byte(value),
	}
}

func GroupCommaExpr(list ...js.IExpr) js.GroupExpr {
	return js.GroupExpr{
		X: js.CommaExpr{
			List: list,
		},
	}
}

// IsDeclaredOnScope check if this Expression is declared on specified scope
func IsDeclaredOnScope(expr *js.Var, scope *js.Scope) (bool, *js.Var) {
	for _, d := range scope.Declared {
		if d == expr {
			return true, d
		}

		if d == expr.Link {
			return true, expr.Link
		}
	}
	return false, nil
}

func AddDispatcers(ast js.INode, globalScope *js.Scope, globaUsedVars *sht.IndexedMap, stack *WalkScopeStack) {
	// fast check
	if hasSideEffect, _ := HasSideEffect(ast); hasSideEffect {
		WalkScoped(IVisitorScopedFunc(func(node js.INode, stack *WalkScopeStack) bool {
			// Essa parte pode ser dinamica, func(INode, *WalkScopeStack) bool
			//// se é um statement, encapsula sua execução para acionar os watchers
			var jsVar *js.Var
			var jsExpr js.IExpr
			var rightAssignmentExpression js.IExpr
			valueChangeBeforeReturn := true

			switch node.(type) {
			case *js.UnaryExpr:
				unaryExpr := node.(*js.UnaryExpr)
				if v, isVar := unaryExpr.X.(*js.Var); isVar {
					jsVar = v
				}
				jsExpr = unaryExpr
				if unaryExpr.Op == js.PostIncrToken || unaryExpr.Op == js.PostDecrToken {
					valueChangeBeforeReturn = false
				}
			case *js.BinaryExpr:
				binaryExpr := node.(*js.BinaryExpr)
				if IsBinaryAssignmentOperator(binaryExpr.Op) {
					if v, isVar := binaryExpr.X.(*js.Var); isVar {
						jsVar = v
						rightAssignmentExpression = binaryExpr.Y
					}
					jsExpr = binaryExpr
				}
			}

			if jsVar != nil {
				if isDeclared, jsVarGlobal := IsDeclaredOnScope(jsVar, globalScope); isDeclared {
					println(jsVarGlobal.JS())
					// mark jsExpr to dispatch
					varIndex := globaUsedVars.Add(jsVarGlobal)
					if valueChangeBeforeReturn {
						// [5, 6, 6] = (value, ++value, value)
						// [5, 6, 6] = (value, value = value + 1, value)
						// _$i(index, variable, Expression)
						stack.Replace(node, CallExpr("_$i", IntegerExpr(varIndex), jsVar, jsExpr))
						if rightAssignmentExpression != nil {
							println(rightAssignmentExpression.JS())

							newStack := &WalkScopeStack{}
							newStack.Push(&WalkScope{
								replace: func(node js.INode, by js.INode) bool {
									if byValue, isSameType := by.(js.IExpr); isSameType {
										if rightAssignmentExpression == node {
											rightAssignmentExpression = byValue
											return true
										}
									}
									return false
								},
							})
							// process Expression
							AddDispatcers(rightAssignmentExpression, globalScope, globaUsedVars, newStack)
							newStack.Pop()

							println(rightAssignmentExpression.JS())
							// dont process child again
							return false
						}
					} else {
						// [5, 5, 6] = (value, value++, value)
						// _$i(index, variable, (Expression, variable))
						stack.Replace(node, CallExpr("_$i", IntegerExpr(varIndex), jsVar, GroupCommaExpr(jsExpr, jsVar)))
					}
				}
			}
			//if jsExpr, isExprStmt := node.(*js.UnaryExpr); isExprStmt {
			//	// JsDispatchUnaryExpr is an update or unary Expression.
			//	// value++, value--, value*=2
			//	if jsVar, isVar := jsExpr.X.(*js.Var); isVar {
			//		if isDeclared, jsVarGlobal := isDeclaredOnScope(jsVar, globalScope); isDeclared {
			//			println(jsVarGlobal.JS())
			//			// mark jsExpr to dispatch
			//			varIndex := globaUsedVars.Add(jsVarGlobal)
			//			stack.Replace(node, jsAstCallExprExpr("_$i", jsAstIntegerExpr(varIndex), jsVar, jsExpr))
			//		}
			//	}
			//}
			return true
		}), ast, stack)
	}

}

// HasSideEffect checks if the Expression has a side effect. Returns on first side effect Expression found.
func HasSideEffect(ast js.INode) (bool, string) {

	hasEffect := false
	expressionJs := ""

	js.Walk(VisitorEnterFunc(func(node js.INode) bool {
		switch node.(type) {
		case *js.UnaryExpr:
			hasEffect = true
			expressionJs = node.JS()
			return false
		case *js.BinaryExpr:
			if IsBinaryAssignmentOperator(node.(*js.BinaryExpr).Op) {
				hasEffect = true
				expressionJs = node.JS()
				return false
			}
		}

		return true
	}), ast)

	return hasEffect, expressionJs
}

//OpenParenToken              // (
//CloseParenToken             // )
//CommaExpr
//GroupExpr
