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
	collab := Collaboration{
		Source:      source,
		Target:      target,
		Cardinality: "1:1",
	}
	sn := source.Base().(*component)
	sn.collaborations = append(sn.collaborations, collab)
	g.collaborations = append(g.collaborations, collab)
}

func (g *Graph) AddCollaboration(source, target Component, feature Feature, description string, cardinality string, cardinalityBy string, flow Flow, step string, stepOrder int, execute string, deliveredBy *MessageBroker) *Collaboration {
	if cardinality == "" {
		cardinality = "1:1"
	}
	collab := Collaboration{
		Source:        source,
		Target:        target,
		Feature:       feature,
		Description:   description,
		Cardinality:   cardinality,
		CardinalityBy: cardinalityBy,
		Flow:          flow,
		Step:          step,
		StepOrder:     stepOrder,
		Execute:       execute,
		DeliveredBy:   deliveredBy,
	}
	sn := source.Base().(*component)
	sn.collaborations = append(sn.collaborations, collab)
	g.collaborations = append(g.collaborations, collab)
	return &g.collaborations[len(g.collaborations)-1]
}

func (g *Graph) LinkPublish(subscribe *Collaboration, publish *Collaboration) {
	subscribe.Publishes = append(subscribe.Publishes, publish)
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

// FindByShortName looks up a component by its short name (e.g. "core-api" instead of "orgs/modo.core-api").
func (g *Graph) FindByShortName(name string) (Component, bool) {
	for _, n := range g.nodes {
		if n.Name() == name {
			return n, true
		}
	}
	return nil, false
}
