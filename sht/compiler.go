package sht

import (
	"bytes"
	"github.com/syntax-framework/shtml/cmn"
	"log"
	"math"
	"regexp"
	"strconv"
)

// Compiler scope of html template being compiled
type Compiler struct {
	System     *TemplateSystem     // Reference to the instance that generated this TemplateSystem compiler
	Directives *Directives         // The directives registered to run in this build
	Assets     map[*cmn.Asset]bool // The possible resources that will be used by this template
	Context    *Context            // allows directives to save context information during compilation
	dynamics   []Dynamic
	sequence   *Sequence
}

// _PrevContext used for previous compilation of the current node
type _PrevContext struct {
	//transcludeDirective           *Directive
	//hasElementTranscludeDirective bool
	//needsNewScope                 bool
	//newScopeDirective             string
	//controllerDirectives          string
	//newIsolateScopeDirective      string
	//templateDirective             string
	MaxPriority int
}

// syntaxDynamicIndexStr used to mark dynamic content locations in html
var syntaxDynamicIndexStr = "____sdi__"

// syntaxDynamicIndexRegex extra space, to be compatible with text and attributes
var syntaxDynamicIndexRegex = regexp.MustCompile("\\s" + syntaxDynamicIndexStr + `([0-9]+)__(="")?`)

func NewCompiler(ts *TemplateSystem) *Compiler {
	return &Compiler{
		System:     ts,
		Directives: ts.Directives.NewChild(),
		dynamics:   []Dynamic{},
		Context:    NewContext(),
	}
}

func (c *Compiler) Compile(template string, filepath string) (*Compiled, error) {
	nodeList, err := Parse(template, filepath)
	if err != nil {
		return nil, err
	}
	return c.compile(nodeList, nil)
}

func (c *Compiler) IsComponente(child *Node) bool {
	// @TODO: Descobir se o node é um Componente registrado
	return false
}

// NextHash Used by components to predictively obtain a hash
func (c *Compiler) NextHash() string {
	if c.sequence == nil {
		c.sequence = &Sequence{}
	}
	return c.sequence.NextHash("")
}

// RegisterAssetJsContent register an anonymous javascript that can be used in this template
func (c *Compiler) RegisterAssetJsContent(content string) *cmn.Asset {
	name := HashXXH64(content)
	asset := &cmn.Asset{
		Content: content,
		Name:    name,
		Etag:    name,
		Size:    int64(len([]byte(content))),
		Type:    cmn.Javascript,
	}

	c.RegisterAsset(asset)

	return asset
}

// RegisterAsset register an asset that can be used in this template
func (c *Compiler) RegisterAsset(asset *cmn.Asset) {
	if c.Assets == nil {
		c.Assets = map[*cmn.Asset]bool{}
	}
	c.Assets[asset] = true
}

// ExtractRoot remove o node do root atual e retorna um novo root para os filhos do node atual
func (c *Compiler) ExtractRoot(node *Node) *Node {

	parent := &Node{Type: DocumentNode}

	parent.FirstChild = node.FirstChild
	parent.LastChild = node.LastChild

	for n := node.FirstChild; n != nil; n = n.NextSibling {
		n.Parent = parent
	}

	node.FirstChild = nil
	node.LastChild = nil

	// remove referencias para filhos
	return parent
}

// SafeRemove remove o node de forma segura
func (c *Compiler) SafeRemove(node *Node) {
	node.Type = TextNode
	node.Data = ""
	node.AttrList = []*Attribute{}
	if node.FirstChild != nil {
		node.FirstChild.Parent = nil
		node.FirstChild = nil
	}
	if node.LastChild != nil {
		node.LastChild.Parent = nil
		node.LastChild = nil
	}
}

// Transverse run callback for node and all its children, until callback returns true
//
// @TODO: Alterar o algoritmo para o formato enter() exit().
// transverse(func(node, enter){ /*before*/ enter() /*after*/  })
func (c *Compiler) Transverse(node *Node, callback func(node *Node) (stop bool)) {
	var f func(*Node)
	f = func(n *Node) {
		if callback(n) {
			return
		}
		for d := n.FirstChild; d != nil; d = d.NextSibling {
			f(d)
		}
	}
	f(node)
}

