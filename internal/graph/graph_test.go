package graph

import (
	"testing"

	"github.com/mcabezas/archlang/internal/ast"
	"github.com/mcabezas/archlang/internal/lexer"
	"github.com/mcabezas/archlang/internal/parser"
	"github.com/mcabezas/archlang/internal/token"
)

func TestBuildSinglePackage(t *testing.T) {
	packages := map[string]*ast.Architecture{
		"users": {
			Statements: []ast.Statement{
				&ast.ServiceStatement{Token: token.Token{Line: 1}, Name: "auth"},
				&ast.ComponentStatement{Token: token.Token{Line: 2}, Name: "users-db"},
				&ast.CollaborationStatement{
					Token:  token.Token{Line: 3},
					Source: ast.ComponentRef{Name: "auth"},
					Target: ast.ComponentRef{Name: "users-db"},
				},
			},
		},
	}

	g, errs := Build(packages)
	if len(errs) > 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}

	if len(g.AllNodes()) != 2 {
		t.Fatalf("expected 2 nodes, got %d", len(g.AllNodes()))
	}

	auth, ok := g.GetNode("users.auth")
	if !ok {
		t.Fatal("expected users.auth to exist")
	}
	if len(auth.Downstreams) != 1 || auth.Downstreams[0] != "users.users-db" {
		t.Fatalf("expected auth downstream to be users.users-db, got %v", auth.Downstreams)
	}
}

func TestBuildCrossPackage(t *testing.T) {
	packages := map[string]*ast.Architecture{
		"orders": {
			Statements: []ast.Statement{
				&ast.ImportStatement{Token: token.Token{Line: 1}, Package: "payments", Alias: "payments"},
				&ast.ServiceStatement{Token: token.Token{Line: 2}, Name: "order-management"},
				&ast.CollaborationStatement{
					Token:  token.Token{Line: 3},
					Source: ast.ComponentRef{Name: "order-management"},
					Target: ast.ComponentRef{Package: "payments", Name: "payment-processing"},
				},
			},
		},
		"payments": {
			Statements: []ast.Statement{
				&ast.ServiceStatement{Token: token.Token{Line: 1}, Name: "payment-processing"},
			},
		},
	}

	g, errs := Build(packages)
	if len(errs) > 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}

	order, _ := g.GetNode("orders.order-management")
	if len(order.Downstreams) != 1 || order.Downstreams[0] != "payments.payment-processing" {
		t.Fatalf("expected downstream payments.payment-processing, got %v", order.Downstreams)
	}

	payment, _ := g.GetNode("payments.payment-processing")
	if len(payment.Upstreams) != 1 || payment.Upstreams[0] != "orders.order-management" {
		t.Fatalf("expected upstream orders.order-management, got %v", payment.Upstreams)
	}
}

func TestBuildWithAlias(t *testing.T) {
	packages := map[string]*ast.Architecture{
		"delivery": {
			Statements: []ast.Statement{
				&ast.ImportStatement{Token: token.Token{Line: 1}, Package: "notifications", Alias: "noti"},
				&ast.ServiceStatement{Token: token.Token{Line: 2}, Name: "tracking"},
				&ast.CollaborationStatement{
					Token:  token.Token{Line: 3},
					Source: ast.ComponentRef{Name: "tracking"},
					Target: ast.ComponentRef{Package: "noti", Name: "push"},
				},
			},
		},
		"notifications": {
			Statements: []ast.Statement{
				&ast.ServiceStatement{Token: token.Token{Line: 1}, Name: "push"},
			},
		},
	}

	g, errs := Build(packages)
	if len(errs) > 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}

	tracking, _ := g.GetNode("delivery.tracking")
	if len(tracking.Downstreams) != 1 || tracking.Downstreams[0] != "notifications.push" {
		t.Fatalf("expected downstream notifications.push, got %v", tracking.Downstreams)
	}
}

func TestBuildUndeclaredLocal(t *testing.T) {
	packages := map[string]*ast.Architecture{
		"orders": {
			Statements: []ast.Statement{
				&ast.ServiceStatement{Token: token.Token{Line: 1}, Name: "order-management"},
				&ast.CollaborationStatement{
					Token:  token.Token{Line: 2},
					Source: ast.ComponentRef{Name: "order-management"},
					Target: ast.ComponentRef{Name: "nonexistent"},
				},
			},
		},
	}

	_, errs := Build(packages)
	if len(errs) == 0 {
		t.Fatal("expected error for undeclared local reference")
	}
}

