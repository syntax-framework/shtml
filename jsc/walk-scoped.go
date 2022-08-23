package jsc

import "github.com/tdewolff/parse/v2/js"

type WalkScope struct {
	replace func(node js.INode, by js.INode) bool
}

type WalkScopeStack struct {
	scopes []*WalkScope
}

func (r *WalkScopeStack) Push(scope *WalkScope) {
	r.scopes = append(r.scopes, scope)
}

func (r *WalkScopeStack) Pop() {
	if len(r.scopes) > 0 {
		r.scopes = r.scopes[:len(r.scopes)-1]
	}
}

func (r *WalkScopeStack) Replace(node js.INode, by js.INode) bool {
	for i := len(r.scopes) - 1; i >= 0; i-- {
		scope := r.scopes[i]
		if scope.replace != nil && scope.replace(node, by) {
			return true
		}
	}
	return false
}

// IVisitorScoped represents the AST Visitor
// Each INode encountered by `Walk` is passed to `Enter`, children nodes will be ignored if the returned IVisitor is nil
// `Exit` is called upon the exit of a node
type IVisitorScoped interface {
	Enter(n js.INode, scope *WalkScopeStack) IVisitorScoped
	Exit(n js.INode)
}

// IVisitorScopedFunc use function as AST Visitor
type IVisitorScopedFunc func(node js.INode, scope *WalkScopeStack) (visitChildren bool)

func (f IVisitorScopedFunc) Enter(node js.INode, scope *WalkScopeStack) IVisitorScoped {
	if f(node, scope) {
		return f
	}
	return nil
}

func (f IVisitorScopedFunc) Exit(node js.INode) {
}

