package token

type TokenType string

const (
	ILLEGAL = "ILLEGAL"
	EOF     = "EOF"

	// Identifiers
	IDENT = "IDENT" // service names, component names

	// Keywords
	COMPONENT     = "COMPONENT"
	SERVICE       = "SERVICE"
	COLLABORATION = "COLLABORATION"
	IMPORT        = "IMPORT"
	AS            = "AS"
	DOMAIN        = "DOMAIN"

	FRONTEND = "FRONTEND"
	INFRA    = "INFRA"

	// Operators
	ARROW  = "->"
	DOT    = "."
	COLON  = ":"
	ASSIGN = "="
	COMMA  = ","

	// Literals
	NUMBER = "NUMBER"

	// Delimiters
	LBRACE = "{"
	RBRACE = "}"
)

type Token struct {
	Type    TokenType
	Literal string
	Line    int
	Column  int
}

var keywords = map[string]TokenType{
	"component":     COMPONENT,
	"service":       SERVICE,
	"collaboration": COLLABORATION,
	"import":        IMPORT,
	"as":            AS,
	"domain":        DOMAIN,
	"frontend":      FRONTEND,
	"infra":         INFRA,
}

func LookupIdent(ident string) TokenType {
	if tok, ok := keywords[ident]; ok {
		return tok
	}
	return IDENT
}
