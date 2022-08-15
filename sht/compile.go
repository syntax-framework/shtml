package sht

import (
	"bytes"
	"golang.org/x/net/html"
	"log"
	"math"
	"regexp"
	"strconv"
)

// Compiler escopo do template html sendo compilado
type Compiler struct {
	System     *TemplateSystem
	Directives *Directives
	filepath   string
	dynamics   []Dynamic
	data       map[string]interface{}
}

// CompileContext used for previous compilation of the current node
type CompileContext struct {
	transcludeDirective           *Directive
	hasElementTranscludeDirective bool
	needsNewScope                 bool
	newScopeDirective             string
	controllerDirectives          string
	newIsolateScopeDirective      string
	templateDirective             string
	maxPriority                   int
}

// syntaxDynamicIndexStr usado para marcar no html locais de conteúdo dinamico
var syntaxDynamicIndexStr = "____sdi__"

// syntaxDynamicIndexRegex extra space, to be compatible with text and attributes
var syntaxDynamicIndexRegex = regexp.MustCompile("\\s" + syntaxDynamicIndexStr + `([0-9]+)__(="")?`)

func NewCompiler(ts *TemplateSystem) *Compiler {
	return &Compiler{
		System:     ts,
		Directives: ts.Directives.NewChild(),
		dynamics:   []Dynamic{},
		data:       map[string]interface{}{},
	}
}

// Get obtém algum parametro do data
func (c *Compiler) Get(key string) (value interface{}) {
	value, exists := c.data[key]
	if !exists {
		return nil
	}
	return value
}

// Set Salva algum dado no data
func (c *Compiler) Set(key string, value interface{}) {
	c.data[key] = value
}

func (c *Compiler) Compile(content string) (*Compiled, error) {
	nodeList, err := ParseHtml(content)
	if err != nil {
		return nil, err
	}
	return c.compile(nodeList, nil)
}

