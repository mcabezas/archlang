package sdk

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/mcabezas/archlang/internal/ast"
	"github.com/mcabezas/archlang/internal/graph"
	"github.com/mcabezas/archlang/internal/lexer"
	"github.com/mcabezas/archlang/internal/parser"
)

type ComponentKind string

const (
	KindComponent ComponentKind = "component"
	KindService   ComponentKind = "service"
)

type Component struct {
	Name        string        `json:"name"`
	Package     string        `json:"package"`
	Kind        ComponentKind `json:"kind"`
	Downstreams []*Component  `json:"downstreams"`
	Upstreams   []*Component  `json:"upstreams"`
}

type Architecture struct {
	components map[string]*Component // keyed by qualified name (pkg.name)
	packages   []string
}

func Compile(dir string) (*Architecture, error) {
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

	g, errs := graph.Build(parsed)
	if len(errs) > 0 {
		return nil, fmt.Errorf("compile errors:\n  %s", strings.Join(errs, "\n  "))
	}

	return fromGraph(g), nil
}

func (a *Architecture) Components() []*Component {
	components := make([]*Component, 0, len(a.components))
	for _, c := range a.components {
		components = append(components, c)
	}
	return components
}

func (a *Architecture) Services() []*Component {
	var services []*Component
	for _, c := range a.components {
		if c.Kind == KindService {
			services = append(services, c)
		}
	}
	return services
}

func (a *Architecture) GetComponent(qualifiedName string) (*Component, bool) {
	c, ok := a.components[qualifiedName]
	return c, ok
}

func (a *Architecture) Packages() []string {
	return a.packages
}

func fromGraph(g *graph.Graph) *Architecture {
	arch := &Architecture{
		components: make(map[string]*Component),
		packages:   g.Packages(),
	}

	for _, n := range g.AllNodes() {
		arch.components[n.QualifiedName()] = &Component{
			Name:        n.Name,
			Package:     n.Package,
			Kind:        ComponentKind(n.Kind),
			Downstreams: []*Component{},
			Upstreams:   []*Component{},
		}
	}

	for _, n := range g.AllNodes() {
		c := arch.components[n.QualifiedName()]
		for _, qn := range n.Downstreams {
			c.Downstreams = append(c.Downstreams, arch.components[qn])
		}
		for _, qn := range n.Upstreams {
			c.Upstreams = append(c.Upstreams, arch.components[qn])
		}
	}

	return arch
}

// discoverPackages walks the directory tree and returns a map of
// package name → concatenated .arch file contents.
// Each subdirectory with .arch files becomes a package named by its
// path relative to the root.
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
