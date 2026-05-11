package graph

type Graph struct {
	nodes          map[string]Component
	nodeQN         map[Component]string
	collaborations []Collaboration
}

func NewGraph() *Graph {
	return &Graph{
		nodes:  make(map[string]Component),
		nodeQN: make(map[Component]string),
	}
}

func (g *Graph) QualifiedNameOf(n Component) string {
	return g.nodeQN[n]
}

func (g *Graph) Register(qn string, n Component) {
	g.nodes[qn] = n
	g.nodeQN[n] = qn
}

func (g *Graph) AddDownstream(source, target Component) {
	sn := source.Base().(*component)
	tn := target.Base().(*component)
	sn.downstreams = append(sn.downstreams, target)
	tn.upstreams = append(tn.upstreams, source)
}

func (g *Graph) AddCollaboration(source, target Component, feature Feature, description string) {
	sn := source.Base().(*component)
	tn := target.Base().(*component)
	sn.downstreams = append(sn.downstreams, target)
	tn.upstreams = append(tn.upstreams, source)
	g.collaborations = append(g.collaborations, Collaboration{
		Source:      source,
		Target:      target,
		Feature:     feature,
		Description: description,
	})
}

func (g *Graph) Collaborations() []Collaboration {
	return g.collaborations
}

func (g *Graph) AllNodes() []Component {
	nodes := make([]Component, 0, len(g.nodes))
	for _, n := range g.nodes {
		nodes = append(nodes, n)
	}
	return nodes
}

func (g *Graph) GetNode(qualifiedName string) (Component, bool) {
	n, ok := g.nodes[qualifiedName]
	return n, ok
}
