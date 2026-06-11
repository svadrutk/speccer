package main

import (
    "fmt"
    "os"
    "speccer/ast"
)

func main() {
    files := []string{
        "/Users/svadrut/Documents/track-id-app/track-id-api/specs/001-i-want-to/data-model.md",
        "/Users/svadrut/Documents/songranker-app/songranker-frontend/BACKEND_SPEC.md",
        "/Users/svadrut/Documents/track-id-app/track-id-api/specs/002-i-would-like/data-model.md",
    }

    for _, f := range files {
        content, err := os.ReadFile(f)
        if err != nil {
            fmt.Printf("ERROR reading %s: %v\n\n", f, err)
            continue
        }
        fmt.Printf("=== %s ===\n", f)
        chunks, err := ast.ParseSpec(content)
        if err != nil {
            fmt.Printf("  Parse error: %v\n\n", err)
            continue
        }
        for i, c := range chunks {
            preview := c.Content
            if len(preview) > 120 {
                preview = preview[:120] + "..."
            }
            fmt.Printf("  Chunk %d: heading=%q domain=%s\n", i, c.Heading, c.Domain)
            fmt.Printf("    content preview: %q\n\n", preview)
        }
        fmt.Println()
    }
}
