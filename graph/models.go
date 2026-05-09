package graph

// Node is the base type for all elements in the architecture graph.
type Node interface {
	Name() string
	Downstreams() []Node
	Upstreams() []Node
	Base() Node
}

type Package struct {
	Node
}

type Domain struct {
	Node
}

type Component struct{ Node }

type Service struct{ Component }
type Infra struct{ Component }

type node struct {
	name        string
	downstreams []Node
	upstreams   []Node
}

func (n *node) Name() string {
	return n.name
}

func (n *node) Base() Node {
	return n
}

func (n *node) Downstreams() []Node {
	return n.downstreams
}

func (n *node) Upstreams() []Node {
	return n.upstreams
}
