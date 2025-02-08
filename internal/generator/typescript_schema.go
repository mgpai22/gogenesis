package generator

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/mgpai22/gogenesis/internal/parser"
)

// GenerateTSSchema generates TypeScript schema lines for a given definition.
// It builds a detailed schema expression (e.g. for enums, maps, lists, objects) based on the structure of def.
func GenerateTSSchema(refName string, def parser.PlutusDefinition, tsTypeName string, chosenNames map[string]string, defs map[string]parser.PlutusDefinition) []string {
	lines := []string{
		"// -----------------------------",
		fmt.Sprintf("// Schema for %s", refName),
	}
	schemaExpr := generateSchemaExpression(def, defs, chosenNames)
	// Sanitize type name (remove spaces)
	sanitizedTypeName := strings.ReplaceAll(tsTypeName, " ", "_")
	if strings.Contains(schemaExpr, sanitizedTypeName+"Schema") {
		lines = append(lines, fmt.Sprintf("export let %sSchema: any;", sanitizedTypeName))
		lines = append(lines, fmt.Sprintf("%sSchema = %s;", sanitizedTypeName, schemaExpr))
	} else {
		lines = append(lines, fmt.Sprintf("export const %sSchema = %s;", sanitizedTypeName, schemaExpr))
	}
	lines = append(lines, "",
		fmt.Sprintf("export type %s = Data.Static<typeof %sSchema>;", sanitizedTypeName, sanitizedTypeName),
		fmt.Sprintf("export const %s = %sSchema as unknown as %s;", sanitizedTypeName, sanitizedTypeName, sanitizedTypeName),
		"",
	)
	return lines
}

//
// --- Schema Expression Generators ---
//

// generateSchemaExpression converts the given definition into a Data.* expression.
// It special-cases wrapped redeemers and uses alternate generators for enums, lists, maps, etc.
func generateSchemaExpression(def parser.PlutusDefinition, defs map[string]parser.PlutusDefinition, chosenNames map[string]string) string {
	// Special case for wrapped redeemer.
	if def.Description == "A redeemer wrapped in an extra constructor to make multi-validator detection possible on-chain." {
		if len(def.AnyOf) > 0 && len(def.AnyOf[0].Fields) > 0 {
			wrappedType := generateRefExpressionForField(def.AnyOf[0].Fields[0], defs, chosenNames)
			return fmt.Sprintf("Data.Enum([\n  Data.Object({ Dummy: Data.Tuple([]) }),\n  Data.Object({ Wrapped: Data.Tuple([%s]) })\n])", wrappedType)
		}
	}
	if len(def.AnyOf) > 0 {
		if len(def.AnyOf) > 1 {
			return generateEnumExpression(def.AnyOf, defs, chosenNames)
		} else if len(def.AnyOf) == 1 {
			// For single-constructor records, flatten if the constructor title matches the parent's title.
			return generateSingleConstructor(def.AnyOf[0], def.Title, defs, chosenNames)
		}
	}
	switch def.DataType {
	case "bytes":
		return "Data.Bytes()"
	case "integer":
		return "Data.Integer()"
	case "map":
		return generateMapExpression(def, defs, chosenNames)
	case "list":
		return generateListExpressionFromDef(def, defs, chosenNames)
	default:
		return "Data.Any()"
	}
}

// generateEnumExpression returns a Data.Enum expression given multiple alternative definitions.
func generateEnumExpression(alts []parser.PlutusDefinition, defs map[string]parser.PlutusDefinition, chosenNames map[string]string) string {
	parts := []string{}
	for _, alt := range alts {
		parts = append(parts, generateConstructorInEnum(alt, defs, chosenNames))
	}
	return fmt.Sprintf("Data.Enum([%s])", strings.Join(parts, ", "))
}

