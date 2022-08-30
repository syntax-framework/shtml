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
		walkScopedReplaceIExpr(n.Value, func(expr js.IExpr) { n.Value = expr }, v, stack)
	case *js.IfStmt:
		WalkScoped(v, n.Body, stack)
		WalkScoped(v, n.Else, stack)
		WalkScoped(v, n.Cond, stack)
	case *js.DoWhileStmt:
		WalkScoped(v, n.Body, stack)

		walkScopedReplaceIExpr(n.Cond, func(expr js.IExpr) { n.Cond = expr }, v, stack)
	case *js.WhileStmt:
		WalkScoped(v, n.Body, stack)

		walkScopedReplaceIExpr(n.Cond, func(expr js.IExpr) { n.Cond = expr }, v, stack)
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

		walkScopedReplaceIExpr(n.Init, func(expr js.IExpr) { n.Init = expr }, v, stack)
		walkScopedReplaceIExpr(n.Cond, func(expr js.IExpr) { n.Cond = expr }, v, stack)
		walkScopedReplaceIExpr(n.Post, func(expr js.IExpr) { n.Post = expr }, v, stack)

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

		walkScopedReplaceIExpr(n.Init, func(expr js.IExpr) { n.Init = expr }, v, stack)
		walkScopedReplaceIExpr(n.Value, func(expr js.IExpr) { n.Value = expr }, v, stack)
	case *js.ForOfStmt:
		if n.Body != nil {
			WalkScoped(v, n.Body, stack)
		}

		walkScopedReplaceIExpr(n.Init, func(expr js.IExpr) { n.Init = expr }, v, stack)
		walkScopedReplaceIExpr(n.Value, func(expr js.IExpr) { n.Value = expr }, v, stack)
	case *js.CaseClause:
		if n.List != nil {
			for i := 0; i < len(n.List); i++ {
				WalkScoped(v, n.List[i], stack)
			}
		}

		walkScopedReplaceIExpr(n.Cond, func(expr js.IExpr) { n.Cond = expr }, v, stack)
	case *js.SwitchStmt:
		if n.List != nil {
			for i := 0; i < len(n.List); i++ {
				WalkScoped(v, &n.List[i], stack)
			}
		}

		walkScopedReplaceIExpr(n.Init, func(expr js.IExpr) { n.Init = expr }, v, stack)
	case *js.BranchStmt:
		return
	case *js.ReturnStmt:
		walkScopedReplaceIExpr(n.Value, func(expr js.IExpr) { n.Value = expr }, v, stack)
	case *js.WithStmt:
		WalkScoped(v, n.Body, stack)
		walkScopedReplaceIExpr(n.Cond, func(expr js.IExpr) { n.Cond = expr }, v, stack)
	case *js.LabelledStmt:
		WalkScoped(v, n.Value, stack)
	case *js.ThrowStmt:
		walkScopedReplaceIExpr(n.Value, func(expr js.IExpr) { n.Value = expr }, v, stack)
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

		walkScopedReplaceIExpr(n.Decl, func(expr js.IExpr) { n.Decl = expr }, v, stack)
	case *js.DirectivePrologueStmt:
		return
	case *js.PropertyName:
		WalkScoped(v, &n.Literal, stack)
		walkScopedReplaceIExpr(n.Computed, func(expr js.IExpr) { n.Computed = expr }, v, stack)
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
		walkScopedReplaceIExpr(n.Default, func(expr js.IExpr) { n.Default = expr }, v, stack)
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
		walkScopedReplaceIExpr(n.Init, func(expr js.IExpr) { n.Init = expr }, v, stack)
	case *js.ClassDecl:
		if n.Name != nil {
			WalkScoped(v, n.Name, stack)
		}

		walkScopedReplaceIExpr(n.Extends, func(expr js.IExpr) { n.Extends = expr }, v, stack)

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
		walkScopedReplaceIExpr(n.Value, func(expr js.IExpr) { n.Value = expr }, v, stack)
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

		walkScopedReplaceIExpr(n.Value, func(expr js.IExpr) { n.Value = expr }, v, stack)
		walkScopedReplaceIExpr(n.Init, func(expr js.IExpr) { n.Init = expr }, v, stack)
	case *js.ObjectExpr:
		if n.List != nil {
			for i := 0; i < len(n.List); i++ {
				WalkScoped(v, &n.List[i], stack)
			}
		}
	case *js.TemplatePart:
		walkScopedReplaceIExpr(n.Expr, func(expr js.IExpr) { n.Expr = expr }, v, stack)
	case *js.TemplateExpr:
		if n.List != nil {
			for i := 0; i < len(n.List); i++ {
				WalkScoped(v, &n.List[i], stack)
			}
		}

		walkScopedReplaceIExpr(n.Tag, func(expr js.IExpr) { n.Tag = expr }, v, stack)
	case *js.GroupExpr:
		walkScopedReplaceIExpr(n.X, func(expr js.IExpr) { n.X = expr }, v, stack)
	case *js.IndexExpr:
		walkScopedReplaceIExpr(n.X, func(expr js.IExpr) { n.X = expr }, v, stack)
		walkScopedReplaceIExpr(n.Y, func(expr js.IExpr) { n.Y = expr }, v, stack)
	case *js.DotExpr:
		walkScopedReplaceIExpr(n.X, func(expr js.IExpr) { n.X = expr }, v, stack)
		WalkScoped(v, &n.Y, stack)
	case *js.NewTargetExpr:
		return
	case *js.ImportMetaExpr:
		return
	case *js.Arg:
		walkScopedReplaceIExpr(n.Value, func(expr js.IExpr) { n.Value = expr }, v, stack)
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

		walkScopedReplaceIExpr(n.X, func(expr js.IExpr) { n.X = expr }, v, stack)
	case *js.CallExpr:
		WalkScoped(v, &n.Args, stack)
		walkScopedReplaceIExpr(n.X, func(expr js.IExpr) { n.X = expr }, v, stack)
	case *js.UnaryExpr:
		walkScopedReplaceIExpr(n.X, func(expr js.IExpr) { n.X = expr }, v, stack)
	case *js.BinaryExpr:
		walkScopedReplaceIExpr(n.X, func(expr js.IExpr) { n.X = expr }, v, stack)
		walkScopedReplaceIExpr(n.Y, func(expr js.IExpr) { n.Y = expr }, v, stack)
	case *js.CondExpr:
		walkScopedReplaceIExpr(n.Cond, func(expr js.IExpr) { n.Cond = expr }, v, stack)
		walkScopedReplaceIExpr(n.X, func(expr js.IExpr) { n.X = expr }, v, stack)
		walkScopedReplaceIExpr(n.Y, func(expr js.IExpr) { n.Y = expr }, v, stack)
	case *js.YieldExpr:
		walkScopedReplaceIExpr(n.X, func(expr js.IExpr) { n.X = expr }, v, stack)
	case *js.ArrowFunc:
		WalkScoped(v, &n.Body, stack)
		WalkScoped(v, &n.Params, stack)
	case *js.CommaExpr:
		for i, item := range n.List {
			walkScopedReplaceIExpr(item, func(expr js.IExpr) { n.List[i] = expr }, v, stack)
		}
	default:
		return
	}
}

func walkScopedReplaceIExpr(actual js.IExpr, replaceFn func(expr js.IExpr), v IVisitorScoped, stack *WalkScopeStack) {
	stack.Push(&WalkScope{
		replace: func(find js.INode, replaceBy js.INode) bool {
			if converted, isSameType := replaceBy.(js.IExpr); actual == find && isSameType {
				replaceFn(converted)
				return true
			}
			return false
		},
	})
	WalkScoped(v, actual, stack)
	stack.Pop()
}
