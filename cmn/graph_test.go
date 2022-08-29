package cmn

import (
  "strings"
  "testing"
)

// tnode represents a single node in the graph with it's dependencies
type tnode struct {
  name         string
  dependencies []*tnode
}

func (n *tnode) addDependency(dependency *tnode) {
  n.dependencies = append(n.dependencies, dependency)
}

func (n *tnode) GetKey() string {
  return n.name
}

func (n *tnode) GetDependencies() []GNode {
  var out []GNode
  for _, dep := range n.dependencies {
    out = append(out, dep)
  }
  return out
}

// newTestNode creates a new node
func newTestNode(name string, deps ...*tnode) *tnode {
  return &tnode{
    name:         name,
    dependencies: deps,
  }
}

// Quando o parse encontra um EmptyStmt, ou quando um EmptyStmt for adicionado de forma programática
func Test_Topological_Sort(t *testing.T) {
  var tests = []struct {
    input  string
    output string
  }{
    {" 2 | 3 | 5 | 7 | 8 | 9 | 10 | 11 | 5,11 | 7,11,8 | 3,8,10 | 8,9 | 11,2,9,10 ", "10, 9, 8, 3, 2, 11, 5, 7"},
  }
  for _, tt := range tests {
    t.Run(tt.input, func(t *testing.T) {

      var nodes []GNode
      byName := map[string]*tnode{}

      definitions := strings.Split(tt.input, "|")
      for _, def := range definitions {
        names := strings.Split(strings.TrimSpace(def), ",")
        var first *tnode
        for i, name := range names {
          name = strings.TrimSpace(name)
          node := byName[name]
          if node == nil {
            node = &tnode{name: name}
            byName[name] = node
            nodes = append(nodes, node)
          }
          if i == 0 {
            first = node
          } else {
            first.addDependency(node)
          }
        }
      }

      sorted, err := GraphResolveDependencies(nodes)
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

// Quando o parse encontra um EmptyStmt, ou quando um EmptyStmt for adicionado de forma programática
func Test_Circular_Dependencies(t *testing.T) {
  var tests = []struct{ input string }{
    {"A | B | C,A | D,B | E,C,D | F,A,B | G,E,F | H,G | A,H"}, // G -> E | E -> C -> A -> H -> G
    {"A | B | C,A | D,B | E,C,D | F,A,B | G,E,F | H,G | A,G"}, // A -> G | G -> E -> C -> A
  }
  for _, tt := range tests {
    t.Run(tt.input, func(t *testing.T) {
      var nodes []GNode
      byName := map[string]*tnode{}

      definitions := strings.Split(tt.input, "|")
      for _, def := range definitions {
        names := strings.Split(strings.TrimSpace(def), ",")
        var first *tnode
        for i, name := range names {
          name = strings.TrimSpace(name)
          node := byName[name]
          if node == nil {
            node = &tnode{name: name}
            byName[name] = node
            nodes = append(nodes, node)
          }
          if i == 0 {
            first = node
          } else {
            first.addDependency(node)
          }
        }
      }

      _, err := GraphResolveDependencies(nodes)
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
