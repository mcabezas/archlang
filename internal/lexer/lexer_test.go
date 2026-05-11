package lexer

import (
	"testing"

	"github.com/mcabezas/archlang/internal/token"
)

func TestNextToken(t *testing.T) {
	input := `component redis
component postgres
component api-gateway

service checkout
service payments
service users

collaboration api-gateway -> checkout
collaboration checkout -> payments
collaboration payments -> postgres
`

	tests := []struct {
		expectedType    token.TokenType
		expectedLiteral string
	}{
		{token.COMPONENT, "component"},
		{token.IDENT, "redis"},
		{token.COMPONENT, "component"},
		{token.IDENT, "postgres"},
		{token.COMPONENT, "component"},
		{token.IDENT, "api-gateway"},

		{token.SERVICE, "service"},
		{token.IDENT, "checkout"},
		{token.SERVICE, "service"},
		{token.IDENT, "payments"},
		{token.SERVICE, "service"},
		{token.IDENT, "users"},

		{token.COLLABORATION, "collaboration"},
		{token.IDENT, "api-gateway"},
		{token.ARROW, "->"},
		{token.IDENT, "checkout"},

		{token.COLLABORATION, "collaboration"},
		{token.IDENT, "checkout"},
		{token.ARROW, "->"},
		{token.IDENT, "payments"},

		{token.COLLABORATION, "collaboration"},
		{token.IDENT, "payments"},
		{token.ARROW, "->"},
		{token.IDENT, "postgres"},

		{token.EOF, ""},
	}

	l := New(input)

	for i, tt := range tests {
		tok := l.NextToken()

		if tok.Type != tt.expectedType {
			t.Fatalf("tests[%d] - tokentype wrong. expected=%q, got=%q",
				i, tt.expectedType, tok.Type)
		}

		if tok.Literal != tt.expectedLiteral {
			t.Fatalf("tests[%d] - literal wrong. expected=%q, got=%q",
				i, tt.expectedLiteral, tok.Literal)
		}
	}
}

func TestComments(t *testing.T) {
	input := `# This is a comment
component redis
# Another comment
service payments
`

	tests := []struct {
		expectedType    token.TokenType
		expectedLiteral string
	}{
		{token.COMPONENT, "component"},
		{token.IDENT, "redis"},
		{token.SERVICE, "service"},
		{token.IDENT, "payments"},
		{token.EOF, ""},
	}

	l := New(input)

	for i, tt := range tests {
		tok := l.NextToken()

		if tok.Type != tt.expectedType {
			t.Fatalf("tests[%d] - tokentype wrong. expected=%q, got=%q",
				i, tt.expectedType, tok.Type)
		}

		if tok.Literal != tt.expectedLiteral {
			t.Fatalf("tests[%d] - literal wrong. expected=%q, got=%q",
				i, tt.expectedLiteral, tok.Literal)
		}
	}
}

func TestLineAndColumn(t *testing.T) {
	input := `component redis
service payments`

	l := New(input)

	tok := l.NextToken() // component
	if tok.Line != 1 || tok.Column != 1 {
		t.Fatalf("expected line=1, col=1, got line=%d, col=%d", tok.Line, tok.Column)
	}

	tok = l.NextToken() // redis
	if tok.Line != 1 || tok.Column != 11 {
		t.Fatalf("expected line=1, col=11, got line=%d, col=%d", tok.Line, tok.Column)
	}

	tok = l.NextToken() // service
	if tok.Line != 2 || tok.Column != 1 {
		t.Fatalf("expected line=2, col=1, got line=%d, col=%d", tok.Line, tok.Column)
	}
}

func TestIllegalToken(t *testing.T) {
	input := `component @invalid`

	l := New(input)

	l.NextToken() // component
	tok := l.NextToken()

	if tok.Type != token.ILLEGAL {
		t.Fatalf("expected ILLEGAL token, got %q", tok.Type)
	}
}

func TestFeatureTokens(t *testing.T) {
	input := `collaboration a -> b {
  feature payments
  feature refunds: handle refund flow
}`

	tests := []struct {
		expectedType    token.TokenType
		expectedLiteral string
	}{
		{token.COLLABORATION, "collaboration"},
		{token.IDENT, "a"},
		{token.ARROW, "->"},
		{token.IDENT, "b"},
		{token.LBRACE, "{"},
		{token.FEATURE, "feature"},
		{token.IDENT, "payments"},
		{token.FEATURE, "feature"},
		{token.IDENT, "refunds"},
		{token.COLON, ":"},
		{token.IDENT, "handle"},
		{token.IDENT, "refund"},
		{token.FLOW, "flow"},
		{token.RBRACE, "}"},
		{token.EOF, ""},
	}

	l := New(input)

	for i, tt := range tests {
		tok := l.NextToken()
		if tok.Type != tt.expectedType {
			t.Fatalf("tests[%d] - tokentype wrong. expected=%q, got=%q (literal=%q)",
				i, tt.expectedType, tok.Type, tok.Literal)
		}
		if tok.Literal != tt.expectedLiteral {
			t.Fatalf("tests[%d] - literal wrong. expected=%q, got=%q",
				i, tt.expectedLiteral, tok.Literal)
		}
	}
}

func TestImportTokens(t *testing.T) {
	input := `import notifications as noti

service tracking

collaboration tracking -> noti.push`

	tests := []struct {
		expectedType    token.TokenType
		expectedLiteral string
	}{
		{token.IMPORT, "import"},
		{token.IDENT, "notifications"},
		{token.AS, "as"},
		{token.IDENT, "noti"},
		{token.SERVICE, "service"},
		{token.IDENT, "tracking"},
		{token.COLLABORATION, "collaboration"},
		{token.IDENT, "tracking"},
		{token.ARROW, "->"},
		{token.IDENT, "noti"},
		{token.DOT, "."},
		{token.IDENT, "push"},
		{token.EOF, ""},
	}

	l := New(input)

	for i, tt := range tests {
		tok := l.NextToken()
		if tok.Type != tt.expectedType {
			t.Fatalf("tests[%d] - tokentype wrong. expected=%q, got=%q",
				i, tt.expectedType, tok.Type)
		}
		if tok.Literal != tt.expectedLiteral {
			t.Fatalf("tests[%d] - literal wrong. expected=%q, got=%q",
				i, tt.expectedLiteral, tok.Literal)
		}
	}
}
