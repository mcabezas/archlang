package parser

import (
	"testing"

	"github.com/mcabezas/archlang/internal/ast"
	"github.com/mcabezas/archlang/internal/lexer"
)

func TestParseComponents(t *testing.T) {
	input := `component redis
component postgres`

	arch := parseInput(t, input)
	assertStatementCount(t, arch, 2)

	tests := []string{"redis", "postgres"}
	for i, name := range tests {
		stmt, ok := arch.Statements[i].(*ast.ComponentStatement)
		if !ok {
			t.Fatalf("statements[%d] not *ast.ComponentStatement, got %T", i, arch.Statements[i])
		}
		if stmt.Name != name {
			t.Fatalf("statements[%d].Name = %q, want %q", i, stmt.Name, name)
		}
	}
}

func TestParseServices(t *testing.T) {
	input := `service checkout
service payments`

	arch := parseInput(t, input)
	assertStatementCount(t, arch, 2)

	tests := []string{"checkout", "payments"}
	for i, name := range tests {
		stmt, ok := arch.Statements[i].(*ast.ServiceStatement)
		if !ok {
			t.Fatalf("statements[%d] not *ast.ServiceStatement, got %T", i, arch.Statements[i])
		}
		if stmt.Name != name {
			t.Fatalf("statements[%d].Name = %q, want %q", i, stmt.Name, name)
		}
	}
}

func TestParseCollaborations(t *testing.T) {
	input := `component redis
service payments
collaboration payments -> redis`

	arch := parseInput(t, input)
	assertStatementCount(t, arch, 3)

	collab, ok := arch.Statements[2].(*ast.CollaborationStatement)
	if !ok {
		t.Fatalf("statements[2] not *ast.CollaborationStatement, got %T", arch.Statements[2])
	}
	if collab.Source.Name != "payments" || collab.Source.Domain != "" {
		t.Fatalf("Source = %+v, want local payments", collab.Source)
	}
	if collab.Target.Name != "redis" || collab.Target.Domain != "" {
		t.Fatalf("Target = %+v, want local redis", collab.Target)
	}
}

func TestParseQualifiedCollaboration(t *testing.T) {
	input := `import payments

service order-management

collaboration order-management -> payments.payment-processing`

	arch := parseInput(t, input)
	assertStatementCount(t, arch, 3)

	collab, ok := arch.Statements[2].(*ast.CollaborationStatement)
	if !ok {
		t.Fatalf("statements[2] not *ast.CollaborationStatement, got %T", arch.Statements[2])
	}
	if collab.Source.Name != "order-management" || collab.Source.Domain != "" {
		t.Fatalf("Source = %+v, want local order-management", collab.Source)
	}
	if collab.Target.Domain != "payments" || collab.Target.Name != "payment-processing" {
		t.Fatalf("Target = %+v, want payments.payment-processing", collab.Target)
	}
}

func TestParseImport(t *testing.T) {
	input := `import notifications`

	arch := parseInput(t, input)
	assertStatementCount(t, arch, 1)

	imp, ok := arch.Statements[0].(*ast.ImportStatement)
	if !ok {
		t.Fatalf("statements[0] not *ast.ImportStatement, got %T", arch.Statements[0])
	}
	if imp.Domain != "notifications" {
		t.Fatalf("Domain = %q, want %q", imp.Domain, "notifications")
	}
	if imp.Alias != "notifications" {
		t.Fatalf("Alias = %q, want %q", imp.Alias, "notifications")
	}
}

func TestParseImportWithAlias(t *testing.T) {
	input := `import notifications as noti`

	arch := parseInput(t, input)
	assertStatementCount(t, arch, 1)

	imp, ok := arch.Statements[0].(*ast.ImportStatement)
	if !ok {
		t.Fatalf("statements[0] not *ast.ImportStatement, got %T", arch.Statements[0])
	}
	if imp.Domain != "notifications" {
		t.Fatalf("Domain = %q, want %q", imp.Domain, "notifications")
	}
	if imp.Alias != "noti" {
		t.Fatalf("Alias = %q, want %q", imp.Alias, "noti")
	}
}

func TestParseFeatureDeclaration(t *testing.T) {
	input := `feature payments: "Process payment during checkout"`

	arch := parseInput(t, input)
	assertStatementCount(t, arch, 1)

	feat, ok := arch.Statements[0].(*ast.FeatureStatement)
	if !ok {
		t.Fatalf("statements[0] not *ast.FeatureStatement, got %T", arch.Statements[0])
	}
	if feat.Name != "payments" {
		t.Fatalf("Name = %q, want %q", feat.Name, "payments")
	}
	if feat.Description != "Process payment during checkout" {
		t.Fatalf("Description = %q, want %q", feat.Description, "Process payment during checkout")
	}
}

