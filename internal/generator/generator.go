package generator

import (
	"bytes"
	"fmt"
	"go/format"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/mcabezas/archlang/internal/ast"
	"github.com/mcabezas/archlang/internal/lexer"
	"github.com/mcabezas/archlang/internal/parser"
)

// Generate discovers .arch files in dir, parses them, builds the graph,
// and produces Go source code containing the hardcoded graph.
func Generate(dir string, packageName string) ([]byte, error) {
	domains, err := discoverDomains(dir)
	if err != nil {
		return nil, err
	}

	if len(domains) == 0 {
		return nil, fmt.Errorf("no .arch files found in %q", dir)
	}

	parsed := make(map[string]*ast.Architecture)
	for domain, input := range domains {
		l := lexer.New(input)
		p := parser.New(l)
		arch := p.Parse()

		if len(p.Errors()) > 0 {
			return nil, fmt.Errorf("parse errors in domain %q:\n  %s", domain, strings.Join(p.Errors(), "\n  "))
		}
		parsed[domain] = arch
	}

	g, errs := buildGraph(parsed)
	if len(errs) > 0 {
		return nil, fmt.Errorf("compile errors:\n  %s", strings.Join(errs, "\n  "))
	}

	return generateCode(g, packageName)
}

// graphNode holds the information needed to generate code for a single node.
type graphNode struct {
	qualifiedName string
	name          string
	domain        string
	isService     bool
	isInfra       bool
	isPublic      bool
	downstreams   []string // qualified names
	upstreams     []string // qualified names
}

// builtGraph is the intermediate representation after building from ASTs.
type builtGraph struct {
	nodes map[string]*graphNode
	order []string // sorted qualified names
}

func buildGraph(allDomains map[string]*ast.Architecture) (*builtGraph, []string) {
	nodes := make(map[string]*graphNode)
	var errors []string

	// Register components, services, and infra
	for domain, arch := range allDomains {
		for _, stmt := range arch.Statements {
			switch s := stmt.(type) {
			case *ast.ComponentStatement:
				qn := domain + "." + s.Name
				if _, exists := nodes[qn]; exists {
					errors = append(errors, fmt.Sprintf("%s: line %d: duplicate declaration %q", domain, s.Token.Line, s.Name))
					continue
				}
				n := &graphNode{
					qualifiedName: qn,
					name:          s.Name,
					domain:        domain,
					isPublic:      s.Public,
				}
				if s.Infra != "" {
					n.isInfra = true
				}
				nodes[qn] = n
			case *ast.ServiceStatement:
				qn := domain + "." + s.Name
				if _, exists := nodes[qn]; exists {
					errors = append(errors, fmt.Sprintf("%s: line %d: duplicate declaration %q", domain, s.Token.Line, s.Name))
					continue
				}
				nodes[qn] = &graphNode{
					qualifiedName: qn,
					name:          s.Name,
					domain:        domain,
					isService:     true,
					isPublic:      s.Public,
				}
			case *ast.InfraStatement:
				qn := domain + "." + s.Name
				if _, exists := nodes[qn]; exists {
					errors = append(errors, fmt.Sprintf("%s: line %d: duplicate declaration %q", domain, s.Token.Line, s.Name))
					continue
				}
				nodes[qn] = &graphNode{
					qualifiedName: qn,
					name:          s.Name,
					domain:        domain,
					isInfra:       true,
					isPublic:      s.Public,
				}
			}
		}
	}

	// Validate imports and wire collaborations
	for domain, arch := range allDomains {
		imports := collectImports(arch)

		for _, imp := range imports {
			found := false
			for other := range allDomains {
				if other == imp.Domain {
					found = true
					break
				}
			}
			if !found {
				errors = append(errors, fmt.Sprintf(
					"%s: line %d: imported domain %q does not exist",
					domain, imp.Token.Line, imp.Domain))
			}
		}

		domainAliases := buildAliasMap(imports)

		for _, stmt := range arch.Statements {
			s, ok := stmt.(*ast.CollaborationStatement)
			if !ok {
				continue
			}

			sourceQN, err := resolveRef(domain, s.Source, domainAliases, s.Token.Line, nodes)
			if err != "" {
				errors = append(errors, fmt.Sprintf("%s: %s", domain, err))
			}

			targetQN, err := resolveRef(domain, s.Target, domainAliases, s.Token.Line, nodes)
			if err != "" {
				errors = append(errors, fmt.Sprintf("%s: %s", domain, err))
			}

			if sourceQN != "" && targetQN != "" {
				nodes[sourceQN].downstreams = append(nodes[sourceQN].downstreams, targetQN)
				nodes[targetQN].upstreams = append(nodes[targetQN].upstreams, sourceQN)
			}
		}
	}

	// Sort for deterministic output
	order := make([]string, 0, len(nodes))
	for qn := range nodes {
		order = append(order, qn)
	}
	sort.Strings(order)

	return &builtGraph{nodes: nodes, order: order}, errors
}

