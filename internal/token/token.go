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
	EVENT         = "EVENT"
	COLLABORATION = "COLLABORATION"
	IMPORT        = "IMPORT"
	AS            = "AS"

	FRONTEND = "FRONTEND"
	INFRA    = "INFRA"
	PUBLIC   = "PUBLIC"
	INTERNAL = "INTERNAL"
	FEATURE     = "FEATURE"
	DESCRIPTION = "DESCRIPTION"
	CARDINALITY = "CARDINALITY"
	FLOW        = "FLOW"
	STEP        = "STEP"
	EXECUTE     = "EXECUTE"
	PUBLISHES   = "PUBLISHES"

	// Operators
	ARROW         = "->"
	REVERSE_ARROW = "<-"
	DOT           = "."
	COLON         = ":"
	ASSIGN        = "="
	COMMA         = ","

	// Literals
	NUMBER = "NUMBER"
	STRING = "STRING"

	// Delimiters
	LBRACE   = "{"
	RBRACE   = "}"
	LBRACKET = "["
	RBRACKET = "]"
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
	"event":         EVENT,
	"collaboration": COLLABORATION,
	"import":        IMPORT,
	"as":            AS,
	"frontend":      FRONTEND,
	"infra":         INFRA,
	"public":        PUBLIC,
	"internal":      INTERNAL,
	"feature":       FEATURE,
	"description":   DESCRIPTION,
	"cardinality":   CARDINALITY,
	"flow":          FLOW,
	"step":          STEP,
	"execute":       EXECUTE,
	"publishes":     PUBLISHES,
}

func LookupIdent(ident string) TokenType {
	if tok, ok := keywords[ident]; ok {
		return tok
	}
	return IDENT
}
