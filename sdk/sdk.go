package sdk

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

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
	Kind        ComponentKind `json:"kind"`
	Downstreams []*Component  `json:"downstreams"`
	Upstreams   []*Component  `json:"upstreams"`
}

type Architecture struct {
	components map[string]*Component
}

func Compile(dir string) (*Architecture, error) {
	input, err := readArchFiles(dir)
	if err != nil {
		return nil, err
	}

	if input == "" {
		return nil, fmt.Errorf("no .arch files found in %q", dir)
	}

	l := lexer.New(input)
	p := parser.New(l)
	parsed := p.Parse()

	if len(p.Errors()) > 0 {
		return nil, fmt.Errorf("parse errors:\n  %s", strings.Join(p.Errors(), "\n  "))
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

func (a *Architecture) GetComponent(name string) (*Component, bool) {
	c, ok := a.components[name]
	return c, ok
}

func fromGraph(g *graph.Graph) *Architecture {
	arch := &Architecture{components: make(map[string]*Component)}

	for _, n := range g.AllNodes() {
		arch.components[n.Name] = &Component{
			Name:        n.Name,
			Kind:        ComponentKind(n.Kind),
			Downstreams: []*Component{},
			Upstreams:   []*Component{},
		}
	}

	for _, n := range g.AllNodes() {
		c := arch.components[n.Name]
		for _, name := range n.Downstreams {
			c.Downstreams = append(c.Downstreams, arch.components[name])
		}
		for _, name := range n.Upstreams {
			c.Upstreams = append(c.Upstreams, arch.components[name])
		}
	}

	return arch
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
