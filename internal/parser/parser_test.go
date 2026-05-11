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

func TestParseCollaborationWithFeature(t *testing.T) {
	input := `feature payments: "handle payment flow"
service a
service b
collaboration a -> b {
  feature payments
}`

	arch := parseInput(t, input)
	assertStatementCount(t, arch, 4)

	collab, ok := arch.Statements[3].(*ast.CollaborationStatement)
	if !ok {
		t.Fatalf("statements[3] not *ast.CollaborationStatement, got %T", arch.Statements[3])
	}
	if collab.Feature != "payments" {
		t.Fatalf("Feature = %q, want %q", collab.Feature, "payments")
	}
}

func TestParseCollaborationWithDescription(t *testing.T) {
	input := `feature payments: "handle payment flow"
service a
service b
collaboration a -> b {
  description: "REST POST /payments"
  feature payments
}`

	arch := parseInput(t, input)
	assertStatementCount(t, arch, 4)

	collab := arch.Statements[3].(*ast.CollaborationStatement)
	if collab.Feature != "payments" {
		t.Fatalf("Feature = %q, want %q", collab.Feature, "payments")
	}
	if collab.Description != "REST POST /payments" {
		t.Fatalf("Description = %q, want %q", collab.Description, "REST POST /payments")
	}
}

func TestParseCollaborationWithInlineDescription(t *testing.T) {
	input := `feature payments: "handle payment flow"
service a
service b
collaboration a -> b {
  feature payments: "REST POST /payments with order payload"
}`

	arch := parseInput(t, input)
	assertStatementCount(t, arch, 4)

	collab := arch.Statements[3].(*ast.CollaborationStatement)
	if collab.Feature != "payments" {
		t.Fatalf("Feature = %q, want %q", collab.Feature, "payments")
	}
	if collab.Description != "REST POST /payments with order payload" {
		t.Fatalf("Description = %q, want %q", collab.Description, "REST POST /payments with order payload")
	}
}

func TestParseCollaborationWithCardinality(t *testing.T) {
	input := `feature payments: "pay"
service a
service b
collaboration a -> b {
  feature payments
  cardinality 1:1
}`

	arch := parseInput(t, input)
	assertStatementCount(t, arch, 4)

	collab := arch.Statements[3].(*ast.CollaborationStatement)
	if collab.Feature != "payments" {
		t.Fatalf("Feature = %q, want %q", collab.Feature, "payments")
	}
	if collab.Cardinality != "1:1" {
		t.Fatalf("Cardinality = %q, want %q", collab.Cardinality, "1:1")
	}
}

func TestParseCollaborationWithCardinalityOneToMany(t *testing.T) {
	input := `feature events: "publish events"
service a
service b
collaboration a -> b {
  feature events: "Publishes order events to multiple consumers"
  cardinality 1:N
}`

	arch := parseInput(t, input)
	assertStatementCount(t, arch, 4)

	collab := arch.Statements[3].(*ast.CollaborationStatement)
	if collab.Feature != "events" {
		t.Fatalf("Feature = %q, want %q", collab.Feature, "events")
	}
	if collab.Description != "Publishes order events to multiple consumers" {
		t.Fatalf("Description = %q, want %q", collab.Description, "Publishes order events to multiple consumers")
	}
	if collab.Cardinality != "1:N" {
		t.Fatalf("Cardinality = %q, want %q", collab.Cardinality, "1:N")
	}
}

func TestParseCollaborationMultipleFeaturesError(t *testing.T) {
	input := `feature payments: "pay"
feature refunds: "refund"
service a
service b
collaboration a -> b {
  feature payments
  feature refunds
}`

	l := lexer.New(input)
	p := New(l)
	p.Parse()

	if len(p.Errors()) == 0 {
		t.Fatal("expected error for multiple features in collaboration block, got none")
	}

	found := false
	for _, err := range p.Errors() {
		if contains(err, "collaboration block can only contain one feature") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected error about one feature per block, got %v", p.Errors())
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

	if collab1.Feature != "payments" {
		t.Fatalf("collab1 feature = %q, want %q", collab1.Feature, "payments")
	}
	if collab2.Feature != "refunds" {
		t.Fatalf("collab2 feature = %q, want %q", collab2.Feature, "refunds")
	}
}

func TestParseCollaborationWithoutFeatures(t *testing.T) {
	input := `service a
service b
collaboration a -> b`

	arch := parseInput(t, input)
	assertStatementCount(t, arch, 3)

	collab := arch.Statements[2].(*ast.CollaborationStatement)
	if collab.Feature != "" {
		t.Fatalf("expected empty feature, got %q", collab.Feature)
	}
	if collab.Description != "" {
		t.Fatalf("expected empty description, got %q", collab.Description)
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