// generateSingleConstructor returns a schema expression for a single constructor.
// If the constructor's title matches the parent's title, its fields are flattened.
func generateSingleConstructor(cons parser.PlutusDefinition, parentTitle string, defs map[string]parser.PlutusDefinition, chosenNames map[string]string) string {
	if len(cons.Fields) == 0 {
		return "Data.Object({}, { hasConstr: true })"
	}
	if parentTitle != "" && cons.Title == parentTitle {
		fieldExprs := []string{}
		for _, f := range cons.Fields {
			fieldExprs = append(fieldExprs, fmt.Sprintf("%s: %s", f.Title, generateRefExpressionForField(f, defs, chosenNames)))
		}
		return fmt.Sprintf("Data.Object({ %s })", strings.Join(fieldExprs, ", "))
	}
	return generateConstructorAsObject(cons, defs, chosenNames)
}

// generateConstructorInEnum returns a constructor wrapped as an object for use in an enum.
func generateConstructorInEnum(cons parser.PlutusDefinition, defs map[string]parser.PlutusDefinition, chosenNames map[string]string) string {
	if len(cons.Fields) == 0 {
		nm := cons.Title
		if nm == "" {
			nm = "Unknown"
		}
		return fmt.Sprintf("Data.Literal(\"%s\")", nm)
	}
	tupleItems := []string{}
	for _, f := range cons.Fields {
		tupleItems = append(tupleItems, generateRefExpressionForField(f, defs, chosenNames))
	}
	return fmt.Sprintf("Data.Object({ %s: Data.Tuple([%s]) })", cons.Title, strings.Join(tupleItems, ", "))
}

// generateConstructorAsObject returns a Data.Object or Data.Tuple expression depending on the fields.
func generateConstructorAsObject(cons parser.PlutusDefinition, defs map[string]parser.PlutusDefinition, chosenNames map[string]string) string {
	constructorTitle := cons.Title
	if constructorTitle == "" {
		constructorTitle = "Unknown"
	}
	if len(cons.Fields) == 0 {
		return "Data.Any()"
	}
	allFieldsHaveTitles := true
	for _, f := range cons.Fields {
		if f.Title == "" {
			allFieldsHaveTitles = false
			break
		}
	}
	if allFieldsHaveTitles {
		tupleItems := []string{}
		for _, f := range cons.Fields {
			tupleItems = append(tupleItems, generateRefExpressionForField(f, defs, chosenNames))
		}
		return fmt.Sprintf("Data.Object({ %s: Data.Tuple([%s]) })", constructorTitle, strings.Join(tupleItems, ", "))
	}
	tupleItems := []string{}
	for _, f := range cons.Fields {
		tupleItems = append(tupleItems, generateRefExpressionForField(f, defs, chosenNames))
	}
	return fmt.Sprintf("Data.Tuple([%s], { hasConstr: true })", strings.Join(tupleItems, ", "))
}

// generateMapExpression builds a Data.Map expression using the key and value schemas.
func generateMapExpression(def parser.PlutusDefinition, defs map[string]parser.PlutusDefinition, chosenNames map[string]string) string {
	if def.Keys == nil || def.Values == nil {
		return "Data.Map(Data.Any(), Data.Any())"
	}
	keysExpr := generateRefExpressionForDef(*def.Keys, defs, chosenNames)
	valuesExpr := generateRefExpressionForDef(*def.Values, defs, chosenNames)
	return fmt.Sprintf("Data.Map(%s, %s)", keysExpr, valuesExpr)
}

// generateListExpressionFromDef builds a Data.Array expression from a list definition.
func generateListExpressionFromDef(def parser.PlutusDefinition, defs map[string]parser.PlutusDefinition, chosenNames map[string]string) string {
	if def.Items == nil {
		return "Data.Array(Data.Any())"
	}
	itemExpr := generateRefExpressionForDef(*def.Items, defs, chosenNames)
	opts := []string{}
	if def.MinItems != 0 {
		opts = append(opts, fmt.Sprintf("minItems: %d", def.MinItems))
	}
	if def.MaxItems != 0 {
		opts = append(opts, fmt.Sprintf("maxItems: %d", def.MaxItems))
	}
	if def.UniqueItems {
		opts = append(opts, "uniqueItems: true")
	}
	if len(opts) > 0 {
		return fmt.Sprintf("Data.Array(%s, { %s })", itemExpr, strings.Join(opts, ", "))
	}
	return fmt.Sprintf("Data.Array(%s)", itemExpr)
}

