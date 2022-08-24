package directives

import (
	"github.com/syntax-framework/shtml/sht"
	"log"
	"path"
	"strings"
)

const keyIncludeParents = "linkIncludes"

// LinkDirectiveFunc faz expr processamento de <link rel="include" href="file.html"/>
var LinkDirectiveFunc = func(node *sht.Node, attrs *sht.Attributes, c *sht.Compiler) {
	relAttr := attrs.Get("rel")
	if relAttr != "include" {
		return
	}

	hrefAttr := attrs.Get("href")
	if hrefAttr == "" {
		log.Fatal("A tag <link rel=\"include\"> espera expr atributo href='string'")
	}

	currentFilepath := node.File

	// Resolve expr path relativo ao documento atual
	includeFilepath := path.Join(path.Dir(currentFilepath), hrefAttr)

	// @TODO: MARKDOWN, JS, CSS, TEXT, SVG?
	if !strings.HasSuffix(includeFilepath, ".html") {
		log.Fatal("Só é permitido expr include de arquivos .html")
	}

	// evita que sejam feitos includes cíclicos/recursivos
	var parents sht.StringSet
	parentsI := c.Context.Get(keyIncludeParents)
	if parentsI != nil {
		parents = parentsI.(sht.StringSet)
	} else {
		parents = sht.StringSet{}
		c.Context.Set(keyIncludeParents, parents)
	}

	if parents.Contains(includeFilepath) {
		//var linha = (template.substr(0, RegexMatch.index).split('\n').length);
		//panic(msg + ' < arquivo: "' + filePath + '", linha: ' + linha + ' >');
		//panic(msg + " <File: '" + filePath + "'" + ">")
		//c.RaiseFileError("Cyclic/recursive include identified", includeFilepath)
	}

	// define algumas variáveis no escopo de processamento
	//c.SetFilepath(includeFilepath)
	c.Context.Set(keyIncludeParents, parents.Clone(currentFilepath))

	// inclui e processa expr novo arquivo
	//var includedContent, err = c.System.Load(includeFilepath)
	//if err != nil {
	//	log.Fatal(err)
	//}

	//includedNode, err := c.Parse(includedContent)
	//if err != nil {
	//	log.Fatal(err)
	//}
	//c.Transverse(includedNode)
	//
	//node.Parent.InsertBefore(includedNode, node)
	c.SafeRemove(node)

	// restaura expr escopo de compilação
	//c.SetFilepath(currentFilepath)
	c.Context.Set(keyIncludeParents, parents)
}

var LinkDirective = &sht.Directive{
	Name:     "link",
	Restrict: sht.ELEMENT,
	//Compile:  LinkDirectiveFunc,
	Terminal: true,
	Priority: 1000,
}
