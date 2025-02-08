package main

import (
	"flag"
	"fmt"
	"log"

	"github.com/mgpai22/gogenesis/internal/generator"
	"github.com/mgpai22/gogenesis/internal/generator/golang"
	"github.com/mgpai22/gogenesis/internal/generator/typescript"
	"github.com/mgpai22/gogenesis/internal/parser"
)

func main() {
	// CLI flags
	jsonPath := flag.String("json", "", "Path to plutus.json")
	outPath := flag.String("out", "./generated", "Output directory for generated files")
	lang := flag.String("lang", "typescript", "Target language (typescript, golang)")
	flag.Parse()

	if *jsonPath == "" {
		log.Fatal("Error: -json flag is required")
	}

	// Parse plutus.json
	plutusData, err := parser.ParsePlutusJSON(*jsonPath)
	if err != nil {
		log.Fatalf("Failed to parse plutus.json: %v", err)
	}

	var codeGen generator.CodeGenerator
	switch *lang {
	case "golang":
		codeGen = golang.NewGoGenerator()
	default:
		codeGen = typescript.NewTypeScriptGenerator()
	}

	opts := generator.GeneratorOptions{
		ReservedNames: nil, // uses defaults if nil
		Language:      *lang,
	}
	g := generator.NewGeneratorWithOptions(*outPath, opts, codeGen)
	if err := g.Generate(plutusData); err != nil {
		log.Fatalf("Code generation failed: %v", err)
	}

	fmt.Println("Code generation completed successfully!")
}
