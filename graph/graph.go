package graph

import "strings"

type Graph struct {
	nodes  map[string]Node
	nodeQN map[Node]string
}

func NewGraph() *Graph {
	return &Graph{
		nodes:  make(map[string]Node),
		nodeQN: make(map[Node]string),
	}
}

func (g *Graph) QualifiedNameOf(n Node) string {
	return g.nodeQN[n]
}

func (g *Graph) Register(qn string, n Node) {
	g.nodes[qn] = n
	g.nodeQN[n] = qn
}

func (g *Graph) AddDownstream(source, target Node) {
	sn := source.Base().(*node)
	tn := target.Base().(*node)
	sn.downstreams = append(sn.downstreams, target)
	tn.upstreams = append(tn.upstreams, source)
}

func (g *Graph) AllNodes() []Node {
	nodes := make([]Node, 0, len(g.nodes))
	for _, n := range g.nodes {
		nodes = append(nodes, n)
	}
	return nodes
}

func (g *Graph) GetNode(qualifiedName string) (Node, bool) {
	n, ok := g.nodes[qualifiedName]
	return n, ok
}

// PackageOf derives the package name from a node's qualified name.
// Returns "" for top-level nodes (Package/Domain).
func (g *Graph) PackageOf(n Node) string {
	qn := g.nodeQN[n]
	if i := strings.LastIndex(qn, "."); i >= 0 {
		return qn[:i]
	}
	return ""
}

// Constructors

func NewPackage(name string) Package {
	return Package{&node{name: name}}
}

func NewDomain(name string) Domain {
	return Domain{&node{name: name}}
}

func NewComponent(name string) Component {
	return Component{&node{name: name}}
}

func NewService(name string) Service {
	return Service{Component{&node{name: name}}}
}

func NewInfra(name string) Infra {
	return Infra{Component{&node{name: name}}}
}
