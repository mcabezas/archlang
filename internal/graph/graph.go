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
	Package     string   `json:"package"`
	Kind        NodeKind `json:"kind"`
	Downstreams []string `json:"downstreams"`
	Upstreams   []string `json:"upstreams"`
}

func (n *Node) QualifiedName() string {
	return n.Package + "." + n.Name
}

type Graph struct {
	nodes    map[string]*Node // keyed by qualified name (pkg.name)
	packages map[string]bool
}

// Build takes a map of package name → parsed AST and produces a unified graph.
func Build(packages map[string]*ast.Architecture) (*Graph, []string) {
	g := &Graph{
		nodes:    make(map[string]*Node),
		packages: make(map[string]bool),
	}
	var errors []string

	for pkg := range packages {
		g.packages[pkg] = true
	}

	// First pass: register all components and services per package
	for pkg, arch := range packages {
		for _, stmt := range arch.Statements {
			switch s := stmt.(type) {
			case *ast.ComponentStatement:
				if err := g.addNode(pkg, s.Name, KindComponent, s.Token.Line); err != "" {
					errors = append(errors, fmt.Sprintf("%s: %s", pkg, err))
				}
			case *ast.ServiceStatement:
				if err := g.addNode(pkg, s.Name, KindService, s.Token.Line); err != "" {
					errors = append(errors, fmt.Sprintf("%s: %s", pkg, err))
				}
			}
		}
	}

	// Second pass: validate imports and wire collaborations
	for pkg, arch := range packages {
		imports := collectImports(arch)

		// Validate imports
		for _, imp := range imports {
			if !g.packages[imp.Package] {
				errors = append(errors, fmt.Sprintf(
					"%s: line %d: imported package %q does not exist",
					pkg, imp.Token.Line, imp.Package))
			}
		}

		aliasMap := buildAliasMap(imports)

		for _, stmt := range arch.Statements {
			s, ok := stmt.(*ast.CollaborationStatement)
			if !ok {
				continue
			}

			sourceQN, err := g.resolveRef(pkg, s.Source, aliasMap, s.Token.Line)
			if err != "" {
				errors = append(errors, fmt.Sprintf("%s: %s", pkg, err))
			}

			targetQN, err := g.resolveRef(pkg, s.Target, aliasMap, s.Token.Line)
			if err != "" {
				errors = append(errors, fmt.Sprintf("%s: %s", pkg, err))
			}

			if sourceQN != "" && targetQN != "" {
				g.nodes[sourceQN].Downstreams = append(g.nodes[sourceQN].Downstreams, targetQN)
				g.nodes[targetQN].Upstreams = append(g.nodes[targetQN].Upstreams, sourceQN)
			}
		}
	}

	return g, errors
}

func (g *Graph) resolveRef(currentPkg string, ref ast.ComponentRef, aliases map[string]string, line int) (string, string) {
	if ref.Package == "" {
		// Local reference
		qn := currentPkg + "." + ref.Name
		if _, ok := g.nodes[qn]; !ok {
			return "", fmt.Sprintf("line %d: undeclared %q in package %q", line, ref.Name, currentPkg)
		}
		return qn, ""
	}

	// Qualified reference: resolve alias to real package name
	realPkg, ok := aliases[ref.Package]
	if !ok {
		return "", fmt.Sprintf("line %d: package alias %q not imported", line, ref.Package)
	}

	qn := realPkg + "." + ref.Name
	if _, ok := g.nodes[qn]; !ok {
		return "", fmt.Sprintf("line %d: undeclared %q in package %q", line, ref.Name, realPkg)
	}
	return qn, ""
}

func collectImports(arch *ast.Architecture) []*ast.ImportStatement {
	var imports []*ast.ImportStatement
	for _, stmt := range arch.Statements {
		if imp, ok := stmt.(*ast.ImportStatement); ok {
			imports = append(imports, imp)
		}
	}
	return imports
}

func buildAliasMap(imports []*ast.ImportStatement) map[string]string {
	m := make(map[string]string)
	for _, imp := range imports {
		m[imp.Alias] = imp.Package
	}
	return m
}

func (g *Graph) addNode(pkg, name string, kind NodeKind, line int) string {
	qn := pkg + "." + name
	if existing, ok := g.nodes[qn]; ok {
		return fmt.Sprintf("line %d: duplicate declaration %q (already declared as %s)",
			line, name, existing.Kind)
	}
	g.nodes[qn] = &Node{
		Name:        name,
		Package:     pkg,
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

func (g *Graph) GetNode(qualifiedName string) (*Node, bool) {
	n, ok := g.nodes[qualifiedName]
	return n, ok
}

func (g *Graph) Packages() []string {
	pkgs := make([]string, 0, len(g.packages))
	for pkg := range g.packages {
		pkgs = append(pkgs, pkg)
	}
	return pkgs
}
