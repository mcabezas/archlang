package graph

import (
	"fmt"

	"github.com/mcabezas/archlang/internal/ast"
)

type NodeKind string

const (
	KindComponent NodeKind = "component"
	KindService   NodeKind = "service"
)

type Node struct {
	Name        string   `json:"name"`
	Kind        NodeKind `json:"kind"`
	Downstreams []string `json:"downstreams"`
	Upstreams   []string `json:"upstreams"`
}

type Graph struct {
	nodes map[string]*Node
}

func Build(arch *ast.Architecture) (*Graph, []string) {
	g := &Graph{nodes: make(map[string]*Node)}
	var errors []string

	// First pass: register all components and services
	for _, stmt := range arch.Statements {
		switch s := stmt.(type) {
		case *ast.ComponentStatement:
			if err := g.addNode(s.Name, KindComponent, s.Token.Line); err != "" {
				errors = append(errors, err)
			}
		case *ast.ServiceStatement:
			if err := g.addNode(s.Name, KindService, s.Token.Line); err != "" {
				errors = append(errors, err)
			}
		}
	}

	// Second pass: register collaborations
	for _, stmt := range arch.Statements {
		s, ok := stmt.(*ast.CollaborationStatement)
		if !ok {
			continue
		}

		source, sourceExists := g.nodes[s.Source]
		if !sourceExists {
			errors = append(errors, fmt.Sprintf(
				"line %d: undeclared component or service %q in collaboration",
				s.Token.Line, s.Source))
		}

		target, targetExists := g.nodes[s.Target]
		if !targetExists {
			errors = append(errors, fmt.Sprintf(
				"line %d: undeclared component or service %q in collaboration",
				s.Token.Line, s.Target))
		}

		if sourceExists && targetExists {
			source.Downstreams = append(source.Downstreams, s.Target)
			target.Upstreams = append(target.Upstreams, s.Source)
		}
	}

	return g, errors
}

func (g *Graph) addNode(name string, kind NodeKind, line int) string {
	if existing, ok := g.nodes[name]; ok {
		return fmt.Sprintf("line %d: duplicate declaration %q (already declared as %s)",
			line, name, existing.Kind)
	}
	g.nodes[name] = &Node{
		Name:        name,
		Kind:        kind,
		Downstreams: []string{},
		Upstreams:   []string{},
	}
	return ""
}

func (g *Graph) AllNodes() []*Node {
	nodes := make([]*Node, 0, len(g.nodes))
	for _, n := range g.nodes {
		nodes = append(nodes, n)
	}
	return nodes
}

func (g *Graph) Services() []*Node {
	var nodes []*Node
	for _, n := range g.nodes {
		if n.Kind == KindService {
			nodes = append(nodes, n)
		}
	}
	return nodes
}

func (g *Graph) GetNode(name string) (*Node, bool) {
	n, ok := g.nodes[name]
	return n, ok
}
