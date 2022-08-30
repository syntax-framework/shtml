package cmn

import (
	"strings"
	"testing"
)

type tnode struct {
	name         string
	dependencies []*tnode
}

func (n *tnode) addDependency(dependency *tnode) {
	n.dependencies = append(n.dependencies, dependency)
}

type ttype struct {
	input  string
	output string
}

func testParseGraphNodes(tt ttype) Assets {
	var nodes Assets
	byName := map[string]*Asset{}

	definitions := strings.Split(tt.input, "|")
	for _, def := range definitions {
		names := strings.Split(strings.TrimSpace(def), ",")
		var first *Asset
		for i, name := range names {
			name = strings.TrimSpace(name)
			node := byName[name]
			if node == nil {
				node = &Asset{Name: name}
				byName[name] = node
				nodes = append(nodes, node)
			}
			if i == 0 {
				first = node
			} else {
				first.Dependencies = append(first.Dependencies, node)
			}
		}
	}
	return nodes
}

func Test_Assets_Graph_Topological_Sort(t *testing.T) {

	var tests = []ttype{
		{" 2 | 3 | 5 | 7 | 8 | 9 | 10 | 11 | 5,11 | 7,11,8 | 3,8,10 | 8,9 | 11,2,9,10 ", "10, 9, 8, 3, 2, 11, 5, 7"},
		{" 1 | 2 | 3 | 4 | 5 | 6 | 7 | 8 | 1,2,3 | 2,4 | 4,5 | 6,5 | 3,5 | 7,8 ", "5, 6, 3, 4, 2, 1, 8, 7"},
	}
	for _, tt := range tests {
		t.Run(tt.output, func(t *testing.T) {
			nodes := testParseGraphNodes(tt)
			sorted, err := nodes.Resolve()
			if err != nil {
				t.Fatal(err)
			} else {
				if actual := debugNodes(sorted, ", "); actual != tt.output {
					t.Errorf("GraphResolveDependencies(nodes) | invalid output\n   actual: %q\n expected: %q", actual, tt.output)
				}
			}
		})
	}
}

func Test_Assets_Graph_Circular_Dependencies(t *testing.T) {
	var tests = []ttype{
		{"A | B | C,A | D,B | E,C,D | F,A,B | G,E,F | H,G | A,H", ""}, // G -> E | E -> C -> A -> H -> G
		{"A | B | C,A | D,B | E,C,D | F,A,B | G,E,F | H,G | A,G", ""}, // A -> G | G -> E -> C -> A
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			nodes := testParseGraphNodes(tt)
			_, err := nodes.Resolve()
			if err == nil {
				t.Errorf("graph.Resolve() | expect to receive error")
			} else {
				errStr := err.Error()
				if !strings.HasPrefix(errStr, "[graph.circulardep]") {
					t.Errorf("graph.Resolve() | invalid error\n expected: [graph.circulardep] .......\n   actual: %s", errStr)
				}
			}
		})
	}
}