// WalkScoped traverses an AST in depth-first order
func WalkScoped(v IVisitorScoped, n js.INode, stack *WalkScopeStack) {
	if n == nil {
		return
	}

	if stack == nil {
		stack = &WalkScopeStack{}
	}

	if v = v.Enter(n, stack); v == nil {
		return
	}

	defer v.Exit(n)

	switch n := n.(type) {
	case *js.AST:
		WalkScoped(v, &n.BlockStmt, stack)
	case *js.Var:
		return
	case *js.BlockStmt:
		if n.List != nil {
			stack.Push(&WalkScope{
				replace: func(node js.INode, by js.INode) bool {
					if byStmt, isStmt := by.(js.IStmt); isStmt {
						for i, stmt := range n.List {
							if stmt == node {
								n.List[i] = byStmt
								return true
							}
						}
					}
					return false
				},
			})
			for i := 0; i < len(n.List); i++ {
				WalkScoped(v, n.List[i], stack)
			}
			stack.Pop()
		}
	case *js.EmptyStmt:
		return
	case *js.ExprStmt:
		WalkScoped(v, n.Value, stack)
	case *js.IfStmt:
		WalkScoped(v, n.Body, stack)
		WalkScoped(v, n.Else, stack)
		WalkScoped(v, n.Cond, stack)
	case *js.DoWhileStmt:
		WalkScoped(v, n.Body, stack)
		WalkScoped(v, n.Cond, stack)
	case *js.WhileStmt:
		WalkScoped(v, n.Body, stack)
		WalkScoped(v, n.Cond, stack)
	case *js.ForStmt:
		if n.Body != nil {
			stack.Push(&WalkScope{
				replace: func(node js.INode, by js.INode) bool {
					if byStmt, isStmt := by.(*js.BlockStmt); isStmt {
						if n.Body == node {
							n.Body = byStmt
							return true
						}
					}
					return false
				},
			})
			WalkScoped(v, n.Body, stack)
			stack.Pop()
		}

		WalkScoped(v, n.Init, stack)
		WalkScoped(v, n.Cond, stack)
		WalkScoped(v, n.Post, stack)
	case *js.ForInStmt:
		if n.Body != nil {
			stack.Push(&WalkScope{
				replace: func(node js.INode, by js.INode) bool {
					if byStmt, isStmt := by.(*js.BlockStmt); isStmt {
						if n.Body == node {
							n.Body = byStmt
							return true
						}
					}
					return false
				},
			})
			WalkScoped(v, n.Body, stack)
			stack.Pop()
		}

		WalkScoped(v, n.Init, stack)
		WalkScoped(v, n.Value, stack)
	case *js.ForOfStmt:
		if n.Body != nil {
			WalkScoped(v, n.Body, stack)
		}

		WalkScoped(v, n.Init, stack)
		WalkScoped(v, n.Value, stack)
	case *js.CaseClause:
		if n.List != nil {
			for i := 0; i < len(n.List); i++ {
				WalkScoped(v, n.List[i], stack)
			}
		}

		WalkScoped(v, n.Cond, stack)
	case *js.SwitchStmt:
		if n.List != nil {
			for i := 0; i < len(n.List); i++ {
				WalkScoped(v, &n.List[i], stack)
			}
		}

		WalkScoped(v, n.Init, stack)
	case *js.BranchStmt:
		return
	case *js.ReturnStmt:
		WalkScoped(v, n.Value, stack)
	case *js.WithStmt:
		WalkScoped(v, n.Body, stack)
		WalkScoped(v, n.Cond, stack)
	case *js.LabelledStmt:
		WalkScoped(v, n.Value, stack)
	case *js.ThrowStmt:
		WalkScoped(v, n.Value, stack)
	case *js.TryStmt:
		if n.Body != nil {
			WalkScoped(v, n.Body, stack)
		}

		if n.Catch != nil {
			WalkScoped(v, n.Catch, stack)
		}

		if n.Finally != nil {
			WalkScoped(v, n.Finally, stack)
		}

		WalkScoped(v, n.Binding, stack)
	case *js.DebuggerStmt:
		return
	case *js.Alias:
		return
	case *js.ImportStmt:
		if n.List != nil {
			for i := 0; i < len(n.List); i++ {
				WalkScoped(v, &n.List[i], stack)
			}
		}
	case *js.ExportStmt:
		if n.List != nil {
			for i := 0; i < len(n.List); i++ {
				WalkScoped(v, &n.List[i], stack)
			}
		}

		WalkScoped(v, n.Decl, stack)
	case *js.DirectivePrologueStmt:
		return
	case *js.PropertyName:
		WalkScoped(v, &n.Literal, stack)
		WalkScoped(v, n.Computed, stack)
	case *js.BindingArray:
		if n.List != nil {
			for i := 0; i < len(n.List); i++ {
				WalkScoped(v, &n.List[i], stack)
			}
		}

		WalkScoped(v, n.Rest, stack)
	case *js.BindingObjectItem:
		if n.Key != nil {
			WalkScoped(v, n.Key, stack)
		}

		WalkScoped(v, &n.Value, stack)
	case *js.BindingObject:
		if n.List != nil {
			for i := 0; i < len(n.List); i++ {
				WalkScoped(v, &n.List[i], stack)
			}
		}

		if n.Rest != nil {
			WalkScoped(v, n.Rest, stack)
		}
	case *js.BindingElement:
		WalkScoped(v, n.Binding, stack)
		WalkScoped(v, n.Default, stack)
	case *js.VarDecl:
		if n.List != nil {
			for i := 0; i < len(n.List); i++ {
				WalkScoped(v, &n.List[i], stack)
			}
		}
	case *js.Params:
		if n.List != nil {
			for i := 0; i < len(n.List); i++ {
				WalkScoped(v, &n.List[i], stack)
			}
		}

		WalkScoped(v, n.Rest, stack)
	case *js.FuncDecl:
		WalkScoped(v, &n.Body, stack)
		WalkScoped(v, &n.Params, stack)

		if n.Name != nil {
			WalkScoped(v, n.Name, stack)
		}
	case *js.MethodDecl:
		WalkScoped(v, &n.Body, stack)
		WalkScoped(v, &n.Params, stack)
		WalkScoped(v, &n.Name, stack)
	case *js.Field:
		WalkScoped(v, &n.Name, stack)
		WalkScoped(v, n.Init, stack)
	case *js.ClassDecl:
		if n.Name != nil {
			WalkScoped(v, n.Name, stack)
		}

		WalkScoped(v, n.Extends, stack)

		for _, item := range n.List {
			if item.StaticBlock != nil {
				WalkScoped(v, item.StaticBlock, stack)
			} else if item.Method != nil {
				WalkScoped(v, item.Method, stack)
			} else {
				WalkScoped(v, &item.Field, stack)
			}
		}
	case *js.LiteralExpr:
		return
	case *js.Element:
		WalkScoped(v, n.Value, stack)
	case *js.ArrayExpr:
		if n.List != nil {
			for i := 0; i < len(n.List); i++ {
				WalkScoped(v, &n.List[i], stack)
			}
		}
	case *js.Property:
		if n.Name != nil {
			WalkScoped(v, n.Name, stack)
		}

		WalkScoped(v, n.Value, stack)
		WalkScoped(v, n.Init, stack)
	case *js.ObjectExpr:
		if n.List != nil {
			for i := 0; i < len(n.List); i++ {
				WalkScoped(v, &n.List[i], stack)
			}
		}
	case *js.TemplatePart:
		WalkScoped(v, n.Expr, stack)
	case *js.TemplateExpr:
		if n.List != nil {
			for i := 0; i < len(n.List); i++ {
				WalkScoped(v, &n.List[i], stack)
			}
		}

		WalkScoped(v, n.Tag, stack)
	case *js.GroupExpr:
		WalkScoped(v, n.X, stack)
	case *js.IndexExpr:
		WalkScoped(v, n.X, stack)
		WalkScoped(v, n.Y, stack)
	case *js.DotExpr:
		WalkScoped(v, n.X, stack)
		WalkScoped(v, &n.Y, stack)
	case *js.NewTargetExpr:
		return
	case *js.ImportMetaExpr:
		return
	case *js.Arg:
		WalkScoped(v, n.Value, stack)
	case *js.Args:
		if n.List != nil {
			for i := 0; i < len(n.List); i++ {
				WalkScoped(v, &n.List[i], stack)
			}
		}
	case *js.NewExpr:
		if n.Args != nil {
			WalkScoped(v, n.Args, stack)
		}

		WalkScoped(v, n.X, stack)
	case *js.CallExpr:
		WalkScoped(v, &n.Args, stack)
		WalkScoped(v, n.X, stack)
	case *js.UnaryExpr:
		WalkScoped(v, n.X, stack)
	case *js.BinaryExpr:
		stack.Push(&WalkScope{replace: canRplaceIExprFn(n.X, func(expr js.IExpr) { n.X = expr })})
		WalkScoped(v, n.X, stack)
		stack.Pop()

		stack.Push(&WalkScope{replace: canRplaceIExprFn(n.Y, func(expr js.IExpr) { n.Y = expr })})
		WalkScoped(v, n.Y, stack)
		stack.Pop()
	case *js.CondExpr:
		WalkScoped(v, n.Cond, stack)
		WalkScoped(v, n.X, stack)
		WalkScoped(v, n.Y, stack)
	case *js.YieldExpr:
		WalkScoped(v, n.X, stack)
	case *js.ArrowFunc:
		WalkScoped(v, &n.Body, stack)
		WalkScoped(v, &n.Params, stack)
	case *js.CommaExpr:
		stack.Push(&WalkScope{
			replace: func(node js.INode, by js.INode) bool {
				if byExpr, isExpr := by.(js.IExpr); isExpr {
					println(by.JS())
					for i, expr := range n.List {
						if expr == node {
							n.List[i] = byExpr
							return true
						}
					}
				}
				return false
			},
		})
		for _, item := range n.List {
			WalkScoped(v, item, stack)
		}
		stack.Pop()
	default:
		return
	}
}

func canRplaceIExpr(actual js.IExpr, find js.INode, replaceBy js.INode, replace func(expr js.IExpr)) bool {
	if converted, isSameType := replaceBy.(js.IExpr); actual == find && isSameType {
		replace(converted)
		return true
	}
	return false
}

func canRplaceIExprFn(actual js.IExpr, replace func(expr js.IExpr)) func(node js.INode, by js.INode) bool {
	return func(node js.INode, by js.INode) bool {
		return canRplaceIExpr(actual, node, by, replace)
	}
}
