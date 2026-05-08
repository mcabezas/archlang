package graph

import (
	"testing"

	"github.com/mcabezas/archlang/internal/lexer"
	"github.com/mcabezas/archlang/internal/parser"
)

func TestBuildGraph(t *testing.T) {
	input := `component redis
service payments
service users
collaboration payments -> redis
collaboration payments -> users`

	g, errors := buildGraph(t, input)
	assertNoErrors(t, errors)

	payments, ok := g.GetNode("payments")
	if !ok {
		t.Fatal("expected node 'payments' to exist")
	}
	if payments.Kind != KindService {
		t.Fatalf("expected kind service, got %s", payments.Kind)
	}
	assertSlice(t, "payments.Downstreams", payments.Downstreams, []string{"redis", "users"})

	redis, ok := g.GetNode("redis")
	if !ok {
		t.Fatal("expected node 'redis' to exist")
	}
	if redis.Kind != KindComponent {
		t.Fatalf("expected kind component, got %s", redis.Kind)
	}
	assertSlice(t, "redis.Upstreams", redis.Upstreams, []string{"payments"})
}

func TestAllNodes(t *testing.T) {
	input := `component redis
service payments
service users`

	g, errors := buildGraph(t, input)
	assertNoErrors(t, errors)

	nodes := g.AllNodes()
	if len(nodes) != 3 {
		t.Fatalf("expected 3 nodes, got %d", len(nodes))
	}
}

func TestServices(t *testing.T) {
	input := `component redis
component postgres
service payments
service users`

	g, errors := buildGraph(t, input)
	assertNoErrors(t, errors)

	services := g.Services()
	if len(services) != 2 {
		t.Fatalf("expected 2 services, got %d", len(services))
	}
}

func TestDuplicateDeclaration(t *testing.T) {
	input := `service payments
service payments`

	_, errors := buildGraph(t, input)
	if len(errors) == 0 {
		t.Fatal("expected duplicate declaration error")
	}
}

func TestDuplicateAcrossKinds(t *testing.T) {
	input := `component redis
service redis`

	_, errors := buildGraph(t, input)
	if len(errors) == 0 {
		t.Fatal("expected duplicate declaration error across kinds")
	}
}

func TestUndeclaredSource(t *testing.T) {
	input := `service payments
collaboration unknown -> payments`

	_, errors := buildGraph(t, input)
	if len(errors) == 0 {
		t.Fatal("expected undeclared component error")
	}
}

func TestUndeclaredTarget(t *testing.T) {
	input := `service payments
collaboration payments -> unknown`

	_, errors := buildGraph(t, input)
	if len(errors) == 0 {
		t.Fatal("expected undeclared component error")
	}
}

func TestCircularCollaboration(t *testing.T) {
	input := `service a
service b
collaboration a -> b
collaboration b -> a`

	g, errors := buildGraph(t, input)
	assertNoErrors(t, errors)

	a, _ := g.GetNode("a")
	assertSlice(t, "a.Downstreams", a.Downstreams, []string{"b"})
	assertSlice(t, "a.Upstreams", a.Upstreams, []string{"b"})
}

func buildGraph(t *testing.T, input string) (*Graph, []string) {
	t.Helper()
	l := lexer.New(input)
	p := parser.New(l)
	arch := p.Parse()
	if len(p.Errors()) > 0 {
		t.Fatalf("parser errors: %v", p.Errors())
	}
	return Build(arch)
}

func assertNoErrors(t *testing.T, errors []string) {
	t.Helper()
	if len(errors) > 0 {
		t.Fatalf("unexpected errors: %v", errors)
	}
}

func assertSlice(t *testing.T, label string, got, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("%s: expected %v, got %v", label, want, got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("%s[%d]: expected %q, got %q", label, i, want[i], got[i])
		}
	}
}
