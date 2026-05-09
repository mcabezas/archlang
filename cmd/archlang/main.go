package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/mcabezas/archlang/internal/generator"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: archlang generate <source-dir> [--out <output-dir>] [--package <name>]")
		os.Exit(1)
	}

	if os.Args[1] != "generate" {
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", os.Args[1])
		os.Exit(1)
	}

	if len(os.Args) < 3 {
		fmt.Fprintln(os.Stderr, "usage: archlang generate <source-dir> [--out <output-dir>] [--package <name>]")
		os.Exit(1)
	}

	sourceDir := os.Args[2]
	outputDir := "."
	packageName := "architecture"

	for i := 3; i < len(os.Args)-1; i++ {
		switch os.Args[i] {
		case "--out":
			outputDir = os.Args[i+1]
			i++
		case "--package":
			packageName = os.Args[i+1]
			i++
		}
	}

	code, err := generator.Generate(sourceDir, packageName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %s\n", err)
		os.Exit(1)
	}

	if err := os.MkdirAll(outputDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "error creating output directory: %s\n", err)
		os.Exit(1)
	}

	outPath := filepath.Join(outputDir, "architecture_gen.go")
	if err := os.WriteFile(outPath, code, 0644); err != nil {
		fmt.Fprintf(os.Stderr, "error writing file: %s\n", err)
		os.Exit(1)
	}

	fmt.Printf("generated %s\n", outPath)
}
