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

	// Operators
	ARROW = "->"

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
}

func LookupIdent(ident string) TokenType {
	if tok, ok := keywords[ident]; ok {
		return tok
	}
	return IDENT
}