func TestBuildUnimportedPackage(t *testing.T) {
	packages := map[string]*ast.Architecture{
		"orders": {
			Statements: []ast.Statement{
				&ast.ServiceStatement{Token: token.Token{Line: 1}, Name: "order-management"},
				&ast.CollaborationStatement{
					Token:  token.Token{Line: 2},
					Source: ast.ComponentRef{Name: "order-management"},
					Target: ast.ComponentRef{Package: "payments", Name: "processing"},
				},
			},
		},
		"payments": {
			Statements: []ast.Statement{
				&ast.ServiceStatement{Token: token.Token{Line: 1}, Name: "processing"},
			},
		},
	}

	_, errs := Build(packages)
	if len(errs) == 0 {
		t.Fatal("expected error for unimported package reference")
	}
}

func TestBuildNonexistentImport(t *testing.T) {
	packages := map[string]*ast.Architecture{
		"orders": {
			Statements: []ast.Statement{
				&ast.ImportStatement{Token: token.Token{Line: 1}, Package: "ghost", Alias: "ghost"},
			},
		},
	}

	_, errs := Build(packages)
	if len(errs) == 0 {
		t.Fatal("expected error for importing nonexistent package")
	}
}

func TestBuildDuplicateNode(t *testing.T) {
	packages := map[string]*ast.Architecture{
		"users": {
			Statements: []ast.Statement{
				&ast.ServiceStatement{Token: token.Token{Line: 1}, Name: "auth"},
				&ast.ServiceStatement{Token: token.Token{Line: 2}, Name: "auth"},
			},
		},
	}

	_, errs := Build(packages)
	if len(errs) == 0 {
		t.Fatal("expected error for duplicate declaration")
	}
}

func TestBuildCircularCollaboration(t *testing.T) {
	packages := map[string]*ast.Architecture{
		"mypackage": {
			Statements: []ast.Statement{
				&ast.ServiceStatement{Token: token.Token{Line: 1}, Name: "a"},
				&ast.ServiceStatement{Token: token.Token{Line: 2}, Name: "b"},
				&ast.CollaborationStatement{
					Token:  token.Token{Line: 3},
					Source: ast.ComponentRef{Name: "a"},
					Target: ast.ComponentRef{Name: "b"},
				},
				&ast.CollaborationStatement{
					Token:  token.Token{Line: 4},
					Source: ast.ComponentRef{Name: "b"},
					Target: ast.ComponentRef{Name: "a"},
				},
			},
		},
	}

	g, errs := Build(packages)
	if len(errs) > 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}

	a, _ := g.GetNode("mypackage.a")
	if len(a.Downstreams) != 1 || a.Downstreams[0] != "mypackage.b" {
		t.Fatalf("expected a downstream mypackage.b, got %v", a.Downstreams)
	}
	if len(a.Upstreams) != 1 || a.Upstreams[0] != "mypackage.b" {
		t.Fatalf("expected a upstream mypackage.b, got %v", a.Upstreams)
	}
}

func TestBuildFromParsedInput(t *testing.T) {
	input := `import payments

service order-management

collaboration order-management -> payments.payment-processing`

	l := lexer.New(input)
	p := parser.New(l)
	arch := p.Parse()
	if len(p.Errors()) > 0 {
		t.Fatalf("parser errors: %v", p.Errors())
	}

	packages := map[string]*ast.Architecture{
		"orders": arch,
		"payments": {
			Statements: []ast.Statement{
				&ast.ServiceStatement{Token: token.Token{Line: 1}, Name: "payment-processing"},
			},
		},
	}

	g, errs := Build(packages)
	if len(errs) > 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}

	order, ok := g.GetNode("orders.order-management")
	if !ok {
		t.Fatal("expected orders.order-management to exist")
	}
	if len(order.Downstreams) != 1 || order.Downstreams[0] != "payments.payment-processing" {
		t.Fatalf("expected downstream payments.payment-processing, got %v", order.Downstreams)
	}
}
