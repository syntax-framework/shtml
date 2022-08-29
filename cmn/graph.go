package cmn

import (
	"bytes"
	"sort"
)

var errorGraphCircular = Err(
	"graph.circulardep",
	"Circular dependency between two nodes was identified", "A: '%s'", "B: '%s'", "PathA: '%s'", "PathB: '%s'",
)

type GNode interface {
	GetKey() string
	GetDependencies() []GNode
}

//
type graph struct {
	nodeList []GNode                     // ALL nodes in this graph, including dependencies
	nodeMap  map[GNode]bool              // ALL nodes in this graph, including dependencies
	path     map[GNode]map[GNode][]GNode // [FROM][TO] => PATH
	err      bool                        // Has circular dependency error
	errA     []GNode                     // Circular dependency path
	errB     []GNode                     // Inverse path of circular dependency
}

// adds a node and checks if there is a circular dependency
func (g *graph) add(node GNode, parentPath []GNode) bool {
	if g.err {
		return true
	}

	if parentPath == nil {
		parentPath = []GNode{}
	}
	for i := 0; i < len(parentPath); i++ {
		// records the path between each parent and the current node
		g.path[parentPath[i]][node] = append(parentPath[i:], node)
		//g.set(parents[i], node, append(parents[i:], node))
	}

	if g.nodeMap[node] != true {
		// Already processed this node
		g.nodeList = append(g.nodeList, node)
		g.path[node] = map[GNode][]GNode{}
	}
	g.nodeMap[node] = true
	//if _, contains := g.path[node]; !contains {
	//}

	nodePath := append(parentPath, node)
	for _, dependency := range node.GetDependencies() {
		// records the path between the current node and its dependencies
		//direct := []GNode{node, dependency}
		//g.path[node][dependency] = direct
		//g.set(node, dependency, direct)

		// circular dependency
		reverse := g.get(dependency, node)
		if reverse != nil {
			g.err = true
			g.errA = []GNode{node, dependency}
			g.errB = reverse
			return true
		}

		if g.add(dependency, nodePath) {
			break
		}
	}

	return false
}

func (g *graph) get(from, to GNode) []GNode {
	if pFrom, existsFrom := g.path[from]; existsFrom {
		if pTo, existsTo := pFrom[to]; existsTo {
			return pTo
		}
	}
	return nil
}

func (g *graph) set(from, to GNode, path []GNode) {
	current := g.path[from][to]
	if current != nil && len(current) < len(path) {
		// já tem um caminho mais curto entre os dois nós
		return
	}
	g.path[from][to] = path
}

// debugNodes debug a path
func debugNodes(nodes []GNode, separator string) string {
	buf := &bytes.Buffer{}
	for i, node := range nodes {
		if i > 0 {
			buf.WriteString(separator)
		}
		buf.WriteString(node.GetKey())
	}
	return buf.String()
}

// GraphResolveDependencies Topological ordering of a directed acyclic graph (DAG)
//
// https://guides.codepath.com/compsci/Graphs
// https://en.wikipedia.org/wiki/Topological_sorting
func GraphResolveDependencies(nodes []GNode) ([]GNode, error) {

	g := &graph{
		path:    map[GNode]map[GNode][]GNode{},
		nodeMap: map[GNode]bool{},
	}
	for _, node := range nodes {
		if g.add(node, nil) {
			return nil, errorGraphCircular(
				g.errA[0].GetKey(),
				g.errB[0].GetKey(),
				debugNodes(g.errA, " -> "),
				debugNodes(g.errB, " -> "),
			)
		}
	}

	nodeList := g.nodeList

	// faz a ordenação de forma que as dependencias estão no inicio da lista, e, o item que não possuir
	sort.Slice(nodeList, func(i, j int) bool {
		A := nodeList[i]
		B := nodeList[j]
		aDependencies, aHasDependencies := g.path[A]
		if aHasDependencies {
			if _, aDependsOnB := aDependencies[B]; aDependsOnB {
				// `A` depends on `B`, `B` need to be loaded first
				return false
			}
		}

		// `A` will be loaded first
		return true
	})

	return nodeList, nil
}