// ExtractRoot remove o node do Root atual e retorna um novo Root para os filhos do node atual
func (c *Compiler) ExtractRoot(node *html.Node) *html.Node {

	parent := &html.Node{Type: html.DocumentNode}

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
func (c *Compiler) SafeRemove(node *html.Node) {
	node.Type = html.TextNode
	node.Data = ""
	node.Attr = []html.Attribute{}
	if node.FirstChild != nil {
		node.FirstChild.Parent = nil
		node.FirstChild = nil
	}
	if node.LastChild != nil {
		node.LastChild.Parent = nil
		node.LastChild = nil
	}
}

// RaiseFileError Permite disparar erro de processamento do arquivo, facilitando o desenvolvimento
func (c *Compiler) RaiseFileError(msg string, filePath string) {
	//var linha = (template.substr(0, RegexMatch.index).split('\n').length);
	//panic(msg + ' < arquivo: "' + filePath + '", linha: ' + linha + ' >');
	panic(msg + " <file: '" + filePath + "'" + ">")
}

// SetFilepath define qual arquivo está sendo processado
func (c *Compiler) SetFilepath(filepath string) {
	c.filepath = filepath
}

// GetFilepath obtém o caminho do arquivo sendo processado atualmente
func (c *Compiler) GetFilepath() string {
	return c.filepath
}

func (c *Compiler) compileNode(node *html.Node, context *CompileContext) (*Compiled, error) {
	return c.compile([]*html.Node{node}, context)
}

// compile compile internal
func (c *Compiler) compile(nodeList []*html.Node, context *CompileContext) (*Compiled, error) {
	c.processNodes(nodeList, context)
	compiled := c.extractCompiled(nodeList)
	return compiled, nil
}

// faz a compilação de todos os html.Node da lista
func (c *Compiler) processNodes(nodeList []*html.Node, prevContext *CompileContext) {
	for _, node := range nodeList {
		if node.Type == html.ElementNode {
			attrs := AttributesFromHtmlNode(node)

			var dynamic *DynamicDirectives

			// get the directives that can be applied on that node
			directives := c.Directives.collect(node, attrs)
			if len(directives) > 0 {
				dynamic = c.compileDirectives(directives, node, attrs, prevContext)
			}

			if dynamic != nil && dynamic.transclude {
				c.replaceNodeByDynamic(node, dynamic)

			} else {
				if dynamic != nil {
					// replace attributes
					_, token := c.addDynamic(dynamic)
					node.Attr = []html.Attribute{{Key: token}}
				}

				childNodes := getChildNodes(node)
				if childNodes != nil || len(childNodes) > 0 {
					c.processNodes(childNodes, prevContext)
				}
			}
		} else if node.Type == html.TextNode {
			c.compileTextNode(node)
		}
	}
}

// compileTextNode verifica se um node do tipo html.TextNode possui conteúdo dinamico e faz sua compilação
func (c *Compiler) compileTextNode(node *html.Node) {
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

// faz a renderização do html.Node e transforma-o em um Compiled
func (c *Compiler) extractCompiled(nodeList []*html.Node) *Compiled {

	prev := nodeList[0]
	doc := &html.Node{Type: html.DocumentNode, FirstChild: prev}
	for _, node := range nodeList {
		node.Parent = doc
		if prev != node {
			prev.NextSibling = node
			node.PrevSibling = prev
		}
	}

	bufferString := bytes.NewBufferString("")

	err := html.Render(bufferString, doc)
	if err != nil {
		log.Fatal(err)
	}
	htmlStr := bufferString.String()

	// aqui faz a segunda fase do processamento, busca os tokens e gera o executável final
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

//func assertNoDuplicate(what string, previousDirective *Directive, directive *Directive, element *html.Node) {
//	if previousDirective != nil {
//		log.Fatal("Multiple directives [{0}, {1}] asking for {3} on: {5}")
//		// previousDirective.Name, directive.name, what, startingTag(element)
//	}
//}

// Once the directives have been collected, their compile functions are executed. This method
// is responsible for inlining directive templates as well as terminating the application
// of the directives if the terminal directive has been reached.
func (c *Compiler) compileDirectives(
	directives []*Directive, node *html.Node, attrs *Attributes, prevContext *CompileContext,
) *DynamicDirectives {

	terminalPriority := math.MinInt

	if prevContext == nil {
		prevContext = &CompileContext{}
	}

	tag := ""
	if node.Type == html.ElementNode {
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
			methods := directive.Compile(node, attrs, c)
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

				contentCompiled, err := c.compileNode(node, &CompileContext{
					maxPriority: terminalPriority,
					// ignoreDirective
					// transcludeFn
				})
				if err != nil {
					log.Fatal(err)
				}
				dynamic.transcludeSlots = map[string]*Compiled{"*": contentCompiled}

				// childTranscludeFn = return compile($compileNodes, transcludeFn, maxPriority, ignoreDirective, previousCompileContext);

			} else {
				if transclude == true {
					childNodes := getChildNodes(node)
					if childNodes != nil || len(childNodes) > 0 {
						c.processNodes(childNodes, prevContext)
					}
					contentCompiled, err := c.compileNode(c.ExtractRoot(node), &CompileContext{
						maxPriority: terminalPriority,
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

	return dynamic
}

// addDynamic adiciona um dynamic no contexto de compilação e retorna seu índice e identificador
func (c *Compiler) addDynamic(dynamic Dynamic) (int, string) {
	index := len(c.dynamics)
	c.dynamics = append(c.dynamics, dynamic)
	return index, syntaxDynamicIndexStr + strconv.Itoa(index) + "__"
}

// replaceNodeByDynamic substitui um node html por um comando dinamico executável
func (c *Compiler) replaceNodeByDynamic(node *html.Node, dynamic Dynamic) {
	_, token := c.addDynamic(dynamic)

	// B) substitui o node por um comentário, que será processado no próximo passo
	node.Type = html.TextNode
	node.Data = " " + token // extra space, see syntaxDynamicIndexRegex

	// deixa todos os filhos órfãos, para não serem renderizados
	for n := node.FirstChild; n != nil; n = n.NextSibling {
		n.Parent = nil
	}

	// remove referencias para filhos
	node.FirstChild = nil
	node.LastChild = nil
}

func getChildNodes(node *html.Node) []*html.Node {
	var childNodes []*html.Node
	for child := node.FirstChild; child != nil; child = child.NextSibling {
		childNodes = append(childNodes, child)
	}
	return childNodes
}