func resolveRef(currentDomain string, ref ast.ComponentRef, aliases map[string]string, line int, nodes map[string]*graphNode) (string, string) {
	if ref.Domain == "" {
		qn := currentDomain + "." + ref.Name
		if _, ok := nodes[qn]; !ok {
			return "", fmt.Sprintf("line %d: undeclared %q in domain %q", line, ref.Name, currentDomain)
		}
		return qn, ""
	}

	realDomain, ok := aliases[ref.Domain]
	if !ok {
		return "", fmt.Sprintf("line %d: domain alias %q not imported", line, ref.Domain)
	}

	qn := realDomain + "." + ref.Name
	if _, ok := nodes[qn]; !ok {
		return "", fmt.Sprintf("line %d: undeclared %q in domain %q", line, ref.Name, realDomain)
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
		m[imp.Alias] = imp.Domain
	}
	return m
}

func generateCode(g *builtGraph, packageName string) ([]byte, error) {
	var buf bytes.Buffer

	buf.WriteString("// Code generated by archlang. DO NOT EDIT.\n")
	fmt.Fprintf(&buf, "package %s\n\n", packageName)
	buf.WriteString("import \"github.com/mcabezas/archlang/graph\"\n\n")

	// Collect unique domains
	domainSet := make(map[string]bool)
	for _, qn := range g.order {
		d := g.nodes[qn].domain
		if d != "" {
			domainSet[d] = true
		}
	}
	var domainNames []string
	for d := range domainSet {
		domainNames = append(domainNames, d)
	}
	sort.Strings(domainNames)

	// Domain constants
	buf.WriteString("const (\n")
	for _, d := range domainNames {
		fmt.Fprintf(&buf, "\t%s graph.Domain = %q\n", toGoName(d), d)
	}
	buf.WriteString(")\n\n")

	// Variable declarations
	buf.WriteString("var (\n")
	for _, qn := range g.order {
		n := g.nodes[qn]
		varName := toGoName(qn)
		domainConst := toGoName(n.domain)

		opts := fmt.Sprintf("graph.WithName(%q), graph.WithDomain(%s)", n.name, domainConst)
		if n.isPublic {
			opts += ", graph.WithVisibility(graph.Public)"
		}

		constructor := "graph.NewComponent"
		if n.isService {
			constructor = "graph.NewService"
		} else if n.isInfra {
			constructor = "graph.NewInfra"
		}

		fmt.Fprintf(&buf, "\t%s = %s(%s)\n", varName, constructor, opts)
	}
	buf.WriteString(")\n\n")

	// AllComponents slice
	buf.WriteString("var AllComponents = []graph.Component{\n")
	for _, qn := range g.order {
		fmt.Fprintf(&buf, "\t%s,\n", toGoName(qn))
	}
	buf.WriteString("}\n\n")

	// Services slice
	var services []string
	for _, qn := range g.order {
		if g.nodes[qn].isService {
			services = append(services, qn)
		}
	}
	buf.WriteString("var AllServices = []graph.Component{\n")
	for _, qn := range services {
		fmt.Fprintf(&buf, "\t%s,\n", toGoName(qn))
	}
	buf.WriteString("}\n\n")

	// Partition into connected components
	components := connectedComponents(g)

	// AllGraphs — one graph per connected component
	buf.WriteString("var AllGraphs = func() []*graph.Graph {\n")
	for i, comp := range components {
		fmt.Fprintf(&buf, "\tg%d := graph.NewGraph()\n", i)
		for _, qn := range comp {
			fmt.Fprintf(&buf, "\tg%d.Register(%q, %s)\n", i, qn, toGoName(qn))
		}
		for _, qn := range comp {
			n := g.nodes[qn]
			for _, dsQN := range n.downstreams {
				fmt.Fprintf(&buf, "\tg%d.AddDownstream(%s, %s)\n", i, toGoName(qn), toGoName(dsQN))
			}
		}
		buf.WriteString("\n")
	}
	buf.WriteString("\treturn []*graph.Graph{")
	for i := range components {
		if i > 0 {
			buf.WriteString(", ")
		}
		fmt.Fprintf(&buf, "g%d", i)
	}
	buf.WriteString("}\n")
	buf.WriteString("}()\n")

	return format.Source(buf.Bytes())
}

