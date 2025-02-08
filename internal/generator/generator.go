package generator

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/mgpai22/gogenesis/internal/parser"
)

// GeneratorOptions holds settings for the master Generator.
type GeneratorOptions struct {
	// ReservedNames is a set of names that must not be used as type names.
	ReservedNames map[string]bool
	// You can add a Language field here if you want to also carry that info.
	Language string
}

var defaultReservedNames = map[string]bool{
	"Data":  true,
	"Dummy": true,
}

// Generator is the master generator that delegates to a CodeGenerator.
type Generator struct {
	OutputDir string
	Options   GeneratorOptions
	CodeGen   CodeGenerator
}

// NewGenerator creates a new Generator with the default CodeGenerator (e.g. for TypeScript).
func NewGenerator(outputDir string) *Generator {
	// Caller will provide a CodeGen override if needed.
	return NewGeneratorWithOptions(outputDir, GeneratorOptions{}, nil)
}

// NewGeneratorWithOptions creates a new Generator with custom options and a chosen CodeGenerator.
func NewGeneratorWithOptions(outputDir string, opts GeneratorOptions, codeGen CodeGenerator) *Generator {
	if opts.ReservedNames == nil {
		opts.ReservedNames = defaultReservedNames
	}
	// // If no CodeGenerator is provided, you could default to a TypeScript one.
	// if codeGen == nil {
	// 	// Here you’d import your typescript package and use its constructor.
	// 	// For example:
	// 	// codeGen = typescript.NewTypeScriptGenerator()
	// 	// (Assuming you imported "github.com/mgpai22/gogenesis/internal/generator/typescript")
	// }
	return &Generator{
		OutputDir: outputDir,
		Options:   opts,
		CodeGen:   codeGen,
	}
}

// Generate precomputes type names, delegates code generation to the CodeGenerator,
// then writes the generated content to a file.
func (g *Generator) Generate(schema *parser.PlutusSchema) error {
	// Precompute unique type names for all definitions.
	usedNames := make(map[string]bool)
	chosenNames := make(map[string]string)
	for refName, def := range schema.Definitions {
		title := def.Title
		if title == "" {
			title = refName
		}
		chosenNames[refName] = g.getUniqueTypeName(title, refName, usedNames)
	}

	code, err := g.CodeGen.Generate(schema, chosenNames)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(g.OutputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	filePath := filepath.Join(g.OutputDir, g.CodeGen.FileName())
	return os.WriteFile(filePath, []byte(code), 0644)
}

// getUniqueTypeName returns a type name that does not collide with existing names.
func (g *Generator) getUniqueTypeName(title, refName string, used map[string]bool) string {
	base := MakeTypeName(title)
	if g.Options.ReservedNames[base] {
		base = MakeTypeName(strings.ReplaceAll(refName, "/", "_"))
		if g.Options.ReservedNames[base] || base == "Data" {
			base = "Plutus" + base
		}
	}
	if !used[base] {
		used[base] = true
		return base
	}
	// Use namespaced version.
	namespaced := MakeTypeName(strings.ReplaceAll(refName, "/", "_"))
	if g.Options.ReservedNames[namespaced] || namespaced == "Data" {
		namespaced = "Plutus" + namespaced
	}
	if !used[namespaced] {
		used[namespaced] = true
		return namespaced
	}
	// Append numeric suffix if still colliding.
	counter := 1
	unique := fmt.Sprintf("%s_%d", namespaced, counter)
	for used[unique] {
		counter++
		unique = fmt.Sprintf("%s_%d", namespaced, counter)
	}
	used[unique] = true
	return unique
}

// MakeTypeName cleans a raw name and returns a TypeScript/Go–friendly type name.
func MakeTypeName(raw string) string {
	raw = strings.ReplaceAll(raw, " ", "_")
	raw = strings.ReplaceAll(raw, "$", "_")
	// Replace all non-word characters with underscore.
	re := regexp.MustCompile(`[^\w]+`)
	raw = re.ReplaceAllString(raw, "_")
	// Handle "~1" if present.
	if strings.Contains(raw, "~1") {
		parts := strings.Split(raw, "~1")
		raw = parts[len(parts)-1]
	}
	if raw == "" {
		return raw
	}
	// Capitalize the first letter.
	first := []rune(raw)[0]
	return strings.ToUpper(string(first)) + raw[1:]
}

// --- Shared Helper Functions ---
//
// The functions below (like CollectDependenciesMemo and GenerateTSSchema)
// were originally only used for TypeScript generation. They’ve been moved here and exported
// so that the TypeScript generator (and eventually others) can reuse them.

// CollectDependenciesMemo returns all direct dependency reference names for the definition identified by refName.
func CollectDependenciesMemo(refName string, defs map[string]parser.PlutusDefinition, memo map[string][]string) []string {
	if deps, ok := memo[refName]; ok {
		return deps
	}
	def := defs[refName]
	depsSet := make(map[string]bool)
	var scan func(v interface{})
	scan = func(v interface{}) {
		switch v := v.(type) {
		case parser.PlutusDefinition:
			if v.Ref != "" {
				r := strings.TrimPrefix(v.Ref, "#/definitions/")
				r = strings.ReplaceAll(r, "~1", "/")
				if strings.HasPrefix(r, "List$") {
					if listDef, ok := defs[r]; ok {
						if listDef.DataType == "map" {
							scan(*listDef.Keys)
							scan(*listDef.Values)
						} else if listDef.Items != nil {
							scan(*listDef.Items)
						}
					}
				} else if _, exists := defs[r]; exists {
					depsSet[r] = true
				}
			}
			for _, alt := range v.AnyOf {
				scan(alt)
			}
			for _, field := range v.Fields {
				scan(field)
			}
			if v.Items != nil {
				scan(*v.Items)
			}
			if v.Keys != nil {
				scan(*v.Keys)
			}
			if v.Values != nil {
				scan(*v.Values)
			}
		case parser.PlutusField:
			if v.Ref != "" {
				r := strings.TrimPrefix(v.Ref, "#/definitions/")
				r = strings.ReplaceAll(r, "~1", "/")
				if strings.HasPrefix(r, "List$") {
					if listDef, ok := defs[r]; ok {
						if listDef.Items != nil {
							scan(*listDef.Items)
						}
					}
				} else if _, exists := defs[r]; exists {
					depsSet[r] = true
				}
			}
			if v.Items != nil {
				scan(*v.Items)
			}
		}
	}
	scan(def)
	deps := []string{}
	for dep := range depsSet {
		deps = append(deps, dep)
	}
	memo[refName] = deps
	return deps
}
