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
	packages, err := discoverPackages(dir)
	if err != nil {
		return nil, err
	}

	if len(packages) == 0 {
		return nil, fmt.Errorf("no .arch files found in %q", dir)
	}

	parsed := make(map[string]*ast.Architecture)
	for pkg, input := range packages {
		l := lexer.New(input)
		p := parser.New(l)
		arch := p.Parse()

		if len(p.Errors()) > 0 {
			return nil, fmt.Errorf("parse errors in package %q:\n  %s", pkg, strings.Join(p.Errors(), "\n  "))
		}
		parsed[pkg] = arch
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
	isDomain      bool
	isPackage     bool
	isService     bool
	isInfra       bool
	downstreams   []string // qualified names
	upstreams     []string // qualified names
}

// builtGraph is the intermediate representation after building from ASTs.
type builtGraph struct {
	nodes map[string]*graphNode
	order []string // sorted qualified names
}

func buildGraph(packages map[string]*ast.Architecture) (*builtGraph, []string) {
	nodes := make(map[string]*graphNode)
	var errors []string

	// Determine domains
	domains := make(map[string]bool)
	for pkg, arch := range packages {
		for _, stmt := range arch.Statements {
			if _, ok := stmt.(*ast.DomainStatement); ok {
				domains[pkg] = true
				break
			}
		}
	}

	// Validate: no nested domains
	for pkg := range domains {
		for parentPkg := range domains {
			if pkg != parentPkg && strings.HasPrefix(pkg, parentPkg+"/") {
				errors = append(errors, fmt.Sprintf(
					"%s: cannot declare domain inside domain %q — nested domains are not allowed",
					pkg, parentPkg))
			}
		}
	}

	// Create package/domain nodes
	for pkg := range packages {
		n := &graphNode{
			qualifiedName: pkg,
			name:          pkg,
		}
		if domains[pkg] {
			n.isDomain = true
		} else {
			n.isPackage = true
		}
		nodes[pkg] = n
	}

	// First pass: register components and services
	for pkg, arch := range packages {
		for _, stmt := range arch.Statements {
			switch s := stmt.(type) {
			case *ast.ComponentStatement:
				qn := pkg + "." + s.Name
				if _, exists := nodes[qn]; exists {
					errors = append(errors, fmt.Sprintf("%s: line %d: duplicate declaration %q", pkg, s.Token.Line, s.Name))
					continue
				}
				n := &graphNode{
					qualifiedName: qn,
					name:          s.Name,
					domain:        pkg,
				}
				if s.Infra != "" {
					n.isInfra = true
				}
				nodes[qn] = n
			case *ast.ServiceStatement:
				qn := pkg + "." + s.Name
				if _, exists := nodes[qn]; exists {
					errors = append(errors, fmt.Sprintf("%s: line %d: duplicate declaration %q", pkg, s.Token.Line, s.Name))
					continue
				}
				nodes[qn] = &graphNode{
					qualifiedName: qn,
					name:          s.Name,
					domain:        pkg,
					isService:     true,
				}
			}
		}
	}

	// Second pass: validate imports and wire collaborations
	for pkg, arch := range packages {
		imports := collectImports(arch)

		for _, imp := range imports {
			if _, ok := nodes[imp.Package]; !ok {
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

			sourceQN, err := resolveRef(pkg, s.Source, aliasMap, s.Token.Line, nodes)
			if err != "" {
				errors = append(errors, fmt.Sprintf("%s: %s", pkg, err))
			}

			targetQN, err := resolveRef(pkg, s.Target, aliasMap, s.Token.Line, nodes)
			if err != "" {
				errors = append(errors, fmt.Sprintf("%s: %s", pkg, err))
			}

			if sourceQN != "" && targetQN != "" {
				nodes[sourceQN].downstreams = append(nodes[sourceQN].downstreams, targetQN)
				nodes[targetQN].upstreams = append(nodes[targetQN].upstreams, sourceQN)
			}
		}
	}

	// Third pass: infer domain-level edges
	edges := make(map[string]bool)
	for _, n := range nodes {
		if n.domain == "" {
			continue
		}
		for _, dsQN := range n.downstreams {
			ds := nodes[dsQN]
			if ds.domain != "" && n.domain != ds.domain {
				edge := n.domain + "->" + ds.domain
				if !edges[edge] {
					edges[edge] = true
					nodes[n.domain].downstreams = append(nodes[n.domain].downstreams, ds.domain)
					nodes[ds.domain].upstreams = append(nodes[ds.domain].upstreams, n.domain)
				}
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

func resolveRef(currentPkg string, ref ast.ComponentRef, aliases map[string]string, line int, nodes map[string]*graphNode) (string, string) {
	if ref.Package == "" {
		qn := currentPkg + "." + ref.Name
		if _, ok := nodes[qn]; !ok {
			return "", fmt.Sprintf("line %d: undeclared %q in package %q", line, ref.Name, currentPkg)
		}
		return qn, ""
	}

	realPkg, ok := aliases[ref.Package]
	if !ok {
		return "", fmt.Sprintf("line %d: package alias %q not imported", line, ref.Package)
	}

	qn := realPkg + "." + ref.Name
	if _, ok := nodes[qn]; !ok {
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

func generateCode(g *builtGraph, packageName string) ([]byte, error) {
	var buf bytes.Buffer

	buf.WriteString("// Code generated by archlang. DO NOT EDIT.\n")
	fmt.Fprintf(&buf, "package %s\n\n", packageName)
	buf.WriteString("import \"github.com/mcabezas/archlang/graph\"\n\n")

	// Variable declarations
	buf.WriteString("var (\n")
	for _, qn := range g.order {
		n := g.nodes[qn]
		varName := toGoName(qn)
		constructor := constructorFor(n)
		fmt.Fprintf(&buf, "\t%s = graph.%s(%q)\n", varName, constructor, n.name)
	}
	buf.WriteString(")\n\n")

	// Helper slices
	buf.WriteString("var AllNodes = []graph.Node{\n")
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
	buf.WriteString("var AllServices = []graph.Node{\n")
	for _, qn := range services {
		fmt.Fprintf(&buf, "\t%s,\n", toGoName(qn))
	}
	buf.WriteString("}\n\n")

	// Graph construction
	buf.WriteString("var Graph = func() *graph.Graph {\n")
	buf.WriteString("\tg := graph.NewGraph()\n")
	for _, qn := range g.order {
		fmt.Fprintf(&buf, "\tg.Register(%q, %s)\n", qn, toGoName(qn))
	}
	buf.WriteString("\n")

	// Wire edges
	for _, qn := range g.order {
		n := g.nodes[qn]
		for _, dsQN := range n.downstreams {
			fmt.Fprintf(&buf, "\tg.AddDownstream(%s, %s)\n", toGoName(qn), toGoName(dsQN))
		}
	}
	buf.WriteString("\treturn g\n")
	buf.WriteString("}()\n")

	return format.Source(buf.Bytes())
}

func constructorFor(n *graphNode) string {
	switch {
	case n.isDomain:
		return "NewDomain"
	case n.isPackage:
		return "NewPackage"
	case n.isService:
		return "NewService"
	case n.isInfra:
		return "NewInfra"
	default:
		return "NewComponent"
	}
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

// discoverPackages walks the directory tree and returns a map of
// package name to concatenated .arch file contents.
func discoverPackages(root string) (map[string]string, error) {
	packages := make(map[string]string)

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

		pkgName := rel
		if pkgName == "." {
			pkgName = filepath.Base(root)
		}

		packages[pkgName] = input
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("cannot walk directory %q: %w", root, err)
	}

	return packages, nil
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
