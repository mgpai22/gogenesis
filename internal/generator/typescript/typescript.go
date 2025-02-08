package typescript

import (
	"sort"
	"strings"

	"github.com/mgpai22/gogenesis/internal/generator"
	"github.com/mgpai22/gogenesis/internal/parser"
)

type TypeScriptGenerator struct{}

func NewTypeScriptGenerator() *TypeScriptGenerator {
	return &TypeScriptGenerator{}
}

func (ts *TypeScriptGenerator) FileName() string {
	return "plutus-types.ts"
}

// Generate returns the generated TypeScript code as a string.
// It orders the definitions (via a DFS using memoized dependency collection),
// delegates schema generation to GenerateTSSchema, and concatenates resulting lines.
func (ts *TypeScriptGenerator) Generate(schema *parser.PlutusSchema, chosenNames map[string]string) (string, error) {
	var builder strings.Builder

	builder.WriteString("// AUTO-GENERATED FILE. DO NOT EDIT MANUALLY.\n")
	builder.WriteString("// Re-generate this by running the code generator script.\n")
	builder.WriteString("import { Data } from '@lucid-evolution/lucid';\n\n")

	// Order definitions using DFS with memoized dependency collection.
	visited := make(map[string]bool)
	var finalOrder []string
	depMemo := make(map[string][]string)

	var visit func(refName string)
	visit = func(refName string) {
		if visited[refName] {
			return
		}
		visited[refName] = true
		for _, dep := range generator.CollectDependenciesMemo(refName, schema.Definitions, depMemo) {
			visit(dep)
		}
		finalOrder = append(finalOrder, refName)
	}

	// Begin with definitions in alphabetical order.
	var refNames []string
	for refName := range schema.Definitions {
		refNames = append(refNames, refName)
	}
	sort.Strings(refNames)
	for _, refName := range refNames {
		visit(refName)
	}

	// Generate a schema for each definition.
	for _, refName := range finalOrder {
		def := schema.Definitions[refName]
		tsTypeName := chosenNames[refName]
		lines := generator.GenerateTSSchema(refName, def, tsTypeName, chosenNames, schema.Definitions)
		for _, line := range lines {
			builder.WriteString(line + "\n")
		}
	}

	return builder.String(), nil
}
