package parser

import (
	"encoding/json"
	"fmt"
	"os"
)

type PlutusDefinition struct {
	Title       string             `json:"title"`
	Description string             `json:"description"`
	DataType    string             `json:"dataType"`
	Fields      []PlutusField      `json:"fields"`
	AnyOf       []PlutusDefinition `json:"anyOf"`
	Ref         string             `json:"$ref"`
	Items       *PlutusDefinition  `json:"items"`
	Keys        *PlutusDefinition  `json:"keys"`
	Values      *PlutusDefinition  `json:"values"`
	MinItems    int                `json:"minItems"`
	MaxItems    int                `json:"maxItems"`
	UniqueItems bool               `json:"uniqueItems"`
	HasConstr   bool               `json:"hasConstr"`
}

type PlutusField struct {
	Title string            `json:"title"`
	Type  string            `json:"type"`
	Ref   string            `json:"$ref"`
	Items *PlutusDefinition `json:"items"`
}

type PlutusSchema struct {
	Definitions map[string]PlutusDefinition `json:"definitions"`
}

func ParsePlutusJSON(filePath string) (*PlutusSchema, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	var schema PlutusSchema
	if err := json.Unmarshal(data, &schema); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	return &schema, nil
}
