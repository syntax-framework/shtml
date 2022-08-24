package sht

import (
	"io/fs"
	"strings"
)

// https://code.tutsplus.com/tutorials/mastering-angularjs-directives--cms-22511

// DirectiveControllerFunc função executada para permitir controlar
type DirectiveControllerFunc func(scope *Scope)

// TranscludeFunc função disponível no método Process quando a directiva possui transclude
type TranscludeFunc func(slot string, preRender func(scope *Scope)) *Rendered

// DirectiveProcessFunc função executada antes da renderização do elemento associado a uma diretiva
// Quando a directiva é transclude, o conteúdo original é substituido em tempo de execução pelo Rendered retornado
type DirectiveProcessFunc func(scope *Scope, attr *Attributes, transclude TranscludeFunc) (rendered *Rendered)

// DirectiveLeaveFunc função executada após a renderização do elemento associado a uma diretiva
type DirectiveLeaveFunc func(scope *Scope)

// DirectiveCompileFunc uma funcão que visita um elemento html e pode realizar ajustes no template em tempo de compilação
type DirectiveCompileFunc func(node *Node, attrs *Attributes, c *Compiler) (*DirectiveMethods, error)

type DirectiveProcessInfo struct {
	name       string
	require    string
	isolate    bool // isolate scope
	terminal   bool // indica que é a função terminal que será executada
	callback   DirectiveProcessFunc
	transclude bool // quando true, o parametro transclude será criado para essa execuçao
}

type DirectiveLeaveInfo struct {
	name     string
	callback DirectiveLeaveFunc
}

// DirectiveRestrict The directive must be found in specific location.
type DirectiveRestrict uint8

const (
	ELEMENT   DirectiveRestrict = 1 << iota // element name
	ATTRIBUTE                               // attribute
)

type DirectiveMethods struct {
	Controller DirectiveControllerFunc
	Process    DirectiveProcessFunc
	Leave      DirectiveLeaveFunc
}

// Directive @TODO: Salvar a referencia de todas as diretivas cadastradas, não permitir que a mesma diretiva seja redefinida ou
// recarregada. Sempre usar a mesma implementação existente em memória.
// Todas as diretivas que possuem conteúdo devem resultar em um Compiled
type Directive struct {
	Name string
	// When there are multiple Directives defined on a single HTML node, sometimes it is necessary to specify the order
	// in which the Directives are applied. The Priority is used to sort the Directives before their Compile functions
	// get called. Directive with greater numerical priority are compiled first. The default priority is 0.
	Priority int
	Restrict DirectiveRestrict
	// Quando possuir template, a diretiva é Terminal
	Template     string
	TemplatePath string
	// Assets e acesso ao fs.Sys, um diretório para permitir carregamento em tempo de compilação
	Assets fs.FS
	// true - transclude the transcludeSlots (i.e. the child nodes) of the directive's element.
	// 'element' - transclude the whole of the directive's element including any directives on this element that are
	// defined at a lower priority than this directive. When used, the template property is ignored.
	// {...} (an object hash): - map elements of the transcludeSlots onto transclusion "slots" in the template.
	Transclude interface{} // true, false, map[string]string
	// If set to true then the current priority will be the last set of Directives which will execute (any Directive at
	// the current priority will still execute as the order of execution on same priority is undefined).
	// Note that expressions and other Directive used in the directive's template will also be excluded from execution.
	// Quando a directiva é terminal:
	//  1. O Transclude fica disponível para o método Process
	//  2. O método Process pode retornar um *Rendered
	Terminal bool
	// false (default): No scope will be created for the directive. The directive will use its root's scope.
	// true: A new child scope that prototypically inherits from its root will be created for the directive's element.
	// If multiple Directives on the same element request a new scope, only one new scope is created.
	Scope      bool
	Compile    DirectiveCompileFunc
	Controller DirectiveControllerFunc
	Process    DirectiveProcessFunc
	Leave      DirectiveLeaveFunc
}

func (d *Directive) Normalize() {
	d.Name = strings.ToLower(strings.TrimSpace(d.Name))
	if d.Priority < 0 {
		d.Priority = 0
	}
}

// DynamicDirectives parte dinamica que executa as diretivas de um Node
type DynamicDirectives struct {
	tag        string
	attrs      *Attributes // template attrs
	scope      bool
	terminal   bool
	transclude bool
	//templateOnThisElement    bool
	//newScopeDirective        bool
	//newIsolateScopeDirective bool
	//transclude               *_TranscludeFn
	//transcludeFn             *_TranscludeFn
	//isComposite              bool
	//composite                []*_RenderComposite
	process         []*DirectiveProcessInfo
	leave           []*DirectiveLeaveInfo
	transcludeSlots map[string]*Compiled
}

//Compile    DirectiveCompileFunc
//Process    DirectiveProcessFunc
//Leave      DirectiveLeaveFunc

func (nd *DynamicDirectives) Exec(scope *Scope) interface{} {
	attrs := nd.attrs.Clone()

	//for each Controller
	//  exec Controller()
	//end

	var rendered *Rendered

	// PROCESS
	if !nd.transclude {
		for _, it := range nd.process {
			// @TODO: Receber scope da execução
			//fnScope := NewRootScope()
			it.callback(scope, attrs, nil)
		}
	} else {
		for _, directive := range nd.process {
			// @TODO: Receber scope da execução
			//fnScope := NewRootScope()
			if !directive.transclude {
				directive.callback(scope, attrs, nil)
			} else {
				rendered = directive.callback(scope, attrs, createTranscludeFn(scope, nd.transcludeSlots))
			}
		}
	}

	// RECURSION
	//if !nd.terminal {
	//	println("recursion")
	//}

	// LEAVE
	for _, directive := range nd.leave {
		//fnScope := NewRootScope()
		directive.callback(scope)
	}

	if nd.transclude {
		return rendered
	} else {
		return attrs.Render()
	}
}

func createTranscludeFn(scope *Scope, slots map[string]*Compiled) TranscludeFunc {
	// @todo: check if it causes any performance issues
	return func(slot string, preRender func(scope *Scope)) *Rendered {
		scopeT := scope
		if slot == "" {
			slot = "*"
		}

		compiled, exist := slots[slot]
		if !exist {
			return nil
		}

		if preRender != nil {
			scopeT = scopeT.New(false, nil)
			preRender(scopeT)
		}

		return compiled.Exec(scopeT)
	}
}
