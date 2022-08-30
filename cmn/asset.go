package cmn

import (
	"bytes"
	"sort"
)

// AssetType represents the resource types handled and known by the syntax
type AssetType uint

const (
	Javascript AssetType = iota
	Stylesheet
)

// Asset a resource used by a template and mapped by syntax
type Asset struct {
	Content        []byte
	Name           string // Unique, non-conflicting name
	Size           int64
	Etag           string
	Url            string
	Type           AssetType
	Integrity      string // https://developer.mozilla.org/en-US/docs/Web/Security/Subresource_Integrity
	CrossOrigin    string
	ReferrerPolicy string
	Filepath       string   // When the asset is a file in the file system
	Dependencies   []*Asset // Dependencies of the asset (imports on js)
	Priority       int      // Developer can set an asset loading priority
}

// Assets utility to resolve dependencies between resources
type Assets []*Asset

// Resolve returns the topological order of the directed acyclic graph (DAG)
//
// https://guides.codepath.com/compsci/Graphs
// https://en.wikipedia.org/wiki/Topological_sorting
func (a Assets) Resolve() ([]*Asset, error) {

	g := &graph{
		set:   map[*Asset]bool{},
		paths: map[*Asset]map[*Asset][]*Asset{},
	}
	for _, node := range a {
		if g.add(node, nil) {
			return nil, errorGraphCircular(
				g.errPathDirect[0].Name,
				g.errPathReverse[0].Name,
				debugNodes(g.errPathDirect, " -> "),
				debugNodes(g.errPathReverse, " -> "),
			)
		}
	}

	nodeList := g.list

	// sort so that the dependencies are at the beginning of the list
	sort.Slice(nodeList, func(i, j int) bool {
		A := nodeList[i]
		B := nodeList[j]
		aDependencies, aHasDependencies := g.paths[A]
		if aHasDependencies {
			if _, aDependsOnB := aDependencies[B]; aDependsOnB {
				// `A` depends on `B`, `B` need to be loaded first
				return false
			}
		}

		bDependencies, bHasDependencies := g.paths[B]
		if bHasDependencies {
			if _, bDependsOnA := bDependencies[A]; bDependsOnA {
				// `B` depends on `A`, `A` need to be loaded first
				return true
			}
		}

		// tiebreaker by priority criteria
		return A.Priority >= B.Priority
	})

	return nodeList, nil
}

var errorGraphCircular = Err(
	"asset.graph.circulardep",
	"Circular dependency between two nodes was identified", "A: '%s'", "B: '%s'", "PathA: '%s'", "PathB: '%s'",
)

// dependency graph (DAG)
type graph struct {
	list           []*Asset                       // ALL nodes in this graph, including dependencies
	set            map[*Asset]bool                // ALL nodes in this graph, including dependencies
	paths          map[*Asset]map[*Asset][]*Asset // [FROM][TO] => PATH
	err            bool                           // Has circular dependency
	errPathDirect  []*Asset                       // Circular dependency path
	errPathReverse []*Asset                       // Inverse path of circular dependency
}

// adds a node and checks if there is a circular dependency
func (g *graph) add(node *Asset, parentPath []*Asset) bool {
	if g.err {
		return true
	}

	if parentPath == nil {
		parentPath = []*Asset{}
	}
	for i := 0; i < len(parentPath); i++ {
		// records the path between each parent and the current node
		g.setPath(parentPath[i], node, append(parentPath[i:], node))
	}

	if g.set[node] != true {
		// Already processed this node
		g.list = append(g.list, node)
		g.paths[node] = map[*Asset][]*Asset{}
	}
	g.set[node] = true

	nodePath := append(parentPath, node)
	for _, dependency := range node.Dependencies {
		// circular dependency
		reverse := g.getPath(dependency, node)
		if reverse != nil {
			g.err = true
			g.errPathDirect = []*Asset{node, dependency}
			g.errPathReverse = reverse
			return true
		}

		if g.add(dependency, nodePath) {
			break
		}
	}

	return false
}

func (g *graph) getPath(from, to *Asset) []*Asset {
	if pFrom, existsFrom := g.paths[from]; existsFrom {
		if pTo, existsTo := pFrom[to]; existsTo {
			return pTo
		}
	}
	return nil
}

func (g *graph) setPath(from, to *Asset, path []*Asset) {
	current := g.paths[from][to]
	if current != nil && len(current) < len(path) {
		// already has a shortest path between the two nodes
		return
	}
	g.paths[from][to] = path
}

// debugNodes debug a path
func debugNodes(nodes []*Asset, separator string) string {
	buf := &bytes.Buffer{}
	for i, node := range nodes {
		if i > 0 {
			buf.WriteString(separator)
		}
		buf.WriteString(node.Name)
	}
	return buf.String()
}
