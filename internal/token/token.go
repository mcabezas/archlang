package token

type TokenType string

const (
	ILLEGAL = "ILLEGAL"
	EOF     = "EOF"

	// Identifiers
	IDENT = "IDENT" // service names, component names

	// Keywords
	COMPONENT        = "COMPONENT"
	SERVICE          = "SERVICE"
	EVENT            = "EVENT"
	COLLABORATION    = "COLLABORATION"
	FRONTEND         = "FRONTEND"
	MESSAGE_BROKER    = "MESSAGE_BROKER"
	BROKER_TECHNOLOGY = "BROKER_TECHNOLOGY"
	CLOUD_PROVIDER    = "CLOUD_PROVIDER"
	TECHNOLOGY        = "TECHNOLOGY"
	CLOUD             = "CLOUD"
	PLATFORM          = "PLATFORM"
	PUBLISHED_AT       = "PUBLISHED_AT"
	DELIVERED_BY       = "DELIVERED_BY"
	PUBLIC           = "PUBLIC"
	INTERNAL         = "INTERNAL"
	FEATURE          = "FEATURE"
	DESCRIPTION      = "DESCRIPTION"
	CARDINALITY      = "CARDINALITY"
	FLOW             = "FLOW"
	STEP             = "STEP"
	EXECUTE          = "EXECUTE"
	PUBLISHES        = "PUBLISHES"

	// Operators
	ARROW         = "->"
	REVERSE_ARROW = "<-"
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
	"component":         COMPONENT,
	"service":           SERVICE,
	"event":             EVENT,
	"collaboration":     COLLABORATION,
	"frontend":          FRONTEND,
	"message_broker":    MESSAGE_BROKER,
	"broker_technology": BROKER_TECHNOLOGY,
	"cloud_provider":    CLOUD_PROVIDER,
	"technology":        TECHNOLOGY,
	"cloud":             CLOUD,
	"platform":          PLATFORM,
	"published_at":      PUBLISHED_AT,
	"delivered_by":      DELIVERED_BY,
	"public":            PUBLIC,
	"internal":          INTERNAL,
	"feature":           FEATURE,
	"description":       DESCRIPTION,
	"cardinality":       CARDINALITY,
	"flow":              FLOW,
	"step":              STEP,
	"execute":           EXECUTE,
	"publishes":         PUBLISHES,
}

func LookupIdent(ident string) TokenType {
	if tok, ok := keywords[ident]; ok {
		return tok
	}
	return IDENT
}