// generateListExpression is a simple wrapper calling generateListExpressionFromDef.
func generateListExpression(def parser.PlutusDefinition, defs map[string]parser.PlutusDefinition, chosenNames map[string]string) string {
	return generateListExpressionFromDef(def, defs, chosenNames)
}

//
// --- Reference Expression Generators ---
//

// generateRefExpressionForDef returns the schema expression for an inline definition.
// It resolves $ref references and falls back to the definition's own type.
func generateRefExpressionForDef(def parser.PlutusDefinition, defs map[string]parser.PlutusDefinition, chosenNames map[string]string) string {
	if def.Ref != "" {
		normalized := normalizeRef(def.Ref)
		if strings.HasPrefix(normalized, "List$") {
			if expr, ok := resolveListReference(normalized, defs, chosenNames); ok {
				return expr
			}
			tsType := makeTypeName(normalized)
			return tsType + "Schema"
		}
		if tsType, found := chosenNames[normalized]; found {
			return tsType + "Schema"
		}
		tsType := makeTypeName(normalized)
		return tsType + "Schema"
	}
	if def.DataType != "" || len(def.AnyOf) > 0 {
		return generateSchemaExpression(def, defs, chosenNames)
	}
	return "Data.Any()"
}

// generateRefExpressionForField returns the schema expression for a field.
func generateRefExpressionForField(field parser.PlutusField, defs map[string]parser.PlutusDefinition, chosenNames map[string]string) string {
	if field.Ref != "" {
		normalized := normalizeRef(field.Ref)
		if strings.HasPrefix(normalized, "List$") {
			if expr, ok := resolveListReference(normalized, defs, chosenNames); ok {
				return expr
			}
			tsType := makeTypeName(normalized)
			return tsType + "Schema"
		}
		if tsType, found := chosenNames[normalized]; found {
			return tsType + "Schema"
		}
		tsType := makeTypeName(normalized)
		return tsType + "Schema"
	}
	if field.Items != nil {
		return generateListExpression(*field.Items, defs, chosenNames)
	}
	return "Data.Any()"
}

//
// --- Helper Functions ---
//

// normalizeRef removes the "#/definitions/" prefix and replaces "~1" with "/".
func normalizeRef(ref string) string {
	ref = strings.TrimPrefix(ref, "#/definitions/")
	return strings.ReplaceAll(ref, "~1", "/")
}

// resolveListReference handles List$ prefixed references.
// It returns a Data.Array or Data.Map expression based on the referenced definition.
func resolveListReference(normalized string, defs map[string]parser.PlutusDefinition, chosenNames map[string]string) (string, bool) {
	listDef, ok := defs[normalized]
	if !ok {
		return "", false
	}
	if listDef.DataType == "map" && listDef.Keys != nil && listDef.Values != nil {
		keysExpr := generateRefExpressionForDef(*listDef.Keys, defs, chosenNames)
		valuesExpr := generateRefExpressionForDef(*listDef.Values, defs, chosenNames)
		return fmt.Sprintf("Data.Map(%s, %s)", keysExpr, valuesExpr), true
	}
	if listDef.Items != nil {
		itemExpr := generateRefExpressionForDef(*listDef.Items, defs, chosenNames)
		return fmt.Sprintf("Data.Array(%s)", itemExpr), true
	}
	return "", false
}

// makeTypeName cleans a raw string to produce a valid TypeScript type name.
func makeTypeName(raw string) string {
	raw = strings.ReplaceAll(raw, " ", "_")
	raw = strings.ReplaceAll(raw, "$", "_")
	re := regexp.MustCompile(`[^\w]+`)
	raw = re.ReplaceAllString(raw, "_")
	if strings.Contains(raw, "~1") {
		parts := strings.Split(raw, "~1")
		raw = parts[len(parts)-1]
	}
	if raw == "" {
		return raw
	}
	first := []rune(raw)[0]
	return strings.ToUpper(string(first)) + raw[1:]
}
