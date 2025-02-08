package generator

import "github.com/mgpai22/gogenesis/internal/parser"

// CodeGenerator is the interface that all language-specific code generators must implement.
type CodeGenerator interface {
	Generate(schema *parser.PlutusSchema, chosenNames map[string]string) (string, error)

	FileName() string
}