func TestParseFeatureWithBacktick(t *testing.T) {
	input := "feature refunds: `Handle refunds\nincluding partial`"

	arch := parseInput(t, input)
	assertStatementCount(t, arch, 1)

	feat := arch.Statements[0].(*ast.FeatureStatement)
	if feat.Name != "refunds" {
		t.Fatalf("Name = %q, want %q", feat.Name, "refunds")
	}
	if feat.Description != "Handle refunds\nincluding partial" {
		t.Fatalf("Description = %q, want %q", feat.Description, "Handle refunds\nincluding partial")
	}
}

func TestParseCollaborationWithFeatures(t *testing.T) {
	input := `feature payments: "handle payment flow"
feature refunds: "handle refunds"
service a
service b
collaboration a -> b {
  feature payments
  feature refunds
}`

	arch := parseInput(t, input)
	assertStatementCount(t, arch, 5)

	collab, ok := arch.Statements[4].(*ast.CollaborationStatement)
	if !ok {
		t.Fatalf("statements[4] not *ast.CollaborationStatement, got %T", arch.Statements[4])
	}
	if len(collab.Features) != 2 {
		t.Fatalf("expected 2 features, got %d", len(collab.Features))
	}
	if collab.Features[0] != "payments" {
		t.Fatalf("Features[0] = %q, want %q", collab.Features[0], "payments")
	}
	if collab.Features[1] != "refunds" {
		t.Fatalf("Features[1] = %q, want %q", collab.Features[1], "refunds")
	}
}

func TestParseDuplicateCollaborationsWithFeatures(t *testing.T) {
	input := `feature payments: "pay"
feature refunds: "refund"
service a
service b
collaboration a -> b {
  feature payments
}
collaboration a -> b {
  feature refunds
}`

	arch := parseInput(t, input)
	assertStatementCount(t, arch, 6)

	collab1 := arch.Statements[4].(*ast.CollaborationStatement)
	collab2 := arch.Statements[5].(*ast.CollaborationStatement)

	if len(collab1.Features) != 1 || collab1.Features[0] != "payments" {
		t.Fatalf("collab1 features wrong: %+v", collab1.Features)
	}
	if len(collab2.Features) != 1 || collab2.Features[0] != "refunds" {
		t.Fatalf("collab2 features wrong: %+v", collab2.Features)
	}
}

func TestParseCollaborationWithoutFeatures(t *testing.T) {
	input := `service a
service b
collaboration a -> b`

	arch := parseInput(t, input)
	assertStatementCount(t, arch, 3)

	collab := arch.Statements[2].(*ast.CollaborationStatement)
	if len(collab.Features) != 0 {
		t.Fatalf("expected 0 features, got %d", len(collab.Features))
	}
}

func TestParseFullArchitecture(t *testing.T) {
	input := `component redis
component postgres
component api-gateway

service checkout
service payments
service users

collaboration api-gateway -> checkout
collaboration checkout -> payments
collaboration payments -> postgres
collaboration users -> redis`

	arch := parseInput(t, input)
	assertStatementCount(t, arch, 10)
}

func TestParseWithComments(t *testing.T) {
	input := `# Infrastructure
component redis

# Services
service payments

# Collaborations
collaboration payments -> redis`

	arch := parseInput(t, input)
	assertStatementCount(t, arch, 3)
}

func TestParseErrors(t *testing.T) {
	tests := []struct {
		input       string
		expectedErr string
	}{
		{
			input:       `component`,
			expectedErr: "expected IDENT, got EOF",
		},
		{
			input:       `collaboration payments redis`,
			expectedErr: "expected ->, got IDENT",
		},
		{
			input:       `collaboration payments ->`,
			expectedErr: "expected IDENT, got EOF",
		},
		{
			input:       `import`,
			expectedErr: "expected IDENT, got EOF",
		},
	}

	for _, tt := range tests {
		l := lexer.New(tt.input)
		p := New(l)
		p.Parse()

		if len(p.Errors()) == 0 {
			t.Fatalf("expected errors for input %q, got none", tt.input)
		}

		found := false
		for _, err := range p.Errors() {
			if contains(err, tt.expectedErr) {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("expected error containing %q, got %v", tt.expectedErr, p.Errors())
		}
	}
}

func parseInput(t *testing.T, input string) *ast.Architecture {
	t.Helper()
	l := lexer.New(input)
	p := New(l)
	arch := p.Parse()

	if len(p.Errors()) > 0 {
		t.Fatalf("parser errors: %v", p.Errors())
	}

	return arch
}

func assertStatementCount(t *testing.T, arch *ast.Architecture, expected int) {
	t.Helper()
	if len(arch.Statements) != expected {
		t.Fatalf("expected %d statements, got %d", expected, len(arch.Statements))
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchString(s, substr)
}

func searchString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
