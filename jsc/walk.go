package jsc

import "github.com/tdewolff/parse/v2/js"

// VisitorEnterFunc use function as AST Visitor
//
// Each INode encountered by `walk` is passed to func, children nodes will be ignored if return false
type VisitorEnterFunc func(node js.INode) (visitChildren bool)

func (f VisitorEnterFunc) Enter(node js.INode) js.IVisitor {
	if f(node) {
		return f
	}
	return nil
}

func (f VisitorEnterFunc) Exit(node js.INode) {
}
