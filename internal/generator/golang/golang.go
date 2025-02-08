package golang

import (
	"fmt"
	"sort"
	"strings"

	"github.com/mgpai22/gogenesis/internal/generator"
	"github.com/mgpai22/gogenesis/internal/parser"
)

type GoGenerator struct{}

func NewGoGenerator() *GoGenerator {
	return &GoGenerator{}
}

func (g *GoGenerator) FileName() string {
	return "plutus_types.go"
}

// Generate returns the generated Go code as a string.
func (g *GoGenerator) Generate(schema *parser.PlutusSchema, chosenNames map[string]string) (string, error) {
	var builder strings.Builder
	builder.WriteString("// AUTO-GENERATED FILE. DO NOT EDIT MANUALLY.\n")
	builder.WriteString("// Re-generate this by running the code generator script.\n\n")
	builder.WriteString("package main\n\n")

	// Order definitions topologically.
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
	var refNames []string
	for refName := range schema.Definitions {
		refNames = append(refNames, refName)
	}
	sort.Strings(refNames)
	for _, refName := range refNames {
		visit(refName)
	}

	// For each definition, generate a Go struct.
	for _, refName := range finalOrder {
		def := schema.Definitions[refName]
		typeName := chosenNames[refName]
		builder.WriteString(fmt.Sprintf("// Definition for %s\n", refName))
		builder.WriteString(fmt.Sprintf("type %s struct {\n", typeName))
		// For simplicity each field becomes an "interface{}" placeholder.
		for _, field := range def.Fields {
			fieldName := field.Title
			if fieldName == "" {
				fieldName = "Field"
			}
			// A simple mapping: if there is a reference, try to use the chosen type.
			fieldType := "interface{}"
			if field.Ref != "" {
				r := strings.TrimPrefix(field.Ref, "#/definitions/")
				r = strings.ReplaceAll(r, "~1", "")
				if tn, ok := chosenNames[r]; ok {
					fieldType = tn
				} else {
					fieldType = generator.MakeTypeName(r)
				}
			}
			builder.WriteString(fmt.Sprintf("    %s %s `json:\"%s\"`\n", fieldName, fieldType, fieldName))
		}
		builder.WriteString("}\n\n")
	}

	return builder.String(), nil
}