func (c *Compiler) compileNode(node *Node, context *_PrevContext) (*Compiled, error) {
	return c.compile([]*Node{node}, context)
}

// compile compile internal
func (c *Compiler) compile(nodeList []*Node, context *_PrevContext) (*Compiled, error) {
	if err := c.processNodes(nodeList, context); err != nil {
		return nil, err
	}
	compiled := c.extractCompiled(nodeList)
	return compiled, nil
}

// processNodes faz a compilação do nodeList
func (c *Compiler) processNodes(nodeList []*Node, prevContext *_PrevContext) error {
	for _, node := range nodeList {
		if node.Type == ElementNode {
			attrs := node.Attributes

			var err error
			var dynamic *DynamicDirectives

			// get the directives that can be applied on that node
			directives := c.Directives.collect(node, attrs)
			if len(directives) > 0 {
				dynamic, err = c.compileDirectives(directives, node, attrs, prevContext)
				if err != nil {
					return err
				}
			}

			if dynamic != nil && dynamic.transclude {
				c.replaceNodeByDynamic(node, dynamic)

			} else {
				if dynamic != nil {
					// replace attributes
					_, token := c.addDynamic(dynamic)
					node.AttrList = []*Attribute{{Name: token}}
				}

				childNodes := node.GetChildNodes()
				if childNodes != nil || len(childNodes) > 0 {
					err = c.processNodes(childNodes, prevContext)
					if err != nil {
						return err
					}
				}
			}
		} else if node.Type == TextNode {
			c.compileTextNode(node)
		}
	}
	return nil
}

// compileTextNode verifica se um node do tipo TextNode possui conteúdo dinamico e faz sua compilação
func (c *Compiler) compileTextNode(node *Node) {
	text := node.Data
	compiled, err := Interpolate(text)
	if err != nil {
		log.Fatal(err)
	}

	// no interpolation found -> ignore
	if compiled == nil {
		return
	}

	out := &bytes.Buffer{}
	for i := 0; i < len(compiled.static); i++ {
		if i == 0 {
			out.WriteString(compiled.static[i])
		} else {
			_, token := c.addDynamic(compiled.dynamics[i-1])
			out.WriteString(" " + token) // extra space, see syntaxDynamicIndexRegex
			out.WriteString(compiled.static[i])
		}
	}
	node.Data = out.String()
}

// faz a renderização do Node e transforma-o em um Compiled
func (c *Compiler) extractCompiled(nodeList []*Node) *Compiled {

	var prev *Node
	root := &Node{Type: DocumentNode}
	for _, node := range nodeList {
		node.Parent = root
		if prev == nil {
			root.FirstChild = node
		} else {
			prev.NextSibling = node
			node.PrevSibling = prev
		}
		prev = node
	}

	htmlStr, err := root.Render()
	if err != nil {
		log.Fatal(err)
	}

	// here it does the second phase of processing, fetches the tokens and generates the final executable
	matches := RegexExecAll(syntaxDynamicIndexRegex, htmlStr)

	compiled := &Compiled{}
	var static []string
	var dynamics []Dynamic

	if len(matches) == 0 {
		// pure string
		static = append(static, htmlStr)
	} else {

		start := 0
		for _, match := range matches {
			static = append(static, htmlStr[start:match.start])
			start = match.end

			var dynamicIndex, _ = strconv.Atoi(match.group[1])
			dynamics = append(dynamics, c.dynamics[dynamicIndex])
		}

		static = append(static, htmlStr[start:])
	}

	compiled.static = static
	compiled.dynamics = dynamics
	return compiled
}

//func assertNoDuplicate(what string, previousDirective *Directive, directive *Directive, element *Node) {
//	if previousDirective != nil {
//		log.Fatal("Multiple directives [{0}, {1}] asking for {3} on: {5}")
//		// previousDirective.Name, directive.name, what, startingTag(element)
//	}
//}