// connectedComponents partitions the graph into connected subgraphs.
// Edges are treated as undirected for connectivity.
func connectedComponents(g *builtGraph) [][]string {
	visited := make(map[string]bool)
	neighbors := make(map[string]map[string]bool)

	// Build undirected adjacency
	for _, qn := range g.order {
		if neighbors[qn] == nil {
			neighbors[qn] = make(map[string]bool)
		}
		n := g.nodes[qn]
		for _, dsQN := range n.downstreams {
			neighbors[qn][dsQN] = true
			if neighbors[dsQN] == nil {
				neighbors[dsQN] = make(map[string]bool)
			}
			neighbors[dsQN][qn] = true
		}
	}

	var components [][]string
	for _, qn := range g.order {
		if visited[qn] {
			continue
		}
		// BFS
		var comp []string
		queue := []string{qn}
		visited[qn] = true
		for len(queue) > 0 {
			cur := queue[0]
			queue = queue[1:]
			comp = append(comp, cur)
			for neighbor := range neighbors[cur] {
				if !visited[neighbor] {
					visited[neighbor] = true
					queue = append(queue, neighbor)
				}
			}
		}
		sort.Strings(comp)
		components = append(components, comp)
	}

	return components
}

func toGoName(name string) string {
	name = strings.ReplaceAll(name, ".", "-")
	name = strings.ReplaceAll(name, "/", "-")
	parts := strings.Split(name, "-")
	for i, part := range parts {
		if len(part) > 0 {
			parts[i] = strings.ToUpper(part[:1]) + part[1:]
		}
	}
	return strings.Join(parts, "")
}

// discoverDomains walks the directory tree and returns a map of
// domain name to concatenated .arch file contents.
func discoverDomains(root string) (map[string]string, error) {
	domains := make(map[string]string)

	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			return nil
		}

		input, err := readArchFiles(path)
		if err != nil {
			return err
		}
		if input == "" {
			return nil
		}

		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}

		domainName := rel
		if domainName == "." {
			domainName = filepath.Base(root)
		}

		domains[domainName] = input
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("cannot walk directory %q: %w", root, err)
	}

	return domains, nil
}

func readArchFiles(dir string) (string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", fmt.Errorf("cannot read directory %q: %w", dir, err)
	}

	var parts []string
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".arch") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, entry.Name()))
		if err != nil {
			return "", fmt.Errorf("cannot read file %q: %w", entry.Name(), err)
		}
		parts = append(parts, string(data))
	}

	return strings.Join(parts, "\n"), nil
}
