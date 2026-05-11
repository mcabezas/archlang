package knowledge

import (
	"github.com/mcabezas/archlang/graph"
)

type documentationEngine struct {
	graphs       []*graph.Graph
	domainGraphs []*graph.Graph
}

func New(graphs []*graph.Graph, domainGraphs []*graph.Graph) Storage {
	return &documentationEngine{graphs: graphs, domainGraphs: domainGraphs}
}

func (e *documentationEngine) ListAll() ([]graph.Component, error) {
	var all []graph.Component
	for _, g := range e.graphs {
		all = append(all, g.AllNodes()...)
	}
	return all, nil
}

func (e *documentationEngine) FindByName(name string, options ...ComponentFilterOption) (graph.Component, error) {
	return e.findBy(e.graphs, name, options...)
}

func (e *documentationEngine) FindByDomain(name string, options ...ComponentFilterOption) (graph.Component, error) {
	return e.findBy(e.domainGraphs, name, options...)
}

func (e *documentationEngine) findBy(graphs []*graph.Graph, name string, options ...ComponentFilterOption) (graph.Component, error) {
	opts := &ComponentFilterOptions{}
	for _, o := range options {
		o(opts)
	}

	for _, g := range graphs {
		if n, ok := g.GetNode(name); ok {
			return &filteredComponent{
				Component:   n,
				downstreams: collectLevels(n, opts.NestedLevels, func(c graph.Component) []graph.Component { return c.Downstreams() }),
				upstreams:   collectLevels(n, opts.UpperLevels, func(c graph.Component) []graph.Component { return c.Upstreams() }),
			}, nil
		}
	}
	return nil, ErrNotFound
}

type filteredComponent struct {
	graph.Component
	downstreams []graph.Component
	upstreams   []graph.Component
}

func (f *filteredComponent) Downstreams() []graph.Component { return f.downstreams }
func (f *filteredComponent) Upstreams() []graph.Component   { return f.upstreams }

func (e *documentationEngine) ListAllDomains() ([]graph.Component, error) {
	var all []graph.Component
	for _, g := range e.domainGraphs {
		all = append(all, g.AllNodes()...)
	}
	return all, nil
}

func collectLevels(root graph.Component, levels int, neighbors func(graph.Component) []graph.Component) []graph.Component {
	if levels <= 0 {
		return neighbors(root)
	}

	seen := make(map[graph.Component]bool)
	current := []graph.Component{root}
	var result []graph.Component

	for level := 0; level < levels && len(current) > 0; level++ {
		var next []graph.Component
		for _, c := range current {
			for _, n := range neighbors(c) {
				if !seen[n] {
					seen[n] = true
					result = append(result, n)
					next = append(next, n)
				}
			}
		}
		current = next
	}

	return result
}