// Once the directives have been collected, their compile functions are executed. This method
// is responsible for inlining directive templates as well as terminating the application
// of the directives if the terminal directive has been reached.
func (c *Compiler) compileDirectives(
	directives []*Directive, node *Node, attrs *Attributes, prevContext *_PrevContext,
) (*DynamicDirectives, error) {

	terminalPriority := math.MinInt

	if prevContext == nil {
		prevContext = &_PrevContext{}
	}

	tag := ""
	if node.Type == ElementNode {
		tag = node.Data
	}

	var leaveInfos []*DirectiveLeaveInfo
	var processInfos []*DirectiveProcessInfo

	dynamic := &DynamicDirectives{
		tag:   tag,
		attrs: attrs,
	}

	//hasTranscludeDirective := false

	// executes all directives on the current element
	for _, directive := range directives {
		if terminalPriority > directive.Priority {
			break // prevent further processing of directives
		}

		leaveFunc := directive.Leave
		processFunc := directive.Process
		transclude := directive.Transclude

		if directive.Compile != nil {
			methods, err := directive.Compile(node, attrs, c)
			if err != nil {
				return nil, err
			}
			if methods != nil {
				if methods.Process != nil {
					processFunc = methods.Process
				}
				if methods.Leave != nil {
					leaveFunc = methods.Leave
				}
			}
		}

		transcludeOnThisDirective := false

		if transclude != nil {
			//hasTranscludeDirective = true

			dynamic.transclude = true
			transcludeOnThisDirective = true

			if transclude == "element" {
				terminalPriority = directive.Priority

				contentCompiled, err := c.compileNode(node, &_PrevContext{
					MaxPriority: terminalPriority,
					// ignoreDirective
					// transcludeFn
				})
				if err != nil {
					log.Fatal(err)
				}
				dynamic.transcludeSlots = map[string]*Compiled{"*": contentCompiled}

				// childTranscludeFn = return compile($compileNodes, transcludeFn, MaxPriority, ignoreDirective, previousCompileContext);

			} else {
				if transclude == true {
					childNodes := node.GetChildNodes()
					if childNodes != nil || len(childNodes) > 0 {
						c.processNodes(childNodes, prevContext)
					}
					contentCompiled, err := c.compileNode(c.ExtractRoot(node), &_PrevContext{
						MaxPriority: terminalPriority,
						// ignoreDirective
						// transcludeFn
						// {needsNewScope: directive.$$isolateScope || directive.$$newScope}
					})
					if err != nil {
						log.Fatal(err)
					}
					dynamic.transcludeSlots = map[string]*Compiled{"*": contentCompiled}
				} else if config, ok := transclude.(map[string]string); ok {
					println(config)
				} else {
					log.Fatal("@TODO: Transclude inválido")
				}
			}
		}

		if processFunc != nil {
			processInfos = append(processInfos, &DirectiveProcessInfo{
				name:       directive.Name,
				callback:   processFunc,
				terminal:   directive.Terminal,
				transclude: transcludeOnThisDirective,
			})
		}

		if leaveFunc != nil {
			leaveInfos = append(leaveInfos, &DirectiveLeaveInfo{
				name:     directive.Name,
				callback: leaveFunc,
			})
		}

		if directive.Terminal {
			dynamic.terminal = true
			if directive.Priority > terminalPriority {
				terminalPriority = directive.Priority
			}
		}
	}

	dynamic.leave = leaveInfos
	dynamic.process = processInfos

	return dynamic, nil
}

// addDynamic adiciona um dynamic no contexto de compilação e retorna seu índice e identificador
func (c *Compiler) addDynamic(dynamic Dynamic) (int, string) {
	index := len(c.dynamics)
	c.dynamics = append(c.dynamics, dynamic)
	return index, syntaxDynamicIndexStr + strconv.Itoa(index) + "__"
}

// replaceNodeByDynamic substitui um node html por um comando dinamico executável
func (c *Compiler) replaceNodeByDynamic(node *Node, dynamic Dynamic) {
	_, token := c.addDynamic(dynamic)

	// B) substitui o node por um comentário, que será processado no próximo passo
	node.Type = TextNode
	node.Data = " " + token // extra space, see syntaxDynamicIndexRegex

	// deixa todos os filhos órfãos, para não serem renderizados
	for n := node.FirstChild; n != nil; n = n.NextSibling {
		n.Parent = nil
	}

	// remove referencias para filhos
	node.FirstChild = nil
	node.LastChild = nil
}
