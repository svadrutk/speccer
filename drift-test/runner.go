package main

import (
	"fmt"
	"os"
	"path/filepath"
	"speccer/ast"
	"speccer/diff"
	"speccer/pyexec"
)

func main() {
	if len(os.Args) < 3 {
		fmt.Println("usage: go run runner.go <spec.md> <workspace> <source_glob...>")
		os.Exit(1)
	}

	specBytes, err := os.ReadFile(os.Args[1])
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR reading spec: %v\n", err)
		os.Exit(1)
	}

	workspaceRoot := os.Args[2]

	chunks, err := ast.ParseSpec(specBytes)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR parsing spec: %v\n", err)
		os.Exit(1)
	}

	var pyFiles []string
	for _, glob := range os.Args[3:] {
		matches, _ := filepath.Glob(filepath.Join(workspaceRoot, glob))
		pyFiles = append(pyFiles, matches...)
	}

	ir, err := pyexec.RunParser(workspaceRoot, pyFiles)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR running parser: %v\n", err)
		os.Exit(1)
	}

	ran := false
	for _, chunk := range chunks {
		if chunk.Domain != "data_schema" {
			continue
		}
		ran = true
		fmt.Printf("--- %s ---\n", chunk.Heading)
		findings := diff.AnalyzeDrift(chunk, ir)
		if len(findings) == 0 {
			fmt.Println("  ✓ No drift detected")
		} else {
			for _, f := range findings {
				fmt.Printf("  [%s] %s\n", f.Severity, f.Message)
			}
		}
	}

	if !ran {
		fmt.Println("  ⚠ No data_schema sections found in spec")
	}
}
