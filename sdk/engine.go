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
			downstream := collectCollaborations(n, opts.NestedLevels,
				func(c graph.Component) []graph.Collaboration { return c.Collaborations() },
				func(c graph.Collaboration) graph.Component { return c.Target })
			upstream := collectCollaborations(n, opts.UpperLevels,
				func(c graph.Component) []graph.Collaboration { return buildReverseMap(g)[c] },
				func(c graph.Collaboration) graph.Component { return c.Source })
			return &filteredComponent{
				Component:      n,
				collaborations: append(downstream, upstream...),
			}, nil
		}
	}
	return nil, ErrNotFound
}

type filteredComponent struct {
	graph.Component
	collaborations []graph.Collaboration
}

func (f *filteredComponent) Collaborations() []graph.Collaboration { return f.collaborations }

func (e *documentationEngine) ListAllDomains() ([]graph.Component, error) {
	var all []graph.Component
	for _, g := range e.domainGraphs {
		all = append(all, g.AllNodes()...)
	}
	return all, nil
}

func (e *documentationEngine) ListFeatures() ([]graph.Feature, error) {
	seen := make(map[string]bool)
	var features []graph.Feature
	for _, g := range e.graphs {
		for _, c := range g.Collaborations() {
			if c.Feature.Name != "" && !seen[c.Feature.Name] {
				seen[c.Feature.Name] = true
				features = append(features, c.Feature)
			}
		}
	}
	return features, nil
}

func (e *documentationEngine) FindByFeature(name string) ([]graph.Component, error) {
	seen := make(map[string]bool)
	var components []graph.Component
	for _, g := range e.graphs {
		for _, c := range g.Collaborations() {
			if c.Feature.Name != name {
				continue
			}
			if sn := g.QualifiedNameOf(c.Source); sn != "" && !seen[sn] {
				seen[sn] = true
				components = append(components, c.Source)
			}
			if tn := g.QualifiedNameOf(c.Target); tn != "" && !seen[tn] {
				seen[tn] = true
				components = append(components, c.Target)
			}
		}
	}
	if len(components) == 0 {
		return nil, ErrNotFound
	}
	return components, nil
}

func collectCollaborations(root graph.Component, levels int, getCollabs func(graph.Component) []graph.Collaboration, getNext func(graph.Collaboration) graph.Component) []graph.Collaboration {
	if levels <= 0 {
		return getCollabs(root)
	}

	seen := make(map[graph.Component]bool)
	seen[root] = true
	current := []graph.Component{root}
	var result []graph.Collaboration

	for level := 0; level < levels && len(current) > 0; level++ {
		var next []graph.Component
		for _, c := range current {
			for _, collab := range getCollabs(c) {
				result = append(result, collab)
				neighbor := getNext(collab)
				if !seen[neighbor] {
					seen[neighbor] = true
					next = append(next, neighbor)
				}
			}
		}
		current = next
	}

	return result
}

func buildReverseMap(g *graph.Graph) map[graph.Component][]graph.Collaboration {
	rm := make(map[graph.Component][]graph.Collaboration)
	for _, c := range g.Collaborations() {
		rm[c.Target] = append(rm[c.Target], c)
	}
	return rm
}
